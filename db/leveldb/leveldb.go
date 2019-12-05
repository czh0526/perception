package leveldb

import (
	"github.com/czh0526/perception/proton/chaindb"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	minCache   = 16
	minHandles = 16
)

type Database struct {
	fn string
	db *leveldb.DB
}

func New(file string, cache int, handles int, namespace string) (*Database, error) {
	if cache < minCache {
		cache = minCache
	}
	if handles < minHandles {
		handles = minHandles
	}

	db, err := leveldb.OpenFile(file, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB,
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, currupted := err.(*errors.ErrCorrupted); currupted {
		db, err = leveldb.RecoverFile(file, nil)
	}

	if err != nil {
		return nil, err
	}

	ldb := &Database{
		fn: file,
		db: db,
	}

	return ldb, nil
}

func (db *Database) Has(key []byte) (bool, error) {
	return db.db.Has(key, nil)
}

func (db *Database) Get(key []byte) ([]byte, error) {
	data, err := db.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (db *Database) Put(key []byte, value []byte) error {
	return db.db.Put(key, value, nil)
}

func (db *Database) Delete(key []byte) error {
	return db.db.Delete(key, nil)
}

func (db *Database) Close() error {
	return db.db.Close()
}

func (db *Database) NewBatch() chaindb.Batch {
	return &batch{
		db: db.db,
		b:  new(leveldb.Batch),
	}
}

func (db *Database) NewIterator() chaindb.Iterator {
	return db.db.NewIterator(new(util.Range), nil)
}

func (db *Database) NewIteratorWithStart(start []byte) chaindb.Iterator {
	return db.db.NewIterator(&util.Range{Start: start}, nil)
}

func (db *Database) NewIteratorWithPrefix(prefix []byte) chaindb.Iterator {
	return db.db.NewIterator(util.BytesPrefix(prefix), nil)
}

type batch struct {
	db   *leveldb.DB
	b    *leveldb.Batch
	size int
}

func (b *batch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}

func (b *batch) Delete(key []byte) error {
	b.b.Delete(key)
	b.size++
	return nil
}

func (b *batch) ValueSize() int {
	return b.size
}

func (b *batch) Write() error {
	return b.db.Write(b.b, nil)
}

func (b *batch) Reset() {
	b.b.Reset()
	b.size = 0
}

// 在 w 上重放 k/v 写操作
func (b *batch) Replay(w chaindb.KeyValueWriter) error {
	return b.b.Replay(&replayer{writer: w})
}

type replayer struct {
	writer  chaindb.KeyValueWriter
	failure error
}

func (r *replayer) Put(key, value []byte) {
	if r.failure != nil {
		return
	}
	r.failure = r.writer.Put(key, value)
}

func (r *replayer) Delete(key []byte) {
	if r.failure != nil {
		return
	}
	r.failure = r.writer.Delete(key)
}
