package backend

import (
	"crypto/ecdsa"
	"github.com/bcos-one/BCOS/accounts"
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/consensus"
	"github.com/bcos-one/BCOS/consensus/dbft"
	"github.com/bcos-one/BCOS/consensus/dbft/dpos"
	"github.com/bcos-one/BCOS/consensus/dbft/pbft"
	"github.com/bcos-one/BCOS/core"
	"github.com/bcos-one/BCOS/core/types"
	"github.com/bcos-one/BCOS/crypto"
	"github.com/bcos-one/BCOS/ethdb"
	"github.com/bcos-one/BCOS/event"
	"github.com/bcos-one/BCOS/log"
	"github.com/bcos-one/BCOS/p2p"
	"github.com/bcos-one/BCOS/params"
	"github.com/hashicorp/golang-lru"
	"sync"
)

const (
	inmemorySignatures = 4096 // Number of recent block signatures to keep in memory

)

type SignerFn func(accounts.Account, []byte) ([]byte, error)

func New(config *params.DbftConfig, privateKey *ecdsa.PrivateKey, db ethdb.Database) consensus.Dbft {
	signatures, _ := lru.NewARC(inmemorySignatures)
	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	backend := &backend{
		config:     config,
		privateKey: privateKey,
		address:    address,
		db:         db,
		dpos:       dpos.New(config, db, signatures, ecrecover),
		signatures: signatures,
	}
	backend.pbft = pbft.New(backend, address)

	return backend
}

type backend struct {
	config *params.DbftConfig

	privateKey *ecdsa.PrivateKey
	address    common.Address
	db         ethdb.Database
	dpos       dbft.DPOS
	pbft       dbft.PBFT

	// the channels for pbft engine notifications
	commitCh          chan *types.Block
	proposedBlockHash common.Hash
	sealMu            sync.Mutex

	// blockchain
	chain        consensus.ChainReader
	currentBlock func() *types.Block
	hasBadBlock  func(hash common.Hash) bool

	signatures *lru.ARCCache // Signatures of recent blocks to speed up mining

	recentMessages *lru.ARCCache // the cache of peer's messages
	knownMessages  *lru.ARCCache // the cache of self messages
}

// Sign implements istanbul.Backend.Sign
func (b *backend) Sign(data []byte) ([]byte, error) {
	hashData := crypto.Keccak256([]byte(data))
	return crypto.Sign(hashData, b.privateKey)
}

func (b *backend) LastProposal() (dbft.Proposal, common.Address) {
	block := b.currentBlock()

	var proposer common.Address
	if block.Number().Cmp(common.Big0) > 0 {
		var err error
		proposer, err = b.Author(block.Header())
		if err != nil {
			log.Error("Failed to get block proposer", "err", err)
			return nil, common.Address{}
		}
	}

	// Return header only block here since we don't need block body
	return block, proposer
}

// Verify implements dbft.Backend.Verify
func (b *backend) Verify(proposal dbft.Proposal) error {
	// Check if the proposal is a valid block
	block := &types.Block{}
	block, ok := proposal.(*types.Block)
	if !ok {
		log.Error("Invalid proposal, %v", proposal)
		return errInvalidProposal
	}

	// check bad block
	if b.hasBadBlock(block.Hash()) {
		return core.ErrBlacklistedHash
	}

	// check block body
	txnHash := types.DeriveSha(block.Transactions())
	uncleHash := types.CalcUncleHash(block.Uncles())
	if txnHash != block.Header().TxHash {
		return errMismatchTxhashes
	}
	if uncleHash != nilUncleHash {
		return errInvalidUncleHash
	}

	// verify the header of proposed block
	err := b.VerifyHeader(b.chain, block.Header(), false)
	// ignore errEmptyCommittedSeals error because we don't have the committed seals yet
	if err == nil || err == errEmptyCommittedSeals {
		return nil
	}

	return err
}

// Validators implements dbft.Backend.Validators
func (b *backend) Validators(proposal dbft.Proposal) dbft.Validators {
	block := proposal.(*types.Block)
	header := block.Header()

	number := proposal.Number().Uint64()

	parent := b.chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return nil
	}

	snap, err := b.dpos.Snapshot(b.chain, number-1, parent.Hash(), nil)
	if err != nil {
		return nil
	}

	return snap.Validators()
}

// CheckSignature implements istanbul.Backend.CheckSignature
func (b *backend) CheckSignature(data []byte, address common.Address, sig []byte) error {
	signer, err := dbft.GetSignatureAddress(data, sig)
	if err != nil {
		log.Error("Failed to get signer address", "err", err)
		return err
	}
	// Compare derived addresses
	if signer != address {
		return errInvalidSignature
	}
	return nil
}

func (b *backend) Commit(proposal dbft.Proposal, seals [][]byte) error {
	// Check if the proposal is a valid block
	block := &types.Block{}
	block, ok := proposal.(*types.Block)
	if !ok {
		log.Error("Invalid proposal, %v", proposal)
		return errInvalidProposal
	}

	h := block.Header()
	// Append seals into extra-data
	err := writeCommittedSeals(h, seals)
	if err != nil {
		return err
	}
	// update block's header
	block = block.WithSeal(h)

	log.Info("Committed", "address", b.address, "hash", proposal.Hash(), "number", proposal.Number().Uint64())
	// - if the proposed and committed blocks are the same, send the proposed hash
	//   to commit channel, which is being watched inside the engine.Seal() function.
	// - otherwise, we try to insert the block.
	// -- if success, the ChainHeadEvent event will be broadcasted, try to build
	//    the next block and the previous Seal() will be stopped.
	// -- otherwise, a error will be returned and a round change event will be fired.
	if b.proposedBlockHash == block.Hash() {
		// feed block hash to Seal() and wait the Seal() result
		b.commitCh <- block
		return nil
	}

	return nil
}

// Sign implements consensus.Handler
func (b *backend) HandleMsg(address common.Address, msg p2p.Msg) (bool, error) {
	return b.pbft.DispatchMsg(address, msg)
}

func (b *backend) SubscribeNewMsgEvent(ch chan<- consensus.PbftMsg) event.Subscription {
	return b.pbft.SubscribeNewMsgEvent(ch)
}
