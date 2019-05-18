// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package wpoa

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/hashicorp/golang-lru"
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/core/types"
	"github.com/bcos-one/BCOS/ethdb"
	"github.com/bcos-one/BCOS/params"
)


// Snapshot is the state of the authorization voting at a given point in time.
type Snapshot struct {
	config   *params.WPoaConfig // Consensus engine parameters to fine tune behavior
	sigcache *lru.ARCCache        // Cache of recent block signatures to speed up ecrecover

	Number   uint64                      `json:"number"`  // Block number where the snapshot was created
	Hash     common.Hash                 `json:"hash"`    // Block hash where the snapshot was created
	Managers map[common.Address]struct{} `json:"managers"` // set of authorized manager at this moment
	Signers  map[common.Address]struct{} `json:"signers"` // Set of authorized signers at this moment
	Recents  map[uint64]common.Address   `json:"recents"` // Set of recent signers for spam protections
}

// signers implements the sort interface to allow sorting a list of addresses
type signers []common.Address

func (s signers) Len() int           { return len(s) }
func (s signers) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s signers) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// newSnapshot creates a new snapshot with the specified startup parameters. This
// method does not initialize the set of recent signers, so only ever use if for
// the genesis block.
func newSnapshot(config *params.WPoaConfig, sigcache *lru.ARCCache, number uint64, hash common.Hash, signers, managers []common.Address) *Snapshot {
	snap := &Snapshot{
		config:   config,
		sigcache: sigcache,
		Number:   number,
		Hash:     hash,
		Managers: make(map[common.Address]struct{}),
		Signers:  make(map[common.Address]struct{}),
		Recents:  make(map[uint64]common.Address),
	}
	for _, signer := range signers {
		snap.Signers[signer] = struct{}{}
	}
	for _, manager := range managers {
		snap.Managers[manager] = struct{}{}
	}
	return snap
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(config *params.WPoaConfig, sigcache *lru.ARCCache, db ethdb.Database, hash common.Hash) (*Snapshot, error) {
	blob, err := db.Get(append([]byte("wpoa-"), hash[:]...))
	if err != nil {
		return nil, err
	}
	snap := new(Snapshot)
	if err := json.Unmarshal(blob, snap); err != nil {
		return nil, err
	}
	snap.config = config
	snap.sigcache = sigcache

	return snap, nil
}

// store inserts the snapshot into the database.
func (s *Snapshot) store(db ethdb.Database) error {
	blob, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Put(append([]byte("wpoa-"), s.Hash[:]...), blob)
}

// copy creates a deep copy of the snapshot, though not the individual votes.
func (s *Snapshot) copy() *Snapshot {
	cpy := &Snapshot{
		config:   s.config,
		sigcache: s.sigcache,
		Number:   s.Number,
		Hash:     s.Hash,
		Managers:  make(map[common.Address]struct{}),
		Signers:   make(map[common.Address]struct{}),
		Recents:   make(map[uint64]common.Address),
	}
	for signer := range s.Signers {
		cpy.Signers[signer] = struct{}{}
	}
	for manager := range s.Managers {
		cpy.Managers[manager] = struct{}{}
	}
	for block, signer := range s.Recents {
		cpy.Recents[block] = signer
	}

	return cpy
}

// validVote returns whether it makes sense to cast the specified vote in the
// given snapshot context (e.g. don't try to add an already authorized signer).
func (s *Snapshot) validVote(address common.Address, authorize bool) bool {
	_, signer := s.Signers[address]
	return (signer && !authorize) || (!signer && authorize)
}


// apply creates a new authorization snapshot by applying the given headers to
// the original one.
func (s *Snapshot) apply(headers []*types.Header) (*Snapshot, error) {
	// Allow passing in no headers for cleaner code
	if len(headers) == 0 {
		return s, nil
	}
	// Sanity check that the headers can be applied
	for i := 0; i < len(headers)-1; i++ {
		if headers[i+1].Number.Uint64() != headers[i].Number.Uint64()+1 {
			return nil, errInvalidVotingChain
		}
	}
	if headers[0].Number.Uint64() != s.Number+1 {
		return nil, errInvalidVotingChain
	}
	// Iterate through the headers and create a new snapshot
	snap := s.copy()

	for _, header := range headers {
		// Remove any votes on checkpoint blocks
		number := header.Number.Uint64()

		// Delete the oldest signer from the recent list to allow it signing again
		if limit := uint64(len(snap.Signers)/2 + 1); number >= limit {
			delete(snap.Recents, number-limit)
		}
		// Resolve the authorization key and check against signers
		signer, err := ecrecover(header, s.sigcache)
		if err != nil {
			return nil, err
		}
		if _, ok := snap.Signers[signer]; !ok {
			return nil, errUnauthorized
		}
		for _, recent := range snap.Recents {
			if recent == signer {
				return nil, errUnauthorized
			}
		}
		snap.Recents[number] = signer


		if  bytes.Equal(header.Nonce[:], nonceNodeChange) {
			headExtra, err := types.ExtractWPoaExtra(header)
			if err != nil {
				return nil, err
			}

			for _, signer := range headExtra.Signers {
				snap.Signers[signer] = struct{}{}
			}

			if len(headExtra.Signers) > 0 {
				snap.Recents = make(map[uint64]common.Address)
			}

			for _, manager := range headExtra.Managers {
				snap.Managers[manager] = struct{}{}
			}

			for _, signer := range headExtra.DiscardSigners {
				delete(snap.Signers, signer)

				// Signer list shrunk, delete any leftover recent caches
				if limit := uint64(len(snap.Signers)/2 + 1); number >= limit {
					delete(snap.Recents, number-limit)
				}
			}

			for _, manager := range headExtra.DiscardManagers {
				delete(snap.Managers, manager)
			}
		}

	}
	snap.Number += uint64(len(headers))
	snap.Hash = headers[len(headers)-1].Hash()

	return snap, nil
}

// signers retrieves the list of authorized signers in ascending order.
func (s *Snapshot) signers() []common.Address {
	sigs := make([]common.Address, 0, len(s.Signers))
	for sig := range s.Signers {
		sigs = append(sigs, sig)
	}
	sort.Sort(signers(sigs))
	return sigs
}

// signers retrieves the list of authorized signers in ascending order.
func (s *Snapshot) managers() []common.Address {
	managers := make([]common.Address, 0, len(s.Managers))
	for manager := range s.Managers {
		managers = append(managers, manager)
	}
	sort.Sort(signers(managers))
	return managers
}

// inturn returns if a signer at a given block height is in-turn or not.
func (s *Snapshot) inturn(number uint64, signer common.Address) bool {
	signers, offset := s.signers(), 0
	for offset < len(signers) && signers[offset] != signer {
		offset++
	}
	return (number % uint64(len(signers))) == uint64(offset)
}

func (s *Snapshot) isManager(address common.Address) bool {
	_, manager := s.Managers[address]

	return manager
}

func (s *Snapshot) isSigner(address common.Address) bool {
	_, signer := s.Signers[address]

	return signer
}

// debug
func (s *Snapshot) String() string {
	res, err := json.Marshal(s)

	if err != nil {
		return err.Error()
	}

	return string(res)
}