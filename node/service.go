package node

import (
	"errors"

	"github.com/czh0526/perception/p2p"
	"github.com/czh0526/perception/proton/chaindb"
	"github.com/czh0526/perception/proton/core/rawdb"
)

type ServiceContext struct {
	config *Config
}

func (ctx *ServiceContext) OpenDatabase(name string, cache int, handles int, namespace string) (chaindb.Database, error) {
	nodeConfig := ctx.config
	if nodeConfig.DataDir == "" {
		return nil, errors.New("--datadir must be set manually.")
	}

	root := nodeConfig.ResolvePath(name)
	return rawdb.NewLevelDBDatabase(root, cache, handles, namespace)
}

type Service interface {
	Protocols() []p2p.Protocol
	Start(*p2p.Server) error
	Stop() error
}
