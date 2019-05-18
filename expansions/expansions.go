package expansions

import (
	"github.com/bcos-one/BCOS/core/state"
	"github.com/bcos-one/BCOS/core/types"
	"github.com/bcos-one/BCOS/expansions/management"
	"github.com/bcos-one/BCOS/expansions/token"
	"github.com/bcos-one/BCOS/params"
)

type ExpansionsService struct {
	*params.ExpansionsConfig
}

func NewExpansions(config *params.ChainConfig) *ExpansionsService {
	return &ExpansionsService{
		config.ExpansionsConfig,
	}
}

func (self *ExpansionsService) ApplyMessage(db *state.StateDB, msg *types.Message) error {
	to := msg.To()
	if to == nil {
		return nil
	}

	switch {
	case self.TokenSupport && *to == self.TokenStorage:
		return token.ApplyTokenOp(self.TokenStorage, db, msg)

	case self.ManageSupport && *to == self.ManageStorage:
		return management.ApplyManageOp(self.ExpansionsConfig, db, msg)

	case self.IcapSupport && *to == self.IcapStorage:
		//TODO
		return nil

	default:
		return nil
	}

	return nil
}


func (self *ExpansionsService) InitGenesis(db *state.StateDB) error {

	if self.ManageSupport {
		management.SetManager(self.ManageStorage, self.Manager, db)
	}
	return nil
}