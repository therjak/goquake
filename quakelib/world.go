package quakelib

//#include "trace.h"
//void PR_ExecuteProgram(int p);
import "C"

import (
	"container/ring"
)

type areaNode struct {
	axis          int
	dist          float32
	children      [2]*areaNode
	triggerEdicts *ring.Ring
	solidEdicts   *ring.Ring
}

var (
	edictToRing map[int]*ring.Ring
	nextLink    *ring.Ring
	gArea       *areaNode
)

func initBoxHull() {
	// TODO
}

// export SV_ClearWorld
func SV_ClearWorld() {
	initBoxHull()
	// gArea = createAreaNode(0, sv.worldmodel.mins, sv.worldmodel.maxs)
}

func InsertLinkBefore() {}
func Edict_From_Area()  {}

// Needs to be called any time an entity changes origin, mins, maxs, or solid
// flags ent->v.modified
// sets ent->v.absmin and ent->v.absmax
// if touchtriggers, calls prog functions for the intersected triggers
// export SV_UnlinkEdict
func SV_UnlinkEdict(e C.int) {
	r, ok := edictToRing[int(e)]
	if !ok {
		return
	}
	if nextLink == r {
		nextLink = nextLink.Next()
	}
	r.Prev().Unlink(1)
}

func SV_TouchLinks(e int, a *areaNode) {
	ev := EntVars(e)

	for l := a.triggerEdicts.Next(); l != a.triggerEdicts; l = nextLink {
		if l == nil {
			// my area got removed out from under me!
			conPrintf("SV_TouchLinks: encountered NULL link!\n")
			break
		}
		nextLink = l.Next()
		touch := l.Value.(int)
		if touch == e {
			continue
		}
		tv := EntVars(touch)
		if tv == nil || tv.Solid != SOLID_TRIGGER {
			continue
		}
		if ev.AbsMin[0] > tv.AbsMin[0] || ev.AbsMin[1] > tv.AbsMin[1] || ev.AbsMin[2] > tv.AbsMin[2] ||
			ev.AbsMax[0] < tv.AbsMax[0] || ev.AbsMax[1] < tv.AbsMax[1] || ev.AbsMax[2] < tv.AbsMax[2] {
			continue
		}

		oldSelf := progsdat.Globals.Self
		oldOther := progsdat.Globals.Other

		progsdat.Globals.Self = int32(touch)
		progsdat.Globals.Other = int32(e)
		progsdat.Globals.Time = sv.time
		C.PR_ExecuteProgram(C.int(tv.Touch))

		progsdat.Globals.Self = oldSelf
		progsdat.Globals.Other = oldOther
	}

	nextLink = nil

	if a.axis == -1 {
		return
	}

	if ev.AbsMax[a.axis] > a.dist {
		SV_TouchLinks(e, a.children[0])
	}
	if ev.AbsMin[a.axis] < a.dist {
		SV_TouchLinks(e, a.children[1])
	}
}

// export SV_LinkEdict
func SV_LinkEdict(e C.int, touchTriggers C.int) {
}
