package backend

import (
	"bytes"
	"errors"
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/consensus"
	"github.com/bcos-one/BCOS/consensus/dbft"
	"github.com/bcos-one/BCOS/core/state"
	"github.com/bcos-one/BCOS/core/types"
	"github.com/bcos-one/BCOS/crypto/sha3"
	"github.com/bcos-one/BCOS/log"
	"github.com/bcos-one/BCOS/rlp"
	"github.com/bcos-one/BCOS/rpc"
	"github.com/hashicorp/golang-lru"
	"math/big"
	"time"
)

var (
	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")
	// errInvalidSignature is returned when given signature is not signed by given
	// address.
	errInvalidSignature = errors.New("invalid signature")
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")
	// errUnauthorized is returned if a header is signed by a non authorized entity.
	errUnauthorized = errors.New("unauthorized")
	// errInvalidDifficulty is returned if the difficulty of a block is not 1
	errInvalidDifficulty = errors.New("invalid difficulty")
	// errInvalidMixDigest is returned if a block's mix digest is not Istanbul digest.
	errInvalidMixDigest = errors.New("invalid dbft mix digest")
	// errInvalidTimestamp is returned if the timestamp of a block is lower than the previous block's timestamp + the minimum block period.
	errInvalidTimestamp = errors.New("invalid timestamp")

	// errInvalidCommittedSeals is returned if the committed seal is not signed by any of parent validators.
	errInvalidCommittedSeals = errors.New("invalid committed seals")
	// errEmptyCommittedSeals is returned if the field of committed seals is zero.
	errEmptyCommittedSeals = errors.New("zero committed seals")
	// errMismatchTxhashes is returned if the TxHash in header is mismatch.
	errMismatchTxhashes = errors.New("mismatch transcations hashes")

	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte suffix signature missing")

	// It is not our turn to generate block
	errNotInTurn = errors.New("not in turn to generate block")

	// errInvalidProposal is returned when a prposal is malformed.
	errInvalidProposal = errors.New("invalid proposal")
)

var (
	emptyNonce = types.BlockNonce{}

	defaultDifficulty = big.NewInt(0).SetUint64(0xFFFFFFFF)
	nilUncleHash      = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.
)

// Author retrieves the Ethereum address of the account that minted the given block.
func (b *backend) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// Protocol implements consensus.Engine.Protocol
func (sb *backend) Protocol() consensus.Protocol {
	return consensus.Protocol{
		Name:     "dbft",
		Versions: []uint{64},
		Lengths:  []uint64{18},
	}
}

// VerifyHeader checks whether a header conforms to the consensus rules of a
// given engine. Verifying the seal may be done optionally here, or explicitly
// via the VerifySeal method.
func (b *backend) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return b.verifyHeader(chain, header, nil)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications (the order is that of
// the input slice).
func (b *backend) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))
	go func() {
		for i, header := range headers {
			err := b.verifyHeader(chain, header, headers[:i])

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of a given engine.
func (b *backend) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errInvalidUncleHash
	}
	return nil
}

// VerifySeal checks whether the crypto seal on a header is valid according to
// the consensus rules of the given engine.
func (b *backend) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	number := header.Number.Uint64()
	parent := chain.GetHeader(header.ParentHash, number-1)

	period := big.NewInt(0).Sub(header.Time, parent.Time)
	if period.Uint64() > b.config.BlockPeriod {
		expect := big.NewInt(0).Sub(defaultDifficulty, period)
		if header.Difficulty.Cmp(expect) != 0 {
			return errInvalidDifficulty
		}
	} else if header.Difficulty.Cmp(defaultDifficulty) != 0 {
		return errInvalidDifficulty
	}

	return b.verifySigner(chain, header, nil)
}

// verifySigner checks whether the signer is in parent's validator set
func (b *backend) verifySigner(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}

	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := b.dpos.Snapshot(chain, number-1, header.ParentHash, parents)
	if err != nil {
		return err
	}

	// resolve the authorization key and check against signers
	signer, err := ecrecover(header, b.signatures)
	if err != nil {
		return err
	}

	if !snap.Inturn(signer, header.Time, header.Number) {
		return errUnauthorized
	}

	return nil
}

// verifyHeader checks whether a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (b *backend) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	if header.Number == nil {
		return errUnknownBlock
	}

	// Don't waste time checking blocks from the future
	if header.Time.Cmp(big.NewInt(time.Now().Unix())) > 0 {
		return consensus.ErrFutureBlock
	}

	// Ensure that the mix digest is zero as we don't have fork protection currently
	if header.MixDigest != types.DbftDigest {
		return errInvalidMixDigest
	}

	// Ensure that the block doesn't contain any uncles which are meaningless in Istanbul
	if header.UncleHash != nilUncleHash {
		return errInvalidUncleHash
	}

	return b.verifyCascadingFields(chain, header, parents)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (b *backend) verifyCascadingFields(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// The genesis block is the always valid dead-end
	number := header.Number.Uint64()
	if number == 0 {
		return nil
	}
	// Ensure that the block's timestamp isn't too close to it's parent
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}
	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}
	if parent.Time.Uint64()+b.config.BlockPeriod > header.Time.Uint64() {
		return errInvalidTimestamp
	}

	// Ensure that the block's difficulty is meaningful (may not be correct at this point)
	if header.Difficulty == nil {
		return errInvalidDifficulty
	}

	period := big.NewInt(0).Sub(header.Time, parent.Time)
	if period.Uint64() > b.config.BlockPeriod {
		expect := big.NewInt(0).Sub(defaultDifficulty, period)
		if header.Difficulty.Cmp(expect) != 0 {
			return errInvalidDifficulty
		}
	} else if header.Difficulty.Cmp(defaultDifficulty) != 0 {
		return errInvalidDifficulty
	}

	if err := b.verifySigner(chain, header, parents); err != nil {
		return err
	}

	return b.verifySeal(chain, header, parents)
}

// verifyCommittedSeals checks whether every committed seal is signed by one of the parent's validators
func (b *backend) verifyCommittedSeals(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	number := header.Number.Uint64()
	// We don't need to verify committed seals in the genesis block
	if number == 0 {
		return nil
	}

	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := b.dpos.Snapshot(chain, number-1, header.ParentHash, parents)
	if err != nil {
		return err
	}

	extra, err := types.ExtractDbftExtra(header)
	if err != nil {
		return err
	}
	// The length of Committed seals should be larger than 0
	if len(extra.CommittedSeal) == 0 {
		return errEmptyCommittedSeals
	}

	validators := snap.Validators()
	// Check whether the committed seals are generated by parent's validators
	validSeal := 0
	proposalSeal := header.Hash().Bytes()
	// 1. Get committed seals from current header
	for _, seal := range extra.CommittedSeal {
		// 2. Get the original address by seal and parent block hash
		addr, err := dbft.GetSignatureAddress(proposalSeal, seal)
		if err != nil {
			log.Error("not a valid address", "err", err)
			return errInvalidSignature
		}
		// Every validator can have only one seal. If more than one seals are signed by a
		// validator, the validator cannot be found and errInvalidCommittedSeals is returned.
		if validators.IsValidator(addr) {
			validSeal += 1
		} else {
			return errInvalidCommittedSeals
		}
	}

	// The length of validSeal should be larger than number of faulty node + 1
	if validSeal <= 2*validators.F() {
		return errInvalidCommittedSeals
	}

	return nil
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements. The method accepts an optional list of parent
// headers that aren't yet part of the local blockchain to generate the snapshots
// from.
func (b *backend) verifySeal(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := b.dpos.Snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}

	// Resolve the authorization key and check against signers
	signer, err := ecrecover(header, b.signatures)
	if err != nil {
		return err
	}

	if !snap.Inturn(signer, header.Time, header.Number) {
		return errUnauthorized
	}

	return nil
}

// Prepare initializes the consensus fields of a block header according to the
// rules of a particular engine. The changes are executed inline.
func (b *backend) Prepare(chain consensus.ChainReader, header *types.Header) error {
	header.Nonce = emptyNonce
	header.MixDigest = types.DbftDigest
	header.Difficulty = defaultDifficulty

	number := header.Number.Uint64()
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	snap, err := b.dpos.Snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}
	header.Time = snap.NextTimeSlot(b.address)
	if header.Time.Uint64() > uint64(time.Now().Unix())+b.config.BlockPeriod {
		return errNotInTurn
	}

	period := big.NewInt(0).Sub(header.Time, parent.Time)
	if period.Uint64() > b.config.BlockPeriod {
		header.Difficulty = big.NewInt(0).Sub(defaultDifficulty, period)
	}

	header.Extra, err = prepareExtra(header)
	if err != nil {
		return err
	}

	return nil
}

// prepareExtra returns a extra-data of the given header and validators
func prepareExtra(header *types.Header) ([]byte, error) {
	var buf bytes.Buffer

	// compensate the lack bytes if header.Extra is not enough IstanbulExtraVanity bytes.
	if len(header.Extra) < types.DbftExtraVanity {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, types.DbftExtraVanity-len(header.Extra))...)
	}
	buf.Write(header.Extra[:types.DbftExtraVanity])

	ist := &types.DbftExtra{
		Seal:          []byte{},
		CommittedSeal: [][]byte{},
	}

	payload, err := rlp.EncodeToBytes(&ist)
	if err != nil {
		return nil, err
	}

	return append(buf.Bytes(), payload...), nil
}

// Finalize runs any post-transaction state modifications (e.g. block rewards)
// and assembles the final block.
//
// Note, the block header and state database might be updated to reflect any
// consensus rules that happen at finalization (e.g. block rewards).
func (b *backend) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
	uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {

	snap, err := b.dpos.Snapshot(chain, header.Number.Uint64()-1, header.ParentHash, nil)
	if err != nil {
		return nil, err
	}
	b.dpos.AccumulateRewards(state, header, snap)
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = nilUncleHash

	return types.NewBlock(header, txs, nil, receipts), nil
}

// Seal generates a new block for the given input block with the local miner's
// seal place on top.
func (b *backend) Seal(chain consensus.ChainReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()
	number := header.Number.Uint64()

	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	//expiredTime := big.NewInt(0).Sub(big.NewInt(time.Now().Unix()), big.NewInt(0).SetUint64(b.config.BlockPeriod))
	//if header.Time.Cmp(expiredTime) < 0 {
	//	return errExpired
	//}

	snap, err := b.dpos.Snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}

	if !snap.Inturn(b.address, header.Time, header.Number) {
		return errNotInTurn
	}

	block, err = b.updateBlock(parent, block)
	if err != nil {
		return err
	}

	delay := time.Unix(block.Header().Time.Int64(), 0).Sub(time.Now())
	select {
	case <-time.After(delay):
	case <-stop:
		return nil
	}

	err = b.propose(snap.Validators(), block)
	if err != nil {
		return err
	}

	go func() {
		b.sealMu.Lock()
		clear := func() {
			b.proposedBlockHash = common.Hash{}
			b.sealMu.Unlock()
		}
		defer clear()

		select {
		case result := <-b.commitCh:
			if result != nil && block.Hash() == result.Hash() {
				results <- result
				return
			}
		case <-stop:
			return
		}
	}()

	return nil
}

func (b *backend) propose(validators dbft.Validators, proposal dbft.Proposal) error {
	b.sealMu.Lock()
	defer b.sealMu.Unlock()
	b.proposedBlockHash = proposal.Hash()

	return b.pbft.StartConsensus(validators, proposal)
}

func (b *backend) SealHash(header *types.Header) common.Hash {
	return sigHash(header)
}

func (b *backend) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	return new(big.Int).Set(big.NewInt(1));
}

func (b *backend) Close() error {
	return nil
}

// APIs returns the RPC APIs this consensus engine provides.
func (b *backend) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "dbft",
		Version:   "1.0",
		Service:   &API{chain: chain, dbft: b},
		Public:    true,
	}}
}

// Start implements consensus.Dbft.Start
func (b *backend) Start(chain consensus.ChainReader, currentBlock func() *types.Block, hasBadBlock func(hash common.Hash) bool) error {
	b.chain = chain
	b.currentBlock = currentBlock
	b.hasBadBlock = hasBadBlock

	// clear previous data
	b.proposedBlockHash = common.Hash{}
	if b.commitCh != nil {
		close(b.commitCh)
	}
	b.commitCh = make(chan *types.Block, 1)

	return b.pbft.Start()
}

// Stop implements consensus.Dbft.Stop
func (b *backend) Stop() error {
	return b.pbft.Stop()
}

// update timestamp and signature of the block based on its number of transactions
func (b *backend) updateBlock(parent *types.Header, block *types.Block) (*types.Block, error) {
	header := block.Header()
	// sign the hash
	seal, err := b.Sign(sigHash(header).Bytes())
	if err != nil {
		return nil, err
	}

	err = writeSeal(header, seal)
	if err != nil {
		return nil, err
	}

	return block.WithSeal(header), nil
}

// sigHash returns the hash which is used as input for the delegated-proof-of-stake
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func sigHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewKeccak256()

	rlp.Encode(hasher, types.DbftFilteredHeader(header, false))
	hasher.Sum(hash[:0])
	return hash
}

// ecrecover extracts the Ethereum account address from a signed header.
func ecrecover(header *types.Header, sigcache *lru.ARCCache) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.Hash()
	if address, known := sigcache.Get(hash); known {
		return address.(common.Address), nil
	}

	// Retrieve the signature from the header extra-data
	dbftExtra, err := types.ExtractDbftExtra(header)
	if err != nil {
		return common.Address{}, err
	}

	addr, err := dbft.GetSignatureAddress(sigHash(header).Bytes(), dbftExtra.Seal)
	if err != nil {
		return addr, err
	}
	sigcache.Add(hash, addr)

	return addr, nil
}

// writeSeal writes the extra-data field of the given header with the given seals.
// suggest to rename to writeSeal.
func writeSeal(h *types.Header, seal []byte) error {
	if len(seal)%types.DbftExtraSeal != 0 {
		return errInvalidSignature
	}

	dbftExtra, err := types.ExtractDbftExtra(h)
	if err != nil {
		return err
	}

	dbftExtra.Seal = seal
	payload, err := rlp.EncodeToBytes(&dbftExtra)
	if err != nil {
		return err
	}

	h.Extra = append(h.Extra[:types.DbftExtraVanity], payload...)
	return nil
}

// writeCommittedSeals writes the extra-data field of a block header with given committed seals.
func writeCommittedSeals(h *types.Header, committedSeals [][]byte) error {
	if len(committedSeals) == 0 {
		return errInvalidCommittedSeals
	}

	for _, seal := range committedSeals {
		if len(seal) != types.DbftExtraSeal {
			return errInvalidCommittedSeals
		}
	}

	istanbulExtra, err := types.ExtractDbftExtra(h)
	if err != nil {
		return err
	}

	istanbulExtra.CommittedSeal = make([][]byte, len(committedSeals))
	copy(istanbulExtra.CommittedSeal, committedSeals)

	payload, err := rlp.EncodeToBytes(&istanbulExtra)
	if err != nil {
		return err
	}

	h.Extra = append(h.Extra[:types.DbftExtraVanity], payload...)
	return nil
}
