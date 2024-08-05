package api

import (
	"blockchain/blockchain"
	"encoding/gob"
	"fmt"
	"net/http"

	"github.com/go-kit/log"
	"github.com/labstack/echo/v4"
)

type TxResponse struct {
	TxCount uint
	Hashes  []string
}

type APIError struct {
	Error string
}

type Block struct {
	Hash          string
	Version       uint32
	DataHash      string
	PrevBlockHash string
	Height        uint32
	Timestamp     int64
	Validator     string
	Signature     string

	TxResponse TxResponse
}

type ServerConfig struct {
	Logger     log.Logger
	ListenAddr string
}

type Server struct {
	txChan chan *blockchain.Transaction
	ServerConfig
	bc *blockchain.Blockchain
}

func NewServer(cfg ServerConfig, bc *blockchain.Blockchain, txChan chan *blockchain.Transaction) *Server {
	return &Server{
		ServerConfig: cfg,
		bc:           bc,
		txChan:       txChan,
	}
}

func (s *Server) Start() error {
	e := echo.New()

	e.POST("/tx", s.handlePostTx)

	return e.Start(s.ListenAddr)
}

func (s *Server) handlePostTx(c echo.Context) error {
	tx := &blockchain.Transaction{}
	if err := gob.NewDecoder(c.Request().Body).Decode(tx); err != nil {
		s.Logger.Log("erorr", err)
		return c.JSON(http.StatusBadRequest, APIError{Error: err.Error()})
	}
	fmt.Printf("received tx: %+v\n", tx)
	s.txChan <- tx

	return nil
}
