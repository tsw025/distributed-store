package main

import (
	"dfs/p2p"
	"log"
)

func main() {
	listenAddr := ":3000"
	tr := p2p.NewTCPTransport(listenAddr)
	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	select {}
}
