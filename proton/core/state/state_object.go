package state

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/czh0526/perception/rlp"

	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/proton/crypto"
)

type Code []byte
type Storage map[common.Hash]common.Hash

var emptyCodeHash = crypto.Keccak256(nil)

type stateObject struct {
	address  common.Address
	addrHash common.Hash
	data     Account
	db       *StateDB

	dbErr error

	trie Trie // Storage trie
	code Code

	originStorage  Storage
	pendingStorage Storage
	dirtyStorage   Storage

	dirtyCode bool
	suicided  bool
	deleted   bool
}

func (s *stateObject) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, s.data)
}

type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash
	CodeHash []byte
}

func newObject(db *StateDB, address common.Address, data Account) *stateObject {
	if data.Balance == nil {
		data.Balance = new(big.Int)
	}
	if data.CodeHash == nil {
		data.CodeHash = emptyCodeHash
	}
	if data.Root == (common.Hash{}) {
		data.Root = emptyRoot
	}

	return &stateObject{
		db:             db,
		address:        address,
		addrHash:       crypto.Keccak256Hash(address[:]),
		data:           data,
		originStorage:  make(Storage),
		pendingStorage: make(Storage),
		dirtyStorage:   make(Storage),
	}
}

func (s *stateObject) Address() common.Address {
	return s.address
}

func (s *stateObject) empty() bool {
	return s.data.Nonce == 0 && s.data.Balance.Sign() == 0 && bytes.Equal(s.data.CodeHash, emptyCodeHash)
}

func (s *stateObject) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (s *stateObject) SetNonce(nonce uint64) {
	s.db.journal.append(nonceChange{
		account: &s.address,
		prev:    s.data.Nonce,
	})
	s.setNonce(nonce)
}

func (s *stateObject) setNonce(nonce uint64) {
	s.data.Nonce = nonce
}

func (s *stateObject) SetCode(codeHash common.Hash, code []byte) {
	prevcode := s.Code(s.db.db)
	s.db.journal.append(codeChange{
		account:  &s.address,
		prevhash: s.CodeHash(),
		prevcode: prevcode,
	})
	s.setCode(codeHash, code)
}

func (s *stateObject) setCode(codeHash common.Hash, code []byte) {
	s.code = code
	s.data.CodeHash = codeHash[:]
	s.dirtyCode = true
}

func (s *stateObject) Code(db Database) []byte {
	if s.code != nil {
		return s.code
	}
	if bytes.Equal(s.CodeHash(), emptyCodeHash) {
		return nil
	}
	code, err := db.ContractCode(s.addrHash, common.BytesToHash(s.CodeHash()))
	if err != nil {
		s.setError(fmt.Errorf("can't load code hash %x: %v", s.CodeHash(), err))
	}
	s.code = code
	return code
}

func (s *stateObject) CodeHash() []byte {
	return s.data.CodeHash
}

func (s *stateObject) Balance() *big.Int {
	return s.data.Balance
}

func (s *stateObject) AddBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.SetBalance(new(big.Int).Add(s.Balance(), amount))
}

func (s *stateObject) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.SetBalance(new(big.Int).Sub(s.Balance(), amount))
}

func (s *stateObject) SetBalance(amount *big.Int) {
	s.db.journal.append(balanceChange{
		account: &s.address,
		prev:    new(big.Int).Set(s.data.Balance),
	})
	s.setBalance(amount)
}

func (s *stateObject) setBalance(amount *big.Int) {
	s.data.Balance = amount
}

func (s *stateObject) SetState(db Database, key, value common.Hash) {
	prev := s.GetState(db, key)
	if prev == value {
		return
	}

	s.db.journal.append(storageChange{
		account:  &s.address,
		key:      key,
		prevalue: prev,
	})
	s.setState(key, value)
}

func (s *stateObject) setState(key, value common.Hash) {
	s.dirtyStorage[key] = value
}

func (s *stateObject) GetState(db Database, key common.Hash) common.Hash {
	value, dirty := s.dirtyStorage[key]
	if dirty {
		return value
	}

	return s.GetCommittedState(db, key)
}

func (s *stateObject) GetCommittedState(db Database, key common.Hash) common.Hash {
	if value, pending := s.pendingStorage[key]; pending {
		return value
	}
	if value, cached := s.originStorage[key]; cached {
		return value
	}

	enc, err := s.getTrie(db).TryGet(key[:])
	if err != nil {
		s.setError(err)
		return common.Hash{}
	}
	var value common.Hash
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			s.setError(err)
		}
		value.SetBytes(content)
	}
	s.originStorage[key] = value
	return value
}

// move data: dirtyStorage => pendingStorage
func (s *stateObject) finalise() {
	for key, value := range s.dirtyStorage {
		s.pendingStorage[key] = value
	}
	if len(s.dirtyStorage) > 0 {
		s.dirtyStorage = make(Storage)
	}
}

// update change: 	1). pendingStorage ==> Trie
//					2). pendingStorage ==> originStorage
func (s *stateObject) updateTrie(db Database) Trie {
	s.finalise()

	tr := s.getTrie(db)
	for key, value := range s.pendingStorage {
		if value == s.originStorage[key] {
			continue
		}
		s.originStorage[key] = value

		if (value == common.Hash{}) {
			s.setError(tr.TryDelete(key[:]))
			continue
		}
		v, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(value[:]))
		s.setError(tr.TryUpdate(key[:], v))
	}
	if len(s.pendingStorage) > 0 {
		s.pendingStorage = make(Storage)
	}
	return tr
}

func (s *stateObject) CommitTrie(db Database) error {
	// write: pendingStorage ==> trie
	s.updateTrie(db)
	if s.dbErr != nil {
		return s.dbErr
	}

	// trie commit
	root, err := s.trie.Commit(nil)
	if err == nil {
		s.data.Root = root
	}
	return err
}

func (s *stateObject) updateRoot(db Database) {
	s.updateTrie(db)
	s.data.Root = s.trie.Hash()
}

func (s *stateObject) getTrie(db Database) Trie {
	if s.trie == nil {
		var err error
		s.trie, err = db.OpenStorageTrie(s.addrHash, s.data.Root)
		if err != nil {
			s.trie, _ = db.OpenStorageTrie(s.addrHash, common.Hash{})
			s.setError(fmt.Errorf("can't create trie: %v", err))
		}
	}
	return s.trie
}
