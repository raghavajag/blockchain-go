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
	RPCDecodeFunc RPCDecodeFunc
	RPCProcessor  RPCProcessor
}
type Server struct {
	TCPTransport *TCPTransport
	peerCh       chan *TCPPeer
	rpcCh        chan RPC
	Address      string
	peerMap      map[net.Addr]*TCPPeer
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
		peerMap:      make(map[net.Addr]*TCPPeer),
	}
	s.TCPTransport.peerCh = peerCh
	if s.RPCProcessor == nil {
		s.RPCProcessor = s
	}
	return s, nil
}
func (s *Server) ProcessMessage(msg *DecodedMessage) error {
	switch t := msg.Data.(type) {
	case *GetStatusMessage:
		return s.processGetStatusMessage(msg.From, t)
	case *StatusMessage:
		return s.processStatusMessage(msg.From, t)
	}

	return nil
}
func (s *Server) processStatusMessage(from net.Addr, data *StatusMessage) error {
	fmt.Printf("Received status message from %s\n", from)
	if data.CurrentHeight <= 1 {
		fmt.Printf("cannot sync blockHeight to low %d from %s \n", data.CurrentHeight, from)
		return nil
	}

	// request blocks from peer
	return nil
}
func (s *Server) processGetStatusMessage(from net.Addr, data *GetStatusMessage) error {
	fmt.Printf("Received get status message from %s\n", from)
	statusMessage := &StatusMessage{
		CurrentHeight: 1,
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
			s.peerMap[peer.conn.RemoteAddr()] = peer
			go peer.readLoop(s.rpcCh)
			if err := s.sendGetStatusMessage(peer); err != nil {
				fmt.Printf("Error while sending get status message: %v", err)
			}

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
