package pack

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

type header struct {
	ID     [4]byte
	Offset int32
	Size   int32
}

type entry struct {
	Name   [56]byte
	Offset int32
	Size   int32
}

type Pack struct {
	f     *os.File
	files map[string]*qfile
	name  string
}

type qfile struct {
	io.SectionReader
}

func (q *qfile) Close() error {
	_, err := q.Seek(0, io.SeekStart)
	return err
}

// GetFile returns a io.SectionReader or nil if the pak has no entry with the
// provided name.
func (p *Pack) GetFile(name string) *qfile {
	return p.files[name]
}

func (p *Pack) String() string {
	return p.name
}

func (p *Pack) Close() error {
	return p.f.Close()
}

func newPack(name string) (*Pack, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return &Pack{f: f, name: name}, nil
}

func (p *Pack) init() error {
	var h header
	if err := binary.Read(p.f, binary.LittleEndian, &h); err != nil {
		return err
	}
	magic := []byte("PACK")
	if !bytes.Equal(magic, h.ID[:]) {
		return errors.New("Not a pack")
	}
	r, err := p.f.Seek(int64(h.Offset), 0)
	if err != nil {
		return err
	}
	if r != int64(h.Offset) {
		return errors.New("Not long enough")
	}
	filenum := h.Size / 64 // 64 is Sizeof(entry)
	p.files = make(map[string]*qfile, filenum)
	for i := int32(0); i < filenum; i++ {
		var e entry
		if err := binary.Read(p.f, binary.LittleEndian, &e); err != nil {
			return err
		}
		n := bytes.IndexByte(e.Name[:], 0)
		name := string(e.Name[:n])
		if p.files[name] != nil {
			return errors.New("files in pack are not unique")
		}
		p.files[name] = &qfile{*io.NewSectionReader(p.f, int64(e.Offset), int64(e.Size))}
	}
	return nil
}

func NewPackReader(name string) (*Pack, error) {
	p, err := newPack(name)
	if err != nil {
		return nil, err
	}
	if err := p.init(); err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}
