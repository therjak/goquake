package quakelib

//#include "trace.h"
//#include "edict.h"
//void PR_ExecuteProgram(int p);
import "C"

import (
	"container/ring"
)

const (
	MOVE_NORMAL = iota
	MOVE_NOMONSTERS
	MOVE_MISSLE
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
//export SV_UnlinkEdict
func SV_UnlinkEdict(e C.int) {
	UnlinkEdict(int(e))
}

func UnlinkEdict(e int) {
	r, ok := edictToRing[e]
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

//export SV_LinkEdict
func SV_LinkEdict(e C.int, touchTriggers C.int) {
	LinkEdict(int(e), touchTriggers != 0)
}

func LinkEdict(e int, touchTriggers bool) {
	UnlinkEdict(e)
	if e == 0 {
		return // don't add the world
	}
	ed := C.EDICT_NUM(C.int(e))
	if ed.free != 0 {
		return
	}
	ev := EntVars(e)

	ev.AbsMin[0] = ev.Origin[0] + ev.Mins[0]
	ev.AbsMin[1] = ev.Origin[1] + ev.Mins[1]
	ev.AbsMin[2] = ev.Origin[2] + ev.Mins[2]
	ev.AbsMax[0] = ev.Origin[0] + ev.Maxs[0]
	ev.AbsMax[1] = ev.Origin[1] + ev.Maxs[1]
	ev.AbsMax[2] = ev.Origin[2] + ev.Maxs[2]

	if (int(ev.Flags) & FL_ITEM) != 0 {
		// make items easier to pick up
		ev.AbsMin[0] -= 15
		ev.AbsMin[1] -= 15
		ev.AbsMax[0] += 15
		ev.AbsMax[1] += 15
	} else {
		// movement is clipped an epsilon away from the actual edge
		// we must fully check even when the bounding boxes don't quite touch
		ev.AbsMin[0] -= 1
		ev.AbsMin[1] -= 1
		ev.AbsMin[2] -= 1
		ev.AbsMax[0] += 1
		ev.AbsMax[1] += 1
		ev.AbsMax[2] += 1
	}

	ed.num_leafs = 0
	if ev.ModelIndex != 0 {
		// TODO:
		// SV_FindTouchedLeafs(e, sv.worldmodel->nodes)
	}
	if ev.Solid == SOLID_NOT {
		return
	}

	node := gArea
	for {
		if node.axis == -1 {
			break
		}
		if ev.AbsMin[node.axis] > node.dist {
			node = node.children[0]
		} else if ev.AbsMax[node.axis] < node.dist {
			node = node.children[1]
		} else {
			break
		}
	}

	r := ring.New(1)
	edictToRing[int(e)] = r
	if ev.Solid == SOLID_TRIGGER {
		node.triggerEdicts.Prev().Link(r)
	} else {
		node.solidEdicts.Prev().Link(r)
	}

	if touchTriggers {
		SV_TouchLinks(e, gArea)
	}
}

type moveClip C.moveclip_t

//export SV_ClipToLinks
func SV_ClipToLinks(clip *moveClip) {
	ClipToLinks(gArea, clip)
}

func ClipToLinks(a *areaNode, clip *moveClip) {
	var next *ring.Ring
	for l := a.solidEdicts.Next(); l != a.solidEdicts; l = next {
		next = l.Next()
		touch := l.Value.(int)
		tv := EntVars(touch)
		if tv.Solid == SOLID_NOT {
			continue
		}
		if touch == int(clip.passedict) {
			continue
		}
		if tv.Solid == SOLID_TRIGGER {
			Error("Trigger in clipping list")
		}
		if clip.Type == MOVE_NOMONSTERS && tv.Solid != SOLID_BSP {
			continue
		}
		if float32(clip.boxmaxs[0]) < tv.AbsMin[0] ||
			float32(clip.boxmaxs[1]) < tv.AbsMin[1] ||
			float32(clip.boxmaxs[2]) < tv.AbsMin[2] ||
			float32(clip.boxmins[0]) > tv.AbsMax[0] ||
			float32(clip.boxmins[1]) > tv.AbsMax[1] ||
			float32(clip.boxmins[2]) > tv.AbsMax[2] {
			continue
		}
		if clip.passedict >= 0 && EntVars(int(clip.passedict)).Size[0] != 0 &&
			tv.Size[0] == 0 {
			continue
		}
		if clip.trace.allsolid != 0 {
			return
		}
		if clip.passedict >= 0 {
			if tv.Owner == int32(clip.passedict) {
				continue
			}
			if EntVars(int(clip.passedict)).Owner == int32(touch) {
				continue
			}
		}
		// TODO: missing stuff
	}

	if a.axis == -1 {
		return
	}
	if float32(clip.boxmaxs[a.axis]) > a.dist {
		ClipToLinks(a.children[0], clip)
	}
	if float32(clip.boxmins[a.axis]) < a.dist {
		ClipToLinks(a.children[1], clip)
	}
}
