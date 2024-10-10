package p2p

import (
	"bytes"
	"fmt"
	"net"
	"sync"
)

// TCPPeer represents the remote node over a TCP established connection.
type TCPPeer struct {
	// conn is the underlying connection of the peer
	conn net.Conn

	// dial, if we dial and retrieve a conn => outbound == true
	// if we accept and retrieve a conn => outbound == false
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransport struct {
	listenAddress string
	listener      net.Listener
	shakeHands    HandshakeFunc
	decoder       Decoder

	mu    sync.Mutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(listAddr string) *TCPTransport {
	return &TCPTransport{
		listenAddress: listAddr,
		shakeHands:    NOPHandshakeFunc,
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	ln, err := net.Listen("tcp", t.listenAddress)
	if err != nil {
		return err
	}

	t.listener = ln
	go t.startAcceptLoop()

	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("TCP accept error: %s\n", err)
		}
		go t.handleConn(conn)
	}
}

type Temp struct{}

func (t *TCPTransport) handleConn(conn net.Conn) {
	peer := NewTCPPeer(conn, true)
	// create handshake
	if err := t.shakeHands(peer); err != nil {
		fmt.Printf("Unable to validate connection%v\n", peer)
		return
	}

	// Read and Decode
	//Read data from io, and decode it.
	fmt.Printf("New incoming connection %v\n", peer)
	buf := new(bytes.Buffer)
	msg := &Temp{}
	for {
		if err := t.decoder.Decode(buf, msg); err != nil {
			fmt.Printf("TCP error: %s\n", err)
			continue
		}
	}

}
