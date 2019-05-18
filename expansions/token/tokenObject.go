package token

import (
	"bytes"
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/core/state"
	"github.com/bcos-one/BCOS/crypto"
	"math/big"
)

const (
	nameIndex = iota
	supplyIndex
	managerIndex
	canIncreaseIndex
	canBurnIndex
	existsIndex
)

type TokenObject struct {
	storage common.Address
	db      *state.StateDB
	hash    common.Hash
}

func NewTokenObject(storage common.Address, tokenid common.Address, db *state.StateDB) *TokenObject {
	return &TokenObject{
		storage: storage,
		db:      db,
		hash:    crypto.Keccak256Hash(tokenid.Hash().Bytes()),
	}
}

func (self *TokenObject) IsExists() (bool) {
	hash := self.db.GetState(self.storage, self.hashAtIndex(existsIndex))

	exists, _ := readBool(hash)
	return exists
}

func (self *TokenObject) GetSupply() (*big.Int) {
	hash := self.db.GetState(self.storage, self.hashAtIndex(supplyIndex))

	return new(big.Int).SetBytes(hash.Bytes())
}

func (self *TokenObject) GetTokenName() string {
	hash := self.db.GetState(self.storage, self.hashAtIndex(nameIndex))


	return string(bytes.TrimLeft(hash.Bytes(), "\x00"))
}

func (self *TokenObject) CanIncrease() bool {
	hash := self.db.GetState(self.storage, self.hashAtIndex(canIncreaseIndex))

	exists, _ := readBool(hash)
	return exists
}

func (self *TokenObject) CanBurn() bool {
	hash := self.db.GetState(self.storage, self.hashAtIndex(canBurnIndex))

	exists, _ := readBool(hash)
	return exists
}

func (self *TokenObject) Manager() common.Address {
	hash := self.db.GetState(self.storage, self.hashAtIndex(managerIndex))

	return common.BytesToAddress(hash.Bytes())
}

func (self *TokenObject) hashAtIndex(index int64) common.Hash {
	  num := new(big.Int).Add(new(big.Int).SetBytes(self.hash.Bytes()), big.NewInt(index))

	  return common.BigToHash(num)
}


func (self *TokenObject) setName(name string) {
	hash := self.hashAtIndex(nameIndex)

	self.db.SetState(self.storage, hash, common.BytesToHash([]byte(name)))
}

func (self *TokenObject) setManager(manager common.Address) {
	hash := self.hashAtIndex(managerIndex)

	self.db.SetState(self.storage, hash, manager.Hash())
}

func (self *TokenObject) setSupply(value *big.Int) {
	hash := self.hashAtIndex(supplyIndex)

	self.db.SetState(self.storage, hash, common.BigToHash(value))
}

func (self *TokenObject) setIncreaseFlag(enable bool) {
	hash := self.hashAtIndex(canIncreaseIndex)

	self.db.SetState(self.storage, hash, bool2Hash(enable))
}

func (self *TokenObject) setBurnFlag(enable bool) {
	hash := self.hashAtIndex(canBurnIndex)

	self.db.SetState(self.storage, hash, bool2Hash(enable))
}

func (self *TokenObject) setExistsFlag(exists bool) {
	hash := self.hashAtIndex(existsIndex)

	self.db.SetState(self.storage, hash, bool2Hash(exists))
}

func bool2Hash(flag bool) common.Hash {
	if flag {
		return common.BytesToHash([]byte{0x01})
	}

	return common.BytesToHash([]byte{0x00})
}

func readBool(hash common.Hash) (bool, error) {
	word := hash.Bytes()
	for _, b := range word[:31] {
		if b != 0 {
			return false, errBadBool
		}
	}
	switch word[31] {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, errBadBool
	}
}