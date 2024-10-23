package p2p

import "net"

// Peer is an interface that represents a remote node
type Peer interface {
	net.Conn
	Send([]byte) error
}

// Transport is anything that handles communication
// between the notes in the network. This can be of the
// for (TCP, UDP, Websockets, ...)
type Transport interface {
	Addr() string
	Dial(string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}
