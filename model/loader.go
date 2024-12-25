// SPDX-License-Identifier: GPL-2.0-or-later

package model

import (
	"encoding/binary"
	"fmt"

	"goquake/filesystem"
)

var (
	loaders map[uint32]LoadFunc
)

func init() {
	loaders = make(map[uint32]LoadFunc)
}

func Load(name string) ([]Model, error) {
	// TODO: move the cache

	file, err := filesystem.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var magic uint32
	err = binary.Read(file, binary.LittleEndian, &magic)
	if err != nil {
		return nil, err
	}

	f, ok := loaders[magic]
	if !ok {
		return nil, fmt.Errorf("File %s has an unknown file format", name)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}
	return f(name, file)
}

type LoadFunc func(string, filesystem.File) ([]Model, error)

func Register(magic uint32, f LoadFunc) {
	loaders[magic] = f
}
