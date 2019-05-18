package token

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/bcos-one/BCOS/accounts/abi"
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/core/state"
	"github.com/bcos-one/BCOS/core/types"
	"github.com/bcos-one/BCOS/crypto"
	"github.com/bcos-one/BCOS/log"
	"math/big"
	"strings"
)

const tokenabi = `[{"constant":true,"inputs":[{"name":"name","type":"string"},{"name":"manager","type":"address"},{"name":"beneficiary","type":"address"},{"name":"supply","type":"uint256"},{"name":"canIncrease","type":"bool"},{"name":"canburn","type":"bool"}],"name":"issue","outputs":[],"payable":false,"stateMutability":"pure","type":"function"},{"constant":true,"inputs":[{"name":"token","type":"address"},{"name":"beneficiary","type":"address"},{"name":"amount","type":"uint256"}],"name":"increase","outputs":[],"payable":false,"stateMutability":"pure","type":"function"},{"constant":true,"inputs":[{"name":"token","type":"address"},{"name":"amount","type":"uint256"}],"name":"burn","outputs":[],"payable":false,"stateMutability":"pure","type":"function"}]`
const maxTokenNameLen = 32

var (
	// function issue(string name, address manager, address beneficiary, uint256 supply, bool canIncrease, bool canburn)
	// web3.sha3("issue(string,address,address,uint256,bool,bool)") = 0x96184f832f1f69d4ce015f4f8db59916c6d21390fd91f3438f87456d8a6420ee
	issueSig, _ = hex.DecodeString("96184f83") //issule

	// function increase(address token, address beneficiary, uint256 amount)
	// web3.sha3("increase(address,address,uint256)") = 0x968071cec2c3fe0eefe02a187a131a794c61aad72a0367ebcd1d00574d4c53e8
	increaseSig, _ = hex.DecodeString("968071ce") // increase

	//function burn(address token, uint256 amount)
	// web3.sha3("burn(address,uint256)") = 0x9dc29fac0ba6d4fc521c69c2b0c636d612e3343bc39ed934429b8876b0d12cba
	burnSig, _ = hex.DecodeString("9dc29fac") //burn
)

var (
	errInvalidInput     = errors.New("invalid input for token operation")
	errInvalidSig       = errors.New("invalid token operation signature")
	errInvalidTokenName = errors.New("invalid token name")
	errUnauthorize      = errors.New("unauthroize")
	errInsufficient     = errors.New("insufficient funds to burn")
	errBadBool          = errors.New("improperly encoded boolean value")
)

func ApplyTokenOp(storage common.Address, db *state.StateDB, msg *types.Message) error {
	input := msg.Data()
	from := msg.From()

	if len(input) < 4 {
		return errInvalidInput
	}

	sig := input[:4]
	switch {
	case bytes.Equal(sig, issueSig):
		return issue(storage, from, msg.Nonce(), db, input[4:])
	case bytes.Equal(sig, increaseSig):
		return increase(storage, from, db, input[4:])
	case bytes.Equal(sig, burnSig):
		return burn(storage, from, db, input[4:])
	default:
		return errInvalidSig
	}

	return nil
}

func issue(storage common.Address, from common.Address, nonce uint64, db *state.StateDB, input []byte) error {
	var (
		name        string
		manager     common.Address
		beneficiary common.Address
		supply      *big.Int
		canBurn     bool
		canIncrease bool
	)
	decoder, _ := abi.JSON(strings.NewReader(tokenabi))

	if err := decoder.UnpackInput(&[]interface{}{&name, &manager, &beneficiary, &supply, &canIncrease, &canBurn}, "issue", input); err != nil {
		return errInvalidInput
	}

	if len(name) > maxTokenNameLen {
		return errInvalidTokenName
	}

	tokenid := crypto.CreateAddress(from, nonce)
	log.Info("issue token", "tokeId", tokenid.String())

	tokenObj := NewTokenObject(storage, tokenid, db)
	tokenObj.setName(name)
	tokenObj.setManager(manager)
	tokenObj.setSupply(supply)
	tokenObj.setIncreaseFlag(canIncrease)
	tokenObj.setBurnFlag(canBurn)
	tokenObj.setExistsFlag(true)

	db.AddTokenBalance(beneficiary, tokenid, supply)
	return nil
}

func increase(storage common.Address, from common.Address, db *state.StateDB, input []byte) error {
	var (
		id          common.Address
		beneficiary common.Address
		value       *big.Int
	)
	decoder, _ := abi.JSON(strings.NewReader(tokenabi))

	if err := decoder.UnpackInput(&[]interface{}{&id, &beneficiary, &value}, "increase", input); err != nil {
		return err
	}

	tokenObj := NewTokenObject(storage, id, db)

	manager := tokenObj.Manager()
	if manager != from || !tokenObj.IsExists() || !tokenObj.CanIncrease() {
		return errUnauthorize
	}

	db.AddTokenBalance(beneficiary, id, value)
	tokenObj.setSupply(new(big.Int).Add(tokenObj.GetSupply(), value))

	return nil
}

func burn(storage common.Address, from common.Address, db *state.StateDB, input []byte) error {
	var (
		id    common.Address
		value *big.Int
	)
	decoder, _ := abi.JSON(strings.NewReader(tokenabi))

	if err := decoder.UnpackInput(&[]interface{}{&id, &value}, "burn", input); err != nil {
		return err
	}

	tokenObj := NewTokenObject(storage, id, db)
	if !tokenObj.IsExists() || !tokenObj.CanBurn() {
		return errUnauthorize
	}

	if db.GetTokenBalance(from, id).Cmp(value) < 0 {
		return errInsufficient
	}

	db.SubTokenBalance(from, id, value)
	tokenObj.setSupply(new(big.Int).Sub(tokenObj.GetSupply(), value))

	return nil
}
