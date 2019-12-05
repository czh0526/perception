package rawdb

import (
	"github.com/czh0526/perception/db/leveldb"
	"github.com/czh0526/perception/db/memorydb"
	"github.com/czh0526/perception/proton/chaindb"
)

func NewMemoryDatabase() chaindb.Database {
	return NewDatabase(memorydb.New())
}

func NewLevelDBDatabase(file string, cache int, handles int, namespace string) (chaindb.Database, error) {
	db, err := leveldb.New(file, cache, handles, namespace)
	if err != nil {
		return nil, err
	}

	return NewDatabase(db), nil
}

type nofreezedb struct {
	chaindb.KeyValueStore
}

// 将 chaindb.KeyValueStore 接口转化成 chaindb.Database
func NewDatabase(db chaindb.KeyValueStore) chaindb.Database {
	return &nofreezedb{
		KeyValueStore: db,
	}
}
