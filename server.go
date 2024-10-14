package main

import (
	"dfs/p2p"
	"fmt"
	"log"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc p2p.PathTransformFunc
	Transport         p2p.Transport
}

type FileServer struct {
	FileServerOpts

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
	}
}
func (s *FileServer) Stop() {
	close(s.quitch)
}

func (s *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to user, quit action.")
		s.Transport.Close()
	}()

	for {
		select {
		case msg := <-s.Transport.Consume():
			fmt.Println(msg)
		case <-s.quitch:
			return
		}
	}
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	s.loop()

	return nil
}
