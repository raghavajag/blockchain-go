package blockchain

import (
	"blockchain/block"
	"blockchain/proofofwork"
	"time"

	"github.com/boltdb/bolt"
)

const dbFile = "data.db"
const blocksBlucket = "blocks"

type Blockchain struct {
	// Blocks []*block.Block
	tip []byte
	DB  *bolt.DB
}
type BlockchainIterator struct {
	currentHash []byte
	DB          *bolt.DB
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.tip, bc.DB}

	return bci
}
func (i *BlockchainIterator) Next() *block.Block {
	var block_ *block.Block

	err := i.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBlucket))
		encodedBlock := b.Get(i.currentHash)
		block_ = block.DeserializeBlock(encodedBlock)

		return nil
	})
	if err != nil {
		panic(err)
	}

	i.currentHash = block_.PrevBlockHash

	return block_
}
func (bc *Blockchain) AddBlock(data string) {
	var lastHash []byte
	err := bc.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBlucket))
		lastHash = b.Get([]byte("l"))
		return nil
	})
	if err != nil {
		panic(err)
	}
	newBlock := NewBlock(data, lastHash)
	err = bc.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBlucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())

		if err != nil {
			panic(err)
		}

		err = b.Put([]byte("l"), newBlock.Hash)
		if err != nil {
			panic(err)
		}

		bc.tip = newBlock.Hash
		return nil
	})
	if err != nil {
		panic(err)
	}

}

func NewBlockchain() *Blockchain {
	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBlucket))

		if b == nil {
			genesis := NewGenesisBlock()
			b, err := tx.CreateBucket([]byte(blocksBlucket))
			if err != nil {
				panic(err)
			}

			err = b.Put(genesis.Hash, genesis.Serialize())
			err = b.Put([]byte("l"), genesis.Hash)
			tip = genesis.Hash
		} else {
			tip = b.Get([]byte("l"))
		}

		return nil
	})

	bc := Blockchain{tip, db}

	return &bc
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
