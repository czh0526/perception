package chaindb

import "io"

type KeyValueReader interface {
	Has(key []byte) (bool, error)
	Get(key []byte) ([]byte, error)
}

type KeyValueWriter interface {
	Put(key []byte, value []byte) error
	Delete(key []byte) error
}

type KeyValueStore interface {
	KeyValueReader
	KeyValueWriter
	Batcher
	Iteratee
	io.Closer
}

/*
	interfaces for rawdb
*/
type AncientReader interface {
	HasAncient(kind string, number uint64) (bool, error)
	Ancient(kind string, number uint64) ([]byte, error)
	Ancients() (uint64, error)
	AncientSize(kind string) (uint64, error)
}

type Reader interface {
	KeyValueReader
}

type Writer interface {
	KeyValueWriter
}

type Database interface {
	Reader
	Writer
	Batcher
	Iteratee
	io.Closer
}
