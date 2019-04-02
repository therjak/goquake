package quakelib

//#include "trace.h"
//#include "edict.h"
//#include "cgo_help.h"
//void PR_ExecuteProgram(int p);
import "C"

import (
	"container/ring"
	"log"
	"quake/math"
	"quake/model"
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
	gArea       *areaNode
)

//export SV_ClearWorld
func SV_ClearWorld() {
	clearWorld()
}

// called after the world model has been loaded, before linking any entities
func clearWorld() {
	edictToRing = make(map[int]*ring.Ring)
	initBoxHull()
	gArea = createAreaNode(0, sv.worldModel.Mins, sv.worldModel.Maxs)
}

func createAreaNode(depth int, mins, maxs math.Vec3) *areaNode {
	if depth == 4 {
		return &areaNode{
			axis: -1,
			// We need a 'root' ring to be able to use Prev()
			triggerEdicts: ring.New(1),
			solidEdicts:   ring.New(1),
		}
	}
	an := &areaNode{
		triggerEdicts: ring.New(1),
		solidEdicts:   ring.New(1),
	}
	s := math.Sub(maxs, mins)
	an.axis = func() int {
		if s.X > s.Y {
			return 0
		}
		return 1
	}()
	an.dist = 0.5 * (maxs.Idx(an.axis) + mins.Idx(an.axis))

	mins1 := mins
	mins2 := mins
	maxs1 := maxs
	maxs2 := maxs

	switch an.axis {
	case 0:
		maxs1.X = an.dist
		mins2.X = an.dist
	case 1:
		maxs1.Y = an.dist
		mins2.Y = an.dist
	case 2:
		maxs1.Z = an.dist
		mins2.Z = an.dist
	}

	an.children[0] = createAreaNode(depth+1, mins2, maxs2)
	an.children[1] = createAreaNode(depth+1, mins1, maxs1)

	return an
}

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
	r.Prev().Unlink(1)
}

func triggerEdicts(e int, a *areaNode) []int {
	ret := []int{}
	ev := EntVars(e)

	for l := a.triggerEdicts.Next(); l != a.triggerEdicts; l = l.Next() {
		if l == nil {
			// my area got removed out from under me!
			log.Printf("triggerEdicts: encountered NULL link!\n")
			break
		}
		touch := l.Value.(int)
		if touch == e {
			continue
		}
		tv := EntVars(touch)
		if tv == nil || tv.Touch == 0 || tv.Solid != SOLID_TRIGGER {
			continue
		}
		if ev.AbsMin[0] > tv.AbsMax[0] ||
			ev.AbsMin[1] > tv.AbsMax[1] ||
			ev.AbsMin[2] > tv.AbsMax[2] ||
			ev.AbsMax[0] < tv.AbsMin[0] ||
			ev.AbsMax[1] < tv.AbsMin[1] ||
			ev.AbsMax[2] < tv.AbsMin[2] {
			continue
		}
		ret = append(ret, touch)
	}

	if a.axis == -1 {
		return ret
	}

	if ev.AbsMax[a.axis] > a.dist {
		ret = append(ret, triggerEdicts(e, a.children[0])...)
	}
	if ev.AbsMin[a.axis] < a.dist {
		ret = append(ret, triggerEdicts(e, a.children[1])...)
	}
	return ret
}

func SV_TouchLinks(e int, a *areaNode) {
	te := triggerEdicts(e, a)
	ev := EntVars(e)

	for _, touch := range te {
		if touch == e {
			continue
		}
		tv := EntVars(touch)
		if tv == nil || tv.Touch == 0 || tv.Solid != SOLID_TRIGGER {
			continue
		}
		if ev.AbsMin[0] > tv.AbsMax[0] ||
			ev.AbsMin[1] > tv.AbsMax[1] ||
			ev.AbsMin[2] > tv.AbsMax[2] ||
			ev.AbsMax[0] < tv.AbsMin[0] ||
			ev.AbsMax[1] < tv.AbsMin[1] ||
			ev.AbsMax[2] < tv.AbsMin[2] {
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
}

//export SV_LinkEdict
func SV_LinkEdict(e C.int, touchTriggers C.int) {
	LinkEdict(int(e), touchTriggers != 0)
}

// Needs to be called any time an entity changes origin, mins, max,
// or solid flags ent.v.modified
// sets the related entvar.absmin and entvar.absmax
// if touchTriggers calls prog functions for the intersected triggers
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
	}

	ed.num_leafs = 0
	if ev.ModelIndex != 0 {
		findTouchedLeafs(e, sv.worldModel.Node)
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
	r.Value = e
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

func findTouchedLeafs(e int, node model.Node) {
	if node.Contents() == CONTENTS_SOLID {
		return
	}
	if node.Contents() < 0 {
		// This is a leaf
		ed := C.EDICT_NUM(C.int(e))
		if ed.num_leafs == C.MAX_ENT_LEAFS {
			return
		}
		leaf := node.(*model.MLeaf)
		leafNum := -1
		for i := 0; i < len(sv.worldModel.Leafs); i++ {
			if sv.worldModel.Leafs[i] == leaf {
				leafNum = i - 1 // why -1 ?
			}
		}

		ed.leafnums[ed.num_leafs] = C.int(leafNum)
		ed.num_leafs++
		return
	}
	n := node.(*model.MNode)
	splitplane := n.Plane
	ev := EntVars(e)
	sides := boxOnPlaneSide(math.VFromA(ev.AbsMin), math.VFromA(ev.AbsMax), splitplane)
	if sides&1 != 0 {
		findTouchedLeafs(e, n.Children[0])
	}
	if sides&2 != 0 {
		findTouchedLeafs(e, n.Children[1])
	}
}

func boxOnPlaneSide(mins, maxs math.Vec3, p *model.Plane) int {
	if p.Type < 3 {
		if p.Dist <= mins.Idx(int(p.Type)) {
			return 1
		}
		if p.Dist >= maxs.Idx(int(p.Type)) {
			return 2
		}
		return 3
	}
	d1, d2 := func() (float32, float32) {
		n := p.Normal
		switch p.SignBits {
		case 0:
			d1 := n.X*maxs.X + n.Y*maxs.Y + n.Z*maxs.Z
			d2 := n.X*mins.X + n.Y*mins.Y + n.Z*mins.Z
			return d1, d2
		case 1:
			d1 := n.X*mins.X + n.Y*maxs.Y + n.Z*maxs.Z
			d2 := n.X*maxs.X + n.Y*mins.Y + n.Z*mins.Z
			return d1, d2
		case 2:
			d1 := n.X*maxs.X + n.Y*mins.Y + n.Z*maxs.Z
			d2 := n.X*mins.X + n.Y*maxs.Y + n.Z*mins.Z
			return d1, d2
		case 3:
			d1 := n.X*mins.X + n.Y*mins.Y + n.Z*maxs.Z
			d2 := n.X*maxs.X + n.Y*maxs.Y + n.Z*mins.Z
			return d1, d2
		case 4:
			d1 := n.X*maxs.X + n.Y*maxs.Y + n.Z*mins.Z
			d2 := n.X*mins.X + n.Y*mins.Y + n.Z*maxs.Z
			return d1, d2
		case 5:
			d1 := n.X*mins.X + n.Y*maxs.Y + n.Z*mins.Z
			d2 := n.X*maxs.X + n.Y*mins.Y + n.Z*maxs.Z
			return d1, d2
		case 6:
			d1 := n.X*maxs.X + n.Y*mins.Y + n.Z*mins.Z
			d2 := n.X*mins.X + n.Y*maxs.Y + n.Z*maxs.Z
			return d1, d2
		case 7:
			d1 := n.X*mins.X + n.Y*mins.Y + n.Z*mins.Z
			d2 := n.X*maxs.X + n.Y*maxs.Y + n.Z*maxs.Z
			return d1, d2
		default:
			Error("BoxOnPlaneSide: Bad signbits")
			return 0, 0
		}
	}()
	sides := 0
	if d1 >= p.Dist {
		sides = 1
	}
	if d2 < p.Dist {
		sides |= 2
	}
	return sides
}

type moveClip struct {
	boxmins, boxmaxs, mins, maxs, mins2, maxs2, start, end math.Vec3
	trace                                                  C.trace_t
	typ, edict                                             int
}

func clipToLinks(a *areaNode, clip *moveClip) {
	for l := a.solidEdicts.Next(); l != a.solidEdicts; l = l.Next() {
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

var (
	boxHull model.Hull
)

func initBoxHull() {
	boxHull.ClipNodes = make([]*model.ClipNode, 6)
	boxHull.Planes = make([]*model.Plane, 6)
	boxHull.FirstClipNode = 0
	boxHull.LastClipNode = 5
	for i := 0; i < 6; i++ {
		boxHull.ClipNodes[i] = &model.ClipNode{}
		boxHull.Planes[i] = &model.Plane{}
		boxHull.ClipNodes[i].Plane = boxHull.Planes[i]
		side := i & 1
		boxHull.ClipNodes[i].Children[side] = CONTENTS_EMPTY
		if i == 5 {
			boxHull.ClipNodes[i].Children[side^1] = CONTENTS_SOLID
		} else {
			boxHull.ClipNodes[i].Children[side^1] = i + 1
		}
		boxHull.Planes[i].Type = byte(i >> 1)
		switch i >> 1 {
		case 0:
			boxHull.Planes[i].Normal.X = 1
		case 1:
			boxHull.Planes[i].Normal.Y = 1
		case 2:
			boxHull.Planes[i].Normal.Z = 1
		}
	}
}

func hullForBox(mins, maxs math.Vec3) *model.Hull {
	boxHull.Planes[0].Dist = maxs.X
	boxHull.Planes[1].Dist = mins.X
	boxHull.Planes[2].Dist = maxs.Y
	boxHull.Planes[3].Dist = mins.Y
	boxHull.Planes[4].Dist = maxs.Z
	boxHull.Planes[5].Dist = mins.Z
	return &boxHull
}

func hullForEntity(ent *progs.EntVars, mins, maxs math.Vec3) (*model.Hull, math.Vec3) {
	if ent.Solid == SOLID_BSP {
		if ent.MoveType != progs.MoveTypePush {
			Error("SOLID_BSP without MOVETYPE_PUSH")
		}
		m := sv.models[int(ent.ModelIndex)]
		if m == nil || m.Type != model.ModBrush {
			Error("MOVETYPE_PUSH with a non bsp model")
		}
		s := maxs.X - mins.X
		h := func() *model.Hull {
			if s < 3 {
				return &m.Hulls[0]
			} else if s <= 32 {
				return &m.Hulls[1]
			}
			return &m.Hulls[2]
		}()
		offset := math.Add(math.Sub(h.ClipMins, mins), math.VFromA(ent.Origin))
		return h, offset
	}
	hullmins := math.Sub(math.VFromA(ent.Mins), maxs)
	hullmaxs := math.Sub(math.VFromA(ent.Maxs), mins)
	origin := math.VFromA(ent.Origin)
	return hullForBox(hullmins, hullmaxs), origin
}

func hullPointContents(h *model.Hull, num int, p math.Vec3) int {
	for num >= 0 {
		if num < h.FirstClipNode || num > h.LastClipNode {
			Error("SV_HullPointContents: bad node number")
		}
		node := h.ClipNodes[num]
		plane := node.Plane
		d := func() float32 {
			if plane.Type < 3 {
				return p.Idx(int(plane.Type)) - plane.Dist
			}
			return math.DoublePrecDot(plane.Normal, p) - plane.Dist
		}()
		if d < 0 {
			num = node.Children[1]
		} else {
			num = node.Children[0]
		}
	}

	return num
}

//TODO: export?
func recursiveHullCheck(h *model.Hull, num int, p1f, p2f float32, p1, p2 math.Vec3, trace *C.trace_t) bool {
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
	if num < h.FirstClipNode || num > h.LastClipNode {
		Error("RecursiveHullCheck: bad node number")
	}
	node := h.ClipNodes[num]
	plane := node.Plane
	t1, t2 := func() (float32, float32) {
		if plane.Type < 3 {
			return (p1.Idx(int(plane.Type)) - plane.Dist),
				(p2.Idx(int(plane.Type)) - plane.Dist)
		} else {
			return math.DoublePrecDot(plane.Normal, p1) - plane.Dist,
				math.DoublePrecDot(plane.Normal, p2) - plane.Dist
		}
	}()
	if t1 >= 0 && t2 >= 0 {
		return recursiveHullCheck(h, node.Children[0], p1f, p2f, p1, p2, trace)
	}
	if t1 < 0 && t2 < 0 {
		return recursiveHullCheck(h, node.Children[1], p1f, p2f, p1, p2, trace)
	}

	// put the crosspoint DIST_EPSILON pixels on the near side
	frac := func() float32 {
		d := t1 - t2
		// In the C implementation DIST_EPSILON is a float64..
		if t1 < 0 {
			return (t1 + DIST_EPSILON) / d
		}
		return (t1 - DIST_EPSILON) / d
	}()
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	midf := (1-frac)*p1f + p2f*frac
	mid := math.Lerp(p1, p2, frac)
	side := func() int {
		if t1 < 0 {
			return 1
		}
		return 0
	}()
	// move up to the node
	if !recursiveHullCheck(h, node.Children[side], p1f, midf, p1, mid, trace) {
		return false
	}
	if hullPointContents(h, node.Children[side^1], mid) != CONTENTS_SOLID {
		return recursiveHullCheck(h, node.Children[side^1], midf, p2f, mid, p2, trace)
	}
	if trace.allsolid != 0 {
		return false // never got out of the solid area
	}
	// the other side of the node is solid, this is the impact point
	if side == 0 {
		trace.plane.normal[0] = C.float(plane.Normal.X)
		trace.plane.normal[1] = C.float(plane.Normal.Y)
		trace.plane.normal[2] = C.float(plane.Normal.Z)
		trace.plane.dist = C.float(plane.Dist)
	} else {
		trace.plane.normal[0] = C.float(-plane.Normal.X)
		trace.plane.normal[1] = C.float(-plane.Normal.Y)
		trace.plane.normal[2] = C.float(-plane.Normal.Z)
		trace.plane.dist = C.float(-plane.Dist)
	}
	for hullPointContents(h, h.FirstClipNode, mid) == CONTENTS_SOLID {
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
		midf = (1-frac)*p1f + p2f*frac
		mid = math.Lerp(p1, p2, frac)
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
	recursiveHullCheck(hull, hull.FirstClipNode, 0, 1, startL, endL, &trace)

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
	c.boxmins = math.Add(min, c.mins)
	c.boxmaxs = math.Add(max, c.maxs)
}

func p2v3(p *C.float) math.Vec3 {
	return math.Vec3{
		X: float32(C.cf(0, p)),
		Y: float32(C.cf(1, p)),
		Z: float32(C.cf(2, p)),
	}
}

//export SV_TestEntityPosition
func SV_TestEntityPosition(ent C.int) C.int {
	return b2i(testEntityPosition(int(ent)))
}

func testEntityPosition(ent int) bool {
	ev := EntVars(ent)
	trace := svMove(math.VFromA(ev.Origin), math.VFromA(ev.Mins),
		math.VFromA(ev.Maxs), math.VFromA(ev.Origin), 0, ent)
	return trace.startsolid != 0
}

// mins and maxs are relative
// if the entire move stays in a solid volume, trace.allsolid will be set
// if the starting point is in a solid, it will be allowed to move out to
// an open area
// nomonsters is used for line of sight or edge testing where monsters
// shouldn't be considered solid objects
// passedict is explicitly excluded from clipping checks (normally NULL)
//export SV_Move
func SV_Move(st, mi, ma, en *C.float, ty C.int, ed C.int) C.trace_t {
	start := p2v3(st)
	mins := p2v3(mi)
	maxs := p2v3(ma)
	end := p2v3(en)
	return svMove(start, mins, maxs, end, int(ty), int(ed))
}

func svMove(start, mins, maxs, end math.Vec3, typ, ed int) C.trace_t {
	clip := moveClip{
		trace: clipMoveToEntity(0, start, mins, maxs, end),
		start: start,
		end:   end,
		mins:  mins,
		maxs:  maxs,
		typ:   typ,
		edict: ed,
		mins2: mins,
		maxs2: maxs,
	}
	if typ == MOVE_MISSLE {
		clip.mins2 = math.Vec3{-15, -15, -15}
		clip.maxs2 = math.Vec3{15, 15, 15}
	}

	// create the bounding box of the entire move
	clip.moveBounds(start, end)

	clipToLinks(gArea, &clip)

	return clip.trace
}
