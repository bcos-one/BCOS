package dbft

import (
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/consensus"
	"github.com/bcos-one/BCOS/core/state"
	"github.com/bcos-one/BCOS/core/types"
	"math/big"
)

type Snapshot interface {
	Validators() Validators

	// Inturn returns if a signer at a given block height is in-turn or not.
	Inturn(validator common.Address, headerTime, number *big.Int) bool

	// Intrun returns next timestamp when validator can sign a block
	NextTimeSlot(validator common.Address) *big.Int
}

type DPOS interface {
	// snapshot retrieves the dpos snapshot at a given point in time.
	Snapshot(chain consensus.ChainReader, number uint64, hash common.Hash, parents []*types.Header) (Snapshot, error)

	// AccumulateRewards credits the coinbase of the given block with the mining reward.
	AccumulateRewards(state *state.StateDB, header *types.Header, snap Snapshot)
}
