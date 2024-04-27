package main

import (
	"blockchain/network"
	"log"
)

func main() {
	mainNode := makeServer("MAIN_NODE", ":3000")
	go mainNode.Start()

	localNode := makeServer("LOCAL_NODE", ":4000")

	go localNode.Start()

	select {}
	// cli := cli.CLI{}
	// cli.Run()
}

func makeServer(id string, addr string) *network.Server {
	opts := network.ServerOpts{
		ListenAddr: addr,
		ID:         id,
	}

	s, err := network.NewServer(opts)
	if err != nil {
		log.Fatal(err)
	}

	return s
}
