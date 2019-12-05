package proton

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/czh0526/perception/node"
	"github.com/czh0526/perception/p2p"
	"github.com/czh0526/perception/proton/chaindb"
	"github.com/czh0526/perception/proton/core"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

type Proton struct {
	config *Config

	networkID       uint64
	protocolManager *ProtocolManager
	host            host.Host

	lock sync.RWMutex
}

func New(ctx *node.ServiceContext, conf *Config) (*Proton, error) {
	var (
		networkID  = conf.NetworkId
		err        error
		chainDb    chaindb.Database
		blockchain *core.BlockChain
	)

	chainDb, err = ctx.OpenDatabase("chaindata", conf.DatabaseCache, conf.DatabaseHandles, "eth/db/chaindata/")
	if err != nil {
		return nil, err
	}

	blockchain, err = core.NewBlockChain(chainDb)
	if err != nil {
		return nil, err
	}

	protocolManager, err := NewProtocolManager(networkID, chainDb, blockchain)
	if err != nil {
		return nil, err
	}

	proton := &Proton{
		config:          conf,
		networkID:       networkID,
		protocolManager: protocolManager,
	}

	return proton, nil
}

func (self *Proton) Start(p2pServer *p2p.Server) error {
	<-p2pServer.Inited
	self.host = p2pServer.Host

	var protoStr string
	protoStr = fmt.Sprintf("/%s/%d", ProtocolName, Protocol_V1)

	self.host.SetStreamHandler(protocol.ID(protoStr), self.streamHandlerV1)
	log.Printf("\t set stream handler for '%s' \n", protoStr)

	self.protocolManager.Start(30)
	return nil
}

func (self *Proton) Stop() error {
	fmt.Println("Service Proton stopped.")
	return nil
}

func (self *Proton) Protocols() []p2p.Protocol {
	protos := make([]p2p.Protocol, 0, len(ProtocolVersions))
	protos = append(protos, self.makeProtocol(Protocol_V1))

	return protos
}

func (self *Proton) makeProtocol(version uint) p2p.Protocol {
	return p2p.Protocol{
		Name:    ProtocolName,
		Version: version,
		Run: func(remoteID libp2p_peer.ID, initial bool) error {
			var (
				p      *peer
				exists bool
			)
			peerIdent := peerIdentifier(remoteID, uint32(version))
			// 已经存在额连接，不处理
			if p, exists = self.protocolManager.peers[peerIdent]; exists {
				return fmt.Errorf("protocolManager has a peer <%s> conn before ...", peerIdent)
			}
			// 应该被动等待的链接，不处理
			if !initial {
				return nil
			}

			stream, err := self.host.NewStream(context.Background(), remoteID, ProtocolID)
			if err != nil {
				log.Printf("libp2p NewStream(%v) error: %v \n", ProtocolID, err)
				return err
			}

			log.Printf("\t\t initial new stream(%v) to %v \n", ProtocolID, remoteID.String())
			p = newPeer(version, stream, remoteID)
			return self.protocolManager.handle(p)
		},
	}
}

func (self *Proton) streamHandlerV1(stream network.Stream) {
	log.Printf("\t\t accept new stream from %v \n", stream.Conn().RemotePeer())
	remoteID := stream.Conn().RemotePeer()
	p := newPeer(Protocol_V1, stream, remoteID)
	//self.protocolManager.addpeer <- p
	go self.protocolManager.handle(p)
}
