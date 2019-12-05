package core

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/common/math"
	"github.com/czh0526/perception/proton/core/rawdb"
	"github.com/czh0526/perception/proton/core/state"
	"github.com/czh0526/perception/proton/core/types"
)

func TestGenerateChainInMemoryDB(t *testing.T) {
	fmt.Println("1). 加载 genesis.json 文件")
	genesisPath := "/Users/czh/Workspace/eth_datadir/genesis.json"
	file, err := os.Open(genesisPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	gspec := new(Genesis)
	if err := json.NewDecoder(file).Decode(gspec); err != nil {
		panic(err)
	}

	fmt.Println("2). 构建一个空的 chaindb.Database 对象")
	db := rawdb.NewMemoryDatabase()
	fmt.Println("3). 将 genesis block & statedb 写入 chaindb.Database")
	genesis, err := gspec.Commit(db)
	if err != nil {
		panic(err)
	}

	fmt.Println("4). 在 genesis block 后面构建6个 Block")
	var (
		header *types.Header
		block  *types.Block = genesis
		root   common.Hash
	)
	blocks := make([]*types.Block, 0)
	statedb, _ := state.New(genesis.Root(), state.NewDatabase(db))
	for i := 1; i <= 6; i++ {
		header = makeHeader(block, statedb)
		block = types.NewBlock(header, []*types.Transaction{}, []*types.Header{})
		fmt.Printf("block %d = %v \n", i, block2Str(block))
		root, err = statedb.Commit(true)
		if err != nil {
			panic(err)
		}
		if err := statedb.Database().TrieDB().Commit(root, false); err != nil {
			panic(err)
		}
		blocks = append(blocks, block)
	}

	// 构建区块链, 插入区块
	// levelDB, err := rawdb.NewLevelDBDatabase("/Users/czh/Workspace/eth_datadir/node3/perception/chaindata", 0, 0, "")
	// if err != nil {
	// 	panic(err)
	// }
	fmt.Println("5). 构建一条 blockchain.")
	blockchain, err := NewBlockChain(db)
	if err != nil {
		panic(err)
	}

	fmt.Println("6). 将6个 Block 插入 blockchain.")
	if i, err := blockchain.InsertChain(blocks); err != nil {
		fmt.Printf("insert error (block %d): %v \n", blocks[i].NumberU64(), err)
		return
	}

	fmt.Println("7). 获取第 n 块的StateDB.")
	header1 := blockchain.GetBlockByNumber(1).Header()
	statedb, err = state.New(header1.Root, state.NewDatabase(db))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("8). 查询账户余额、验证插入的区块是否正确")
	addr := common.HexToAddress("559702c299028e859086d860c64a69b3b6c14c0d")
	balance := statedb.GetBalance(addr)
	expected, ok := math.ParseBig256("100000000000000000000000")
	if !ok {
		t.Fatal("数字转换错误 ...")
	}
	if balance.Cmp(expected) != 0 {
		t.Fatal("账户余额对不上 ...")
	}
	fmt.Printf("%v ==> %v \n", addr.Hex(), balance)
}

func header2Str(header *types.Header) string {
	return "{" +
		fmt.Sprintf("\n\t\tParentHash: 0x%0x", header.ParentHash) +
		fmt.Sprintf("\n\t\tRoot: 0x%0x", header.Root) +
		fmt.Sprintf("\n\t\tNumber: %d", header.Number) +
		fmt.Sprintf("\n\t\tTime: %d", header.Time) +
		"\n\t}"
}

func block2Str(block *types.Block) string {
	return fmt.Sprintf(`"%0x": {`, block.Hash()) +
		"\n\theader: " + header2Str(block.Header()) +
		fmt.Sprintf("\n\ttransactions: %d", len(block.Body().Transactions)) +
		"\n}"
}

func TestGenerateChainInLevelDB(t *testing.T) {

	fmt.Println("1). 打开一个现有的 chaindb.Database.")
	levelDB, err := rawdb.NewLevelDBDatabase("/Users/czh/Workspace/perception_datadir/node3/perception/chaindata", 0, 0, "")
	if err != nil {
		panic(err)
	}

	fmt.Println("2). 基于levelDB, 构建一条 blockchain.")
	blockchain, err := NewBlockChain(levelDB)
	if err != nil {
		panic(err)
	}

	fmt.Println("3). 读取 blockchain 的 head header ")
	currentBlock := blockchain.CurrentBlock()
	if currentBlock == nil {
		t.Fatalf("can not read head header. ")
	}

	fmt.Printf("4). 在 block %d 后面构建6个 Block \n", currentBlock.NumberU64())
	var (
		header *types.Header
		block  *types.Block = currentBlock
		root   common.Hash
	)
	blocks := make([]*types.Block, 0)
	statedb, _ := state.New(currentBlock.Root(), state.NewDatabase(levelDB))
	for i := 1; i <= 1000; i++ {
		header = makeHeader(block, statedb)
		block = types.NewBlock(header, []*types.Transaction{}, []*types.Header{})
		fmt.Printf("block %d = %v \n", i, block2Str(block))
		root, err = statedb.Commit(true)
		if err != nil {
			panic(err)
		}
		if err := statedb.Database().TrieDB().Commit(root, false); err != nil {
			panic(err)
		}
		blocks = append(blocks, block)
	}

	fmt.Println("5). 将6个 Block 插入 blockchain.")
	if i, err := blockchain.InsertChain(blocks); err != nil {
		fmt.Printf("insert error (block %d): %v \n", blocks[i].NumberU64(), err)
		return
	}

	fmt.Println("6). 获取第 n 块的StateDB.")
	header1 := blockchain.GetBlockByNumber(5).Header()
	statedb, err = state.New(header1.Root, state.NewDatabase(levelDB))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("8). 查询账户余额、验证插入的区块是否正确")
	addr := common.HexToAddress("559702c299028e859086d860c64a69b3b6c14c0d")
	balance := statedb.GetBalance(addr)
	expected, ok := math.ParseBig256("100000000000000000000000")
	if !ok {
		t.Fatal("数字转换错误 ...")
	}
	if balance.Cmp(expected) != 0 {
		t.Fatal("账户余额对不上 ...")
	}
	fmt.Printf("%v ==> %v \n", addr.Hex(), balance)
}
