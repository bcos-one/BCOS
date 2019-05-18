package dpos

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bcos-one/BCOS/accounts/abi"
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/common/math"
	"github.com/bcos-one/BCOS/consensus"
	"github.com/bcos-one/BCOS/consensus/dbft"
	"github.com/bcos-one/BCOS/core"
	"github.com/bcos-one/BCOS/core/state"
	"github.com/bcos-one/BCOS/core/types"
	"github.com/bcos-one/BCOS/core/vm"
	"github.com/bcos-one/BCOS/ethdb"
	"github.com/bcos-one/BCOS/log"
	"github.com/bcos-one/BCOS/params"
	"github.com/hashicorp/golang-lru"
	"math/big"
	"strings"
)

const (
	checkpointInterval = 1024 // Number of blocks after which to save the vote snapshot to the database

	inmemorySnapshots = 128 // Number of recent vote snapshots to keep in memory
)

var (
	// errInvalidVotingChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	errInvalidVotingChain = errors.New("invalid voting chain")
	// errUnauthorizedSigner is returned if a header is signed by a non-authorized entity.
	errUnauthorizedSigner = errors.New("unauthorized signer")

	// errRecentlySigned is returned if a header is signed by an authorized entity
	// that already signed a header recently, thus is temporarily not allowed to.
	errRecentlySigned = errors.New("recently signed")
)

var (
	signerBlockReward = big.NewInt(5e+18) // Block reward in wei for successfully mining a block first year
)

type recoverFunc func(header *types.Header, sigcache *lru.ARCCache) (common.Address, error)

type DPos struct {
	config *params.DbftConfig
	db     ethdb.Database

	sigcache *lru.ARCCache // Cache of recent block signatures to speed up ecrecover
	recents  *lru.ARCCache // Snapshots for recent block to speed up reorgs
	recover  recoverFunc
}

func New(config *params.DbftConfig, db ethdb.Database, sigcache *lru.ARCCache, recover recoverFunc) dbft.DPOS {
	recents, _ := lru.NewARC(inmemorySnapshots)

	dpos := &DPos{
		config:   config,
		db:       db,
		recents:  recents,
		sigcache: sigcache,
		recover:  recover,
	}

	return dpos
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (d *DPos) Snapshot(chain consensus.ChainReader, number uint64, hash common.Hash, parents []*types.Header) (dbft.Snapshot, error) {
	var (
		headers []*types.Header
		snap    *Snapshot
	)

	for snap == nil {
		// If an in-memory snapshot was found, use that
		if s, ok := d.recents.Get(hash); ok {
			snap = s.(*Snapshot)
			break
		}

		// If an on-disk checkpoint snapshot can be found, use that
		if number%checkpointInterval == 0 {
			if s, err := d.loadSnapshot(d.db, hash); err == nil {
				log.Trace("Loaded voting snapshot from disk", "Number", number, "Hash", hash)
				snap = s
				break
			}
		}

		if number == 0 || number%d.config.Epoch == 0 {
			checkpoint := chain.GetHeaderByNumber(number)
			if checkpoint != nil {
				hash := checkpoint.Hash()

				//extra, err := types.ExtractDbftExtra(checkpoint)
				//if err != nil {
				//	return nil, err
				//}
				var validators []common.Address

				blockchain, ok := chain.(*core.BlockChain)
				if !ok {
					return nil, errInvalidVotingChain
				}

				validators, err := getCandidates(blockchain, blockchain.Config(), checkpoint)
				if err != nil {
					return nil, err
				}

				snap = newSnapshot(d, number, hash, validators, checkpoint.Time.Uint64())
				if err := snap.store(d.db); err != nil {
					return nil, err
				}
				log.Info("Stored checkpoint snapshot to disk", "Number", number, "Hash", hash)
				break
			}
		}

		// No snapshot for this header, gather the header and move backward
		var header *types.Header
		if len(parents) > 0 {
			// If we have explicit parents, pick from there (enforced)
			header = parents[len(parents)-1]
			if header.Hash() != hash || header.Number.Uint64() != number {
				return nil, consensus.ErrUnknownAncestor
			}
			parents = parents[:len(parents)-1]
		} else {
			// No explicit parents (or no more left), reach out to the database
			header = chain.GetHeader(hash, number)
			if header == nil {
				return nil, consensus.ErrUnknownAncestor
			}
		}
		headers = append(headers, header)
		number, hash = number-1, header.ParentHash
	}

	// Previous snapshot found, apply any pending headers on top of it
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}
	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}
	d.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if snap.Number%checkpointInterval == 0 && len(headers) > 0 {
		if err = snap.store(d.db); err != nil {
			return nil, err
		}
		log.Trace("Stored voting snapshot to disk", "Number", snap.Number, "Hash", snap.Hash)
	}
	return snap, err
}

// loadSnapshot loads an existing snapshot from the database.
func (d *DPos) loadSnapshot(db ethdb.Database, hash common.Hash) (*Snapshot, error) {
	blob, err := db.Get(append([]byte("dpos-"), hash[:]...))
	if err != nil {
		return nil, err
	}
	snap := new(Snapshot)
	if err := json.Unmarshal(blob, snap); err != nil {
		return nil, err
	}
	snap.dpos = d

	return snap, nil
}

// 31536000  365 * 24 * 3600
func (d *DPos) AccumulateRewards(state *state.StateDB, header *types.Header, snap dbft.Snapshot) {
	yearCount := header.Number.Uint64() / (31536000 / d.config.BlockPeriod)
	blockReward := new(big.Int).Rsh(signerBlockReward, uint(yearCount))

	state.AddBalance(header.Coinbase, blockReward)
}

func getCandidates(chain *core.BlockChain, config *params.ChainConfig, header *types.Header) ([]common.Address, error) {
	state, err := chain.StateAt(header.Root)
	if err != nil {
		return nil, err
	}

	var (
		gp = new(core.GasPool).AddGas(math.MaxUint32)
		to = types.VoteContract

		// 06a49fce = web3.sha3("getCandidates()")
		msg = types.NewMessage(
			common.HexToAddress("0x0000000000000000000000000000000000000000"),
			&to,
			0,
			big.NewInt(0),
			header.GasLimit,
			new(big.Int),
			common.Hex2Bytes("b7ab4db5"),
			false)

		evmContext = core.NewEVMContext(msg, header, chain, nil)
		vmenv      = vm.NewEVM(evmContext, state, config, vm.Config{})
	)

	result, _, _, err := core.NewStateTransition(vmenv, msg, gp).TransitionDb()
	if err != nil {
		return nil, err
	}

	var validators []common.Address
	log.Debug("validator result", "result", fmt.Sprintf("%x", result))
	if validators, err = unpackValidator(result); err != nil {
		return nil, err
	}

	return validators, nil
}

func unpackValidator(enc []byte) ([]common.Address, error) {
	const definition = `[{"constant":true,"inputs":[],"name":"getValidators","outputs":[{"name":"","type":"address[]"}],"payable":false,"stateMutability":"view","type":"function"}]`
	abi, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		return nil, err
	}
	validators := []common.Address{}
	if err := abi.Unpack(&validators, "getValidators", enc); err != nil {
		return nil, err
	}

	return validators, nil
}
