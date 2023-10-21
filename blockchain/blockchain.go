package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/boltdb/bolt"
)

const dbFile = "blockchain_%s.db"
const blocksBucket = "blocks"
const genesisCoinbaseData = "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"

type Blockchain struct {
	tip []byte
	DB  *bolt.DB
}
type BlockchainIterator struct {
	currentHash []byte
	DB          *bolt.DB
}

/*
The Iterator method is called on a Blockchain object and returns a new BlockchainIterator object.
It initializes the iterator with the tip (current block hash) of the blockchain and the database instance.
*/
func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.tip, bc.DB}

	return bci
}

/*
The Next method is called on a BlockchainIterator object and returns the next block
in the blockchain. It retrieves the encoded block data from the database using the current block hash
stored in the iterator, deserializes it into a Block object, and updates the iterator's current hash
to the previous block hash of the retrieved block. Finally, it returns the deserialized block.
*/
func (i *BlockchainIterator) Next() *Block {
	var block *Block

	err := i.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	i.currentHash = block.PrevBlockHash

	return block
}

/*
FindUnspentTransactions returns a list of transactions containing unspent outputs.
It iterates through the bc from tip and goes backwards.
For each tx in a block, it checks if any of the tx output (Vout) can be unlocked (spent)
with the provided address. Additionally, it keeps track of spent tx outputs (spentTXOs)
to prevent double spending. This ensure that if an output has already been spent, it won't
be considered as an unspent output again during subsequent interations.
*/
// func (bc *Blockchain) FindUnspentTransactions(address string) []Transaction {
// 	var unspentTXs []Transaction
// 	spentTXOs := make(map[string][]int)
// 	bci := bc.Iterator()

// 	for {
// 		block := bci.Next()

// 		for _, tx := range block.Transactions {
// 			txID := hex.EncodeToString(tx.ID)

// 		Outputs:
// 			for outIdx, out := range tx.Vout {
// 				// Was the output spent?
// 				if spentTXOs[txID] != nil {
// 					for _, spentOut := range spentTXOs[txID] {
// 						if spentOut == outIdx {
// 							continue Outputs
// 						}
// 					}
// 				}

// 				if out.CanBeUnlockedWith(address) {
// 					unspentTXs = append(unspentTXs, *tx)
// 				}
// 			}

// 			if !tx.IsCoinbase() {
// 				for _, in := range tx.Vin {
// 					if in.CanUnlockOutputWith(address) {
// 						inTxID := hex.EncodeToString(in.Txid)
// 						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
// 					}
// 				}
// 			}
// 		}

// 		if len(block.PrevBlockHash) == 0 { // genesis block
// 			break
// 		}
// 	}

// 	return unspentTXs
// }

// FindSpendableOutputs finds and returns unspent outputs to reference in inputs
// func (bc *Blockchain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
// 	unspentOutputs := make(map[string][]int)
// 	unspentTXs := bc.FindUnspentTransactions(address)
// 	accumulated := 0

// Work:
// 	for _, tx := range unspentTXs {
// 		txID := hex.EncodeToString(tx.ID)

// 		for outIdx, out := range tx.Vout {
// 			if out.CanBeUnlockedWith(address) && accumulated < amount {
// 				accumulated += out.Value
// 				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)

// 				if accumulated >= amount {
// 					break Work
// 				}
// 			}
// 		}
// 	}

// 	return accumulated, unspentOutputs
// }
func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}

// MineBlock mines a new block with the provided transactions
func (bc *Blockchain) MineBlock(transactions []*Transaction) *Block {
	var lastHash []byte
	var lastHeight int

	for _, tx := range transactions {
		// TODO: ignore transaction if it's not valid
		if bc.VerifyTransaction(tx) != true {
			log.Panic("ERROR: Invalid transaction")
		}
	}

	err := bc.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))

		blockData := b.Get(lastHash)
		block := DeserializeBlock(blockData)

		lastHeight = block.Height

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(transactions, lastHash, lastHeight+1)

	err = bc.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), newBlock.Hash)
		if err != nil {
			log.Panic(err)
		}

		bc.tip = newBlock.Hash

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return newBlock
}

// func NewUTXOTransaction(from string, to string, amount int, bc *Blockchain) *Transaction {
// 	var inputs []TXInput
// 	var outputs []TXOutput
// 	acc, validOutputs := bc.FindSpendableOutputs(from, amount)

// 	if acc < amount {
// 		log.Panic("ERROR: Not enough funds")
// 	}

// 	for txid, outs := range validOutputs {
// 		txID, err := hex.DecodeString(txid)
// 		if err != nil {
// 			log.Panic("Error occured while decoding the Tx ID")
// 		}
// 		for _, out := range outs {
// 			input := TXInput{txID, out, from}
// 			inputs = append(inputs, input)
// 		}
// 	}
// 	outputs = append(outputs, TXOutput{amount, to})
// 	if acc > amount {
// 		outputs = append(outputs, TXOutput{acc - amount, from})
// 	}
// 	tx := Transaction{nil, inputs, outputs}
// 	tx.SetID()
// 	return &tx
// }

// FindUTXO finds and returns all unspent transaction outputs
// func (bc *Blockchain) FindUTXO(address string) []TXOutput {
// 	var UTXOs []TXOutput
// 	unspentTransactions := bc.FindUnspentTransactions(address)

// 	for _, tx := range unspentTransactions {
// 		for _, out := range tx.Vout {
// 			if out.CanBeUnlockedWith(address) {
// 				UTXOs = append(UTXOs, out)
// 			}
// 		}
// 	}

// 	return UTXOs
// }

func dbExists(dbFile string) bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

// responsible for creating a new instance of the Blockchain struct and initializing it with new tip
func NewBlockchain(nodeID string) *Blockchain {
	dbFile := fmt.Sprintf(dbFile, nodeID)
	if dbExists(dbFile) == false {
		fmt.Println("No existing blockchain found. Create one first.")
		os.Exit(1)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		tip = b.Get([]byte("l"))
		fmt.Printf("tip: %x\n", tip)
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{tip, db}

	return &bc
}

/*
responsible for creating a new blockchain instance and initializing it with a genesis block.
*/
func CreateBlockchain(address, nodeID string) *Blockchain {
	dbFile := fmt.Sprintf(dbFile, nodeID)
	if dbExists(dbFile) {
		fmt.Println("Blockchain already exists.")
		os.Exit(1)
	}

	var tip []byte

	cbtx := NewCoinbaseTX(address, genesisCoinbaseData)
	genesis := NewGenesisBlock(cbtx)

	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			log.Panic(err)
		}

		err = b.Put(genesis.Hash, genesis.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), genesis.Hash)
		if err != nil {
			log.Panic(err)
		}
		tip = genesis.Hash

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{tip, db}

	return &bc
}

// SignTransaction signs inputs of a Transaction
func (bc *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs)
}

// FindTransaction finds a transaction by its ID
func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction is not found")
}
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		dataToVerify := fmt.Sprintf("%x\n", txCopy)

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if ecdsa.Verify(&rawPubKey, []byte(dataToVerify), &r, &s) == false {
			return false
		}
		txCopy.Vin[inID].PubKey = nil
	}

	return true
}
func (bc *Blockchain) FindUTXO() map[string]TXOutputs {
	UTXO := make(map[string]TXOutputs)
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Vout {
				// Was the output spent?
				if spentTXOs[txID] != nil {
					for _, spentOutIdx := range spentTXOs[txID] {
						if spentOutIdx == outIdx {
							continue Outputs
						}
					}
				}

				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}

			if tx.IsCoinbase() == false {
				for _, in := range tx.Vin {
					inTxID := hex.EncodeToString(in.Txid)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return UTXO
}
