package main

import (
	"blockchain/network"
	"log"
)

func main() {
	mainNode := makeServer("MAIN_NODE", ":3000", []string{":4000"}, 3)
	go mainNode.Start()

	localNode := makeServer("LOCAL_NODE", ":4000", []string{}, 10)

	go localNode.Start()

	select {}
}

func makeServer(id string, addr string, seedNodes []string, multiplier int) *network.Server {
	opts := network.ServerOpts{
		ListenAddr: addr,
		ID:         id,
		SeedNodes:  seedNodes,
		Multiplier: multiplier,
	}

	s, err := network.NewServer(opts)
	if err != nil {
		log.Fatal(err)
	}

	return s
}
