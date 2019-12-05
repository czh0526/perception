package rawdb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/proton/chaindb"
	"github.com/czh0526/perception/proton/core/types"
	"github.com/czh0526/perception/rlp"
)

// HeaderNumber:  hash ==> number
func ReadHeaderNumber(db chaindb.KeyValueReader, hash common.Hash) *uint64 {
	data, _ := db.Get(headerNumberKey(hash))
	if len(data) == 0 {
		return nil
	}
	number := binary.BigEndian.Uint64(data)
	return &number
}

func WriteHeaderNumber(db chaindb.KeyValueWriter, hash common.Hash, number uint64) {
	key := headerNumberKey(hash)
	enc := encodeBlockNumber(number)
	if err := db.Put(key, enc); err != nil {
		panic(fmt.Sprintf("Failed to store hash to number mapping, err = %v", err))
	}
}

func ReadCanonicalHash(db chaindb.Reader, number uint64) common.Hash {
	data, _ := db.Get(headerHashKey(number))
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// CanonicalHash:  number ==> hash
func WriteCanonicalHash(db chaindb.KeyValueWriter, hash common.Hash, number uint64) {
	key := headerHashKey(number)
	if err := db.Put(key, hash.Bytes()); err != nil {
		panic(fmt.Sprintf("Failed to store number to hash mapping, err = %v", err))
	}
}

func ReadBlock(db chaindb.Reader, hash common.Hash, number uint64) *types.Block {
	header := ReadHeader(db, hash, number)
	if header == nil {
		return nil
	}

	body := ReadBody(db, hash, number)
	if body == nil {
		return nil
	}

	return types.NewBlockWithHeader(header).WithBody(body.Transactions)
}

func WriteBlock(db chaindb.KeyValueWriter, block *types.Block) {
	WriteBody(db, block.Hash(), block.NumberU64(), block.Body())
	WriteHeader(db, block.Header())
}

func ReadHeader(db chaindb.Reader, hash common.Hash, number uint64) *types.Header {
	// read rlp encoded bytes
	data := ReadHeaderRLP(db, hash, number)
	if len(data) == 0 {
		return nil
	}
	// decode rlp encoded bytes
	header := new(types.Header)
	if err := rlp.Decode(bytes.NewReader(data), header); err != nil {
		fmt.Printf("Invalid block header RLP, hash = %v, err = %v \n", hash, err)
		return nil
	}

	return header
}

func ReadHeaderRLP(db chaindb.Reader, hash common.Hash, number uint64) rlp.RawValue {
	data, _ := db.Get(headerKey(number, hash))
	if len(data) == 0 {
		return rlp.RawValue{}
	}
	return data
}

func WriteHeader(db chaindb.KeyValueWriter, header *types.Header) {
	var (
		hash   = header.Hash()
		number = header.Number.Uint64()
	)
	WriteHeaderNumber(db, hash, number)

	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		panic(fmt.Sprintf("Failed to RLP encode header, err = %v", err))
	}
	key := headerKey(number, hash)
	if err := db.Put(key, data); err != nil {
		panic(fmt.Sprintf("Failed to store header, err = %v", err))
	}
}

func ReadBody(db chaindb.Reader, hash common.Hash, number uint64) *types.Body {
	data := ReadBodyRLP(db, hash, number)
	if len(data) == 0 {
		return nil
	}
	body := new(types.Body)
	if err := rlp.Decode(bytes.NewReader(data), body); err != nil {
		fmt.Printf("Invalid block body RLP, hash = %v, err = %v \n", hash, err)
		return nil
	}
	return body
}

func ReadBodyRLP(db chaindb.Reader, hash common.Hash, number uint64) rlp.RawValue {
	data, _ := db.Get(blockBodyKey(number, hash))
	if len(data) == 0 {
		return rlp.RawValue{}
	}
	return data
}

func WriteBody(db chaindb.KeyValueWriter, hash common.Hash, number uint64, body *types.Body) {
	data, err := rlp.EncodeToBytes(body)
	if err != nil {
		panic(fmt.Sprintf("Failed to RLP encode body, err = %v", err))
	}
	WriteBodyRLP(db, hash, number, data)
}

func WriteBodyRLP(db chaindb.KeyValueWriter, hash common.Hash, number uint64, rlp rlp.RawValue) {
	key := blockBodyKey(number, hash)
	if err := db.Put(key, rlp); err != nil {
		panic(fmt.Sprintf("Failed to store block body, err = %v", err))
	}
}

/*
func ReadTd(db chaindb.Reader, hash common.Hash, number uint64) *big.Int {
	data := ReadTdRLP(db, hash, number)
	if len(data) == 0 {
		return nil
	}
	td := new(big.Int)
	if err := rlp.Decode(bytes.NewReader(data), td); err != nil {
		fmt.Printf("Invalid block TD RLP, hash = %v, err = %v \n", hash, err)
		return nil
	}
	return td
}

func ReadTdRLP(db chaindb.Reader, hash common.Hash, number uint64) rlp.RawValue {
	data, _ := db.Get(headerTDKey(number, hash))
	if len(data) == 0 {
		return rlp.RawValue{}
	}
	return data
}
*/

func ReadHeadHeaderHash(db chaindb.KeyValueReader) common.Hash {
	data, _ := db.Get(headHeaderKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

func WriteHeadHeaderHash(db chaindb.KeyValueWriter, hash common.Hash) {
	if err := db.Put(headHeaderKey, hash.Bytes()); err != nil {
		panic(fmt.Sprintf("Failed to store last header's hash, err = %v", err))
	}
}

func ReadHeadBlockHash(db chaindb.KeyValueReader) common.Hash {
	data, _ := db.Get(headBlockKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

func WriteHeadBlockHash(db chaindb.KeyValueWriter, hash common.Hash) {
	if err := db.Put(headBlockKey, hash.Bytes()); err != nil {
		panic(fmt.Sprintf("Failed to store last block's hash, err = %v", err))
	}
}

func DeleteHeader(db chaindb.KeyValueWriter, hash common.Hash, number uint64) {
	deleteHeaderWithoutNumber(db, hash, number)
	if err := db.Delete(headerNumberKey(hash)); err != nil {
		log.Fatalf("Failed to delete hash to number mapping, err = %v \n", err)
	}
}

func deleteHeaderWithoutNumber(db chaindb.KeyValueWriter, hash common.Hash, number uint64) {
	if err := db.Delete(headerKey(number, hash)); err != nil {
		log.Fatalf("Failed to delete header, err = %v", err)
	}
}

func DeleteCanonicalHash(db chaindb.KeyValueWriter, number uint64) {
	if err := db.Delete(headerHashKey(number)); err != nil {
		log.Fatalf("Failed to delete number to hash mapping, err = %v", err)
	}
}

func DeleteBody(db chaindb.KeyValueWriter, hash common.Hash, number uint64) {
	if err := db.Delete(blockBodyKey(number, hash)); err != nil {
		log.Fatalf("Failed to delete block body, err = %v", err)
	}
}
