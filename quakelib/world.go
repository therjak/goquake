package quakelib

//#include "trace.h"
//#include "edict.h"
//#include "cgo_help.h"
import "C"

import (
	"container/ring"
	"log"
	"quake/conlog"
	"quake/math"
	"quake/math/vec"
	"quake/model"
	"quake/progs"
	"runtime/debug"
)

const (
	MOVE_NORMAL = iota
	MOVE_NOMONSTERS
	MOVE_MISSILE
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

// called after the world model has been loaded, before linking any entities
func clearWorld() {
	edictToRing = make(map[int]*ring.Ring)
	initBoxHull()
	gArea = createAreaNode(0, sv.worldModel.Mins, sv.worldModel.Maxs)
}

func createAreaNode(depth int, mins, maxs vec.Vec3) *areaNode {
	if depth == 4 {
		return &areaNode{
			axis: -1,
			// We need a 'root' ring to be able to use Prev()
			triggerEdicts: &ring.Ring{},
			solidEdicts:   &ring.Ring{},
		}
	}
	an := &areaNode{
		triggerEdicts: &ring.Ring{},
		solidEdicts:   &ring.Ring{},
	}
	s := vec.Sub(maxs, mins)
	an.axis = func() int {
		if s[0] > s[1] {
			return 0
		}
		return 1
	}()
	an.dist = 0.5 * (maxs[an.axis] + mins[an.axis])

	mins1 := mins
	mins2 := mins
	maxs1 := maxs
	maxs2 := maxs

	switch an.axis {
	case 0:
		maxs1[0] = an.dist
		mins2[0] = an.dist
	case 1:
		maxs1[1] = an.dist
		mins2[1] = an.dist
	case 2:
		maxs1[2] = an.dist
		mins2[2] = an.dist
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
		PRExecuteProgram(tv.Touch)

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
	ed := edictNum(e)
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
		// because movement is clipped an epsilon away from an actual edge,
		// we must fully check even when bounding boxes don't quite touch

		// Therjak: this just breaks a lot of stuff, why?
		ev.AbsMin[0] -= 1
		ev.AbsMin[1] -= 1
		ev.AbsMin[2] -= 1
		ev.AbsMax[0] += 1
		ev.AbsMax[1] += 1
		ev.AbsMax[2] += 1
	}

	ed.num_leafs = 0
	if ev.ModelIndex != 0 {
		findTouchedLeafs(e, sv.worldModel.Node, sv.worldModel)
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

	r := &ring.Ring{Value: e}
	edictToRing[e] = r
	if ev.Solid == SOLID_TRIGGER {
		node.triggerEdicts.Prev().Link(r)
	} else {
		node.solidEdicts.Prev().Link(r)
	}

	if touchTriggers {
		SV_TouchLinks(e, gArea)
	}
}

func findTouchedLeafs(e int, node model.Node, world *model.QModel) {
	if node.Contents() == model.CONTENTS_SOLID {
		return
	}
	if node.Contents() < 0 {
		// This is a leaf
		ed := edictNum(e)
		if ed.num_leafs == C.MAX_ENT_LEAFS {
			return
		}
		leaf := node.(*model.MLeaf)
		leafNum := -2
		for i, l := range world.Leafs { // := 0; i < len(world.Leafs); i++ {
			if l == leaf {
				leafNum = i - 1 // why -1 ?
				if i == 0 {
					log.Printf("Got leafnum -1")
				}
			}
		}
		if leafNum == -2 {
			log.Printf("Got leafnum -2, %d, len(leafs)= %d", leaf.Index, len(world.Leafs))
			debug.PrintStack()
			leafNum = leaf.Index - 1
		}

		ed.leafnums[ed.num_leafs] = C.int(leafNum)
		ed.num_leafs++
		return
	}
	n := node.(*model.MNode)
	splitplane := n.Plane
	ev := EntVars(e)
	sides := boxOnPlaneSide(vec.VFromA(ev.AbsMin), vec.VFromA(ev.AbsMax), splitplane)
	if sides&1 != 0 {
		findTouchedLeafs(e, n.Children[0], world)
	}
	if sides&2 != 0 {
		findTouchedLeafs(e, n.Children[1], world)
	}
}

func boxOnPlaneSide(mins, maxs vec.Vec3, p *model.Plane) int {
	if p.Type < 3 {
		if p.Dist <= mins[int(p.Type)] {
			return 1
		}
		if p.Dist >= maxs[int(p.Type)] {
			return 2
		}
		return 3
	}
	d1, d2 := func() (float32, float32) {
		n := p.Normal
		switch p.SignBits {
		case 0:
			d1 := n[0]*maxs[0] + n[1]*maxs[1] + n[2]*maxs[2]
			d2 := n[0]*mins[0] + n[1]*mins[1] + n[2]*mins[2]
			return d1, d2
		case 1:
			d1 := n[0]*mins[0] + n[1]*maxs[1] + n[2]*maxs[2]
			d2 := n[0]*maxs[0] + n[1]*mins[1] + n[2]*mins[2]
			return d1, d2
		case 2:
			d1 := n[0]*maxs[0] + n[1]*mins[1] + n[2]*maxs[2]
			d2 := n[0]*mins[0] + n[1]*maxs[1] + n[2]*mins[2]
			return d1, d2
		case 3:
			d1 := n[0]*mins[0] + n[1]*mins[1] + n[2]*maxs[2]
			d2 := n[0]*maxs[0] + n[1]*maxs[1] + n[2]*mins[2]
			return d1, d2
		case 4:
			d1 := n[0]*maxs[0] + n[1]*maxs[1] + n[2]*mins[2]
			d2 := n[0]*mins[0] + n[1]*mins[1] + n[2]*maxs[2]
			return d1, d2
		case 5:
			d1 := n[0]*mins[0] + n[1]*maxs[1] + n[2]*mins[2]
			d2 := n[0]*maxs[0] + n[1]*mins[1] + n[2]*maxs[2]
			return d1, d2
		case 6:
			d1 := n[0]*maxs[0] + n[1]*mins[1] + n[2]*mins[2]
			d2 := n[0]*mins[0] + n[1]*maxs[1] + n[2]*maxs[2]
			return d1, d2
		case 7:
			d1 := n[0]*mins[0] + n[1]*mins[1] + n[2]*mins[2]
			d2 := n[0]*maxs[0] + n[1]*maxs[1] + n[2]*maxs[2]
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
	boxmins, boxmaxs, mins, maxs, mins2, maxs2, start, end vec.Vec3
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

		if clip.boxmaxs[0] < tv.AbsMin[0] ||
			clip.boxmaxs[1] < tv.AbsMin[1] ||
			clip.boxmaxs[2] < tv.AbsMin[2] ||
			clip.boxmins[0] > tv.AbsMax[0] ||
			clip.boxmins[1] > tv.AbsMax[1] ||
			clip.boxmins[2] > tv.AbsMax[2] {
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
			if (int(tv.Flags) & FL_MONSTER) != 0 {
				// this just makes monstern easier to hit with missiles
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
		}
	}

	if a.axis == -1 {
		return
	}
	if clip.boxmaxs[a.axis] > a.dist {
		clipToLinks(a.children[0], clip)
	}
	if clip.boxmins[a.axis] < a.dist {
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
		boxHull.ClipNodes[i].Children[side] = model.CONTENTS_EMPTY
		if i == 5 {
			boxHull.ClipNodes[i].Children[side^1] = model.CONTENTS_SOLID
		} else {
			boxHull.ClipNodes[i].Children[side^1] = i + 1
		}
		boxHull.Planes[i].Type = byte(i >> 1)
		switch i >> 1 {
		case 0:
			boxHull.Planes[i].Normal[0] = 1
		case 1:
			boxHull.Planes[i].Normal[1] = 1
		case 2:
			boxHull.Planes[i].Normal[2] = 1
		}
	}
}

func hullForBox(mins, maxs vec.Vec3) *model.Hull {
	boxHull.Planes[0].Dist = maxs[0]
	boxHull.Planes[1].Dist = mins[0]
	boxHull.Planes[2].Dist = maxs[1]
	boxHull.Planes[3].Dist = mins[1]
	boxHull.Planes[4].Dist = maxs[2]
	boxHull.Planes[5].Dist = mins[2]
	return &boxHull
}

func hullForEntity(ent *progs.EntVars, mins, maxs vec.Vec3) (*model.Hull, vec.Vec3) {
	if ent.Solid == SOLID_BSP {
		if ent.MoveType != progs.MoveTypePush {
			Error("SOLID_BSP without MOVETYPE_PUSH")
		}
		m := sv.models[int(ent.ModelIndex)]
		if m == nil || m.Type != model.ModBrush {
			Error("MOVETYPE_PUSH with a non bsp model")
		}
		s := maxs[0] - mins[0]
		h := func() *model.Hull {
			if s < 3 {
				return &m.Hulls[0]
			} else if s <= 32 {
				return &m.Hulls[1]
			}
			return &m.Hulls[2]
		}()
		offset := vec.Add(vec.Sub(h.ClipMins, mins), vec.VFromA(ent.Origin))
		return h, offset
	}
	hullmins := vec.Sub(vec.VFromA(ent.Mins), maxs)
	hullmaxs := vec.Sub(vec.VFromA(ent.Maxs), mins)
	origin := vec.VFromA(ent.Origin)
	return hullForBox(hullmins, hullmaxs), origin
}

func hullPointContents(h *model.Hull, num int, p vec.Vec3) int {
	for num >= 0 {
		if num < h.FirstClipNode || num > h.LastClipNode {
			Error("SV_HullPointContents: bad node number")
		}
		node := h.ClipNodes[num]
		plane := node.Plane
		d := func() float32 {
			if plane.Type < 3 {
				return p[int(plane.Type)] - plane.Dist
			}
			return vec.DoublePrecDot(plane.Normal, p) - plane.Dist
		}()
		if d < 0 {
			num = node.Children[1]
		} else {
			num = node.Children[0]
		}
	}

	return num
}

//export SV_PointContents
func SV_PointContents(p *C.float) C.int {
	return C.int(pointContents(p2v3(p)))
}

func pointContents(p vec.Vec3) int {
	return hullPointContents(&sv.worldModel.Hulls[0], 0, p)
}

//TODO: export?
func recursiveHullCheck(h *model.Hull, num int, p1f, p2f float32, p1, p2 vec.Vec3, trace *C.trace_t) bool {
	if num < 0 { // check for empty
		if num != model.CONTENTS_SOLID {
			trace.allsolid = b2i(false)
			if num == model.CONTENTS_EMPTY {
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
			return (p1[int(plane.Type)] - plane.Dist),
				(p2[int(plane.Type)] - plane.Dist)
		} else {
			return vec.DoublePrecDot(plane.Normal, p1) - plane.Dist,
				vec.DoublePrecDot(plane.Normal, p2) - plane.Dist
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
	frac = math.Clamp32(0, frac, 1)
	midf := math.Lerp(p1f, p2f, frac)
	mid := vec.Lerp(p1, p2, frac)
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
	if hullPointContents(h, node.Children[side^1], mid) != model.CONTENTS_SOLID {
		return recursiveHullCheck(h, node.Children[side^1], midf, p2f, mid, p2, trace)
	}
	if trace.allsolid != 0 {
		return false // never got out of the solid area
	}
	// the other side of the node is solid, this is the impact point
	if side == 0 {
		trace.plane.normal[0] = C.float(plane.Normal[0])
		trace.plane.normal[1] = C.float(plane.Normal[1])
		trace.plane.normal[2] = C.float(plane.Normal[2])
		trace.plane.dist = C.float(plane.Dist)
	} else {
		trace.plane.normal[0] = C.float(-plane.Normal[0])
		trace.plane.normal[1] = C.float(-plane.Normal[1])
		trace.plane.normal[2] = C.float(-plane.Normal[2])
		trace.plane.dist = C.float(-plane.Dist)
	}
	for hullPointContents(h, h.FirstClipNode, mid) == model.CONTENTS_SOLID {
		// shouldn't really happen, but does occasionally
		frac -= 0.1
		if frac < 0 {
			trace.fraction = C.float(midf)
			trace.endpos[0] = C.float(mid[0])
			trace.endpos[1] = C.float(mid[1])
			trace.endpos[2] = C.float(mid[2])
			conlog.DPrintf("backup past 0\n")
			return false
		}
		midf = math.Lerp(p1f, p2f, frac)
		mid = vec.Lerp(p1, p2, frac)
	}
	trace.fraction = C.float(midf)
	trace.endpos[0] = C.float(mid[0])
	trace.endpos[1] = C.float(mid[1])
	trace.endpos[2] = C.float(mid[2])

	return false
}

func clipMoveToEntity(ent int, start, mins, maxs, end vec.Vec3) C.trace_t {
	var trace C.trace_t
	trace.fraction = 1
	trace.allsolid = b2i(true)
	trace.endpos[0] = C.float(end[0])
	trace.endpos[1] = C.float(end[1])
	trace.endpos[2] = C.float(end[2])
	hull, offset := hullForEntity(EntVars(ent), mins, maxs)
	startL := vec.Sub(start, offset)
	endL := vec.Sub(end, offset)
	recursiveHullCheck(hull, hull.FirstClipNode, 0, 1, startL, endL, &trace)

	if trace.fraction != 1 {
		trace.endpos[0] += C.float(offset[0])
		trace.endpos[1] += C.float(offset[1])
		trace.endpos[2] += C.float(offset[2])
	}
	if trace.fraction < 1 || trace.startsolid != 0 {
		trace.entn = C.int(ent)
		trace.entp = b2i(true)
	}
	return trace
}

func (c *moveClip) moveBounds(s, e vec.Vec3) {
	min, max := vec.MinMax(s, e)
	c.boxmins = vec.Sub(vec.Add(min, c.mins2), vec.Vec3{1, 1, 1})
	c.boxmaxs = vec.Add(vec.Add(max, c.maxs2), vec.Vec3{1, 1, 1})
}

func p2v3(p *C.float) vec.Vec3 {
	return vec.Vec3{
		float32(C.cf(0, p)),
		float32(C.cf(1, p)),
		float32(C.cf(2, p)),
	}
}

//export SV_TestEntityPosition
func SV_TestEntityPosition(ent C.int) C.int {
	return b2i(testEntityPosition(int(ent)))
}

func testEntityPosition(ent int) bool {
	ev := EntVars(ent)
	trace := svMove(vec.VFromA(ev.Origin), vec.VFromA(ev.Mins),
		vec.VFromA(ev.Maxs), vec.VFromA(ev.Origin), MOVE_NORMAL, ent)
	return trace.startsolid != 0
}

// mins and maxs are relative
// if the entire move stays in a solid volume, trace.allsolid will be set
// if the starting point is in a solid, it will be allowed to move out to
// an open area
// nomonsters is used for line of sight or edge testing where monsters
// shouldn't be considered solid objects
// passedict is explicitly excluded from clipping checks (normally NULL)
func svMove(start, mins, maxs, end vec.Vec3, typ, ed int) C.trace_t {
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
	if typ == MOVE_MISSILE {
		clip.mins2 = vec.Vec3{-15, -15, -15}
		clip.maxs2 = vec.Vec3{15, 15, 15}
	}

	// create the bounding box of the entire move
	clip.moveBounds(start, end)

	clipToLinks(gArea, &clip)

	return clip.trace
}

const (
	kStepSize = 18
)

//Returns false if any part of the bottom of the entity is off an edge that
//is not a staircase.
func checkBottom(ent int) bool {
	ev := EntVars(ent)
	o := vec.VFromA(ev.Origin)
	mins := vec.Add(o, vec.VFromA(ev.Mins))
	maxs := vec.Add(o, vec.VFromA(ev.Maxs))

	// if all of the points under the corners are solid world, don't bother
	// with the tougher checks
	d := []vec.Vec3{
		vec.Vec3{mins[0], mins[1], mins[2] - 1},
		vec.Vec3{mins[0], maxs[1], mins[2] - 1},
		vec.Vec3{maxs[0], mins[1], mins[2] - 1},
		vec.Vec3{maxs[0], maxs[1], mins[2] - 1},
	}
	for _, start := range d {
		if pointContents(start) != model.CONTENTS_SOLID {
			return expensiveCheckBottom(ent, mins, maxs)
		}
	}
	return true
}

func expensiveCheckBottom(ent int, mins, maxs vec.Vec3) bool {
	level := mins[2]
	below := mins[2] - 2*kStepSize
	start := vec.Vec3{
		(mins[0] + maxs[0]) * 0.5,
		(mins[1] + maxs[1]) * 0.5,
		level,
	}
	stop := vec.Vec3{start[0], start[1], below}
	trace := svMove(start, vec.Vec3{}, vec.Vec3{}, stop, MOVE_NOMONSTERS, ent)

	if trace.fraction == 1.0 {
		return false
	}
	mid := trace.endpos[2]
	bottom := trace.endpos[2]

	d := []vec.Vec3{
		vec.Vec3{mins[0], mins[1], 0},
		vec.Vec3{mins[0], maxs[1], 0},
		vec.Vec3{maxs[0], mins[1], 0},
		vec.Vec3{maxs[0], maxs[1], 0},
	}

	for _, p := range d {
		start := vec.Vec3{p[0], p[1], level}
		stop := vec.Vec3{p[0], p[1], below}
		trace := svMove(start, vec.Vec3{}, vec.Vec3{}, stop, MOVE_NOMONSTERS, ent)

		if trace.fraction != 1.0 && trace.endpos[2] > bottom {
			bottom = trace.endpos[2]
		}
		if trace.fraction == 1.0 || mid-trace.endpos[2] > kStepSize {
			return false
		}
	}
	return true
}
