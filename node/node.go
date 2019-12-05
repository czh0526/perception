package node

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/czh0526/perception/proton/chaindb"
	"github.com/czh0526/perception/proton/core/rawdb"

	"github.com/czh0526/perception/p2p"
)

const (
	datadirPrivateKey      = "nodekey"
	datadirDefaultKeyStore = "keystore"
	datadirNodeDatabase    = "nodes"
)

type Node struct {
	config *Config
	server *p2p.Server

	serviceFuncs []ServiceConstructor
	services     map[reflect.Type]Service

	stop chan struct{}
	lock sync.RWMutex
}

func New(conf *Config) (*Node, error) {
	if conf.DataDir != "" {
		absDataDir, err := filepath.Abs(conf.DataDir)
		if err != nil {
			return nil, err
		}
		conf.DataDir = absDataDir
	}

	if strings.ContainsAny(conf.Name, `/\`) {
		return nil, errors.New(`Config.Name must bot contain '/' or '\'`)
	}
	if conf.Name == datadirDefaultKeyStore {
		return nil, errors.New(fmt.Sprintf(`Config.Name cannot be "%s"`, datadirDefaultKeyStore))
	}
	if strings.HasSuffix(conf.Name, ".ipc") {
		return nil, errors.New(`Config.Name cannot end with ".ipc"`)
	}

	return &Node{
		config: conf,
	}, nil
}

type ServiceConstructor func(ctx *ServiceContext) (Service, error)

func (n *Node) Register(constructor ServiceConstructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.serviceFuncs = append(n.serviceFuncs, constructor)
	return nil
}

func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// 构建 p2p.Server 对象
	var p2pConfig = &n.config.P2P
	p2pConfig.PrivateKey = n.config.NodeKey()
	p2pConfig.Name = "Sweet Windy"
	running := p2p.NewServer(p2pConfig)

	// 构建 service 列表
	services := make(map[reflect.Type]Service)
	for _, constructor := range n.serviceFuncs {
		ctx := &ServiceContext{
			config: n.config,
		}
		service, err := constructor(ctx)
		if err != nil {
			return err
		}
		kind := reflect.TypeOf(service)
		if _, exists := services[kind]; exists {
			return errors.New(fmt.Sprintf("duplicate Service(%s) error.", kind))
		}
		services[kind] = service
	}

	// 加入服务模块
	for _, service := range services {
		running.Protocols = append(running.Protocols, service.Protocols()...)
	}
	if err := running.Init(); err != nil {
		return err
	}

	go running.Start()
	log.Printf("Starting p2p Server %q.\n", p2pConfig.Name)

	// 启动 Service 列表
	for kind, service := range services {
		if err := service.Start(running); err != nil {
			return errors.New(fmt.Sprintf("start Service(%s) error: %v", kind, err))
		}
	}

	n.services = services
	n.server = running
	n.stop = make(chan struct{})
	return nil
}

func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.server == nil {
		return errors.New("Node has stopped.")
	}

	// 停止服务模块
	for kind, service := range n.services {
		if err := service.Stop(); err != nil {
			return errors.New(fmt.Sprintf("stop Service(%s) error: %v", kind, err))
		}
	}

	// 停止 p2p server
	n.server.Stop()
	n.server = nil

	// 通知 Wait() 函数
	close(n.stop)
	return nil
}

func (n *Node) Wait() {
	n.lock.RLock()
	stop := n.stop
	defer n.lock.RUnlock()

	<-stop
}

func (n *Node) Close() error {
	if err := n.Stop(); err != nil {
		return err
	}
	return nil
}

func (n *Node) OpenDatabase(name string, cache, handles int, namespace string) (chaindb.Database, error) {
	if n.config.DataDir == "" {
		return nil, errors.New("--datadir must be set.")
	}
	root := n.config.ResolvePath(name)
	return rawdb.NewLevelDBDatabase(root, cache, handles, namespace)
}
