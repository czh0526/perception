package state

import (
	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/proton/chaindb"
	"github.com/czh0526/perception/proton/trie"
	lru "github.com/hashicorp/golang-lru"
)

const (
	codeSizeCacheSize = 100000
)

type Database interface {
	OpenTrie(root common.Hash) (Trie, error)
	OpenStorageTrie(addrHash, root common.Hash) (Trie, error)
	ContractCode(addrHash, codeHash common.Hash) ([]byte, error)
	TrieDB() *trie.Database
}

type Trie interface {
	GetKey([]byte) []byte

	TryGet(key []byte) ([]byte, error)

	TryUpdate(key, value []byte) error

	TryDelete(key []byte) error

	Hash() common.Hash

	Commit(onleaf trie.LeafCallback) (common.Hash, error)

	NodeIterator(startKey []byte) trie.NodeIterator
}

func NewDatabase(db chaindb.Database) Database {
	return NewDatabaseWithCache(db, 0)
}

func NewDatabaseWithCache(db chaindb.Database, cache int) Database {
	csc, _ := lru.New(codeSizeCacheSize)
	return &cachingDB{
		db:            trie.NewDatabaseWithCache(db, cache),
		codeSizeCache: csc,
	}
}

type cachingDB struct {
	db            *trie.Database
	codeSizeCache *lru.Cache
}

func (db *cachingDB) OpenTrie(root common.Hash) (Trie, error) {
	return trie.NewSecure(root, db.db)
}

func (db *cachingDB) OpenStorageTrie(addrHash, root common.Hash) (Trie, error) {
	return trie.NewSecure(root, db.db)
}

func (db *cachingDB) ContractCode(addrHash, codeHash common.Hash) ([]byte, error) {
	code, err := db.db.Node(codeHash)
	if err == nil {
		db.codeSizeCache.Add(codeHash, len(code))
	}
	return code, err
}

func (db *cachingDB) TrieDB() *trie.Database {
	return db.db
}
