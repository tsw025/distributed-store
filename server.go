package main

import (
	"bytes"
	"dfs/p2p"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc p2p.PathTransformFunc
	Transport         p2p.Transport
	BootStrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store  *p2p.Store
	quitch chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := p2p.StoreOpts{
		PathTransformFunc: opts.PathTransformFunc,
		Root:              opts.StorageRoot,
	}
	return &FileServer{
		store:          p2p.NewStore(storeOpts),
		FileServerOpts: opts,
		quitch:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}
func (s *FileServer) Stop() {
	close(s.quitch)
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p
	log.Printf("connected with remote: %s", p.RemoteAddr())
	return nil
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

type MessageGetFile struct {
	Key string
}

func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(key) {
		return s.store.Read(key)
	}

	fmt.Printf("dont have file (%s) locally, fetching from network: \n", key)
	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(3 * time.Second)
	for _, peer := range s.peers {
		fmt.Println("receiving stream from peer: ", peer.RemoteAddr())
		fileBuffer := new(bytes.Buffer)
		n, err := io.CopyN(fileBuffer, peer, 22)
		if err != nil {
			return nil, err
		}

		fmt.Println("received bytes over the network: ", n)
		fmt.Println(fileBuffer.String())
	}

	select {}

	return nil, nil
}

// This will be the intial point where the client will store the file into the server.
func (s *FileServer) Store(key string, r io.Reader) error {
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)
	// Here, the TeeReader will read from the source, and simultaneously write to the buffer.
	// So the will hold the original source, and one in the buffer.
	// We do this because we need to read the data twice, when saving and broadcasting.
	// We can't read from io.Reader twice.
	size, err := s.store.Write(key, tee)
	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}

	// Broadcast msg.
	if err := s.broadcast(&msg); err != nil {
		return err
	}
	time.Sleep(3 * time.Millisecond)

	//TODO: use a multiwriter here.
	// Send the actual data as a stream, so that the broadcast channels
	// and will continue to read from the peers(handleMessageStoreFile).
	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingStream})
		n, err := io.Copy(peer, fileBuffer)
		if err != nil {
			return err
		}
		fmt.Println("received and written bytes to the disk: ", n)
	}

	return nil
}

// This will continuously process the incoming messages, and handle shutdown signal.
func (s *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to user, quit action.")
		err := s.Transport.Close()
		if err != nil {
			return
		}
	}()

	// This will listen to the channels and process messages appropriately.
	for {
		select {
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("decoding error: ", err)
			}
			if err := s.handleMessage(rpc.From, &msg); err != nil {
				log.Print("handle message error: ", err)
			}
		case <-s.quitch:
			return
		}
	}
}

func (s *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	}
	return nil
}

// The handleMessageStoreFile will retrieve the data written to the TCP Conn on the peers.
// and will write to the network, and finally close the steam wait group.
func (s *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}
	n, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes to disk\n", s.Transport.Addr(), n)

	peer.(*p2p.TCPPeer).Wg.Done()
	return nil
}

func (s *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !s.store.Has(msg.Key) {
		return fmt.Errorf("need to server file (%s) but it does not exist on disk", msg.Key)
	}

	fmt.Printf("serving file (%s) over the network\n", msg.Key)

	r, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not in map\n", from)
	}

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("written %d bytes over the network to %s\n", n, from)
	return nil
}

func (s *FileServer) bootStrapNetwork() error {
	for _, addr := range s.BootStrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			fmt.Println("attempting to connect with remote: ", addr)
			if err := s.Transport.Dial(addr); err != nil {
				log.Println("dial error: ", err)
			}
		}(addr)
	}

	return nil
}

func (s *FileServer) Start() error {
	// This creates the local node, the inbound connection, by a go routine.
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	// This open ups the outbound connections (remote), by a go routine.
	s.bootStrapNetwork()
	s.loop()
	return nil
}

func (s *FileServer) stream(msg *Message) error {
	peers := []io.Writer{}

	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(&msg)
}

// Right now the broadcast will do is send message to the network.
// In specific, this will send the message over the TCP peer configured.
// So far we are only broadcasting the metadata, when using Store, and GET.
// Thats why we use the Incoming Message.
func (s *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			log.Println(err)
		}
	}

	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}
