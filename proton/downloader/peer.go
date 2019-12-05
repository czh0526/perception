package downloader

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/event"
)

const (
	maxLackingHashes  = 4096 // Maximum number of entries allowed on the list or lacking items
	measurementImpact = 0.1  // The impact a single measurement has on a peer's final throughput value.
)

var (
	errAlreadyFetching   = errors.New("aready fetching blocks from peer")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

type Peer interface {
	RequestBlocksByNumber(uint64, int) error
	RequestNodeData([]common.Hash) error
}

type peerConnection struct {
	id   string
	peer Peer

	blockIdle       int32
	stateIdle       int32
	blockThroughput float64
	stateThroughput float64
	rtt             time.Duration
	blockStarted    time.Time
	stateStarted    time.Time

	version int
	lock    sync.RWMutex
}

func newPeerConnection(id string, version int, peer Peer) *peerConnection {
	return &peerConnection{
		id:      id,
		peer:    peer,
		version: version,
	}
}

type peerSet struct {
	peers        map[string]*peerConnection
	newPeerFeed  event.Feed
	peerDropFeed event.Feed
	lock         sync.RWMutex
}

func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peerConnection),
	}
}

func (ps *peerSet) Register(p *peerConnection) error {
	ps.lock.Lock()

	if _, exists := ps.peers[p.id]; exists {
		ps.lock.Unlock()
		return errAlreadyRegistered
	}
	ps.peers[p.id] = p
	ps.lock.Unlock()

	ps.newPeerFeed.Send(p)
	return nil
}

func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()

	p, exists := ps.peers[id]
	if !exists {
		ps.lock.Unlock()
		return errNotRegistered
	}

	delete(ps.peers, id)
	ps.lock.Unlock()

	ps.peerDropFeed.Send(p)
	return nil
}

func (ps *peerSet) SubscribeNewPeers(ch chan<- *peerConnection) event.Subscription {
	return ps.newPeerFeed.Subscribe(ch)
}

func (ps *peerSet) SubscribePeerDrops(ch chan<- *peerConnection) event.Subscription {
	return ps.peerDropFeed.Subscribe(ch)
}

func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

func (ps *peerSet) Peer(id string) *peerConnection {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

func (ps *peerSet) BlockIdlePeers() ([]*peerConnection, int) {
	idle := func(p *peerConnection) bool {
		return atomic.LoadInt32(&p.blockIdle) == 0
	}
	throughput := func(p *peerConnection) float64 {
		p.lock.RLock()
		defer p.lock.RUnlock()
		return p.blockThroughput
	}
	return ps.idlePeers(idle, throughput)
}

func (ps *peerSet) NodeDataIdlePeers() ([]*peerConnection, int) {
	idle := func(p *peerConnection) bool {
		return atomic.LoadInt32(&p.stateIdle) == 0
	}
	throughput := func(p *peerConnection) float64 {
		p.lock.RLock()
		defer p.lock.RUnlock()
		return p.stateThroughput
	}
	return ps.idlePeers(idle, throughput)
}

func (ps *peerSet) idlePeers(idleCheck func(*peerConnection) bool, throughput func(*peerConnection) float64) ([]*peerConnection, int) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	// 筛选
	idle, total := make([]*peerConnection, 0, len(ps.peers)), 0
	for _, p := range ps.peers {
		if idleCheck(p) {
			idle = append(idle, p)
		}
		total++

	}

	// 排序
	for i := 0; i < len(idle); i++ {
		for j := i + 1; j < len(idle); j++ {
			if throughput(idle[i]) < throughput(idle[j]) {
				idle[i], idle[j] = idle[j], idle[i]
			}
		}
	}
	return idle, total
}

func (p *peerConnection) SetBlocksIdle(delivered int) {
	p.setIdle(p.blockStarted, delivered, &p.blockThroughput, &p.blockIdle)
}

func (p *peerConnection) SetNodeDataIdle(delivered int) {
	p.setIdle(p.stateStarted, delivered, &p.stateThroughput, &p.stateIdle)
}

func (p *peerConnection) setIdle(started time.Time, delivered int, throughput *float64, idle *int32) {
	defer atomic.StoreInt32(idle, 0)

	p.lock.Lock()
	defer p.lock.Unlock()

	if delivered == 0 {
		*throughput = 0
		return
	}

	elapsed := time.Since(started) + 1
	measured := float64(delivered) / (float64(elapsed) / float64(time.Second))

	*throughput = (1-measurementImpact)*(*throughput) + measurementImpact*measured
	p.rtt = time.Duration((1-measurementImpact)*float64(p.rtt) + measurementImpact*float64(elapsed))
}
