package blockchain

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

const walletFile = "wallet_%s.dat"

type Wallets struct {
	Wallets map[string]*Wallet
}

func (ws *Wallets) GetAddresses() map[string]string {
	walletsWithTags := make(map[string]string)

	for address, wallet := range ws.Wallets {
		walletsWithTags[address] = wallet.Tag
	}

	return walletsWithTags
}

// NewWallets creates Wallets and fills it from a file if it exists
func NewWallets(nodeID string) (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFromFile(nodeID)

	return &wallets, err
}

// LoadFromFile loads wallets from the file
func (ws *Wallets) LoadFromFile(nodeID string) error {
	walletFile := fmt.Sprintf(walletFile, nodeID)
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}

	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		log.Panic(err)
	}
	ws.Wallets = wallets.Wallets

	return nil
}

func (ws *Wallets) CreateWallet(tag string) string {
	wallet := NewWallet(tag)
	address := fmt.Sprintf("%s", wallet.GetAddress())
	ws.Wallets[address] = wallet
	return address
}

func (ws Wallets) SaveToFile(nodeID string) {
	var content bytes.Buffer
	walletFile := fmt.Sprintf(walletFile, nodeID)

	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}
func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}
func (ws Wallets) LoadFromFiles() error {
	files, err := ioutil.ReadDir(".")
	if err != nil {
		log.Panic(err)
	}
	for _, file := range files {
		fileName := file.Name()
		if isWalletFile(fileName) {
			nodeID := extractNodeID(fileName)
			wallets := Wallets{}

			err := wallets.LoadFromFile(nodeID)
			if err != nil {
				return err
			}

			for address, wallet := range wallets.Wallets {
				ws.Wallets[address] = wallet
			}
		}
	}
	return nil
}

func isWalletFile(fileName string) bool {
	return strings.HasPrefix(fileName, "wallet_") && strings.HasSuffix(fileName, ".dat")
}

func extractNodeID(fileName string) string {
	parts := strings.Split(fileName, "_")
	if len(parts) == 2 {
		return parts[1][:len(parts[1])-4]
	}
	return ""
}
