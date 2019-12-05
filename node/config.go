package node

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/czh0526/perception/p2p"
	"github.com/libp2p/go-libp2p-core/crypto"
)

type Config struct {
	Name        string
	Version     string
	DataDir     string
	P2P         p2p.Config
	KeyStoreDir string
	IPCPath     string
}

func (c *Config) NodeKey() crypto.PrivKey {
	// 启动命令行中指定了 --nodekey
	if c.P2P.PrivateKey != nil {
		return c.P2P.PrivateKey
	}

	// 从 datadir 中读取 nodekey
	keyFilename := c.ResolvePath(datadirPrivateKey)
	if _, statErr := os.Stat(keyFilename); statErr == nil {
		if keyFile, openErr := os.Open(keyFilename); openErr == nil {
			data := make([]byte, 36)
			if _, readErr := io.ReadFull(keyFile, data); readErr == nil {
				key, unmarshalErr := crypto.UnmarshalPrivateKey(data)
				if unmarshalErr == nil {
					return key
				}
				fmt.Printf("*). unmarshal nodekey error: %v, regenarate new nodekey file. \n", unmarshalErr)
			}
		}
	}

	// 读取失败：产生一个新的 nodekey, 并保存在 DataDir 目录
	os.Remove(keyFilename)

	key, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate private key: %v", err))
	}

	instanceDir := c.instanceDir()
	if err := os.MkdirAll(instanceDir, 0700); err != nil {
		panic(fmt.Sprintf("Failed to create parent directory for nodekey: %v", err))
	}

	keyFilename = filepath.Join(instanceDir, datadirPrivateKey)
	data, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal nodekey: %v", err))
	}

	keyFile, err := os.Create(keyFilename)
	if err != nil {
		panic(fmt.Sprintf("Failed to create nodekey file: %v", err))
	}

	if _, err := keyFile.Write(data); err != nil {
		panic(fmt.Sprintf("Failed to write nodekey file: %v", err))
	}

	keyFile.Close()
	return key
}

func (c *Config) ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(c.instanceDir(), path)
}

func (c *Config) instanceDir() string {
	return filepath.Join(c.DataDir, c.name())
}

func (c *Config) name() string {
	if c.Name == "" {
		progname := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
		if progname == "" {
			panic("empty executable name, setConfig.Name")
		}
		return progname
	}
	return c.Name
}
