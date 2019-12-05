package downloader

import (
	"errors"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/proton/chaindb"
	"github.com/czh0526/perception/proton/core/types"
)

var (
	MaxBlockFetch = 128
)

var (
	errBusy                    = errors.New("busy")
	errUnknownPeer             = errors.New("peer is unknown or unhealthy")
	errBadPeer                 = errors.New("action from bad peer ignored")
	errStallingPeer            = errors.New("peer is stalling")
	errUnsyncedPeer            = errors.New("unsynced peer")
	errNoPeers                 = errors.New("no peers to keep download active")
	errTimeout                 = errors.New("timeout")
	errEmptyHeaderSet          = errors.New("empty header set by peer")
	errPeersUnavailable        = errors.New("no peers available or all tried for download")
	errInvalidAncestor         = errors.New("retrieved ancestor is invalid")
	errInvalidChain            = errors.New("retrieved hash chain is invalid")
	errInvalidBlock            = errors.New("retrieved block is invalid")
	errInvalidBody             = errors.New("retrieved block body is invalid")
	errInvalidReceipt          = errors.New("retrieved receipt is invalid")
	errCancelBlockFetch        = errors.New("block download canceled (requested)")
	errCancelHeaderFetch       = errors.New("block header download canceled (requested)")
	errCancelBodyFetch         = errors.New("block body download canceled (requested)")
	errCancelReceiptFetch      = errors.New("receipt download canceled (requested)")
	errCancelStateFetch        = errors.New("state data download canceled (requested)")
	errCancelHeaderProcessing  = errors.New("header processing canceled (requested)")
	errCancelContentProcessing = errors.New("content processing canceled (requested)")
	errNoSyncActive            = errors.New("no sync active")
	errTooOld                  = errors.New("peer doesn't speak recent enough protocol version (need version >= 62)")
)

type Downloader struct {
	peers *peerSet

	blockchain BlockChain
	chaindb    chaindb.Database

	dropPeer peerDropFn
	queue    sortedBlocks

	blockCh     chan dataPack
	blockProcCh chan []*types.Block

	cancelPeer string
	cancelCh   chan struct{}
	cancelLock sync.RWMutex
	cancelWg   sync.WaitGroup
}

type BlockChain interface {
	CurrentHeader() *types.Header
	CurrentBlock() *types.Block
	InsertChain([]*types.Block) (int, error)
}

func New(chainDb chaindb.Database, blockChain BlockChain, dropPeer peerDropFn) *Downloader {
	dl := &Downloader{
		chaindb:     chainDb,
		blockchain:  blockChain,
		dropPeer:    dropPeer,
		blockCh:     make(chan dataPack, 1),
		blockProcCh: make(chan []*types.Block, 1),
		peers:       newPeerSet(),
	}
	return dl
}

func (d *Downloader) RegisterPeer(id string, version int, peer Peer) error {
	if err := d.peers.Register(newPeerConnection(id, version, peer)); err != nil {
		return err
	}
	return nil
}

func (d *Downloader) UnregisterPeer(id string) error {
	if err := d.peers.Unregister(id); err != nil {
		return err
	}
	d.cancelLock.RLock()
	master := id == d.cancelPeer
	d.cancelLock.RUnlock()

	if master {
		d.cancel()
	}
	return nil
}

func (d *Downloader) Synchronise(id string, head common.Hash, number *big.Int) error {
	err := d.synchronise(id, head, number)
	switch err {
	case nil:
	case errBusy:
	case errTimeout, errBadPeer, errStallingPeer, errUnsyncedPeer,
		errEmptyHeaderSet, errPeersUnavailable, errTooOld, errInvalidChain:
		if d.dropPeer != nil {
			d.dropPeer(id)
		}
	default:
		log.Printf("Synchronisation failed, retrying, err = %v \n", err)
	}
	return err
}

func (d *Downloader) synchronise(id string, hash common.Hash, number *big.Int) error {

	d.cancelLock.Lock()
	d.cancelCh = make(chan struct{})
	d.cancelPeer = id
	d.cancelLock.Unlock()
	defer d.Cancel()

	p := d.peers.Peer(id)
	if p == nil {
		return errUnknownPeer
	}
	return d.syncWithPeer(p, hash, number)
}

func (d *Downloader) syncWithPeer(p *peerConnection, remoteHeadHash common.Hash, remoteHeadNumber *big.Int) (err error) {

	localHeadNumber := d.blockchain.CurrentHeader().Number
	log.Printf("5). Downloader.syncWithPeer() started, local head = %v, remote head = %v \n", localHeadNumber, remoteHeadNumber)
	origin := new(big.Int).Add(localHeadNumber, big.NewInt(1))
	if localHeadNumber.Cmp(remoteHeadNumber) > 0 {
		return nil
	}

	fetchers := []func() error{
		func() error {
			return d.fetchBlocks(p, origin.Uint64(), remoteHeadNumber.Uint64())
		},
	}

	return d.spawnSync(fetchers)
}

func (d *Downloader) Cancel() {
	d.cancel()
	d.cancelWg.Wait()
}

func (d *Downloader) cancel() {
	d.cancelLock.Lock()
	if d.cancelCh != nil {
		select {
		case <-d.cancelCh:
		default:
			close(d.cancelCh)
		}
	}
	d.cancelLock.Unlock()
}

func (d *Downloader) spawnSync(fetchers []func() error) error {
	errc := make(chan error, len(fetchers))
	// 启动例程组
	d.cancelWg.Add(len(fetchers))
	for _, fn := range fetchers {
		fn := fn
		go func() {
			defer d.cancelWg.Done()
			errc <- fn()
		}()
	}

	// 监测到任意一个例程出错后，退出循环
	var err error
	for i := 0; i < len(fetchers); i++ {
		if err = <-errc; err != nil {
			break
		}
	}

	// 取消本次 Downloading
	d.Cancel()
	return err
}

func (d *Downloader) fetchBlocks(p *peerConnection, from uint64, end uint64) error {
	log.Printf("\t\t downloader fetchBlocks(%d -> %d) ", from, end)
	defer log.Printf("\t\t fetchBlocks() terminated.")

	timeout := time.NewTimer(0)
	<-timeout.C
	defer timeout.Stop()

	var ttl time.Duration
	getBlocks := func(from uint64, end uint64) {
		ttl = d.requestTTL()
		timeout.Reset(ttl)

		total := int(end - from + 1)
		if total > MaxBlockFetch {
			total = MaxBlockFetch
		}
		go func() {
			p.peer.RequestBlocksByNumber(from, total)
		}()
	}

	getBlocks(from, end)
	for {
		select {
		case <-d.cancelCh:
			return errCancelBlockFetch

		case packet := <-d.blockCh:
			if packet.PeerId() != p.id {
				log.Printf("Received block from incorrect peer: %v \n", packet.PeerId())
				break
			}

			timeout.Stop()

			if packet.Items() == 0 {
				log.Println("No more headers available")
				select {
				case d.blockProcCh <- nil:
					return nil
				case <-d.cancelCh:
					return errCancelBlockFetch
				}
			}

			blocks := packet.(*blockPack).blocks
			if len(blocks) > 0 {
				d.processBlocks(blocks)
				from += uint64(len(blocks))
				if from < end {
					getBlocks(from, end)
				} else {
					return nil
				}
			} else {
				select {
				case <-time.After(3 * time.Second):
					if from < end {
						getBlocks(from, end)
					} else {
						return nil
					}
				case <-d.cancelCh:
					return errCancelBlockFetch
				}
			}
		}
	}
}

func (d *Downloader) requestTTL() time.Duration {
	return time.Second
}

func (d *Downloader) DeliverBlocks(id string, blocks []*types.Block) (err error) {
	log.Printf("\t\t downloader.DeliverBlocks (%d -> %d) \n", blocks[0].NumberU64(), blocks[len(blocks)-1].NumberU64())
	return d.deliver(id, d.blockCh, &blockPack{id, blocks})
}

func (d *Downloader) deliver(id string, destCh chan dataPack, packet dataPack) (err error) {
	d.cancelLock.RLock()
	cancel := d.cancelCh
	d.cancelLock.RUnlock()
	if cancel == nil {
		return errNoSyncActive
	}
	select {
	case destCh <- packet:
		return nil
	case <-cancel:
		return errNoSyncActive
	}
}

func (d *Downloader) processBlocks(blocks []*types.Block) {

	if len(blocks) == 0 {
		return
	}

	inserted := 0
	log.Printf("block[0].Number = %v, chain current block = %v", blocks[0].NumberU64(), d.blockchain.CurrentBlock().NumberU64())
	if blocks[0].NumberU64() == d.blockchain.CurrentBlock().NumberU64()+1 {
		inserted, _ = d.blockchain.InsertChain(blocks)
		log.Printf("inserted = %v", inserted)
		if inserted > 0 {
			futureBlocks := d.queue.peekContinuousBlocks()
			n, _ := d.blockchain.InsertChain(futureBlocks)
			log.Printf("n = %v", n)
			d.queue.pop(n)
		}
	}
	log.Printf("blockchain head block is: %v \n", d.blockchain.CurrentBlock().NumberU64())
	log.Printf("blockchain head header is: %v \n", d.blockchain.CurrentHeader().Number)

	if inserted == 0 {
		d.queue.insert(blocks)
	}
}
