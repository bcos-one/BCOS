package wpoa

import (
	"strings"
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/consensus"
	"github.com/bcos-one/BCOS/core/state"
	"github.com/bcos-one/BCOS/core/types"
	"github.com/bcos-one/BCOS/log"
)

const (
	 // bcos custom transaction data
	 // wtx:version:category:data
	wtxPrefix      = "WTX"
	wtxVersion     = "1"

	wtxCategoryAddSigner       = "AddSigner"
	wtxCategoryRemoveSigner    = "RemoveSigner"
	wtxCategoryAddManager      = "AddManager"
	wtxCategoryRemoveManager   = "RemoveManager"
)

func (w *WPoa) processCustomTx(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction) (*types.WPoaExtra, error) {
	number := header.Number.Uint64()
	change := false
	// Assemble the voting snapshot to check which votes make sense
	snap, err := w.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return nil, err
	}

	headExtra := &types.WPoaExtra{
		Managers: 		 []common.Address{},
		Signers:  		 []common.Address{},
		DiscardManagers: []common.Address{},
		DiscardSigners:  []common.Address{},
	}

	for _, tx := range txs {
		txSender, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
		if err != nil {
			return nil, nil
		}

		to := tx.To()
		if !snap.isManager(txSender) {
			if to == nil {
				// 发布合约的类型
				log.Error("TxSender is not manager,so that it can't create contract", "hash", tx.Hash())
				return nil, errManagerAddress
			}
			continue
		}

		txData := string(tx.Data())
		if len(txData) < len(wtxPrefix) || !strings.HasPrefix(txData, wtxPrefix+":") {
			continue
		}

		txDataInfo := strings.Split(txData, ":")
		//wtx:version:category:data
		if len(txDataInfo) != 4 || txDataInfo[1] != wtxVersion  {
			continue
		}

		change = true

		address := common.HexToAddress(txDataInfo[3])
		switch txDataInfo[2] {
		case wtxCategoryAddSigner:
			if !snap.isSigner(address) {
				//snap.Signers[address] = struct {}{}
				headExtra.Signers = append(headExtra.Signers, address)
			}
		case wtxCategoryRemoveSigner:
			if snap.isSigner(address) {
				//delete(snap.Signers, address)
				headExtra.DiscardSigners = append(headExtra.DiscardSigners, address)
			}
		case wtxCategoryAddManager:
			if !snap.isManager(address) {
				snap.Managers[address] = struct {}{}
				headExtra.Managers = append(headExtra.Managers, address)
			}
		case wtxCategoryRemoveManager:
			if snap.isManager(address) {
				delete(snap.Managers, address)
				headExtra.DiscardManagers = append(headExtra.DiscardManagers, address)
			}
		default:
			continue
		}
	}

	if (change){
		return headExtra, nil
	}

	return nil, nil
}
