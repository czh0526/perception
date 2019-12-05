package core

import (
	"errors"
	"sync/atomic"

	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/proton/chaindb"
	"github.com/czh0526/perception/proton/core/rawdb"
	"github.com/czh0526/perception/proton/core/types"
)

type HeaderChain struct {
	chainDb chaindb.Database

	genesisHeader *types.Header

	currentHeader     atomic.Value
	currentHeaderHash common.Hash
}

type (
	UpdateHeadBlocksCallback func(chaindb.KeyValueWriter, *types.Header)

	DeleteBlockContentCallback func(chaindb.KeyValueWriter, common.Hash, uint64)
)

func NewHeaderChain(chainDb chaindb.Database) (*HeaderChain, error) {
	/*
		seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			return nil, err
		}
	*/

	hc := &HeaderChain{
		chainDb: chainDb,
	}
	hc.genesisHeader = hc.GetHeaderByNumber(0)
	if hc.genesisHeader == nil {
		return nil, errors.New("genesis header can't be found.")
	}

	hc.currentHeader.Store(hc.genesisHeader)
	if head := rawdb.ReadHeadBlockHash(chainDb); head != (common.Hash{}) {
		if chead := hc.GetHeaderByHash(head); chead != nil {
			hc.currentHeader.Store(chead)
		}
	}

	hc.currentHeaderHash = hc.CurrentHeader().Hash()
	return hc, nil
}

func (hc *HeaderChain) CurrentHeader() *types.Header {
	return hc.currentHeader.Load().(*types.Header)
}

func (hc *HeaderChain) SetCurrentHeader(head *types.Header) {
	rawdb.WriteHeadHeaderHash(hc.chainDb, head.Hash())
	hc.currentHeader.Store(head)
	hc.currentHeaderHash = head.Hash()
}

func (hc *HeaderChain) GetHeaderByHash(hash common.Hash) *types.Header {
	number := rawdb.ReadHeaderNumber(hc.chainDb, hash)
	if number == nil {
		return nil
	}
	return hc.GetHeader(hash, *number)
}

func (hc *HeaderChain) GetHeaderByNumber(number uint64) *types.Header {
	hash := rawdb.ReadCanonicalHash(hc.chainDb, number)
	if (hash == common.Hash{}) {
		return nil
	}
	return hc.GetHeader(hash, number)
}

func (hc *HeaderChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	header := rawdb.ReadHeader(hc.chainDb, hash, number)
	if header == nil {
		return nil
	}
	return header
}

func (hc *HeaderChain) SetGenesis(head *types.Header) {
	hc.genesisHeader = head
}

func (hc *HeaderChain) SetHead(head uint64, updateFn UpdateHeadBlocksCallback, delFn DeleteBlockContentCallback) {
	var (
		parentHash common.Hash
		batch      = hc.chainDb.NewBatch()
	)
	for hdr := hc.CurrentHeader(); hdr != nil && hdr.Number.Uint64() > head; hdr = hc.CurrentHeader() {
		hash, num := hdr.Hash(), hdr.Number.Uint64()

		// 得到 parent header && parent hash
		parent := hc.GetHeader(hdr.ParentHash, num-1)
		if parent == nil {
			parent = hc.genesisHeader
		}
		parentHash = hdr.ParentHash

		// reset head block
		if updateFn != nil {
			updateFn(hc.chainDb, parent)
		}

		// reset head header
		rawdb.WriteHeadHeaderHash(hc.chainDb, parentHash)

		if delFn != nil {
			delFn(batch, hash, num)
		}

		rawdb.DeleteHeader(batch, hash, num)
		rawdb.DeleteCanonicalHash(batch, num)

		hc.currentHeader.Store(parent)
		hc.currentHeaderHash = parentHash
	}
	batch.Write()
}
