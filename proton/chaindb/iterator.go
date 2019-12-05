package chaindb

type Iterator interface {
	Next() bool
	Error() error
	Key() []byte
	Value() []byte
	Release()
}

type Iteratee interface {
	NewIterator() Iterator
	NewIteratorWithStart(start []byte) Iterator
	NewIteratorWithPrefix(prefix []byte) Iterator
}
