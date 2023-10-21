package cli

import (
	"blockchain/blockchain"
	"blockchain/utils"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

type CLI struct {
	BC *blockchain.Blockchain
}

const nodeID = "5000"

func (cli *CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  getbalance -address ADDRESS - Get balance of ADDRESS")
	fmt.Println("  createblockchain -address ADDRESS - Create a blockchain and send genesis block reward to ADDRESS")
	fmt.Println("  printchain - Print all the blocks of the blockchain")
	fmt.Println("  createwallet - Generates a new key-pair and saves it into the wallet file")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT  - Send AMOUNT of coins from FROM address to To")
	fmt.Println("  getwallets Get all wallet address")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}
func (cli *CLI) send(from, to string, amount int, nodeID string) {
	if !blockchain.ValidateAddress(from) {
		log.Panic("ERROR: Sender address is not valid")
	}
	if !blockchain.ValidateAddress(to) {
		log.Panic("ERROR: Recipient address is not valid")
	}
	bc := blockchain.NewBlockchain(nodeID)
	UTXOSet := blockchain.UTXOSet{Blockchain: bc}
	defer bc.DB.Close()

	wallets, err := blockchain.NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)
	tx := blockchain.NewUTXOTransaction(&wallet, to, amount, &UTXOSet)

	cbTx := blockchain.NewCoinbaseTX(from, "")
	txs := []*blockchain.Transaction{cbTx, tx}

	newBlock := bc.MineBlock(txs)
	UTXOSet.Update(newBlock)
	fmt.Println("Success!")
}

func (cli *CLI) getBalance(address, nodeID string) {
	if !blockchain.ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := blockchain.NewBlockchain(nodeID)
	UTXOSet := blockchain.UTXOSet{Blockchain: bc}
	defer bc.DB.Close()

	balance := 0
	pubKeyHash := utils.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := UTXOSet.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of '%s': %d\n", address, balance)
}
func (cli *CLI) createBlockchain(address, nodeID string) {
	if !blockchain.ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := blockchain.CreateBlockchain(address, nodeID)
	defer bc.DB.Close()

	UTXOSet := blockchain.UTXOSet{Blockchain: bc}
	UTXOSet.Reindex()

	fmt.Println("Done!")
}

func (cli *CLI) printChain() {
	bc := blockchain.NewBlockchain("")
	defer bc.DB.Close()

	bci := bc.Iterator()
	for {
		block := bci.Next()

		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Hash: %x\n", block.Hash)
		pow := blockchain.NewProofOfWork(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}
func (cli *CLI) createWallet(nodeID string, tag string) {
	wallets, _ := blockchain.NewWallets(nodeID)
	address := wallets.CreateWallet(tag)
	wallets.SaveToFile(nodeID)
	fmt.Printf("Your new address: %s\n", address)
}
func (cli *CLI) getWallets() {
	wallets, err := blockchain.NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()
	for address, tag := range addresses {
		fmt.Printf("Wallet Address %s %s \n", address, tag)
	}
}

// Run parses command line arguments and processes commands
func (cli *CLI) Run() {
	cli.validateArgs()

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	getWallets := flag.NewFlagSet("getwallets", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	createWalletTag := createWalletCmd.String("tag", "", "Tag for wallet")
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("To", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")

	switch os.Args[1] {
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}

	case "getwallets":
		err := getWallets.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress, nodeID)
	}
	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockchain(*createBlockchainAddress, nodeID)
	}
	if createWalletCmd.Parsed() {
		if *createWalletTag == "" {
			createWalletCmd.Usage()
			os.Exit(1)
		}
		cli.createWallet(nodeID, *createWalletTag)
	}
	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}
		cli.send(*sendFrom, *sendTo, *sendAmount, nodeID)
	}

	if getWallets.Parsed() {
		cli.getWallets()
	}
}
