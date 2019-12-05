package core

import (
	"math/big"
	"time"

	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/proton/chaindb"
	"github.com/czh0526/perception/proton/core/state"
	"github.com/czh0526/perception/proton/core/types"
)

type BlockGen struct {
	i       int
	parent  *types.Block
	chain   []*types.Block
	header  *types.Header
	statedb *state.StateDB
}

func GenerateChain(parent *types.Block, db chaindb.Database, n int, gen func(int, *BlockGen)) []*types.Block {

	blocks := make([]*types.Block, n)
	genblock := func(i int, parent *types.Block, statedb *state.StateDB) *types.Block {
		b := &BlockGen{i: i, chain: blocks, parent: parent, statedb: statedb}
		b.header = makeHeader(parent, statedb)

		if gen != nil {
			gen(i, b)
		}

		b.header.Root = statedb.IntermediateRoot(true)
		block := types.NewBlock(b.header, []*types.Transaction{}, []*types.Header{})
		return block
	}

	for i := 0; i < n; i++ {
		statedb, err := state.New(parent.Root(), state.NewDatabase(db))
		if err != nil {
			panic(err)
		}
		block := genblock(i, parent, statedb)
		blocks[i] = block
		parent = block
	}
	return blocks
}

func makeHeader(parent *types.Block, state *state.StateDB) *types.Header {
	var timestamp uint64
	if parent.Time() == 0 {
		timestamp = uint64(time.Now().Unix())
	} else {
		timestamp = parent.Time() + 10
	}

	return &types.Header{
		Root:       state.IntermediateRoot(true),
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		Time:       timestamp,
	}
}
