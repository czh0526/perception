package rawdb

import (
	"encoding/binary"

	"github.com/czh0526/perception/common"
)

var (
	headHeaderKey = []byte("LastHeader")
	headBlockKey  = []byte("LastBlock")

	headerNumberPrefix = []byte("H") // 'H' + hash -> num (uint64 big endian)
	headerPrefix       = []byte("h") // 'h' + num (uint64 big endian) + hash -> header
	headerTDSuffix     = []byte("t") // 'h' + num (uint64 big endian) + hash + 't' -> td
	headerHashSuffix   = []byte("n") // 'h' + num (uint64 big endian) + 'n' -> hash

	blockBodyPrefix = []byte("b")
)

const (
	freezerHeaderTable     = "headers"
	freezerHashTable       = "hashes"
	freezerBodiesTable     = "bodies"
	freezerReceiptTable    = "receipts"
	freezerDifficultyTable = "diffs"
)

func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// headerHashKey = 'h' + <num> + 'n'
func headerHashKey(number uint64) []byte {
	return append(append(headerPrefix, encodeBlockNumber(number)...), headerHashSuffix...)
}

// headerNumberKey = 'H' + <hash>
func headerNumberKey(hash common.Hash) []byte {
	return append(headerNumberPrefix, hash.Bytes()...)
}

// headerKey = 'h' + <num> + <hash>
func headerKey(number uint64, hash common.Hash) []byte {
	return append(append(headerPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// headerTDKey = 'h' + <num> + <hash> + 't'
func headerTDKey(number uint64, hash common.Hash) []byte {
	return append(headerKey(number, hash), headerTDSuffix...)
}

// blockBodyKey = 'b' + <num> + <hash>
func blockBodyKey(number uint64, hash common.Hash) []byte {
	return append(append(blockBodyPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}
