package p2p

import (
	crypto "github.com/libp2p/go-libp2p-core/crypto"
)

type Config struct {
	Name           string
	PrivateKey     crypto.PrivKey `toml:"-"`
	MaxPeers       int
	ListenAddr     string
	BootstrapPeers []string
}
