package quakelib

//#include "trace.h"
import "C"

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

// Needs to be called any time an entity changes origin, mins, maxs, or solid
// flags ent->v.modified
// sets ent->v.absmin and ent->v.absmax
// if touchtriggers, calls prog functions for the intersected triggers
// export SV_UnlinkEdict
func SV_UnlinkEdict(e C.int) {
}

func SV_TouchLinks(e int, a *AreaNode) {
}

// export SV_LinkEdict
func SV_LinkEdict(e C.int, touchTriggers C.int) {
}
