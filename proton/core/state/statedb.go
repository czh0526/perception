package state

import (
	"fmt"
	"math/big"

	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/proton/crypto"
	"github.com/czh0526/perception/rlp"
)

var (
	emptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	emptyCode = crypto.Keccak256Hash(nil)
)

type revision struct {
	id           int
	journalIndex int
}

type StateDB struct {
	db   Database
	trie Trie

	stateObjects        map[common.Address]*stateObject
	stateObjectsPending map[common.Address]struct{}
	stateObjectsDirty   map[common.Address]struct{}

	dbErr  error
	refund uint64 // ?

	journal        *journal
	validRevisions []revision
}

func New(root common.Hash, db Database) (*StateDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}

	return &StateDB{
		db:                  db,
		trie:                tr,
		stateObjects:        make(map[common.Address]*stateObject),
		stateObjectsPending: make(map[common.Address]struct{}),
		stateObjectsDirty:   make(map[common.Address]struct{}),
		journal:             newJournal(),
	}, nil
}

func (self *StateDB) Database() Database {
	return self.db
}

func (self *StateDB) Trie() Trie {
	return self.trie
}

func (self *StateDB) GetBalance(addr common.Address) *big.Int {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance()
	}
	return common.Big0
}

func (self *StateDB) AddBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

func (self *StateDB) SubBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(amount)
	}
}

func (self *StateDB) SetBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetBalance(amount)
	}
}

func (self *StateDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (self *StateDB) SetCode(addr common.Address, code []byte) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(crypto.Keccak256Hash(code), code)
	}
}

func (self *StateDB) SetState(addr common.Address, key, value common.Hash) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(self.db, key, value)
	}
}

// 更新数据: journal ==> stateObjects
func (s *StateDB) Finalise(deleteEmptyObjects bool) {
	for addr := range s.journal.dirties {
		obj, exist := s.stateObjects[addr]
		if !exist {
			continue
		}

		if obj.suicided || (deleteEmptyObjects && obj.empty()) {
			obj.deleted = true
		} else {
			obj.finalise()
		}
		s.stateObjectsPending[addr] = struct{}{}
		s.stateObjectsDirty[addr] = struct{}{}
	}
	s.clearJournalAndRefund()
}

func (s *StateDB) clearJournalAndRefund() {
	if len(s.journal.entries) > 0 {
		s.journal = newJournal()
		s.refund = 0
	}
	s.validRevisions = s.validRevisions[:0]
}

// 更新数据: stateObjectsPending ==> Trie
func (s *StateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	s.Finalise(deleteEmptyObjects)

	for addr := range s.stateObjectsPending {
		obj := s.stateObjects[addr]
		if obj.deleted {
			// 更新 statedb 数据
			s.deleteStateObject(obj)
		} else {
			// 更新 stateObject 数据
			obj.updateRoot(s.db)
			// 更新 statedb 数据
			s.updateStateObject(obj)
		}
	}
	if len(s.stateObjectsPending) > 0 {
		s.stateObjectsPending = make(map[common.Address]struct{})
	}

	return s.trie.Hash()
}

// write the trie
func (s *StateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	// Finalize pending changes && merge data into trie
	s.IntermediateRoot(deleteEmptyObjects)

	for addr := range s.stateObjectsDirty {
		if obj := s.stateObjects[addr]; !obj.deleted {
			if obj.code != nil && obj.dirtyCode {
				s.db.TrieDB().InsertBlob(common.BytesToHash(obj.CodeHash()), obj.code)
				obj.dirtyCode = false
			}

			if err := obj.CommitTrie(s.db); err != nil {
				return common.Hash{}, err
			}
		}
	}

	// reset dirty
	if len(s.stateObjectsDirty) > 0 {
		s.stateObjectsDirty = make(map[common.Address]struct{})
	}

	return s.trie.Commit(func(leaf []byte, parent common.Hash) error {
		var account Account
		if err := rlp.DecodeBytes(leaf, &account); err != nil {
			return nil
		}
		if account.Root != emptyRoot {
			s.db.TrieDB().Reference(account.Root, parent)
		}
		code := common.BytesToHash(account.CodeHash)
		if code != emptyCode {
			s.db.TrieDB().Reference(code, parent)
		}
		return nil
	})
}

func (self *StateDB) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (self *StateDB) getDeletedStateObject(addr common.Address) *stateObject {
	if obj := self.stateObjects[addr]; obj != nil {
		return obj
	}

	enc, err := self.trie.TryGet(addr[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil
	}

	var data Account
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		fmt.Printf("Failed to decode state object, addr = %0x, err = %v \n", addr, err)
		return nil
	}

	obj := newObject(self, addr, data)
	self.setStateObject(obj)
	return obj
}

func (self *StateDB) getStateObject(addr common.Address) *stateObject {
	if obj := self.getDeletedStateObject(addr); obj != nil && !obj.deleted {
		return obj
	}
	return nil
}

func (self *StateDB) setStateObject(object *stateObject) {
	self.stateObjects[object.Address()] = object
}

func (self *StateDB) createObject(addr common.Address) (newobj, prev *stateObject) {
	prev = self.getDeletedStateObject(addr)

	newobj = newObject(self, addr, Account{})
	newobj.setNonce(0)
	if prev == nil {
		self.journal.append(createObjectChange{account: &addr})
	} else {
		self.journal.append(resetObjectChange{prev: prev})
	}
	self.setStateObject(newobj)
	return newobj, prev
}

func (self *StateDB) deleteStateObject(obj *stateObject) {
	addr := obj.Address()
	self.setError(self.trie.TryDelete(addr[:]))
}

func (self *StateDB) updateStateObject(obj *stateObject) {
	addr := obj.Address()

	data, err := rlp.EncodeToBytes(obj)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v.", addr[:], err))
	}
	err = self.trie.TryUpdate(addr[:], data)
	self.setError(err)
}

func (self *StateDB) GetOrNewStateObject(addr common.Address) *stateObject {
	stateObject := self.getStateObject(addr)
	if stateObject == nil {
		stateObject, _ = self.createObject(addr)
	}
	return stateObject
}
