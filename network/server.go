package network

import (
	"fmt"
	"net"
	"time"
)

type ServerOpts struct {
	APIListenAddr string
	ListenAddr    string
	SeedNodes     []string
	TCPTransport  *TCPTransport
	ID            string
}
type Server struct {
	TCPTransport *TCPTransport
	peerCh       chan *TCPPeer
	rpcCh        chan RPC
	Address      string
	ServerOpts
}

func NewServer(opts ServerOpts) (*Server, error) {
	peerCh := make(chan *TCPPeer)
	rpcCh := make(chan RPC)
	tr := NewTCPTransport(opts.ListenAddr, peerCh)
	s := &Server{
		TCPTransport: tr,
		peerCh:       peerCh,
		ServerOpts:   opts,
		rpcCh:        rpcCh,
	}
	s.TCPTransport.peerCh = peerCh
	return s, nil
}
func (s *Server) Start() {
	s.TCPTransport.Start()
	fmt.Printf("Node listening on %s\n", s.ListenAddr)
	time.Sleep(2 * time.Second)
	s.bootstrapNetwork()
	s.loop()
}
func (s *Server) loop() {
	for {
		select {
		case peer := <-s.peerCh:
			fmt.Printf("Peer %s connected\n", peer.conn.RemoteAddr())
			go peer.readLoop(s.rpcCh)
		case rpc := <-s.rpcCh:
			fmt.Printf("incoming rpc message: %v", rpc)
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
