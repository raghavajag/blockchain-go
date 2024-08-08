package network

import (
	"blockchain/api"
	"blockchain/blockchain"
	"blockchain/utils"
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-kit/log"
)

var defaultBlockTime = 5 * time.Second

type ServerOpts struct {
	APIListenAddr string
	ListenAddr    string
	SeedNodes     []string
	TCPTransport  *TCPTransport
	ID            string
	Logger        log.Logger
	RPCDecodeFunc RPCDecodeFunc
	RPCProcessor  RPCProcessor
	BlockTime     time.Duration
}
type Server struct {
	TCPTransport         *TCPTransport
	txChan               chan *blockchain.Transaction
	peerCh               chan *TCPPeer
	rpcCh                chan RPC
	WalletAddress        string
	Wallet               blockchain.Wallet
	mempool              map[string]blockchain.Transaction
	peerMap              map[net.Addr]*TCPPeer
	Blockchain           *blockchain.Blockchain
	isValidator          bool
	processedBlocks      map[string]bool
	processedBlocksMutex sync.RWMutex
	ServerOpts
}

func NewServer(opts ServerOpts) (*Server, error) {
	if opts.BlockTime == time.Duration(0) {
		opts.BlockTime = defaultBlockTime
	}
	if opts.RPCDecodeFunc == nil {
		opts.RPCDecodeFunc = DefaultRPCDecodeFunc
	}
	if opts.Logger == nil {
		opts.Logger = log.NewLogfmtLogger(os.Stderr)
		opts.Logger = log.With(opts.Logger, "addr", opts.ID)
	}
	wallets, _ := blockchain.NewWallets(opts.ID)
	address := wallets.CreateWallet(opts.ID)
	wallets.SaveToFile(opts.ID)
	fmt.Printf("Your new address: %s\n", address)
	var bc *blockchain.Blockchain

	if opts.ID == "MAIN_NODE" {
		// Create a new blockchain for the main node
		bc = blockchain.CreateBlockchain(address, opts.ID)
		// defer bc.DB.Close()
	} else {
		// Copy the blockchain from the main node and rename it for the local node
		mainNodeBlockchainFile := "blockchain_MAIN_NODE.db"
		localNodeBlockchainFile := "blockchain_" + opts.ID + ".db"

		// Copy the main node's blockchain file to the local node
		sourceFile, err := os.Open(mainNodeBlockchainFile)
		if err != nil {
			return nil, err
		}
		defer sourceFile.Close()

		destinationFile, err := os.Create(localNodeBlockchainFile)
		if err != nil {
			return nil, err
		}
		defer destinationFile.Close()

		_, err = io.Copy(destinationFile, sourceFile)
		if err != nil {
			return nil, err
		}

		bc = blockchain.NewBlockchain(opts.ID)
	}
	UTXOSet := blockchain.UTXOSet{Blockchain: bc}
	UTXOSet.Reindex()
	peerCh := make(chan *TCPPeer)
	rpcCh := make(chan RPC)
	tr := NewTCPTransport(opts.ListenAddr, peerCh)

	// Channel being used to communicate between the JSON RPC server
	// and the node that will process this message.
	txChan := make(chan *blockchain.Transaction)
	if len(opts.APIListenAddr) > 0 {
		apiServerCfg := api.ServerConfig{
			Logger:     opts.Logger,
			ListenAddr: opts.APIListenAddr,
		}
		apiServer := api.NewServer(apiServerCfg, bc, txChan)
		go apiServer.Start()

		opts.Logger.Log("msg", "JSON API server running", "port", opts.APIListenAddr)
	}
	s := &Server{
		txChan:          txChan,
		TCPTransport:    tr,
		peerCh:          peerCh,
		ServerOpts:      opts,
		rpcCh:           rpcCh,
		peerMap:         make(map[net.Addr]*TCPPeer),
		Blockchain:      bc,
		isValidator:     opts.ID == "MAIN_NODE",
		WalletAddress:   address,
		Wallet:          wallets.GetWallet(address),
		mempool:         make(map[string]blockchain.Transaction),
		processedBlocks: make(map[string]bool),
	}
	s.TCPTransport.peerCh = peerCh
	if s.RPCProcessor == nil {
		s.RPCProcessor = s
	}
	if s.isValidator {
		go s.validatorLoop()
	}
	return s, nil
}
func (s *Server) validatorLoop() {
	ticker := time.NewTicker(s.BlockTime)

	s.Logger.Log("msg", "Starting validator loop", "blockTime", s.BlockTime)

	for {
		fmt.Println("creating new block")

		if err := s.createNewBlock(); err != nil {
			s.Logger.Log("create block error", err)
		}

		<-ticker.C
	}
}

func (s *Server) createNewBlock() error {
	// get transactions from mempool
	// verify each tx
	// add coinbase tx
	// mine block
	// update utxo_bucket
	// broadcast blocks
	// clear mempool
	var txs []*blockchain.Transaction

	for _, tx := range s.mempool {
		if s.Blockchain.VerifyTransaction(&tx) {
			txs = append(txs, &tx)
		}
	}

	if len(txs) == 0 {
		s.Logger.Log("msg", "All Transactions are invalid")
		return nil
	}

	cbTx := blockchain.NewCoinbaseTX(s.WalletAddress, "")
	txs = append(txs, cbTx)

	newBlock := s.Blockchain.MineBlock(txs)
	fmt.Printf("\n mined block %v\n", newBlock)
	UTXOSet := blockchain.UTXOSet{Blockchain: s.Blockchain}
	UTXOSet.Update(newBlock)

	for _, tx := range txs {
		delete(s.mempool, hex.EncodeToString(tx.ID))
	}
	s.Logger.Log("msg", "Broadcasting new block", "hash", newBlock.Hash)
	if err := s.broadcastBlock(newBlock); err != nil {
		s.Logger.Log("error", err)
		return err
	}

	s.mempool = make(map[string]blockchain.Transaction)
	s.Logger.Log("msg", "New Block mined", "hash", newBlock.Hash)
	return nil
}
func (s *Server) broadcastBlock(b *blockchain.Block) error {
	buf := &bytes.Buffer{}
	if err := gob.NewEncoder(buf).Encode(b); err != nil {
		return err
	}
	msg := NewMessage(MessageTypeBlock, buf.Bytes())
	return s.broadcast(msg.Bytes())
}
func (s *Server) broadcast(payload []byte) error {
	for netAddr, peer := range s.peerMap {
		fmt.Printf("sending to peer %s\n", netAddr)
		if err := peer.Send(payload); err != nil {
			fmt.Printf("peer send error => addr %s [err: %s]\n", netAddr, err)
		}
	}

	return nil
}
func (s *Server) processTransaction(tx *blockchain.Transaction) error {
	if _, exists := s.mempool[hex.EncodeToString(tx.ID)]; exists {
		return nil
	}
	// s.Logger.Log(
	//     "msg", "adding new tx to mempool",
	//     "hash", hash,
	//     "mempoolPending", s.mempool.PendingCount(),
	// )

	s.mempool[hex.EncodeToString(tx.ID)] = *tx
	s.Logger.Log("msg", "new transaction added to mempool", "tx", tx)
	s.Logger.Log("msg", "mempool size", "size", len(s.mempool))

	return nil
}
func (s *Server) isBlockProcessed(b *blockchain.Block) bool {
	s.processedBlocksMutex.RLock()
	defer s.processedBlocksMutex.RUnlock()

	_, exists := s.processedBlocks[string(b.Hash)]
	return exists
}
func (s *Server) markBlockAsProcessed(b *blockchain.Block) {
	s.processedBlocksMutex.Lock()
	defer s.processedBlocksMutex.Unlock()

	s.processedBlocks[string(b.Hash)] = true
}
func (s *Server) processBlock(b *blockchain.Block) error {
	if s.isBlockProcessed(b) {
		return nil
	}
	if err := s.Blockchain.AddBlock(b); err != nil {
		s.Logger.Log("error", err.Error())
		return err
	}
	UTXOSet := blockchain.UTXOSet{Blockchain: s.Blockchain}
	UTXOSet.Reindex()

	s.markBlockAsProcessed(b)

	go s.broadcastBlock(b)
	return nil
}
func (s *Server) ProcessMessage(msg *DecodedMessage) error {
	switch t := msg.Data.(type) {
	case *blockchain.Block:
		return s.processBlock(t)
	case *blockchain.Transaction:
		return s.processTransaction(t)
	case *GetStatusMessage:
		return s.processGetStatusMessage(msg.From, t)
	case *StatusMessage:
		return s.processStatusMessage(msg.From, t)
	case *GetBlocksMessage:
		return s.processGetBlocksMessage(msg.From, t)
	case *BlocksMessage:
		return s.processBlocksMessage(msg.From, t)
	}
	return nil
}
func (s *Server) processBlocksMessage(from net.Addr, data *BlocksMessage) error {
	s.Logger.Log("msg", "received BLOCKS!", "from", from)

	for _, block := range data.Blocks {
		if err := s.Blockchain.AddBlock(block); err != nil {
			s.Logger.Log("error", err.Error())
			return err
		}
	}
	UTXOSet := blockchain.UTXOSet{Blockchain: s.Blockchain}
	UTXOSet.Reindex()
	return nil
}
func (s *Server) processGetBlocksMessage(from net.Addr, data *GetBlocksMessage) error {
	s.Logger.Log("msg", "received getBlocks message", "from", from)

	var (
		blocks    = []*blockchain.Block{}
		ourHeight = s.Blockchain.GetBestHeight()
	)

	if data.To == 0 {
		for i := int(data.From); i <= int(ourHeight); i++ {
			block, err := s.Blockchain.GetBlockByHeight(i)
			if err != nil {
				return err
			}

			blocks = append(blocks, block)
		}
	}

	blocksMsg := &BlocksMessage{
		Blocks: blocks,
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(blocksMsg); err != nil {
		return err
	}

	msg := NewMessage(MessageTypeBlocks, buf.Bytes())
	peer, ok := s.peerMap[from]
	if !ok {
		return fmt.Errorf("peer %s not known", from)
	}

	return peer.Send(msg.Bytes())
}
func (s *Server) processStatusMessage(from net.Addr, data *StatusMessage) error {
	s.Logger.Log("msg", "Received status message", "from", from, "height", data.CurrentHeight)
	s.Logger.Log("msg", "Current height", "height", s.Blockchain.GetBestHeight())
	if data.CurrentHeight <= uint32(s.Blockchain.GetBestHeight()) {
		s.Logger.Log("msg", "cannot sync blockHeight to low", "height", data.CurrentHeight, "from", from)
		return nil
	}
	go s.requestBlocksLoop(from)
	return nil
}
func (s *Server) requestBlocksLoop(peer net.Addr) error {
	ticker := time.NewTicker(3 * time.Second)

	for {
		ourHeight := uint32(s.Blockchain.GetBestHeight())

		s.Logger.Log("msg", "requesting new blocks", "requesting height", ourHeight+1)

		// In this case we are 100% sure that the node has blocks heigher than us.
		getBlocksMessage := &GetBlocksMessage{
			From: ourHeight + 1,
			To:   0,
		}

		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(getBlocksMessage); err != nil {
			return err
		}
		msg := NewMessage(MessageTypeGetBlocks, buf.Bytes())
		peer, ok := s.peerMap[peer]
		if !ok {
			return fmt.Errorf("peer %s not known", peer.conn.RemoteAddr())
		}

		if err := peer.Send(msg.Bytes()); err != nil {
			s.Logger.Log("error", "failed to send to peer", "err", err, "peer", peer)
		}

		<-ticker.C
	}
}
func (s *Server) processGetStatusMessage(from net.Addr, data *GetStatusMessage) error {
	fmt.Printf("Received get status message from %s\n", from)
	statusMessage := &StatusMessage{
		CurrentHeight: uint32(s.Blockchain.GetBestHeight()),
		ID:            s.ID,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(statusMessage); err != nil {
		return err
	}
	peer, ok := s.peerMap[from]
	if !ok {
		return fmt.Errorf("peer not found")
	}
	msg := NewMessage(MessageTypeStatus, buf.Bytes())
	return peer.Send(msg.Bytes())
}
func (s *Server) Start() {
	s.TCPTransport.Start()
	s.Logger.Log("msg", "accepting TCP connection on", "addr", s.ListenAddr, "id", s.ID)
	time.Sleep(2 * time.Second)
	s.bootstrapNetwork()
	go s.loop()
}
func (s *Server) loop() {
	for {
		select {
		case peer := <-s.peerCh:
			fmt.Printf("Peer %s connected\n", peer.conn.RemoteAddr())
			s.peerMap[peer.conn.RemoteAddr()] = peer
			go peer.readLoop(s.rpcCh)
			if err := s.sendGetStatusMessage(peer); err != nil {
				s.Logger.Log("err", err)
			}
			s.Logger.Log("msg", "peer added to the server", "outgoing", peer.Outgoing, "addr", peer.conn.RemoteAddr())
		case tx := <-s.txChan:
			if err := s.processTransaction(tx); err != nil {
				s.Logger.Log("process TX error", err)
			}
		case rpc := <-s.rpcCh:
			msg, err := DefaultRPCDecodeFunc(rpc)

			if err != nil {
				s.Logger.Log("RPC error", err)
				continue
			}
			if err := s.RPCProcessor.ProcessMessage(msg); err != nil {
				s.Logger.Log("error", err)
				continue
			}
		}
	}
}
func (s *Server) sendGetStatusMessage(peer *TCPPeer) error {
	var (
		getStatusMsg = new(GetStatusMessage)
		buf          = new(bytes.Buffer)
	)
	if err := gob.NewEncoder(buf).Encode(getStatusMsg); err != nil {
		return err
	}
	msg := NewMessage(MessageTypeGetStatus, buf.Bytes())
	return peer.Send(msg.Bytes())
}
func (s *Server) bootstrapNetwork() {
	for _, addr := range s.SeedNodes {
		fmt.Printf("trying to connect to node %s\n", addr)
		go func(addr string) {
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				fmt.Println("failed to connect to seed node")
				return
			}

			s.peerCh <- &TCPPeer{
				conn: conn,
			}
		}(addr)
	}
}
func (s *Server) SendTransaction(toAddress string, amount int) error {
	UTXOSet := blockchain.UTXOSet{Blockchain: s.Blockchain}
	tx := blockchain.NewUTXOTransaction(&s.Wallet, toAddress, amount, &UTXOSet)
	buf := &bytes.Buffer{}
	if err := gob.NewEncoder(buf).Encode(tx); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/tx", buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.Logger.Log("msg", "failed to send transaction", "error", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server responded with status code: %d", resp.StatusCode)
	}

	return nil
}
func (s *Server) GetBalance() {
	if !blockchain.ValidateAddress(s.WalletAddress) {
		s.Logger.Log("error", "Address is not valid")
	}
	UTXOSet := blockchain.UTXOSet{Blockchain: s.Blockchain}

	balance := 0
	pubKeyHash := utils.Base58Decode([]byte(s.WalletAddress))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := UTXOSet.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}
	s.Logger.Log("msg", "Balance", "address", s.WalletAddress, "balance", balance)
}
