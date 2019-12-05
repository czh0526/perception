package proton

import (
	"fmt"
	"math/big"

	"github.com/czh0526/perception/common"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
)

const (
	Protocol_V1  uint = 1
	ProtocolName      = "proton"
)

var ProtocolID = protocol.ID(fmt.Sprintf("/%s/%d", ProtocolName, Protocol_V1))
var ProtocolVersions = make(map[uint]network.StreamHandler)

const protocolMaxMsgSize = 10 * 1024 * 1024

const (
	StatusMsg         = 0x10
	NewBlockHashesMsg = 0x11
	TxMsg             = 0x12
	GetBlocksMsg      = 0x13
	BlocksMsg         = 0x14
	GetNodeDataMsg    = 0x15
	NodeDataMsg       = 0x16
)

type statusData struct {
	ProtocolVersion uint32
	NetworkId       uint64
	CurrentBlock    common.Hash
	BlockNumber     *big.Int
	GenesisBlock    common.Hash
}

type getBlocksData struct {
	From   uint64
	Amount uint64
}
