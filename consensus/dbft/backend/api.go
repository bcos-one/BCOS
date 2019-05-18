package backend

import (
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/consensus"
	"github.com/bcos-one/BCOS/core/types"
	"github.com/bcos-one/BCOS/rpc"
)

type API struct {
	chain    consensus.ChainReader
	dbft     *backend
}

// GetValidators retrieves the list of authorized validators at the specified block.
func (api *API) GetValidators(number *rpc.BlockNumber) ([]common.Address, error) {
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}

	dpos := api.dbft.dpos
	snap, err := dpos.Snapshot(api.chain, header.Number.Uint64(),  header.Hash(), nil)
	if err != nil {
		return nil, err
	}

	return snap.Validators(), nil
}
