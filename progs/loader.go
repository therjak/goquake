package progs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"

	"github.com/therjak/goquake/crc"
	"github.com/therjak/goquake/filesystem"
)

type prog struct {
	CRC uint16
	// The progs.dat expects an edict to have EdictSize 32bit values
	EdictSize   int
	Header      *Header
	Functions   []Function
	Statements  []Statement
	GlobalDefs  []Def
	FieldDefs   []Def
	Globals     *GlobalVars
	RawGlobalsI []int32
	RawGlobalsF []float32
	Alpha       bool
	Strings     map[int32]string
}

func loadProgs() (*prog, error) {
	var crcVal uint16
	b, err := filesystem.GetFileContents("progs.dat")
	if err != nil {
		return nil, fmt.Errorf("Could not load progs.dat, %v", err)
	}
	crcVal = crc.Update(b)
	r := bytes.NewReader(b)
	hdr, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	st, err := readStatements(hdr, r)
	if err != nil {
		return nil, fmt.Errorf("Could not read statements: %v", err)
	}
	fu, err := readFunctions(hdr, r)
	if err != nil {
		return nil, fmt.Errorf("Could not read functions: %v", err)
	}
	gl, rgli, rglf, err := readGlobals(hdr, r)
	if err != nil {
		return nil, fmt.Errorf("Could not read globals: %v", err)
	}
	fd, err := readFieldDefs(hdr, r)
	if err != nil {
		return nil, fmt.Errorf("Could not read field defs: %v", err)
	}
	gd, err := readGlobalDefs(hdr, r)
	if err != nil {
		return nil, fmt.Errorf("Could not read global defs: %v", err)
	}
	sr, err := readStrings(hdr, r)
	if err != nil {
		return nil, fmt.Errorf("Could not read strings: %v", err)
	}

	a := false
	for _, f := range fd {
		n, ok := sr[f.SName]
		if ok && n == "alpha" {
			a = true
			break
		}
	}

	ez := int(hdr.EntityFields)

	return &prog{
		CRC:         crcVal,
		EdictSize:   ez,
		Header:      hdr,
		Functions:   fu,
		Statements:  st,
		GlobalDefs:  gd,
		FieldDefs:   fd,
		Globals:     gl,
		RawGlobalsI: rgli,
		RawGlobalsF: rglf,
		Alpha:       a,
		Strings:     sr,
	}, nil
}

func readHeader(file io.ReadSeeker) (*Header, error) {
	var v Header
	file.Seek(0, io.SeekStart)
	if err := binary.Read(file, binary.LittleEndian, &v); err != nil {
		return nil, fmt.Errorf("Could not read progs %v", err)
	}
	if v.Version != ProgVersion {
		return nil, fmt.Errorf("ProgVersion must be %v but is %v", ProgVersion, v.Version)
	}
	if v.CRC != ProgHeaderCRC {
		return nil, fmt.Errorf("progdefs.h is out of date")
	}
	return &v, nil
}

func readStatements(pr *Header, file io.ReadSeeker) ([]Statement, error) {
	v := make([]Statement, pr.NumStatements)
	_, err := file.Seek(int64(pr.OffsetStatements), io.SeekStart)
	if err != nil {
		return nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func readGlobalDefs(pr *Header, file io.ReadSeeker) ([]Def, error) {
	v := make([]Def, pr.NumGlobalDefs)
	_, err := file.Seek(int64(pr.OffsetGlobalDefs), io.SeekStart)
	if err != nil {
		return nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func readFieldDefs(pr *Header, file io.ReadSeeker) ([]Def, error) {
	v := make([]Def, pr.NumFieldDefs)
	_, err := file.Seek(int64(pr.OffsetFieldDefs), io.SeekStart)
	if err != nil {
		return nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func readFunctions(pr *Header, file io.ReadSeeker) ([]Function, error) {
	v := make([]Function, pr.NumFunctions)
	_, err := file.Seek(int64(pr.OffsetFunctions), io.SeekStart)
	if err != nil {
		return nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func readGlobals(pr *Header, file io.ReadSeeker) (*GlobalVars, []int32, []float32, error) {
	v := make([]int32, pr.NumGlobals)
	_, err := file.Seek(int64(pr.OffsetGlobals), io.SeekStart)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &v); err != nil {
		return nil, nil, nil, err
	}
	gp := unsafe.Pointer(&v[0])
	vf := *(*[]float32)(unsafe.Pointer(&v))
	return (*GlobalVars)(gp), v, vf, nil
}

func readStrings(pr *Header, file io.ReadSeeker) (map[int32]string, error) {
	_, err := file.Seek(int64(pr.OffsetStrings), io.SeekStart)
	if err != nil {
		return nil, err
	}
	b := make([]byte, pr.NumStrings)
	if _, err := file.Read(b); err != nil {
		return nil, err
	}
	bs := bytes.Split(b, []byte{0x00})
	m := make(map[int32]string)
	idx := int32(0)
	for _, s := range bs {
		m[idx] = string(s)
		// log.Printf("ProgsString: [%d] '%X' l:%d", idx, s, len(s))
		idx += int32(len(s)) + 1
	}
	return m, nil
}
