package wad

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"quake/filesystem"
	"strings"
)

const (
	magic = 'W' | 'A'<<8 | 'D'<<16 | '2'<<24
)

type header struct {
	m          [4]byte
	entryCount uint32
	dirOffset  uint32
}

type lump struct {
	offset      int32
	dsize       int32
	size        int32
	t           byte
	compression byte
	dummy       int16
	name        [16]byte
}

var (
	lumps []lump
	data  []byte
)

type QPic struct {
	width  int32
	height int32
	data   []byte
}

type qPicHeader struct {
	width  int32
	height int32
}

func LoadWad() error {
	file, err := filesystem.GetFile("gfx.wad")
	if err != nil {
		return err
	}
	defer file.Close()
	data, err = ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	buf := bytes.NewReader(data)
	h := header{}
	binary.Read(buf, binary.LittleEndian, &h)
	if h.m != [4]byte{'W', 'A', 'D', '2'} {
		return fmt.Errorf("Wad file doesn't have WAD2 id\n")
	}
	lumps = make([]lump, h.entryCount)
	_, err = buf.Seek(int64(h.dirOffset), io.SeekStart)
	if err != nil {
		return err
	}
	err = binary.Read(buf, binary.LittleEndian, &lumps)
	if err != nil {
		return err
	}
	return nil
}

func GetLump(n string) ([]byte, error) {
	name := strings.ToLower(n)
	log.Printf("name: %v", name)
	return nil, nil
}
