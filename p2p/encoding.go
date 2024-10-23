package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(reader io.Reader, msg *RPC) error
}

type GOBDecoder struct{}

// GOBDecoder is a small struct and the Decode method does not modify the receiver, using a value receiver can simplify the code without significant performance impact.
func (dec GOBDecoder) Decode(r io.Reader, msg *RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

type NOPDecoder struct {
}

func (dec NOPDecoder) Decode(r io.Reader, msg *RPC) error {
	peekBuf := make([]byte, 1)

	if _, err := r.Read(peekBuf); err != nil {
		return nil
	}

	// In case of a stream, we are not decoding what is being sent over the network.
	// We are just setting Stream true, so we can handle that in our logic
	stream := peekBuf[0] == IncomingStream
	if stream {
		msg.Stream = true
		return nil
	}

	buf := make([]byte, 1028)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}

	msg.Payload = buf[:n]
	return nil
}
