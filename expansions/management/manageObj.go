package management

import (
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/core/state"
	"github.com/bcos-one/BCOS/crypto"
)


var (
	managerIndex   = common.BytesToHash([]byte{0x0}).Bytes()
	whitelistIndex = common.BytesToHash([]byte{0x1}).Bytes()
)

type ManageObj struct {
	storage common.Address
	from    common.Address
	db      *state.StateDB
}

func NewManageObj(storage common.Address, from common.Address, db *state.StateDB) *ManageObj {
	return &ManageObj{
		storage: storage,
		from:    from,
		db:      db,
	}
}

func SetManager(storage common.Address, manager common.Address, db *state.StateDB) error {
	obj := NewManageObj(storage, common.Address{}, db)

	obj.setManager(manager)
	return nil
}

func (self *ManageObj) SetManager(manager common.Address) error {
	if !self.IsManager(self.from) {
		return errUnauthorize
	}
	self.setManager(manager)
	return nil
}

func (self *ManageObj) setManager(manager common.Address) {
	hashM := crypto.Keccak256Hash(append(manager.Hash().Bytes(), managerIndex[:]...))

	self.db.SetState(self.storage, hashM, bool2Hash(true))
}

func (self *ManageObj) DelManager(manager common.Address) error {
	if !self.IsManager(self.from) {
		return errUnauthorize
	}
	hashM := crypto.Keccak256Hash(append(manager.Hash().Bytes(), managerIndex[:]...))

	self.db.SetState(self.storage, hashM, bool2Hash(false))
	return nil
}

func (self *ManageObj) IsManager(manager common.Address) bool {
	hashM := crypto.Keccak256Hash(append(manager.Hash().Bytes(), managerIndex[:]...))

	hash := self.db.GetState(self.storage, hashM)

	ok, _ := readBool(hash)
	return ok
}

func (self *ManageObj) SetTokenWhiteList(tokenId common.Address) error {
	if !self.IsManager(self.from) {
		return errUnauthorize
	}
	hashW := crypto.Keccak256Hash(append(tokenId.Hash().Bytes(), whitelistIndex[:]...))

	self.db.SetState(self.storage, hashW, bool2Hash(true))

	return nil
}

func (self *ManageObj) DelTokenWhiteList(tokenId common.Address) error {
	if !self.IsManager(self.from) {
		return errUnauthorize
	}
	hashW := crypto.Keccak256Hash(append(tokenId.Hash().Bytes(), whitelistIndex[:]...))


	self.db.SetState(self.storage, hashW, bool2Hash(false))

	return nil
}

func (self *ManageObj) IsTokenInWhiteList(tokenId common.Address) bool {
	hashW := crypto.Keccak256Hash(append(tokenId.Hash().Bytes(), whitelistIndex[:]...))
	hash := self.db.GetState(self.storage, hashW)

	ok, _ := readBool(hash)
	return ok
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