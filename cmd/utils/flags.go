package utils

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/czh0526/perception/node"
	"github.com/czh0526/perception/p2p"
	"github.com/czh0526/perception/proton"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli"
)

var (
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory for databases and keystore",
	}
	KeyStoreDirFlag = cli.StringFlag{
		Name:  "keystore",
		Usage: "Directory for the keystore (default = inside the datadir)",
	}
	NodeKeyFileFlag = cli.StringFlag{
		Name:  "nodekey",
		Usage: "P2P node key file",
	}
	ListenPortFlag = cli.IntFlag{
		Name:  "port",
		Usage: "Network listening port",
		Value: 10000,
	}
	NetworkIdFlag = cli.Uint64Flag{
		Name:  "networkid",
		Usage: "Network identifier",
		Value: proton.DefaultConfig.NetworkId,
	}
	BootnodesFlag = cli.StringFlag{
		Name:  "bootnodes",
		Usage: "Comma separated node urls for P2P discovery bootstrap.",
		Value: "",
	}
)

func SetProtonConfig(ctx *cli.Context, stack *node.Node, conf *proton.Config) {
	if ctx.GlobalIsSet(NetworkIdFlag.Name) {
		conf.NetworkId = ctx.GlobalUint64(NetworkIdFlag.Name)
	}
}

func SetNodeConfig(ctx *cli.Context, conf *node.Config) {
	SetP2PConfig(ctx, &conf.P2P)

	if ctx.GlobalIsSet(DataDirFlag.Name) {
		conf.DataDir = ctx.GlobalString(DataDirFlag.Name)
	}

	if ctx.GlobalIsSet(KeyStoreDirFlag.Name) {
		conf.KeyStoreDir = ctx.GlobalString(KeyStoreDirFlag.Name)
	}
}

func SetP2PConfig(ctx *cli.Context, conf *p2p.Config) {
	setNodeKey(ctx, conf)
	setListenAddress(ctx, conf)
	setBootNodes(ctx, conf)
}

func setNodeKey(ctx *cli.Context, conf *p2p.Config) {
	var (
		filename = ctx.GlobalString(NodeKeyFileFlag.Name)
		keyfile  *os.File
		key      crypto.PrivKey
		data     []byte
		err      error
	)

	if filename != "" {
		keyfile, err = os.Open(filename)
		if err != nil {
			panic(err)
		}
		defer keyfile.Close()

		data = make([]byte, 36)
		if _, err = io.ReadFull(keyfile, data); err != nil {
			panic(err)
		}

		if key, err = crypto.UnmarshalPrivateKey(data); err != nil {
			panic(err)
		}

		conf.PrivateKey = key
	}
}

func setListenAddress(ctx *cli.Context, conf *p2p.Config) {
	if ctx.GlobalIsSet(ListenPortFlag.Name) {
		conf.ListenAddr = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", ctx.GlobalInt(ListenPortFlag.Name))
	}
}

func setBootNodes(ctx *cli.Context, conf *p2p.Config) {
	urls := []string{}
	switch {
	case ctx.GlobalIsSet(BootnodesFlag.Name):
		urls = strings.Split(ctx.GlobalString(BootnodesFlag.Name), ",")
	case conf.BootstrapPeers != nil:
		return
	}

	for _, url := range urls {
		maddr, err := ma.NewMultiaddr(url)
		if err != nil {
			continue
		}
		_, err = peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			continue
		}
		conf.BootstrapPeers = append(conf.BootstrapPeers, url)
	}
}
