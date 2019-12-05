package main

import (
	"fmt"
	"reflect"
	"unicode"

	"github.com/czh0526/perception/cmd/utils"
	"github.com/czh0526/perception/node"
	"github.com/czh0526/perception/proton"
	"github.com/naoina/toml"
	"github.com/urfave/cli"
)

type Config struct {
	Node   node.Config
	Proton proton.Config
}

func makeConfigNode(ctx *cli.Context) (*node.Node, Config) {
	conf := Config{
		Proton: proton.DefaultConfig,
		Node:   node.DefaultConfig,
	}

	// 为 node.Config 设置 Flags
	utils.SetNodeConfig(ctx, &conf.Node)
	node, err := node.New(&conf.Node)
	if err != nil {
		panic(err)
	}

	utils.SetProtonConfig(ctx, node, &conf.Proton)

	return node, conf
}

func makeFullNode(ctx *cli.Context) *node.Node {
	nd, conf := makeConfigNode(ctx)
	// if err := tomlSettings.NewEncoder(os.Stdout).Encode(conf); err != nil {
	// 	fmt.Printf("print config file err: %v \n", err)
	// }
	// fmt.Println()

	err := nd.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		proton, err := proton.New(ctx, &conf.Proton)
		return proton, err
	})
	if err != nil {
		fmt.Printf("Failed to register the Proton service: %v", err)
		return nil
	}
	return nd
}

var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}
