package core

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/czh0526/perception/common/hexutil"
	"github.com/czh0526/perception/common/math"
	"github.com/czh0526/perception/proton/core/rawdb"
	"github.com/czh0526/perception/proton/core/state"
	"github.com/czh0526/perception/proton/core/types"

	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/proton/chaindb"
)

type Genesis struct {
	Nonce      uint64         `json:"nonce"`
	Timestamp  uint64         `json:"timestamp"`
	ExtraData  []byte         `json:"extraData"`
	GasLimit   uint64         `json:"gasLimit" gencodec:"required"`
	Difficulty *big.Int       `json:"difficulty" gencodec:"required"`
	Mixhash    common.Hash    `json:"mixHash"`
	Coinbase   common.Address `json:"coinbase"`
	Alloc      GenesisAlloc   `json:"alloc" gencodec:"required"`
	Number     uint64         `json:"number"`
	GasUsed    uint64         `json:"gasUsed"`
	ParentHash common.Hash    `json:"parentHash"`
}

type GenesisAlloc map[common.Address]GenesisAccount

type GenesisAccount struct {
	Code    []byte                      `json:"code,omitempty"`
	Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
	Balance *big.Int                    `json:"balance" gencodec:"required"`
	Nonce   uint64                      `json:"nonce,omitempty"`
}

type genesisSpecMarshaling struct {
	Nonce      math.HexOrDecimal64
	Timestamp  math.HexOrDecimal64
	ExtraData  hexutil.Bytes
	GasLimit   math.HexOrDecimal64
	GasUsed    math.HexOrDecimal64
	Number     math.HexOrDecimal64
	Difficulty *math.HexOrDecimal256
	Alloc      map[common.UnprefixedAddress]GenesisAccount
}

type genesisAccountMarshaling struct {
	Code    hexutil.Bytes
	Balance *math.HexOrDecimal256
	Nonce   math.HexOrDecimal64
	Storage map[storageJSON]storageJSON
}

type storageJSON common.Hash

func SetupGenesisBlock(db chaindb.Database, genesis *Genesis) (common.Hash, error) {
	return SetupGenesisBlockWithOverride(db, genesis)
}

func SetupGenesisBlockWithOverride(db chaindb.Database, genesis *Genesis) (common.Hash, error) {

	stored := rawdb.ReadCanonicalHash(db, 0)
	if (stored == common.Hash{}) {
		if genesis == nil {
			return common.Hash{}, errors.New("genesis block is nil.")
		}
		block, err := genesis.Commit(db)
		if err != nil {
			return common.Hash{}, err
		}
		return block.Hash(), nil
	}

	return common.Hash{}, errors.New("genesis block has already existed in database")
}

func (g *Genesis) Commit(db chaindb.Database) (*types.Block, error) {
	// write stateDB
	block := g.ToBlock(db)
	if block.Number().Sign() != 0 {
		return nil, fmt.Errorf("can't commit genesis block with number > 0")
	}

	// write block
	rawdb.WriteBlock(db, block)
	// write blockchain
	rawdb.WriteCanonicalHash(db, block.Hash(), block.NumberU64())
	rawdb.WriteHeadBlockHash(db, block.Hash())
	rawdb.WriteHeadHeaderHash(db, block.Hash())
	return block, nil
}

func (g *Genesis) ToBlock(db chaindb.Database) *types.Block {
	if db == nil {
		panic("chaindb is nil.")
	}
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(db))
	for addr, account := range g.Alloc {
		statedb.AddBalance(addr, account.Balance)
		statedb.SetCode(addr, account.Code)
		statedb.SetNonce(addr, account.Nonce)
		for key, value := range account.Storage {
			statedb.SetState(addr, key, value)
		}
	}
	root := statedb.IntermediateRoot(false)
	head := &types.Header{
		Number:     new(big.Int).SetUint64(g.Number),
		ParentHash: g.ParentHash,
		Root:       root,
	}

	statedb.Commit(false)
	statedb.Database().TrieDB().Commit(root, true)

	return types.NewBlock(head, nil, nil)
}
