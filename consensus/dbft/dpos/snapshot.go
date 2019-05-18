package dpos

import (
	"encoding/json"
	"fmt"
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/consensus/dbft"
	"github.com/bcos-one/BCOS/core/types"
	"github.com/bcos-one/BCOS/ethdb"
	"math/big"
	"time"
)

type Snapshot struct {
	dpos          *DPos
	Number        uint64                    `json:"Number"`        // Block Number where the snapshot was created
	Hash          common.Hash               `json:"Hash"`          // Block Hash where the snapshot was created
	Validator     []common.Address          `json:"Validator"`     // Set of authorized Validator at this moment
	Recents       map[uint64]common.Address `json:"Recents"`       // Set of recent signers for spam protections
	LoopStartTime uint64                    `json:"LoopStartTime"` // Start Time of the current loop
}

func newSnapshot(dpos *DPos, number uint64, hash common.Hash, validators []common.Address, loopStartTime uint64) *Snapshot {
	snap := &Snapshot{
		dpos:          dpos,
		Number:        number,
		Hash:          hash,
		Validator:     validators,
		Recents:       make(map[uint64]common.Address),
		LoopStartTime: loopStartTime,
	}

	return snap
}

func (s *Snapshot) Validators() dbft.Validators {
	return s.Validator
}

func (s *Snapshot) Inturn(validator common.Address, headerTime *big.Int, blockNumber *big.Int) bool {
	period := s.dpos.config.BlockPeriod
	time := headerTime.Uint64()
	number := blockNumber.Uint64()

	for seen, recent := range s.Recents {
		if recent == validator {
			if limit := uint64(len(s.Validator)/2 + 1); number < limit || seen > number-limit {
				return false
			}
		}
	}

	loopIndex := int((time-s.LoopStartTime)/period) % len(s.Validator)

	if s.Validator[loopIndex] == validator {
		return true
	}

	return false
}

func (s *Snapshot) NextTimeSlot(signer common.Address) *big.Int {
	period := s.dpos.config.BlockPeriod

	loopCount := ((uint64(time.Now().Unix()) - s.LoopStartTime) / period) / uint64(len(s.Validator))

	loopindex := uint64(0)
	for index, validator := range s.Validator {
		if signer == validator {
			loopindex = uint64(index)
			break;
		}
	}

	current := s.LoopStartTime + loopCount*uint64(len(s.Validator))*period

	var nexttime uint64
	if current+loopindex*period > uint64(time.Now().Unix()) {
		nexttime = current + loopindex*period
	} else {
		nexttime = current + loopindex*period + uint64(len(s.Validator))*period
	}

	return big.NewInt(0).SetUint64(nexttime)
}

// store inserts the snapshot into the database.
func (s *Snapshot) store(db ethdb.Database) error {
	blob, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Put(append([]byte("dpos-"), s.Hash[:]...), blob)
}

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
		number := header.Number.Uint64()
		// Delete the oldest signer from the recent list to allow it signing again
		if limit := uint64(len(snap.Validator)/2 + 1); number >= limit {
			delete(snap.Recents, number-limit)
		}

		// Resolve the authorization key and check against signers
		validator, err := snap.dpos.recover(header, snap.dpos.sigcache)
		if err != nil {
			return nil, err
		}
		for _, recent := range snap.Recents {
			if recent == validator {
				return nil, errRecentlySigned
			}
		}
		snap.Recents[number] = validator
	}

	snap.Number += uint64(len(headers))
	snap.Hash = headers[len(headers)-1].Hash()

	return snap, nil
}

// copy creates a deep copy of the snapshot, though not the individual votes.
func (s *Snapshot) copy() *Snapshot {
	cpy := &Snapshot{
		dpos:          s.dpos,
		Number:        s.Number,
		Hash:          s.Hash,
		LoopStartTime: s.LoopStartTime,
		Validator:     s.Validator,
		Recents:       make(map[uint64]common.Address),
	}
	for block, signer := range s.Recents {
		cpy.Recents[block] = signer
	}

	return cpy
}

func (s *Snapshot) String() string {
	str := fmt.Sprintf("DPOS Snapshot Number: %d, Hash: %v, loopstartTime: %d, Validator: [", s.Number, s.Hash.Hex(), s.LoopStartTime)
	for _, validator := range s.Validator {
		str = str + validator.Hex() + ","
	}
	str += "]"

	return str
}
