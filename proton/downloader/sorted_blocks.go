package downloader

import (
	"sort"

	"github.com/czh0526/perception/proton/core/types"
)

type sortedBlocks []*types.Block

func (sb sortedBlocks) Len() int {
	return len(sb)
}

func (sb sortedBlocks) Swap(i, j int) {
	sb[i], sb[j] = sb[j], sb[i]
}

func (sb sortedBlocks) Less(i, j int) bool {
	return sb[i].NumberU64() < sb[j].NumberU64()
}

func (sb sortedBlocks) insert(blocks []*types.Block) {
	sb = append(sb, blocks...)
	sort.Sort(sb)
}

func (sb sortedBlocks) pop(n int) {
	sb = sb[n:]
}

func (sb sortedBlocks) peekContinuousBlocks() []*types.Block {
	var parentNum uint64 = 0
	var (
		pos   int
		block *types.Block
	)
	for pos, block = range sb {
		if parentNum == 0 {
			parentNum = block.NumberU64()
		} else {
			if parentNum+1 != block.NumberU64() {
				break
			}
		}
	}

	return sb[:pos]
}
