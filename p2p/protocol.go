package p2p

import (
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p-core/peer"
)

type Protocol struct {
	Name    string
	Version uint
	Run     func(remoteID peer.ID, initial bool) error
}

func (p Protocol) Cap() Cap {
	return Cap{p.Name, p.Version}
}

type Cap struct {
	Name    string
	Version uint
}

func (cap Cap) String() string {
	return fmt.Sprintf("%s/%d", cap.Name, cap.Version)
}

type capsByNameAndVersion []Cap

func (cs capsByNameAndVersion) Len() int      { return len(cs) }
func (cs capsByNameAndVersion) Swap(i, j int) { cs[i], cs[j] = cs[j], cs[i] }
func (cs capsByNameAndVersion) Less(i, j int) bool {
	return cs[i].Name < cs[j].Name || (cs[i].Name == cs[j].Name && cs[i].Version < cs[j].Version)
}

func startProtocols(peerId peer.ID, protos map[string]Protocol, initial bool) {
	// 启动全部的协议模块
	for _, proto := range protos {
		proto := proto
		go func() {
			err := proto.Run(peerId, initial)
			log.Printf("Protocol %s/%d exit with error: %v", proto.Name, proto.Version, err)
		}()
	}
}
