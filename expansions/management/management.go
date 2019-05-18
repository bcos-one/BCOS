package management

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/bcos-one/BCOS/accounts/abi"
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/core/state"
	"github.com/bcos-one/BCOS/core/types"
	"github.com/bcos-one/BCOS/expansions/token"
	"github.com/bcos-one/BCOS/params"
	"strings"
)

const manageAbi = `[{"constant":false,"inputs":[{"name":"tokenid","type":"address"}],"name":"setWhiteList","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"tokenid","type":"address"}],"name":"delWhiteList","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`

var (
	errBadBool      = errors.New("improperly encoded boolean value")
	errUnauthorize  = errors.New("unauthroize")
	errInvalidInput = errors.New("invalid input for management operation")
	errInvalidSig   = errors.New("invalid management operation signature")
	errInvalidToke  = errors.New("invalid token id")
)

var (
	// web3.sha3("setWhiteList(address)") = 0x39e899ee395d44f557ba5a068df16ea3b8131626e0acaa595abf2304eb1f6212
	setWlSig, _ = hex.DecodeString("39e899ee")

	// web3.sha3("delWhiteList(address)") = 0x605e5ee189a8a221d02b35a141f8fa5f96a4a4a6d8b2e3d1f5bd9a953b8dd72e
	delWlSig, _ = hex.DecodeString("605e5ee1")
)

func ApplyManageOp(config *params.ExpansionsConfig, db *state.StateDB, msg *types.Message) error {
	input := msg.Data()
	from := msg.From()

	if len(input) < 4 {
		return errInvalidInput
	}

	sig := input[:4]
	switch {
	case bytes.Equal(sig, setWlSig):
		return addWhiteList(config, from, db, input[4:])
	case bytes.Equal(sig, delWlSig):
		return delWhiteList(config, from, db, input[4:])
	default:
		return errInvalidSig
	}

	return nil
}

func addWhiteList(config *params.ExpansionsConfig, from common.Address, db *state.StateDB, input []byte) error {
	var tokenid common.Address
	decoder, _ := abi.JSON(strings.NewReader(manageAbi))

	if err := decoder.UnpackInput(&tokenid, "setWhiteList", input); err != nil {
		return errInvalidInput
	}

	tokenObj := token.NewTokenObject(config.TokenStorage, tokenid, db)
	if !tokenObj.IsExists() {
		return errInvalidToke
	}

	manageObj := NewManageObj(config.ManageStorage, from, db)
	return manageObj.SetTokenWhiteList(tokenid)
}

func delWhiteList(config *params.ExpansionsConfig, from common.Address, db *state.StateDB, input []byte) error {
	var tokenid common.Address
	decoder, _ := abi.JSON(strings.NewReader(manageAbi))

	if err := decoder.UnpackInput(&tokenid, "delWhiteList", input); err != nil {
		return errInvalidInput
	}

	tokenObj := token.NewTokenObject(config.TokenStorage, tokenid, db)
	if !tokenObj.IsExists() {
		return errInvalidToke
	}

	manageObj := NewManageObj(config.ManageStorage, from, db)
	return manageObj.DelTokenWhiteList(tokenid)
}
