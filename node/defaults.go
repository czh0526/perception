package node

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/czh0526/perception/p2p"
)

var DefaultConfig = Config{
	Name:    "perception",
	Version: "0.0.1",
	DataDir: DefaultDataDir(),
	IPCPath: "perception.ipc",
	P2P: p2p.Config{
		Name:       "perception",
		ListenAddr: "/ip4/0.0.0.0/tcp/10000",
		MaxPeers:   50,
	},
}

func DefaultDataDir() string {
	home := homeDir()
	return filepath.Join(home, ".perception")
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
