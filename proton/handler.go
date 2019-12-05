package proton

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/czh0526/perception/p2p"
	"github.com/czh0526/perception/proton/chaindb"
	"github.com/czh0526/perception/proton/core"
	"github.com/czh0526/perception/proton/core/types"
	"github.com/czh0526/perception/proton/downloader"
)

type ProtocolManager struct {
	networkID  uint64
	blockchain *core.BlockChain
	maxPeers   int
	peers      map[string]*peer
	downloader *downloader.Downloader
}

func NewProtocolManager(networkID uint64, chainDb chaindb.Database, blockChain *core.BlockChain) (*ProtocolManager, error) {
	manager := &ProtocolManager{
		networkID:  networkID,
		blockchain: blockChain,
		peers:      make(map[string]*peer),
	}
	manager.downloader = downloader.New(chainDb, blockChain, manager.removePeer)

	return manager, nil
}

func (pm *ProtocolManager) Start(maxPeers int) {
	pm.maxPeers = maxPeers

	go pm.syncer()
}

func (pm *ProtocolManager) Stop() {
}

func (pm *ProtocolManager) syncer() {

	forceSync := time.NewTicker(10 * time.Second)
	defer forceSync.Stop()
	for {
		select {
		case <-forceSync.C:
			go pm.synchronise(bestPeer(pm.peers))
		}
	}
}

func bestPeer(peers map[string]*peer) *peer {
	var (
		bestPeer *peer
		bestNum  *big.Int = big.NewInt(0)
	)
	for _, p := range peers {
		if _, num := p.Head(); bestNum.Cmp(big.NewInt(0)) == 0 || num.Cmp(bestNum) > 0 {
			bestPeer, bestNum = p, num
		}
	}
	return bestPeer
}

func (pm *ProtocolManager) removePeer(id string) {
	// Short circuit if the peer was already removed
	peer, exists := pm.peers[id]
	if !exists {
		return
	}

	// Unregister the peer from the downloader and Ethereum peer set
	pm.downloader.UnregisterPeer(id)
	delete(pm.peers, id)

	// Hard disconnect at the networking layer
	if peer != nil {
		peer.close(p2p.DiscUselessPeer)
	}
}

func (pm *ProtocolManager) synchronise(peer *peer) {
	if peer == nil {
		return
	}

	// 读取对端的最新区块
	pHead, pNumber := peer.Head()
	// 读取本地的最新区块
	headBlk := pm.blockchain.CurrentBlock()

	log.Printf("[before downloader.Synchronise]: local head = %v, remote head = %v \n", headBlk.NumberU64(), pNumber.Uint64())
	defer func() {
		headBlk = pm.blockchain.CurrentBlock()
		log.Printf("[after downloader.Synchronise]: local head = %v, remote head = %v \n", headBlk.NumberU64(), pNumber.Uint64())
	}()

	// 判断本地链条长度，决定是否启动下载区块的流程。
	if headBlk.Number().Cmp(pNumber) >= 0 {
		return
	}

	if err := pm.downloader.Synchronise(peer.Identifier(), pHead, pNumber); err != nil {
		return
	}

	if headBlk.NumberU64() > 0 {
		fmt.Println(">>>>>>>>>>>>>. 区块同步完成后的处理，需要完善...")
	}
}

func (pm *ProtocolManager) handle(p *peer) error {
	var (
		genesis = pm.blockchain.Genesis()
		head    = pm.blockchain.CurrentHeader()
	)

	log.Printf("3). do 'proton' Handshake ... \n")
	if err := p.Handshake(pm.networkID, head.Hash(), head.Number, genesis.Hash()); err != nil {
		log.Printf("\t\t proton handshake error: %v \n", err)
		return err
	}
	log.Println("\t\t finish proton handshake.")

	log.Printf("4). register proton peer ... \n")
	if _, exists := pm.peers[p.Identifier()]; exists {
		return fmt.Errorf("peer %q has exists.", p.Identifier())
	}
	pm.peers[p.Identifier()] = p
	if err := pm.downloader.RegisterPeer(p.Identifier(), int(p.version), p); err != nil {
		log.Printf("\t\t proton downloader register peer, err = %v", err)
		return err
	}
	log.Println("\t\t finish register in downloader.")

	for {
		if err := pm.handleMsg(p); err != nil {
			log.Printf("Proton message handling failed, err = %v \n", err)
			return err
		}
	}
}

func (pm *ProtocolManager) handleMsg(p *peer) error {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}

	switch {
	case msg.Code == StatusMsg:
		return errors.New("unexpected status msg.")

	case msg.Code == GetBlocksMsg:
		var query getBlocksData
		if err := msg.Decode(&query); err != nil {
			return err
		}
		var (
			blocks []*types.Block
			pos    uint64 = 0
		)
		for len(blocks) < int(query.Amount) && len(blocks) < downloader.MaxBlockFetch {
			var block *types.Block
			block = pm.blockchain.GetBlockByNumber(query.From + pos)
			blocks = append(blocks, block)
			pos++
		}

		return p.SendBlocks(blocks)

	case msg.Code == BlocksMsg:
		var blocks []*types.Block
		if err := msg.Decode(&blocks); err != nil {
			return fmt.Errorf("decode msg error: %v", err)
		}

		if len(blocks) > 0 {
			err := pm.downloader.DeliverBlocks(p.Identifier(), blocks)
			if err != nil {
				log.Printf("Failed to deliver blocks, err = %v", err)
			}
		}

	default:
		fmt.Printf("recv msg: %v", msg)
	}
	return nil
}
