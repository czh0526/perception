package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"os"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	ma "github.com/multiformats/go-multiaddr"
)

func main() {
	var (
		port        = flag.Int("port", 10000, "listen address")
		genKey      = flag.String("genkey", "", "generate a node key")
		nodeKeyFile = flag.String("nodekey", "", "private key filename")

		data    []byte
		nodeKey crypto.PrivKey
		keyfile *os.File
		err     error
	)
	flag.Parse()

	switch {
	case *genKey != "":
		nodeKey, _, err = crypto.GenerateSecp256k1Key(rand.Reader)
		if err != nil {
			panic(fmt.Sprintf("cauld not generate key: %v", err))
		}
		if data, err = crypto.MarshalPrivateKey(nodeKey); err != nil {
			panic(fmt.Sprintf("marshal private key error: %v", err))
		}

		if keyfile, err = os.Create(*genKey); err != nil {
			panic(fmt.Sprintf("create nodekey error: %v", err))
		}
		if _, err := keyfile.Write(data); err != nil {
			panic(fmt.Sprintf("save private key error: %v", err))
		}
		keyfile.Close()

		return

	case *nodeKeyFile != "":
		if keyfile, err = os.Open(*nodeKeyFile); err != nil {
			panic(fmt.Sprintf("open nodekey error: %v", err))
		}
		data = make([]byte, 36)
		if _, err = keyfile.Read(data); err != nil {
			panic(fmt.Sprintf("read nodekey error: %v", err))
		}
		if nodeKey, err = crypto.UnmarshalPrivateKey(data); err != nil {
			panic(fmt.Sprintf("load nodekey error: %v", err))
		}

	default:
		fmt.Println("Usage: ")
		fmt.Println("\t 1). bootnode --genkey <filename> ")
		fmt.Println("\t 2). bootnode --nodekey <filename> --port <port>")
		return
	}

	libp2pListenAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port))
	if err != nil {
		panic(fmt.Sprintf("create libp2p listen port error: %v", err))
	}

	ctx := context.Background()
	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(libp2pListenAddr),
		libp2p.Identity(nodeKey),
	)
	if err != nil {
		panic(fmt.Sprintf("new libp2p host error: %v", err))
	}

	kadDHT, err := dht.New(ctx, host)
	if err != nil {
		panic(fmt.Sprintf("new kadDHT error: %v", err))
	}
	if err = kadDHT.Bootstrap(ctx); err != nil {
		panic(fmt.Sprintf("kadDHT Bootstrap error: %v", err))
	}

	fmt.Printf("bootnode <%v@%v> started.", host.ID(), host.Addrs())
	select {}
}
