package p2p

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "bestpictures"
	pathKey := CASPathTransformFunc(key)
	expectedOriginalName := "b2f8f1dd50fdeec113ac1b5066d1d3a10f70f1dc"
	expectedPathName := "b2f8f/1dd50/fdeec/113ac/1b506/6d1d3/a10f7/0f1dc"
	if pathKey.PathName != expectedPathName {
		t.Errorf("have %s want %s", pathKey.PathName, expectedPathName)
	}
	if pathKey.Filename != expectedOriginalName {
		t.Errorf("have %s want %s", pathKey.Filename, expectedOriginalName)
	}
}

func TestStoreWrite(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)
	data := bytes.NewReader([]byte("some jpg bytes"))
	err := s.writeStream("myspecialPicture", data)
	if err != nil {
		t.Error(err)
	}
}

func TestStoreRead(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	//Setup with data.
	s := NewStore(opts)
	data := bytes.NewReader([]byte("some jpg bytes"))
	err := s.writeStream("myspecialPicture", data)

	//Test Read
	r, err := s.Read("myspecialPicture")
	if err != nil {
		t.Errorf("failed to read the file: %s", err)
	}
	expectedValue := "some jpg bytes"
	b, _ := io.ReadAll(r)
	assert.Equal(t, expectedValue, string(b))
}

func TestStoreDeleteKey(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	//Setup with data.
	key := "myspecialPicture"
	s := NewStore(opts)
	data := bytes.NewReader([]byte("some jpg bytes"))
	err := s.writeStream(key, data)
	if err != nil {
		t.Error(err)
	}

	if err := s.Delete(key); err != nil {
		t.Error(err)
	}
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	//Setup with data.
	key := "myspecialPicture"
	s := NewStore(opts)
	data := bytes.NewReader([]byte("some jpg bytes"))

	err := s.writeStream(key, data)
	if err != nil {
		t.Error(err)
	}

	// Read

	r, err := s.Read("myspecialPicture")
	if err != nil {
		t.Errorf("failed to read the file: %s", err)
	}
	expectedValue := "some jpg bytes"
	b, _ := io.ReadAll(r)
	assert.Equal(t, expectedValue, string(b))

	//Delete
	if err := s.Delete(key); err != nil {
		t.Error(err)
	}
}
