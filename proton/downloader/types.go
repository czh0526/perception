package downloader

import (
	"fmt"

	"github.com/czh0526/perception/proton/core/types"
)

type peerDropFn func(id string)

type dataPack interface {
	PeerId() string
	Items() int
	Stats() string
}

type blockPack struct {
	peerID string
	blocks []*types.Block
}

func (p *blockPack) PeerId() string {
	return p.peerID
}

func (p *blockPack) Items() int {
	return len(p.blocks)
}

func (p *blockPack) Stats() string {
	return fmt.Sprintf("%d", len(p.blocks))
}
