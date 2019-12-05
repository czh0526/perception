package p2p

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

func TestServer(t *testing.T) {

	server1 := makeServer("server_1", "/ip4/0.0.0.0/tcp/10001", "/Users/czh/Workspace/perception_datadir/node1/perception/nodekey")
	server2 := makeServer("server_2", "/ip4/0.0.0.0/tcp/10002", "/Users/czh/Workspace/perception_datadir/node2/perception/nodekey")
	server3 := makeServer("server_3", "/ip4/0.0.0.0/tcp/10003", "/Users/czh/Workspace/perception_datadir/node3/perception/nodekey")

	go server1.Start()
	go server2.Start()
	go server3.Start()

	select {}
}

var (
	bootstrapPeers = []string{
		"/ip4/0.0.0.0/tcp/10000/ipfs/16Uiu2HAm3f3prVE7MkqeuzDBNC5GoNkoeR91NVz7MaL3HGSD9tzD",
	}
)

func makeServer(name string, listenAddr string, keyfile string) *Server {
	// 构建 Config 对象
	config := Config{
		Name:           name,
		PrivateKey:     nil,
		MaxPeers:       30,
		ListenAddr:     listenAddr,
		BootstrapPeers: bootstrapPeers,
	}

	fillPrivateKey(&config, keyfile)

	// 构建 Server 对象
	server := NewServer(&config)

	// 初始化 Server 对象
	server.Init()

	// 构建一个测试的协议模块
	svc := makeDummyService("DummyService", 1, server.Host)
	svc.Init(server)

	server.Protocols = append(server.Protocols, Protocol{
		Name:    "DummyService",
		Version: 1,
		Run: func(remoteID peer.ID, initial bool) error {
			err := svc.StartTalk(remoteID, initial)
			fmt.Printf("Protocol.Run() error: %v \n", err)
			return err
		},
	})

	server.setupLocalNode()
	return server
}

func fillPrivateKey(config *Config, keyfilePath string) error {
	keyfile, err := os.Open(keyfilePath)
	if err != nil {
		return err
	}

	data := make([]byte, 36)
	if _, err := keyfile.Read(data); err != nil {
		return err
	}
	fmt.Printf("************ %0x \n", data)

	if config.PrivateKey, err = crypto.UnmarshalPrivateKey(data); err != nil {
		return err
	}
	return nil
}

type DummyService struct {
	Name    string
	Version uint
	peers   map[peer.ID]*Peer
	host    host.Host
}

func makeDummyService(name string, version uint, host host.Host) *DummyService {
	return &DummyService{
		Name:    name,
		Version: version,
		peers:   make(map[peer.ID]*Peer),
		host:    host,
	}
}

func (ps *DummyService) Init(server *Server) {
	server.Host.SetStreamHandler(ps.ProtocolID(), ps.StreamHandler)
}

func (ps *DummyService) StreamHandler(stream network.Stream) {
	go func() {
		rw := NewProtoRW(stream)
		for {
			msg, err := rw.ReadMsg()
			if err != nil {
				fmt.Printf("DummyService.StreamHandler, read msg error: %v \n", err)
			}

			err = rw.WriteMsg(msg)
			if err != nil {
				fmt.Printf("DummyService.StreamHandler, write msg error: %v \n", err)
			}
		}
	}()
}

func (ps *DummyService) ProtocolID() protocol.ID {
	return protocol.ID(fmt.Sprintf("/%s/%d", ps.Name, ps.Version))
}

func (ps *DummyService) StartTalk(remoteID peer.ID, initial bool) error {

	if _, exists := ps.peers[remoteID]; !exists {
		if !initial {
			return nil
		}

		fmt.Println("begin new stream from proton/1 ...")
		ProtocolID := protocol.ID(fmt.Sprintf("/%s/%d", ps.Name, ps.Version))
		stream, err := ps.host.NewStream(context.Background(), remoteID, ProtocolID)
		if err != nil {
			return err
		}
		log.Printf("\t\t initial new stream(%v) to %v \n", ProtocolID, remoteID.String())
		rw := NewProtoRW(stream)
		for {
			if err := Send(rw, uint64(44), "Dummy Service Msg."); err != nil {
				fmt.Printf("DummyService send msg error: %v \n", err)
			}
			time.Sleep(3 * time.Second)
		}
	}
	return nil
}
