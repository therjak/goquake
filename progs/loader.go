package progs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"quake/crc"
	"quake/filesystem"
)

type LoadedProg struct {
	CRC uint16
	// The progs.dat expects an edict to have EdictSize 32bit values
	EdictSize  int
	Header     *Header
	Functions  []Function
	Statements []Statement
	GlobalDefs []Def
	FieldDefs  []Def
	Globals    *GlobalVars
	Alpha      bool
}

func LoadProgs() (*LoadedProg, error) {
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
	gl, err := readGlobals(hdr, r)
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
	// pr_strings?
	// alpha
	a := false

	ez := int(hdr.EntityFields)

	return &LoadedProg{
		CRC:        crcVal,
		EdictSize:  ez,
		Header:     hdr,
		Functions:  fu,
		Statements: st,
		GlobalDefs: gd,
		FieldDefs:  fd,
		Globals:    gl,
		Alpha:      a,
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

func readGlobals(pr *Header, file io.ReadSeeker) (*GlobalVars, error) {
	var v GlobalVars
	_, err := file.Seek(int64(pr.OffsetGlobals), io.SeekStart)
	if err != nil {
		return nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
