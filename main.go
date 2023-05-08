package main

import (
	"blockchain/blockchain"
	"blockchain/cli"
)

func main() {
	bc := blockchain.NewBlockchain()
	defer bc.DB.Close()

	cli := cli.CLI{BC: bc}
	cli.Run()
}
