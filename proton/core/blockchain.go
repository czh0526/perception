package core

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/proton/chaindb"
	"github.com/czh0526/perception/proton/core/rawdb"
	"github.com/czh0526/perception/proton/core/state"
	"github.com/czh0526/perception/proton/core/types"
)

type BlockChain struct {
	db chaindb.Database

	genesisBlock *types.Block
	hc           *HeaderChain

	chainmu      sync.RWMutex
	stateCache   state.Database
	currentBlock atomic.Value
}

func NewBlockChain(db chaindb.Database) (*BlockChain, error) {
	bc := &BlockChain{
		db:         db,
		stateCache: state.NewDatabaseWithCache(db, 256),
	}

	// 初始化 header chain
	var err error
	bc.hc, err = NewHeaderChain(db)
	if err != nil {
		return nil, err
	}

	bc.genesisBlock = bc.GetBlockByNumber(0)
	if bc.genesisBlock == nil {
		return nil, errors.New("No genesis block.")
	}

	var nilBlock *types.Block
	bc.currentBlock.Store(nilBlock)

	if err := bc.loadLastState(); err != nil {
		return nil, err
	}

	return bc, nil
}

func (bc *BlockChain) State() (*state.StateDB, error) {
	return bc.StateAt(bc.CurrentBlock().Root())
}

func (bc *BlockChain) StateAt(root common.Hash) (*state.StateDB, error) {
	return state.New(root, bc.stateCache)
}

func (bc *BlockChain) loadLastState() error {
	head := rawdb.ReadHeadBlockHash(bc.db)
	if head == (common.Hash{}) {
		log.Println("empty database, resetting chain.")
		return bc.Reset()
	}

	currentBlock := bc.GetBlockByHash(head)
	if currentBlock == nil {
		log.Printf("Head block missing, resetting chain, hash = %v \n", head)
		return bc.Reset()
	}

	if _, err := state.New(currentBlock.Root(), bc.stateCache); err != nil {
		log.Printf("Head state missing, repairing chain, number = %d, hash = 0x%0x \n", currentBlock.Number(), currentBlock.Hash())
		if err := bc.repair(&currentBlock); err != nil {
			return err
		}
		rawdb.WriteHeadBlockHash(bc.db, currentBlock.Hash())
	}

	// 数据库一切正常，设置 blockchain
	bc.currentBlock.Store(currentBlock)

	currentHeader := currentBlock.Header()
	if head := rawdb.ReadHeadHeaderHash(bc.db); head != (common.Hash{}) {
		if header := bc.GetHeaderByHash(head); header != nil {
			currentHeader = header
		}
	}
	bc.hc.SetCurrentHeader(currentHeader)

	return nil
}

func (bc *BlockChain) Genesis() *types.Block {
	return bc.genesisBlock
}

func (bc *BlockChain) GetBlockByNumber(number uint64) *types.Block {
	hash := rawdb.ReadCanonicalHash(bc.db, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return bc.GetBlock(hash, number)
}

func (bc *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	number := rawdb.ReadHeaderNumber(bc.db, hash)
	if number == nil {
		return nil
	}
	return bc.GetBlock(hash, *number)
}

func (bc *BlockChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	// TODO: add data cache
	block := rawdb.ReadBlock(bc.db, hash, number)
	if block == nil {
		return nil
	}
	return block
}

func (bc *BlockChain) CurrentHeader() *types.Header {
	return bc.hc.CurrentHeader()
}

func (bc *BlockChain) CurrentBlock() *types.Block {
	return bc.currentBlock.Load().(*types.Block)
}

func (bc *BlockChain) Reset() error {
	return bc.ResetWithGenesisBlock(bc.genesisBlock)
}

func (bc *BlockChain) ResetWithGenesisBlock(genesis *types.Block) error {
	// 删除全部的 Block 和 Header
	if err := bc.SetHead(0); err != nil {
		return err
	}

	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()

	// 向 chaindb 中写入 Header && Body
	rawdb.WriteBlock(bc.db, genesis)

	// 修改 blockchain
	bc.genesisBlock = genesis
	bc.insert(bc.genesisBlock)
	bc.currentBlock.Store(bc.genesisBlock)

	// 修改 blockchain header
	bc.hc.SetGenesis(bc.genesisBlock.Header())
	bc.hc.SetCurrentHeader(bc.genesisBlock.Header())

	return nil
}

func (bc *BlockChain) SetHead(head uint64) error {
	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()

	// 1). 修改 db 中的 head block
	// 2). 修改 blockchain 中的 current block
	updateFn := func(db chaindb.KeyValueWriter, header *types.Header) {
		if currentBlock := bc.CurrentBlock(); currentBlock != nil && header.Number.Uint64() < currentBlock.NumberU64() {
			newHeadBlock := bc.GetBlock(header.Hash(), header.Number.Uint64())
			if newHeadBlock == nil {
				newHeadBlock = bc.genesisBlock
			} else {
				if _, err := state.New(newHeadBlock.Root(), bc.stateCache); err != nil {
					newHeadBlock = bc.genesisBlock
				}
			}
			rawdb.WriteHeadBlockHash(db, newHeadBlock.Hash())
			bc.currentBlock.Store(newHeadBlock)
		}
	}

	// 删除 block body
	delFn := func(db chaindb.KeyValueWriter, hash common.Hash, num uint64) {
		rawdb.DeleteBody(db, hash, num)
	}

	bc.hc.SetHead(head, updateFn, delFn)
	return bc.loadLastState()
}

func (bc *BlockChain) GetHeaderByHash(hash common.Hash) *types.Header {
	return bc.hc.GetHeaderByHash(hash)
}

func (bc *BlockChain) InsertChain(chain []*types.Block) (int, error) {
	if len(chain) == 0 {
		return 0, nil
	}

	var (
		block, prev *types.Block
	)

	// 检验插入的区块是否连续
	for i := 1; i < len(chain); i++ {
		block = chain[i]
		prev = chain[i-1]
		if block.NumberU64() != prev.NumberU64()+1 || block.ParentHash() != prev.Hash() {
			log.Fatalf("Non contiguous block insert, number = %d, hash = %0x, parent = %0x, prevnumber = %v, prevhash = %0x",
				block.Number(), block.Hash(), block.ParentHash(), prev.Number(), prev.Hash())

			return 0, fmt.Errorf("Non contiguous block insert, number = %d, hash = %0x, parent = %0x, prevnumber = %v, prevhash = %0x",
				block.Number(), block.Hash(), block.ParentHash(), prev.Number(), prev.Hash())
		}
	}

	return bc.insertChain(chain, true)
}

func (bc *BlockChain) insertChain(chain []*types.Block, verifySeals bool) (int, error) {
	for i, blk := range chain {
		block := blk
		statedb, err := state.New(block.Root(), bc.stateCache)
		if err != nil {
			return i, err
		}
		if err := bc.writeBlockWithState(block, statedb); err != nil {
			return i, err
		}
	}
	return len(chain), nil
}

func (bc *BlockChain) writeBlockWithState(block *types.Block, statedb *state.StateDB) error {
	// 写入 Block
	rawdb.WriteBlock(bc.db, block)
	//log.Printf("Write block %d into databse. ", block.NumberU64())

	// 写入 StateDB
	root, err := statedb.Commit(true)
	if err != nil {
		return err
	}
	triedb := bc.stateCache.TrieDB()
	if err := triedb.Commit(root, false); err != nil {
		return err
	}
	log.Printf("Update Global-State[Trie] into database.")

	bc.insert(block)
	log.Printf("Update BlockChain's variables into database.")
	return nil
}

func (bc *BlockChain) insert(block *types.Block) {
	//log.Printf("block %d's canonical hash = %0x, block hash = %0x", block.NumberU64(), rawdb.ReadCanonicalHash(bc.db, block.NumberU64()), block.Hash())
	updateHeads := rawdb.ReadCanonicalHash(bc.db, block.NumberU64()) != block.Hash()

	rawdb.WriteCanonicalHash(bc.db, block.Hash(), block.NumberU64())
	//log.Printf("write block %d with canonical Hash.", block.NumberU64())
	rawdb.WriteHeadBlockHash(bc.db, block.Hash())
	//log.Printf("write head block = %d.", block.NumberU64())
	bc.currentBlock.Store(block)

	if updateHeads {
		bc.hc.SetCurrentHeader(block.Header())
		//log.Printf("write current header = %d.", block.Header().Number)
	}
	//currentBlockNumber := bc.CurrentBlock().NumberU64()
	//log.Printf("chain current block = %d, chain current header = %d", currentBlockNumber, bc.CurrentHeader().Number)
}

func (bc *BlockChain) repair(head **types.Block) error {
	for {
		if _, err := state.New((*head).Root(), bc.stateCache); err == nil {
			log.Printf("Rewound blockchain to past state, num = %v, hash = %v", (*head).Number(), (*head).Hash())
			return nil
		}

		block := bc.GetBlock((*head).ParentHash(), (*head).NumberU64()-1)
		if block == nil {
			return fmt.Errorf("missing block %d [%x]", (*head).NumberU64()-1, (*head).ParentHash())
		}
	}
}
