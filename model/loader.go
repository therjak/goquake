package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"quake/filesystem"
)

var (
	loaders map[uint32]LoadFunc
)

func init() {
	loaders = make(map[uint32]LoadFunc)
}

func Load(name string) ([]*QModel, error) {
	// TODO: move the cache

	b, err := filesystem.GetFileContents(name)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewReader(b)
	var magic uint32
	err = binary.Read(buf, binary.LittleEndian, &magic)
	if err != nil {
		return nil, err
	}

	f, ok := loaders[magic]
	if !ok {
		return nil, fmt.Errorf("File %s has an unknown file format", name)
	}
	return f(name, b)
}

type LoadFunc func(string, []byte) ([]*QModel, error)

func Register(magic uint32, f LoadFunc) {
	loaders[magic] = f
}
