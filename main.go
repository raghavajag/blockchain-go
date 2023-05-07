package main

import (
	"blockchain/blockchain"
	"fmt"
)

func main() {
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Send 1 BTC to dev")
	bc.AddBlock("Send 2 more BTC to yv")

	for _, block := range bc.Blocks {
		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Println()
	}
}
