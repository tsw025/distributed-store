package p2p

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const defaultRootFolderName = "ggnetwork"

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:]) // convert a fixed [] byte to a slice

	var result strings.Builder
	blocksize := 5
	sliceLen := len(hashStr) / blocksize

	for i := 0; i < sliceLen; i++ {
		if i > 0 {
			result.WriteString("/")
		}
		from, to := i*blocksize, (i*blocksize)+blocksize
		result.WriteString(hashStr[from:to])
	}

	return PathKey{
		Filename: hashStr,
		PathName: result.String(),
	}
}

type PathTransformFunc func(string) PathKey

type PathKey struct {
	PathName string
	Filename string
}

func (p *PathKey) FullPath() string {
	return filepath.Join(p.PathName, p.Filename)
}

type StoreOpts struct {
	PathTransformFunc PathTransformFunc
	Root              string
}

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		Filename: key,
		PathName: key,
	}
}

type Store struct {
	StoreOpts StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}
	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Write(key string, r io.Reader) error {
	return s.writeStream(key, r)
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)
	return buf, err

}

func (s *Store) writeStream(filePath string, r io.Reader) error {
	pathKey := s.StoreOpts.PathTransformFunc(filePath)
	pathNameWithRoot := filepath.Join(s.StoreOpts.Root, pathKey.PathName)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return err
	}

	fullPathWithRoot := filepath.Join(s.StoreOpts.Root, pathKey.FullPath())
	f, err := os.Create(filepath.Join(fullPathWithRoot))
	if err != nil {
		return err
	}

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}

	log.Printf("written (%d) byes to disk: %s", n, fullPathWithRoot)
	return nil
}
func (s *Store) Has(key string) bool {
	pathKey := s.StoreOpts.PathTransformFunc(key)
	fullpathWithRoot := filepath.Join(s.StoreOpts.Root, pathKey.FullPath())

	_, err := os.Stat(fullpathWithRoot)
	return !errors.Is(err, os.ErrNotExist)

}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.StoreOpts.PathTransformFunc(key)
	return os.Open(filepath.Join(s.StoreOpts.Root, pathKey.FullPath()))
}

func (s *Store) CleanUp() error {
	return os.RemoveAll(s.StoreOpts.Root)
}

func (s *Store) Delete(key string) error {
	pathKey := s.StoreOpts.PathTransformFunc(key)
	defer func() {
		log.Printf("deleted [%s] from disk", pathKey.Filename)
	}()
	return s.deleteFullPath(filepath.Join(s.StoreOpts.Root, pathKey.FullPath()))
}

// deleteFullPath doesn't delete the root folder for safety.
func (s *Store) deleteFullPath(path string) error {
	pathSlice := strings.Split(path, "/")
	for i := len(pathSlice); i > 1; i-- {
		if err := os.RemoveAll(strings.Join(pathSlice[:i], string(os.PathSeparator))); err != nil {
			return err
		}
	}
	return nil
}
