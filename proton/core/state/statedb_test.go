package state

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/db/memorydb"
	"github.com/czh0526/perception/proton/core/rawdb"
	"github.com/czh0526/perception/proton/trie"
)

func TestEmptyRoot(t *testing.T) {
	sec_trie, err := trie.NewSecure(common.Hash{}, trie.NewDatabase(memorydb.New()))
	if err != nil {
		t.Fatal(err)
	}
	rootHash := sec_trie.Hash()
	if !bytes.Equal(rootHash.Bytes(), emptyRoot.Bytes()) {
		t.Fatalf("not incoresponding with emptyRoot, %x != %x", rootHash, emptyRoot)
	}

	fmt.Println("wonderfully, rootHash == emptyRoot.")
}

func TestUpdateLeaks(t *testing.T) {
	fmt.Println("1). 构建一个空的 chaindb.Database 对象。")
	db := rawdb.NewMemoryDatabase()
	fmt.Println("2). 在 chaindb.Database 之上构建 state.StateDB 对象")
	state, _ := New(common.Hash{}, NewDatabase(db))

	fmt.Println("3). 增加几个 stateObject 对象.")
	for i := byte(0); i < 5; i++ {
		addr := common.BytesToAddress([]byte{i})
		state.AddBalance(addr, big.NewInt(int64(11*i)))
		state.SetNonce(addr, uint64(43*i))
	}

	root := state.IntermediateRoot(false)
	fmt.Printf("4). statedb.intermediateRoot(), root = %v \n", root.Hex())

	root, err := state.Commit(false)
	if err != nil {
		t.Errorf("can not commit state, err = %v", err)
	}
	fmt.Printf("5). statedb.Commit(), root = %v \n", root.Hex())

	if err := state.Database().TrieDB().Commit(root, false); err != nil {
		t.Errorf("can not commit trie %v to persistent database", root.Hex())
	}
	fmt.Println("6). trie.Commit()")

	fmt.Println("检查 chaindb.Database 对象中是否被写入数据")
	it := db.NewIterator()
	i := 1
	for it.Next() {
		fmt.Printf("%d). %x -> %x \n", i, it.Key(), it.Value())
		i++
	}
	it.Release()

	state2, err := New(root, NewDatabase(db))
	if err != nil {
		panic(err)
	}
	for i := byte(0); i < 5; i++ {
		addr := common.BytesToAddress([]byte{i})
		balance := state2.GetBalance(addr)
		fmt.Printf("%v ==> %v \n", addr, balance)
	}
}
