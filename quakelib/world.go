package quakelib

import (
	"container/ring"
)

var (
	edictToRing map[int]*ring.Ring
)

type AreaNode struct {
	axis          int
	dist          float32
	children      [2]*AreaNode
	triggerEdicts *ring.Ring
	solidEdicts   *ring.Ring
}

func InsertLinkBefore() {}
func Edict_From_Area()  {}
func SV_UnlinkEdict()   {}
