// SPDX-License-Identifier: GPL-2.0-or-later

package wad

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"goquake/filesystem"
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
	Width  int
	Height int
	Data   []byte
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

func getPics(ls []lump, data []byte) (map[string]*QPic, error) {
	p := make(map[string]*QPic)
	for _, l := range ls {
		if l.Typ != typQPic {
			continue
		}
		d := data[l.Offset : l.Offset+l.Size]
		q := &QPic{
			Width:  int(binary.LittleEndian.Uint32(d[0:])),
			Height: int(binary.LittleEndian.Uint32(d[4:])),
			Data:   d[8:],
		}
		ln := bytes.IndexByte(l.Name[:], 0)
		if ln == -1 {
			ln = len(l.Name)
		}
		name := string(l.Name[:ln])
		p[name] = q
	}
	return p, nil
}

var (
	consoleChars []byte
	pics         map[string]*QPic
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
	pics, err = getPics(lumps, data)
	if err != nil {
		return err
	}
	return nil
}

func GetPic(n string) *QPic {
	name := strings.ToLower(n)
	return pics[name]
}

func GetConsoleChars() []byte {
	return consoleChars
}
