package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"time"
)

type Count int
type ServerOpts struct {
	APIListenAddr string
	ListenAddr    string
	SeedNodes     []string
	TCPTransport  *TCPTransport
	ID            string
	Multiplier    int
	RPCDecodeFunc RPCDecodeFunc
	RPCProcessor  RPCProcessor
}
type Server struct {
	TCPTransport *TCPTransport
	peerCh       chan *TCPPeer
	rpcCh        chan RPC
	Address      string
	count        Count
	ServerOpts
}

func NewServer(opts ServerOpts) (*Server, error) {
	if opts.RPCDecodeFunc == nil {
		opts.RPCDecodeFunc = DefaultRPCDecodeFunc
	}
	peerCh := make(chan *TCPPeer)
	rpcCh := make(chan RPC)
	tr := NewTCPTransport(opts.ListenAddr, peerCh)
	s := &Server{
		TCPTransport: tr,
		peerCh:       peerCh,
		ServerOpts:   opts,
		rpcCh:        rpcCh,
		count:        1,
	}
	s.TCPTransport.peerCh = peerCh
	if s.RPCProcessor == nil {
		s.RPCProcessor = s
	}
	return s, nil
}
func (s *Server) ProcessMessage(msg *DecodedMessage) error {
	switch t := msg.Data.(type) {
	case GetCountMessage:
		fmt.Printf("Received count: %v from %s\n", t.Count, msg.From)
	}
	return nil
}
func (s *Server) startCounting() {
	for {
		time.Sleep(1 * time.Second)
		s.count = Count(int(s.count) * s.ServerOpts.Multiplier)
	}
}
func (s *Server) Start() {
	go s.startCounting()
	s.TCPTransport.Start()
	fmt.Printf("Node listening on %s\n", s.ListenAddr)
	time.Sleep(2 * time.Second)
	s.bootstrapNetwork()
	go s.loop()
}
func (s *Server) loop() {
	for {
		select {
		case peer := <-s.peerCh:
			fmt.Printf("Peer %s connected\n", peer.conn.RemoteAddr())
			go peer.readLoop(s.rpcCh)
			go s.sendCount(peer)

		case rpc := <-s.rpcCh:
			msg, err := DefaultRPCDecodeFunc(rpc)
			if err != nil {
				fmt.Printf("Error while decoding: %v", err)
				continue
			}
			if err := s.RPCProcessor.ProcessMessage(msg); err != nil {
				fmt.Printf("Error while processing message: %v", err)
				continue
			}
		}
	}
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
func (s *Server) sendCount(peer *TCPPeer) error {
	var (
		getCountMsg = &GetCountMessage{Count: s.count}
		buf         = new(bytes.Buffer)
	)
	if err := gob.NewEncoder(buf).Encode(getCountMsg); err != nil {
		return err
	}
	msg := NewMessage(MessageTypeGetCount, buf.Bytes())
	return peer.Send(msg.Bytes())
}
