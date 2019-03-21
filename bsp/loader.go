package bsp

import (
	"log"
	"quake/filesystem"
	qm "quake/model"
)

var (
	polyMagic   = [4]byte{'I', 'D', 'P', 'O'}
	spriteMagic = [4]byte{'I', 'D', 'S', 'P'}
)

func LoadModel(name string) (*qm.QModel, error) {
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
		// LoadBrushModel, this should be a .bsp
	}
	return nil, nil
}
