package bsp

import (
	"bytes"
	"encoding/binary"
	"log"
	"quake/filesystem"
	qm "quake/model"
)

var (
	polyMagic   = [4]byte{'I', 'D', 'P', 'O'}
	spriteMagic = [4]byte{'I', 'D', 'S', 'P'}
)

func LoadModel(name string) (*qm.QModel, error) {
	// TODO: Add cache

	b, err := filesystem.GetFileContents(name)
	if err != nil {
		return nil, err
	}
	var magic [4]byte
	copy(magic[:], b)
	switch magic {
	case polyMagic:
		log.Printf("Got a poly %s", name)
	// LoadAliasModel, this is a .mdl
	case spriteMagic:
		log.Printf("Got a sprite %s", name)
		// LoadSpriteModel, this is a .spr
	default:
		log.Printf("Got a bsp %s", name)
		return LoadBSP(name, b)
	}
	return nil, nil
}

const (
	bspVersion       = 29
	bsp2Version_2psb = 'B'<<24 | 'S'<<16 | 'P'<<8 | '2'
	bsp2Version_bsp2 = '2'<<24 | 'P'<<16 | 'S'<<8 | 'B'
)

func LoadBSP(name string, data []byte) (*qm.QModel, error) {
	ret := &qm.QModel{
		Name: name,
		Type: qm.ModBrush,
		// Mins, Maxs, ClipMins, Clipmaxs Vec3
		// NumSubmodels int
		// leafs []*qm.MLeaf
		// Node qm.Node
		// Hulls [4]qm.Hull
	}
	buf := bytes.NewReader(data)
	h := header{}
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return nil, err
	}
	switch h.Version {
	case bspVersion:
		log.Printf("Got V0 bsp: %v", h)
		// read leafs
		// read nodes
		// read clipnodes
		// read 'submodels', submodel[0] is the 'map'
		// HeadNode [0] == first bsp node index
		// [1] == first clip node index
		// [2] == last clip node index
		// [3] usually 0

	case bsp2Version_2psb:
		log.Printf("Got V1 bsp: %v", h)
	case bsp2Version_bsp2:
		log.Printf("Got V2 bsp: %v", h)
	default:
		log.Printf("Version %v", h.Version)
	}

	return ret, nil
}
