package blockchain

import (
	"blockchain/block"
	"blockchain/proofofwork"
	"time"
)

type Blockchain struct {
	Blocks []*block.Block
}

func (bc *Blockchain) AddBlock(data string) {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(data, prevBlock.Hash)
	bc.Blocks = append(bc.Blocks, newBlock)
}

func NewBlockchain() *Blockchain {
	return &Blockchain{[]*block.Block{NewGenesisBlock()}}
}

func NewGenesisBlock() *block.Block {
	return NewBlock("Genesis Block", []byte{})
}

func NewBlock(data string, prevBlockHash []byte) *block.Block {
	block := &block.Block{
		Timestamp:     time.Now().Unix(),
		Data:          []byte(data),
		PrevBlockHash: prevBlockHash,
		Hash:          []byte{},
		Nonce:         0}
	pow := proofofwork.NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}
