package main

import (
	"blockchain/network"
	"log"
)

func main() {
	mainNode := makeServer("MAIN_NODE", ":3000", []string{":4000"}, ":8080")
	go mainNode.Start()

	localNode := makeServer("LOCAL_NODE_1", ":4000", []string{""}, "")

	go localNode.Start()

	select {}
}

func makeServer(id string, addr string, seedNodes []string, apiListenAddr string) *network.Server {
	opts := network.ServerOpts{
		ListenAddr:    addr,
		ID:            id,
		SeedNodes:     seedNodes,
		APIListenAddr: apiListenAddr,
	}

	s, err := network.NewServer(opts)
	if err != nil {
		log.Fatal(err)
	}

	return s
}
