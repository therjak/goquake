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

	typPalette    = 0x40
	typQPic       = 0x42 // 66
	typMipTex     = 0x44
	typConsolePic = 0x45
)

type header struct {
	M          [4]byte
	EntryCount uint32
	DirOffset  uint32
}

type lump struct {
	Offset      int32
	Dsize       int32
	Size        int32
	Typ         byte
	Compression byte
	Dummy       int16
	Name        [16]byte
}

type QPic struct {
	Width  int32
	Height int32
	Data   []byte
}

type qPicHeader struct {
	width  int32
	height int32
}

func getWad() ([]byte, error) {
	file, err := filesystem.GetFile("gfx.wad")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return ioutil.ReadAll(file)
}

func getLumps(data []byte) ([]lump, error) {
	buf := bytes.NewReader(data)
	h := header{}
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return nil, err
	}
	if h.M != [4]byte{'W', 'A', 'D', '2'} {
		return nil, fmt.Errorf("Wad file doesn't have WAD2 id\n")
	}
	lumps := make([]lump, h.EntryCount)
	_, err = buf.Seek(int64(h.DirOffset), io.SeekStart)
	if err != nil {
		return nil, err
	}
	err = binary.Read(buf, binary.LittleEndian, &lumps)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(lumps); i++ {
		copy(lumps[i].Name[:], strings.ToLower(string(lumps[i].Name[:])))
	}
	return lumps, nil
}

const (
	consoleCharsLump = "conchars"
)

func getConChars(ls []lump, data []byte) ([]byte, error) {
	for _, l := range ls {
		if strings.HasPrefix(string(l.Name[:]), consoleCharsLump) {
			return data[l.Offset : l.Offset+l.Size], nil
		}
	}
	return nil, fmt.Errorf("Could not find %v texture", consoleCharsLump)
}

var (
	consoleChars []byte
)

func Load() error {
	data, err := getWad()
	if err != nil {
		return err
	}
	lumps, err := getLumps(data)
	if err != nil {
		return err
	}
	consoleChars, err = getConChars(lumps, data)
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

func GetConsoleChars() []byte {
	return consoleChars
}
