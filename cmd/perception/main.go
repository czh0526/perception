package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/czh0526/perception/cmd/utils"

	"github.com/czh0526/perception/node"
	"github.com/urfave/cli"
)

var (
	app = cli.NewApp()
)

func init() {
	app.Name = "Perception"
	app.Usage = "Perception help you to get in touch with p2p network."
	app.Version = "0.0.1"
	app.Action = perception
	app.Flags = []cli.Flag{
		utils.DataDirFlag,
		utils.ListenPortFlag,
		utils.BootnodesFlag,
	}
	app.Commands = []cli.Command{
		initProtonCommand,
	}
}

func main() {

	app.Run(os.Args)
}

func perception(ctx *cli.Context) error {
	// 构建 Node
	node := makeFullNode(ctx)
	defer node.Stop()

	// 启动 Node
	startNode(node)

	// 等待 Node 停止
	node.Wait()
	return nil
}

func startNode(n *node.Node) {
	// 启动 Node 对象
	if err := n.Start(); err != nil {
		panic(err)
	}

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)

		<-sigc
		fmt.Println("Got interrupt, shutting down...")
		go n.Stop()
	}()
}
