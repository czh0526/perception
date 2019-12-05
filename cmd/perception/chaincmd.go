package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/czh0526/perception/cmd/utils"
	"github.com/czh0526/perception/proton/core"
	"github.com/urfave/cli"
)

var (
	initProtonCommand = cli.Command{
		Action:   initGenesis,
		Name:     "init_proton",
		Usage:    "init a new genesis block",
		Category: "Blockchain Commands",
		Flags: []cli.Flag{
			utils.DataDirFlag,
		},
	}
)

func initGenesis(ctx *cli.Context) error {

	genesisPath := ctx.Args().First()
	if len(genesisPath) == 0 {
		fmt.Println("Error: Must supply path to genesis JSON file.")
		return errors.New("args not supplied.")
	}

	file, err := os.Open(genesisPath)
	if err != nil {
		fmt.Printf("Error: Failed to read genesis file: %v. \n", err)
		return err
	}
	defer file.Close()

	genesis := new(core.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		fmt.Printf("Error: Failed to decode genesis file: %v. \n", err)
		return err
	}

	stack := makeFullNode(ctx)
	defer stack.Close()

	for _, name := range []string{"chaindata"} {
		chaindb, err := stack.OpenDatabase(name, 0, 0, "")
		if err != nil {
			fmt.Printf("Error: Failed to open database: %v. \n", err)
			return err
		}

		hash, err := core.SetupGenesisBlock(chaindb, genesis)
		if err != nil {
			fmt.Printf("Error: Failed to setup genesis block: %v. \n", err)
			return err
		}

		chaindb.Close()
		fmt.Printf("Successfully wrote genesis state, database = %v, hash = %0x. \n", name, hash)

	}
	return nil
}
