package proton

import (
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/p2p"
	"github.com/czh0526/perception/proton/core/types"
	"github.com/libp2p/go-libp2p-core/network"
	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"
)

const (
	handshakeTimeout = 5 * time.Second
)

type peer struct {
	version     uint32
	remoteID    libp2p_peer.ID
	rw          *p2p.ProtoRW
	lock        sync.RWMutex
	head        common.Hash
	blockNumber *big.Int
}

func newPeer(version uint, stream network.Stream, remoteID libp2p_peer.ID) *peer {
	return &peer{
		version:  uint32(version),
		remoteID: remoteID,
		rw:       p2p.NewProtoRW(stream),
	}
}

func (p *peer) Identifier() string {
	return peerIdentifier(p.remoteID, p.version)
}

func peerIdentifier(remoteID libp2p_peer.ID, version uint32) string {
	return fmt.Sprintf("%s_%d", remoteID.String(), version)
}

func (p *peer) Handshake(network uint64, head common.Hash, blockNumber *big.Int, genesis common.Hash) error {
	var status statusData = statusData{}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := p2p.Send(p.rw, StatusMsg, &statusData{
			ProtocolVersion: uint32(p.version),
			NetworkId:       network,
			GenesisBlock:    genesis,
			CurrentBlock:    head,
			BlockNumber:     blockNumber,
		}); err != nil {
			log.Printf("\t\t send Proton status msg, error = %v \n", err)
		}
	}()

	if err := p.readStatus(network, &status, genesis); err != nil {
		log.Printf("\t\t read Proton status msg, error = %v \n", err)
		return err
	}
	log.Printf("\t\t status.NetworkId ==>  %v - %v \n", network, status.NetworkId)
	log.Printf("\t\t status.GenesisBlock ==> %0x - %0x \n", genesis, status.GenesisBlock)
	log.Printf("\t\t status.CurrentBlock ==> %0x - %0x \n", head, status.CurrentBlock)
	log.Printf("\t\t status.BlockNumber ==> %d - %0d \n", blockNumber, status.BlockNumber)

	p.head, p.blockNumber = status.CurrentBlock, status.BlockNumber
	wg.Wait()
	return nil
}

func (p *peer) readStatus(network uint64, status *statusData, genesis common.Hash) (err error) {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != StatusMsg {
		return fmt.Errorf("first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}
	if msg.Size > protocolMaxMsgSize {
		return fmt.Errorf("msg size is too big (%v > %v)", msg.Size, protocolMaxMsgSize)
	}

	if err := msg.Decode(&status); err != nil {
		return fmt.Errorf("handshake msg decode error: %v", err)
	}

	if status.GenesisBlock != genesis {
		return fmt.Errorf("genesis block mismatch: %x != %x", status.GenesisBlock[:8], genesis[:8])
	}

	if status.NetworkId != network {
		return fmt.Errorf("network id mismatch, %x != %x", status.NetworkId, network)
	}

	if status.ProtocolVersion != p.version {
		return fmt.Errorf("version mismatch, %d != %d", status.ProtocolVersion, p.version)
	}

	return nil
}

func (p *peer) close(err error) {
	p.rw.Close(err)
}

func (p *peer) Head() (hash common.Hash, number *big.Int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.head[:])
	return hash, new(big.Int).Set(p.blockNumber)
}

func (p *peer) RequestBlocksByNumber(from uint64, amount int) error {
	err := p2p.Send(p.rw, GetBlocksMsg, &getBlocksData{From: from, Amount: uint64(amount)})
	return err
}

func (p *peer) SendBlocks(blocks []*types.Block) error {
	return p2p.Send(p.rw, BlocksMsg, blocks)
}

func (p *peer) RequestNodeData(hashes []common.Hash) error {
	return p2p.Send(p.rw, GetNodeDataMsg, hashes)
}
