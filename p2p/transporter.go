package p2p

// Peer is an interface that represents a remote node
type Peer interface {
}

// Transport is anything that handles communication
// between the notes in the network. This can be of the
// for (TCP, UDP, Websockets, ...)
type Transport interface {
	ListenAndAccept() error
}
