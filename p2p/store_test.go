package p2p

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	//Setup with data.
	return NewStore(opts)
}

func tearDown(t *testing.T, s *Store) {
	if err := s.CleanUp(); err != nil {
		t.Error(err)
	}
}

func TestPathTransformFunc(t *testing.T) {
	key := "bestpictures"
	pathKey := CASPathTransformFunc(key)
	expectedFilename := "b2f8f1dd50fdeec113ac1b5066d1d3a10f70f1dc"
	expectedPathName := "b2f8f/1dd50/fdeec/113ac/1b506/6d1d3/a10f7/0f1dc"
	if pathKey.PathName != expectedPathName {
		t.Errorf("have %s want %s", pathKey.PathName, expectedPathName)
	}
	if pathKey.Filename != expectedFilename {
		t.Errorf("have %s want %s", pathKey.Filename, expectedFilename)
	}
}

func TestStore(t *testing.T) {
	s := newStore()
	defer tearDown(t, s)
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("foo_%d", i)
		data := []byte("some jpg bytes")

		_, err := s.writeStream(key, bytes.NewReader(data))
		if err != nil {
			t.Error(err)
		}

		if ok := s.Has(key); ok == false {
			t.Errorf("file [%s] NOT exists.", key)
		}

		// Read
		r, err := s.Read(key)
		if err != nil {
			t.Errorf("failed to read the file: %s", err)
		}
		b, _ := io.ReadAll(r)
		assert.Equal(t, string(data), string(b))

		//Delete
		if err := s.Delete(key); err != nil {
			t.Error(err)
		}
		if ok := s.Has(key); ok == true {
			t.Errorf("file [%s] still exists.", key)
		}
	}
}
