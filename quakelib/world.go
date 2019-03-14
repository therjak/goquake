package quakelib

//#include "trace.h"
//#include "edict.h"
//#include "cgo_help.h"
//void PR_ExecuteProgram(int p);
import "C"

import (
	"container/ring"
	"quake/math"
	"quake/progs"
)

const (
	MOVE_NORMAL = iota
	MOVE_NOMONSTERS
	MOVE_MISSLE
)

const (
	CONTENTS_EMPTY        = -1
	CONTENTS_SOLID        = -2
	CONTENTS_WATER        = -3
	CONTENTS_SLIME        = -4
	CONTENTS_LAVA         = -5
	CONTENTS_SKY          = -6
	CONTENTS_ORIGIN       = -7
	CONTENTS_CLIP         = -8
	CONTENTS_CURRENT_0    = -9
	CONTENTS_CURRENT_90   = -10
	CONTENTS_CURRENT_180  = -11
	CONTENTS_CURRENT_270  = -12
	CONTENTS_CURRENT_UP   = -13
	CONTENTS_CURRENT_DOWN = -14
)

const (
	DIST_EPSILON = 0.03125 // (1/32) to keep floating point happy
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

// export SV_ClearWorld
func SV_ClearWorld() {
	initBoxHull()
	// gArea = createAreaNode(0, sv.worldmodel.mins, sv.worldmodel.maxs)
}

/*
func InsertLinkBefore() {}
func Edict_From_Area()  {}
*/

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

type moveClip struct {
	boxmins, boxmaxs, mins, maxs, mins2, maxs2, start, end math.Vec3
	trace                                                  C.trace_t
	typ, edict                                             int
}

func clipToLinks(a *areaNode, clip *moveClip) {
	var next *ring.Ring
	for l := a.solidEdicts.Next(); l != a.solidEdicts; l = next {
		next = l.Next()
		touch := l.Value.(int)
		tv := EntVars(touch)
		if tv.Solid == SOLID_NOT {
			continue
		}
		if touch == clip.edict {
			continue
		}
		if tv.Solid == SOLID_TRIGGER {
			Error("Trigger in clipping list")
		}
		if clip.typ == MOVE_NOMONSTERS && tv.Solid != SOLID_BSP {
			continue
		}
		if clip.boxmaxs.X < tv.AbsMin[0] ||
			clip.boxmaxs.Y < tv.AbsMin[1] ||
			clip.boxmaxs.Z < tv.AbsMin[2] ||
			clip.boxmins.X > tv.AbsMax[0] ||
			clip.boxmins.Y > tv.AbsMax[1] ||
			clip.boxmins.Z > tv.AbsMax[2] {
			continue
		}
		if clip.edict >= 0 && EntVars(clip.edict).Size[0] != 0 &&
			tv.Size[0] == 0 {
			continue
		}
		if clip.trace.allsolid != 0 {
			return
		}
		if clip.edict >= 0 {
			if tv.Owner == int32(clip.edict) {
				continue
			}
			if EntVars(clip.edict).Owner == int32(touch) {
				continue
			}
		}
		trace := func() C.trace_t {
			if int(tv.Flags)&FL_MONSTER != 0 {
				return clipMoveToEntity(touch, clip.start, clip.mins2, clip.maxs2, clip.end)
			}
			return clipMoveToEntity(touch, clip.start, clip.mins, clip.maxs, clip.end)
		}()
		if trace.allsolid != 0 || trace.startsolid != 0 ||
			trace.fraction < clip.trace.fraction {
			trace.entn = C.int(touch)
			trace.entp = b2i(true)
			if clip.trace.startsolid != 0 {
				clip.trace = trace
				clip.trace.startsolid = b2i(true)
			} else {
				clip.trace = trace
			}
		} else {
			clip.trace.startsolid = b2i(true)
		}
	}

	if a.axis == -1 {
		return
	}
	if clip.boxmaxs.Idx(a.axis) > a.dist {
		clipToLinks(a.children[0], clip)
	}
	if clip.boxmins.Idx(a.axis) < a.dist {
		clipToLinks(a.children[1], clip)
	}
}

type mPlane struct {
	normal   math.Vec3
	dist     float32
	typ      int // was byte
	signBits int // was byte
	// pad [2]byte
}

type mClipNode struct {
	planeNum int
	children [2]int
}

type hull struct {
	clipNodes     []mClipNode
	planes        []mPlane
	firstClipNode int
	lastClipNode  int
	clipMins      math.Vec3
	clipMaxs      math.Vec3
}

var (
	boxHull hull
)

func initBoxHull() {
	boxHull.clipNodes = make([]mClipNode, 6)
	boxHull.planes = make([]mPlane, 6)
	boxHull.firstClipNode = 0
	boxHull.lastClipNode = 5
	for i := 0; i < 6; i++ {
		boxHull.clipNodes[i].planeNum = i
		side := i & 1
		boxHull.clipNodes[i].children[side] = CONTENTS_EMPTY
		if i == 5 {
			boxHull.clipNodes[i].children[side^1] = CONTENTS_SOLID
		} else {
			boxHull.clipNodes[i].children[side^1] = i + 1
		}
		boxHull.planes[i].typ = i >> 1
		switch i >> 1 {
		case 0:
			boxHull.planes[i].normal.X = 1
		case 1:
			boxHull.planes[i].normal.Y = 1
		case 2:
			boxHull.planes[i].normal.Z = 1
		}
	}
}

func hullForBox(mins, maxs math.Vec3) *hull {
	boxHull.planes[0].dist = maxs.X
	boxHull.planes[1].dist = mins.X
	boxHull.planes[2].dist = maxs.Y
	boxHull.planes[3].dist = mins.Y
	boxHull.planes[4].dist = maxs.Z
	boxHull.planes[5].dist = mins.Z
	return &boxHull
}

func hullForEntity(ent *progs.EntVars, mins, maxs math.Vec3) (*hull, math.Vec3) {
	return nil, math.Vec3{}
}

func hullPointContents(h *hull, num int, p math.Vec3) int {
	for num >= 0 {
		if num < h.firstClipNode || num > h.lastClipNode {
			Error("SV_HullPointContents: bad node number")
		}
		node := h.clipNodes[num]
		plane := h.planes[node.planeNum]
		d := func() float32 {
			if plane.typ < 3 {
				return p.Idx(plane.typ) - plane.dist
			}
			return math.DoublePrecDot(plane.normal, p) - plane.dist
		}()
		if d < 0 {
			num = node.children[1]
		} else {
			num = node.children[0]
		}
	}

	return num
}

func recursiveHullCheck(h *hull, num int, p1f, p2f float32, p1, p2 math.Vec3, trace *C.trace_t) bool {
	if num < 0 { // check for empty
		if num != CONTENTS_SOLID {
			trace.allsolid = b2i(false)
			if num == CONTENTS_EMPTY {
				trace.inopen = b2i(true)
			} else {
				trace.inwater = b2i(true)
			}
		} else {
			trace.startsolid = b2i(true)
		}
		return true
	}
	if num < h.firstClipNode || num > h.lastClipNode {
		Error("RecursiveHullCheck: bad node number")
	}
	node := h.clipNodes[num]
	plane := h.planes[node.planeNum]
	t1, t2 := func() (float32, float32) {
		if plane.typ < 3 {
			return (p1.Idx(plane.typ) - plane.dist),
				(p2.Idx(plane.typ) - plane.dist)
		} else {
			return math.DoublePrecDot(plane.normal, p1) - plane.dist,
				math.DoublePrecDot(plane.normal, p2) - plane.dist
		}
	}()
	if t1 >= 0 && t2 >= 0 {
		return recursiveHullCheck(h, node.children[0], p1f, p2f, p1, p2, trace)
	}
	if t1 < 0 && t2 < 0 {
		return recursiveHullCheck(h, node.children[1], p1f, p2f, p1, p2, trace)
	}
	// put the crosspoint DIST_EPSILON pixels on the near side
	frac := func() float32 {
		if t1 < 0 {
			return (t1 + DIST_EPSILON) / (t1 - t2)
		}
		return (t1 - DIST_EPSILON) / (t1 - t2)
	}()
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	midf := p1f + (p2f-p1f)*frac
	mid := func() math.Vec3 {
		t := math.Sub(p2, p1)
		t = t.Scale(frac)
		return math.Add(p1, t)
	}()
	side := func() int {
		if t1 < 0 {
			return 1
		}
		return 0
	}()
	// move up to the node
	if !recursiveHullCheck(h, node.children[side], p1f, midf, p1, mid, trace) {
		return false
	}
	if hullPointContents(h, node.children[side^1], mid) != CONTENTS_SOLID {
		return recursiveHullCheck(h, node.children[side^1], midf, p2f, mid, p2, trace)
	}
	if trace.allsolid != 0 {
		return false // never got out of the solid area
	}
	// the other side of the node is solid, this is the impact point
	if side == 0 {
		trace.plane.normal[0] = C.float(plane.normal.X)
		trace.plane.normal[1] = C.float(plane.normal.Y)
		trace.plane.normal[2] = C.float(plane.normal.Z)
		trace.plane.dist = C.float(plane.dist)
	} else {
		trace.plane.normal[0] = C.float(-plane.normal.X)
		trace.plane.normal[1] = C.float(-plane.normal.Y)
		trace.plane.normal[2] = C.float(-plane.normal.Z)
		trace.plane.dist = C.float(-plane.dist)
	}
	for hullPointContents(h, h.firstClipNode, mid) == CONTENTS_SOLID {
		// shouldn't really happen, but does occasionally
		frac -= 0.1
		if frac < 0 {
			trace.fraction = C.float(midf)
			trace.endpos[0] = C.float(mid.X)
			trace.endpos[1] = C.float(mid.Y)
			trace.endpos[2] = C.float(mid.Z)
			conPrintf("backup past 0\n")
			return false
		}
		midf = p1f + (p2f-p1f)*frac
		mid = func() math.Vec3 {
			t := math.Sub(p2, p1)
			t = t.Scale(frac)
			return math.Add(p1, t)
		}()
	}
	trace.fraction = C.float(midf)
	trace.endpos[0] = C.float(mid.X)
	trace.endpos[1] = C.float(mid.Y)
	trace.endpos[2] = C.float(mid.Z)

	return false
}

func clipMoveToEntity(ent int, start, mins, maxs, end math.Vec3) C.trace_t {
	var trace C.trace_t
	trace.fraction = 1
	trace.allsolid = b2i(true)
	trace.endpos[0] = C.float(end.X)
	trace.endpos[1] = C.float(end.Y)
	trace.endpos[2] = C.float(end.Z)
	hull, offset := hullForEntity(EntVars(ent), mins, maxs)
	startL := math.Sub(start, offset)
	endL := math.Sub(end, offset)
	recursiveHullCheck(hull, hull.firstClipNode, 0, 1, startL, endL, &trace)

	if trace.fraction != 1 {
		trace.endpos[0] += C.float(offset.X)
		trace.endpos[1] += C.float(offset.Y)
		trace.endpos[2] += C.float(offset.Z)
	}
	if trace.fraction < 1 || trace.startsolid != 0 {
		trace.entn = C.int(ent)
		trace.entp = b2i(true)
	}
	return trace
}

func (c *moveClip) moveBounds(s, e math.Vec3) {
	min, max := math.MinMax(s, e)
	c.boxmins = math.Add(min, math.Add(c.mins, math.Vec3{-1, -1, -1}))
	c.boxmaxs = math.Add(max, math.Add(c.maxs, math.Vec3{1, 1, 1}))
}

func p2v3(p *C.float) math.Vec3 {
	return math.Vec3{
		X: float32(C.cf(0, p)),
		Y: float32(C.cf(1, p)),
		Z: float32(C.cf(2, p)),
	}
}

//export SV_Move
func SV_Move(st, mi, ma, en *C.float, ty C.int, ed C.int) C.trace_t {
	start := p2v3(st)
	mins := p2v3(mi)
	maxs := p2v3(ma)
	end := p2v3(en)

	clip := moveClip{
		trace: clipMoveToEntity(0, start, mins, maxs, end),
		start: start,
		end:   end,
		mins:  mins,
		maxs:  maxs,
		typ:   int(ty),
		edict: int(ed),
		mins2: mins,
		maxs2: maxs,
	}
	if ty == MOVE_MISSLE {
		clip.mins2 = math.Vec3{-15, -15, -15}
		clip.maxs2 = math.Vec3{15, 15, 15}
	}

	// create the bounding box of the entire move
	clip.moveBounds(start, end)

	clipToLinks(gArea, &clip)

	return clip.trace
}
