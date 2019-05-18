// Copyright 2014 The go-ethereum Authors
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

package state

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/crypto"
	"github.com/bcos-one/BCOS/rlp"
)

var emptyCodeHash = crypto.Keccak256(nil)

type Code []byte

func (self Code) String() string {
	return string(self) //strings.Join(Disassemble(self), " ")
}

type Storage map[common.Hash]common.Hash

func (self Storage) String() (str string) {
	for key, value := range self {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (self Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range self {
		cpy[key] = value
	}

	return cpy
}

type tokenBalance map[common.Address]*big.Int

func (self tokenBalance) Copy() tokenBalance {
	cpy := make(tokenBalance)
	for key, value := range self {
		cpy[key] = value
	}

	return cpy
}

func (self tokenBalance) String() (str string) {
	for key, value := range self {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

// stateObject represents an Ethereum account which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateObject struct {
	address  common.Address
	addrHash common.Hash // hash of ethereum address of the account
	data     Account
	db       *StateDB

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access
	code Code // contract bytecode, which gets set when code is loaded

	originStorage Storage // Storage cache of original entries to dedup rewrites
	dirtyStorage  Storage // Storage entries that need to be flushed to disk

	tokenbalanceTrie   Trie         //tokenBalance trie, which becomes non-nil on first access
	originTokenBalance tokenBalance // tokenBalance cache of original entries to dedup rewrites
	dirtyTokenBalance  tokenBalance // tokenBalance entries that need to be flushed to disk

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	dirtyCode bool // true if the code was updated
	suicided  bool
	deleted   bool
}

// empty returns whether the account is considered empty.
func (self *stateObject) empty() bool {
	return self.data.Nonce == 0 &&
		self.data.Balance.Sign() == 0 &&
		bytes.Equal(self.data.CodeHash, emptyCodeHash) &&
		len(self.dirtyTokenBalance) == 0 && len(self.originTokenBalance) == 0 &&
		len(self.originStorage) == 0 && len(self.dirtyStorage) == 0
}

// Account is the Ethereum consensus representation of accounts.
// These objects are stored in the main account trie.
type Account struct {
	Nonce            uint64
	Balance          *big.Int
	Root             common.Hash // merkle root of the storage trie
	TokenBalanceRoot common.Hash // merkle root of the token balance trie
	CodeHash         []byte
	TokenSupport     common.Address // only set for contract address
}

// newObject creates a state object.
func newObject(db *StateDB, address common.Address, data Account) *stateObject {
	if data.Balance == nil {
		data.Balance = new(big.Int)
	}
	if data.CodeHash == nil {
		data.CodeHash = emptyCodeHash
	}
	return &stateObject{
		db:                 db,
		address:            address,
		addrHash:           crypto.Keccak256Hash(address[:]),
		data:               data,
		originStorage:      make(Storage),
		dirtyStorage:       make(Storage),
		originTokenBalance: make(tokenBalance),
		dirtyTokenBalance:  make(tokenBalance),
	}
}

// EncodeRLP implements rlp.Encoder.
func (self *stateObject) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, self.data)
}

// setError remembers the first non-nil error it is called with.
func (self *stateObject) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (self *stateObject) markSuicided() {
	self.suicided = true
}

func (self *stateObject) touch() {
	self.db.journal.append(touchChange{
		account: &self.address,
	})
	if self.address == ripemd {
		// Explicitly put it in the dirty-cache, which is otherwise generated from
		// flattened journals.
		self.db.journal.dirty(self.address)
	}
}

func (self *stateObject) getTrie(db Database) Trie {
	if self.trie == nil {
		var err error
		self.trie, err = db.OpenStorageTrie(self.addrHash, self.data.Root)
		if err != nil {
			self.trie, _ = db.OpenStorageTrie(self.addrHash, common.Hash{})
			self.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return self.trie
}

func (self *stateObject) getTokenBalanceTrie(db Database) Trie {
	if self.tokenbalanceTrie == nil {
		var err error
		self.tokenbalanceTrie, err = db.OpenStorageTrie(self.addrHash, self.data.TokenBalanceRoot)
		if err != nil {
			self.tokenbalanceTrie, _ = db.OpenStorageTrie(self.addrHash, common.Hash{})
			self.setError(fmt.Errorf("can't create token balance trie: %v", err))
		}
	}

	return self.tokenbalanceTrie
}

// GetState retrieves a value from the account storage trie.
func (self *stateObject) GetState(db Database, key common.Hash) common.Hash {
	// If we have a dirty value for this state entry, return it
	value, dirty := self.dirtyStorage[key]
	if dirty {
		return value
	}
	// Otherwise return the entry's original value
	return self.GetCommittedState(db, key)
}

// GetCommittedState retrieves a value from the committed account storage trie.
func (self *stateObject) GetCommittedState(db Database, key common.Hash) common.Hash {
	// If we have the original value cached, return that
	value, cached := self.originStorage[key]
	if cached {
		return value
	}
	// Otherwise load the value from the database
	enc, err := self.getTrie(db).TryGet(key[:])
	if err != nil {
		self.setError(err)
		return common.Hash{}
	}
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			self.setError(err)
		}
		value.SetBytes(content)
	}
	self.originStorage[key] = value
	return value
}

// SetState updates a value in account storage.
func (self *stateObject) SetState(db Database, key, value common.Hash) {
	// If the new value is the same as old, don't set
	prev := self.GetState(db, key)
	if prev == value {
		return
	}
	// New value is different, update and journal the change
	self.db.journal.append(storageChange{
		account:  &self.address,
		key:      key,
		prevalue: prev,
	})
	self.setState(key, value)
}

func (self *stateObject) setState(key, value common.Hash) {
	self.dirtyStorage[key] = value
}

// updateTrie writes cached storage modifications into the object's storage trie.
func (self *stateObject) updateTrie(db Database) Trie {
	tr := self.getTrie(db)
	for key, value := range self.dirtyStorage {
		delete(self.dirtyStorage, key)

		// Skip noop changes, persist actual changes
		if value == self.originStorage[key] {
			continue
		}
		self.originStorage[key] = value

		if (value == common.Hash{}) {
			self.setError(tr.TryDelete(key[:]))
			continue
		}
		// Encoding []byte cannot fail, ok to ignore the error.
		v, _ := rlp.EncodeToBytes(bytes.TrimLeft(value[:], "\x00"))
		self.setError(tr.TryUpdate(key[:], v))
	}
	return tr
}

// UpdateRoot sets the trie root to the current root hash of
func (self *stateObject) updateRoot(db Database) {
	self.updateTrie(db)
	self.updateTokenBalance(db)
	self.data.Root = self.trie.Hash()
	self.data.TokenBalanceRoot = self.tokenbalanceTrie.Hash()
}

// CommitTrie the storage trie of the object to db.
// This updates the trie root.
func (self *stateObject) CommitTrie(db Database) error {
	self.updateTrie(db)
	if self.dbErr != nil {
		return self.dbErr
	}
	root, err := self.trie.Commit(nil)
	if err == nil {
		self.data.Root = root
	}
	return err
}

func (self *stateObject) updateTokenBalance(db Database) Trie {
	tr := self.getTokenBalanceTrie(db)
	for key, value := range self.dirtyTokenBalance {
		delete(self.dirtyTokenBalance, key)

		//Skip noop changes, persist actual changes
		if self.originTokenBalance[key] != nil && self.originTokenBalance[key].Cmp(value) == 0 {
			continue
		}
		self.originTokenBalance[key] = value

		if value.Cmp(common.Big0) == 0 {
			self.setError(tr.TryDelete(key[:]))
		}
		// Encoding []byte cannot fail, ok to ignore the error.
		v, _ := rlp.EncodeToBytes(value)
		self.setError(tr.TryUpdate(key[:], v))
	}

	return tr
}

// CommitTokenBalance the token balance trie of the object to db.
// This updates the tokenbalanceTrie root.
func (self *stateObject) CommitTokenBalance(db Database) error {
	self.updateTokenBalance(db)
	if self.dbErr != nil {
		return self.dbErr
	}
	root, err := self.tokenbalanceTrie.Commit(nil)
	if err == nil {
		self.data.TokenBalanceRoot = root
	}

	return err
}

// AddBalance removes amount from c's balance.
// It is used to add funds to the destination account of a transfer.
func (self *stateObject) AddBalance(amount *big.Int) {
	// EIP158: We must check emptiness for the objects such that the account
	// clearing (0,0,0 objects) can take effect.
	if amount.Sign() == 0 {
		if self.empty() {
			self.touch()
		}

		return
	}
	self.SetBalance(new(big.Int).Add(self.Balance(), amount))
}

// SubBalance removes amount from c's balance.
// It is used to remove funds from the origin account of a transfer.
func (self *stateObject) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	self.SetBalance(new(big.Int).Sub(self.Balance(), amount))
}

func (self *stateObject) SetBalance(amount *big.Int) {
	self.db.journal.append(balanceChange{
		account: &self.address,
		prev:    new(big.Int).Set(self.data.Balance),
	})
	self.setBalance(amount)
}

func (self *stateObject) setBalance(amount *big.Int) {
	self.data.Balance = amount
}

// Return the gas back to the origin. Used by the Virtual machine or Closures
func (self *stateObject) ReturnGas(gas *big.Int) {}

func (self *stateObject) deepCopy(db *StateDB) *stateObject {
	stateObject := newObject(db, self.address, self.data)
	if self.trie != nil {
		stateObject.trie = db.db.CopyTrie(self.trie)
	}
	if self.tokenbalanceTrie != nil {
		stateObject.tokenbalanceTrie = db.db.CopyTrie(self.tokenbalanceTrie)
	}

	stateObject.code = self.code
	stateObject.dirtyTokenBalance = self.dirtyTokenBalance.Copy()
	stateObject.originTokenBalance = self.originTokenBalance.Copy()
	stateObject.dirtyStorage = self.dirtyStorage.Copy()
	stateObject.originStorage = self.originStorage.Copy()

	stateObject.suicided = self.suicided
	stateObject.dirtyCode = self.dirtyCode
	stateObject.deleted = self.deleted
	return stateObject
}

//
// Attribute accessors
//

// Returns the address of the contract/account
func (self *stateObject) Address() common.Address {
	return self.address
}

// Code returns the contract code associated with this object, if any.
func (self *stateObject) Code(db Database) []byte {
	if self.code != nil {
		return self.code
	}
	if bytes.Equal(self.CodeHash(), emptyCodeHash) {
		return nil
	}
	code, err := db.ContractCode(self.addrHash, common.BytesToHash(self.CodeHash()))
	if err != nil {
		self.setError(fmt.Errorf("can't load code hash %x: %v", self.CodeHash(), err))
	}
	self.code = code
	return code
}

func (self *stateObject) SetCode(codeHash common.Hash, code []byte) {
	prevcode := self.Code(self.db.db)
	self.db.journal.append(codeChange{
		account:  &self.address,
		prevhash: self.CodeHash(),
		prevcode: prevcode,
	})
	self.setCode(codeHash, code)
}

func (self *stateObject) setCode(codeHash common.Hash, code []byte) {
	self.code = code
	self.data.CodeHash = codeHash[:]
	self.dirtyCode = true
}

func (self *stateObject) SetTokenSupport(token common.Address) {
	prev := self.TokenSupport()
	self.db.journal.append(tokenSupportChange{
		account: &self.address,
		prev:    *prev,
	})

	self.setTokenSupport(token)
}

func (self *stateObject) setTokenSupport(token common.Address) {
	self.data.TokenSupport = token
}

func (self *stateObject) SetNonce(nonce uint64) {
	self.db.journal.append(nonceChange{
		account: &self.address,
		prev:    self.data.Nonce,
	})
	self.setNonce(nonce)
}

func (self *stateObject) setNonce(nonce uint64) {
	self.data.Nonce = nonce
}

func (self *stateObject) CodeHash() []byte {
	return self.data.CodeHash
}

func (self *stateObject) TokenSupport() *common.Address {
	return &self.data.TokenSupport
}

func (self *stateObject) Balance() *big.Int {
	return self.data.Balance
}

func (self *stateObject) Nonce() uint64 {
	return self.data.Nonce
}

// Never called, but must be present to allow stateObject to be used
// as a vm.Account interface that also satisfies the vm.ContractRef
// interface. Interfaces are awesome.
func (self *stateObject) Value() *big.Int {
	panic("Value on stateObject should never be called")
}

func (self *stateObject) TokenBalance(db Database, key common.Address) *big.Int {
	amount, dirty := self.dirtyTokenBalance[key]
	if dirty {
		return amount
	}
	return self.GetCommittedTokenBalance(db, key)
}

func (self *stateObject) GetCommittedTokenBalance(db Database, key common.Address) *big.Int {
	// If we have the original value cached, return that
	amount, cached := self.originTokenBalance[key]
	if cached {
		return amount
	}

	// Otherwise load the value from the database
	enc, err := self.getTokenBalanceTrie(db).TryGet(key[:])
	if err != nil {
		self.setError(err)
		return common.Big0
	}
	if len(enc) > 0 {
		var value big.Int
		err := rlp.DecodeBytes(enc, &value)
		if err != nil {
			self.setError(err)
		}
		amount = &value
	}

	if amount == nil {
		amount = common.Big0
	}

	self.originTokenBalance[key] = amount
	return amount
}

func (self *stateObject) AddTokenBalance(db Database, key common.Address, amount *big.Int) {
	// EIP158: We must check emptiness for the objects such that the account
	// clearing (0,0,0 objects) can take effect.
	if amount.Sign() == 0 {
		if self.empty() {
			self.touch()
		}

		return
	}

	self.SetTokenBalance(db, key, new(big.Int).Add(self.TokenBalance(db, key), amount))
}

func (self *stateObject) SubTokenBalance(db Database, key common.Address, amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}

	self.SetTokenBalance(db, key, new(big.Int).Sub(self.TokenBalance(db, key), amount))
}

func (self *stateObject) SetTokenBalance(db Database, key common.Address, amount *big.Int) {
	prev := self.TokenBalance(db, key)
	if prev.Cmp(amount) == 0 {
		return
	}

	self.db.journal.append(tokenChange{
		account: &self.address,
		id:      key,
		prev:    amount,
	})
	self.setTokenBalance(key, amount)
}

func (self *stateObject) setTokenBalance(key common.Address, amount *big.Int) {
	self.dirtyTokenBalance[key] = amount
}
