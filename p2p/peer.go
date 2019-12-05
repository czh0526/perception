package p2p

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/czh0526/perception/rlp"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	handshakeMsg = 0x00
	discMsg      = 0x01
	pingMsg      = 0x02
	pongMsg      = 0x03
)

type DiscReason uint

const (
	DiscRequested DiscReason = iota
	DiscNetworkError
	DiscProtocolError
	DiscUselessPeer
	DiscTooManyPeers
	DiscAlreadyConnected
	DiscIncompatibleVersion
	DiscInvalidIdentity
	DiscQuitting
	DiscUnexpectedIdentity
	DiscSelf
	DiscReadTimeout
	DiscSubprotocolError = 0x10
)

var discReasonToString = [...]string{
	DiscRequested:           "disconnect requested",
	DiscNetworkError:        "network error",
	DiscProtocolError:       "breach of protocol",
	DiscUselessPeer:         "useless peer",
	DiscTooManyPeers:        "too many peers",
	DiscAlreadyConnected:    "already connected",
	DiscIncompatibleVersion: "incompatible p2p protocol version",
	DiscInvalidIdentity:     "invalid node identity",
	DiscQuitting:            "client quitting",
	DiscUnexpectedIdentity:  "unexpected identity",
	DiscSelf:                "connected to self",
	DiscReadTimeout:         "read timeout",
	DiscSubprotocolError:    "subprotocol error",
}

func (d DiscReason) String() string {
	if len(discReasonToString) < int(d) {
		return fmt.Sprintf("unknown disconnect reason %d", d)
	}
	return discReasonToString[d]
}

func (d DiscReason) Error() string {
	return d.String()
}

const (
	baseProtocolVersion    = 1
	baseProtocolLength     = uint64(16)
	baseProtocolMaxMsgSize = 2 * 1024

	pingInterval = 10 * time.Second
)

type protoHandshake struct {
	Name    string
	Version uint64
	Caps    []Cap
	PubKey  []byte // secp256k1 public key
}

type Peer struct {
	ID         peer.ID
	RemoteAddr []ma.Multiaddr
	protoRW    *ProtoRW
	running    map[string]Protocol

	protoErr chan error
	disc     chan error    // 控制开关
	closed   chan struct{} // 指示灯
	wg       sync.WaitGroup
}

func matchProtocols(remoteId peer.ID, protocols []Protocol, caps []Cap) map[string]Protocol {
	sort.Sort(capsByNameAndVersion(caps))
	result := make(map[string]Protocol)

outer:
	for _, cap := range caps {
		for _, proto := range protocols {
			if cap.Name == proto.Name && cap.Version == proto.Version {
				result[cap.Name] = proto
			}

			continue outer
		}
	}
	return result
}

func newPeer(remoteAddr peer.AddrInfo, protoRW *ProtoRW, caps []Cap, protocols []Protocol) *Peer {
	log.Printf("2). match protocol\n")
	matches := matchProtocols(remoteAddr.ID, protocols, caps)
	for _, proto := range matches {
		log.Printf("\t\t protocol match ==> '/%s/%d' \n", proto.Name, proto.Version)
	}
	p := &Peer{
		ID:         remoteAddr.ID,
		RemoteAddr: remoteAddr.Addrs,
		protoRW:    protoRW,
		running:    matches,
		protoErr:   make(chan error, len(matches)+1), // protocols + pingLoop
		disc:       make(chan error),
		closed:     make(chan struct{}),
	}

	return p
}

func (p *Peer) run(initial bool) error {
	var (
		err    error
		reason error
	)

	p.wg.Add(2)
	go p.pingLoop()
	go p.readLoop()

	startProtocols(p.ID, p.running, initial)

loop:
	for {
		select {
		case err = <-p.protoErr:
			if err != nil {
				reason = err
				break loop
			}
		case err = <-p.disc:
			reason = fmt.Errorf("Disconnect conn, reason = %v.", err)
			break loop
		}
	}

	close(p.closed)
	p.protoRW.Close(reason)
	p.wg.Wait()
	log.Printf("p2p peer run() terminate, reason = %v", reason)
	return err
}

func (p *Peer) Disconnect(reason DiscReason) {
	select {
	case p.disc <- reason:
	case <-p.closed:
	}
}

func (p *Peer) pingLoop() {
	ping := time.NewTimer(pingInterval)
	defer p.wg.Done()
	defer ping.Stop()

	for {
		select {
		case <-ping.C:
			if err := SendItems(p.protoRW, pingMsg); err != nil {
				p.protoErr <- fmt.Errorf("pingLoop error: %v", err)
				return
			}
			ping.Reset(pingInterval)

		case <-p.closed:
			return
		}
	}
}

func (p *Peer) readLoop() {
	defer p.wg.Done()
	for {
		msg, err := p.protoRW.ReadMsg()
		if err != nil {
			p.protoErr <- err
			return
		}
		msg.ReceivedAt = time.Now()
		if err = p.handle(msg); err != nil {
			p.protoErr <- err
			return
		}
	}
}

func (p *Peer) handle(msg Msg) error {
	switch {
	case msg.Code == pingMsg:
		go SendItems(p.protoRW, pongMsg)

	case msg.Code == pongMsg:
		return nil

	case msg.Code == discMsg:
		var reason [1]DiscReason
		rlp.Decode(msg.Payload, &reason)
		return reason[0]

	default:
		return errors.New(fmt.Sprintf("unknown msg: %v \n", msg))
	}
	return nil
}
