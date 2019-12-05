package p2p

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	rt "github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	kad_dht "github.com/libp2p/go-libp2p-kad-dht"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	TOPIC_PERCEPTION = "/perception/msg/0.0.1"
	PROTO_PERCEPTION = protocol.ID(TOPIC_PERCEPTION)
)

type Server struct {
	Config           *Config
	Host             host.Host
	Routing          rt.Routing
	RoutingDiscovery *discovery.RoutingDiscovery

	Protocols    []Protocol
	ourHandshake *protoHandshake
	Peers        map[peer.ID]*Peer
	peerChan     chan peer.AddrInfo

	lock   sync.Mutex
	Inited chan struct{}
}

func NewServer(config *Config) *Server {
	return &Server{
		Config:   config,
		Peers:    make(map[peer.ID]*Peer),
		peerChan: make(chan peer.AddrInfo),
		Inited:   make(chan struct{}),
	}
}

func (srv *Server) Init() error {
	ctx := context.Background()

	privKey := srv.Config.PrivateKey
	if privKey == nil {
		return errors.New("Server.Private must be set to a non-nil key.")
	}

	if err := srv.setupLocalNode(); err != nil {
		return fmt.Errorf("p2pServer seupLocalNode error: %v", err)
	}

	listenAddr, err := ma.NewMultiaddr(srv.Config.ListenAddr)
	if err != nil {
		return fmt.Errorf("create listen address error: %v", err)
	}

	// 构建 libp2p Host
	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(listenAddr),
		libp2p.Identity(privKey),
	)
	if err != nil {
		return fmt.Errorf("new libp2p host error: %v", err)
	}

	// 启动 KAD_DHT 模块
	kadDHT, err := kad_dht.New(ctx, host)
	if err != nil {
		return fmt.Errorf("new KAD_DHT error: %v", err)
	}
	if err = kadDHT.Bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap KAD_DHT error: %v", err)
	}

	// 连接 bootnodes, 构建libp2p swarm网络
	srv.connectBootNodes(ctx, host, srv.Config.BootstrapPeers)

	// 公布自己的身份，等待连接
	routingDiscovery := discovery.NewRoutingDiscovery(kadDHT)
	routingDiscovery.Advertise(ctx, TOPIC_PERCEPTION)
	log.Println("Announcing ourselves")
	log.Printf("\t <%s> ==> %v \n", host.ID(), TOPIC_PERCEPTION)

	srv.Host = host
	srv.Routing = kadDHT
	srv.RoutingDiscovery = routingDiscovery
	close(srv.Inited)

	return nil
}

func (srv *Server) Start() {

	ctx := context.Background()
	host := srv.Host

	// 设置协议处理部分
	host.SetStreamHandler(PROTO_PERCEPTION, srv.streamHandler)
	go srv.findPeers(ctx, TOPIC_PERCEPTION)

	for {
		//log.Printf("\t\t server has %d peers. \n", len(srv.Peers))
		select {
		case addrInfo := <-srv.peerChan:
			if _, exists := srv.Peers[addrInfo.ID]; exists {
				continue
			}

			log.Printf("\t find new peer ==> %v:%v \n", addrInfo.ID, addrInfo.Addrs)
			// 与同一主题组的 Host 建立连接
			stream, err := host.NewStream(ctx, addrInfo.ID, PROTO_PERCEPTION)
			if err != nil {
				fmt.Printf("Establish a stream to <%v> failed. \n", addrInfo.Addrs)
				break
			}
			log.Printf("\t new stream ==> %v \n", stream.Conn().RemoteMultiaddr())

			peer, err := createPeer(stream, srv.ourHandshake, srv.Protocols)
			if err != nil {
				fmt.Printf("create p2p Peer error: %v. \n", err)
				break
			}
			srv.Peers[peer.ID] = peer
			go srv.runPeer(peer, true)
		}
	}
}

func (srv *Server) connectBootNodes(ctx context.Context, host host.Host, addrs []string) {
	for _, addr := range addrs {
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			continue
		}
		addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			continue
		}
		go func(addrInfo *peer.AddrInfo) {
			if err := host.Connect(ctx, *addrInfo); err != nil {
				log.Printf("\t host connect bootnode<%v> error: %v \n", addrInfo.String(), err)
				return
			}
			log.Printf("\t host connect bootnode<%v> success. \n", addrInfo.String())
		}(addrInfo)
	}
}

func (srv *Server) findPeers(ctx context.Context, topic string) {

	for {
		peerChan, err := srv.RoutingDiscovery.FindPeers(ctx, topic)
		if err != nil {
			log.Printf("routingDiscovery.FindPeers() error: %v \n", err)
			continue
		}

		for addrInfo := range peerChan {
			if srv.Host.ID() == addrInfo.ID {
				continue
			}

			//log.Printf("find peer ==> %v:%v \n", addrInfo.ID, addrInfo.Addrs)

			// 将远端地址写入处理管道
			srv.peerChan <- addrInfo
		}
		time.Sleep(10 * time.Second)
	}

}

func createPeer(stream network.Stream, handshake *protoHandshake, protos []Protocol) (*Peer, error) {
	conn := NewProtoRW(stream)
	remoteCaps, _, err := fetchRemoteCaps(conn, handshake)
	if err != nil {
		fmt.Printf("fetch remote peer <%v> caps failed. \n", stream.Conn().RemoteMultiaddr())
		return nil, err
	}

	addrInfo := peer.AddrInfo{
		ID:    stream.Conn().RemotePeer(),
		Addrs: []ma.Multiaddr{stream.Conn().RemoteMultiaddr()},
	}
	return newPeer(addrInfo, conn, remoteCaps, protos), nil

}

func (srv *Server) streamHandler(stream network.Stream) {
	p, err := createPeer(stream, srv.ourHandshake, srv.Protocols)
	if err != nil {
		log.Printf("p2pServer handle stream error: %v", err)
		return
	}

	srv.Peers[p.ID] = p
	go srv.runPeer(p, false)
}

func (srv *Server) setupLocalNode() error {
	pubkey, _ := srv.Config.PrivateKey.GetPublic().Raw()
	srv.ourHandshake = &protoHandshake{Name: srv.Config.Name, Version: baseProtocolVersion, PubKey: pubkey}
	for _, p := range srv.Protocols {
		srv.ourHandshake.Caps = append(srv.ourHandshake.Caps, p.Cap())
	}
	sort.Sort(capsByNameAndVersion(srv.ourHandshake.Caps))

	return nil
}

func fetchRemoteCaps(rw *ProtoRW, handshake *protoHandshake) ([]Cap, string, error) {
	remoteHandshake, err := perceptionHandshake(rw, handshake)
	if err != nil {
		fmt.Println("Failed proto handshake", "err", err)
		return nil, "", err
	}
	return remoteHandshake.Caps, remoteHandshake.Name, nil
}

func (srv *Server) runPeer(p *Peer, initial bool) {
	if err := p.run(initial); err != nil {
		log.Printf("peer <%v> exit with error: %v\n", p.ID, err)
	}
	srv.lock.Lock()
	defer srv.lock.Unlock()
	defer func() {
		delete(srv.Peers, p.ID)
		log.Printf("p2p server delete a peer.")
	}()
}

func (srv *Server) Stop() {
	srv.lock.Lock()
	srv.Host.Close()
	srv.lock.Unlock()
}
