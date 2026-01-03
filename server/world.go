// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"log"
	"runtime/debug"

	"goquake/bsp"
	"goquake/math/vec"
	"goquake/model"
	"goquake/progs"
	"goquake/ring"
)

const (
	MOVE_NORMAL = iota
	MOVE_NOMONSTERS
	MOVE_MISSILE
)

type areaNode struct {
	axis          int
	dist          float32
	children      [2]*areaNode
	triggerEdicts *ring.Ring[int]
	solidEdicts   *ring.Ring[int]
}

var (
	edictToRing map[int]*ring.Ring[int]
	gArea       *areaNode

	boxHull bsp.Hull
)

// called after the world model has been loaded, before linking any entities
func (s *Server) clearWorld() {
	edictToRing = make(map[int]*ring.Ring[int])
	initBoxHull()
	gArea = createAreaNode(0, s.worldModel.Mins(), s.worldModel.Maxs())
}

func createAreaNode(depth int, mins, maxs vec.Vec3) *areaNode {
	if depth == 4 {
		return &areaNode{
			axis: -1,
			// We need a 'root' ring to be able to use Prev()
			triggerEdicts: &ring.Ring[int]{},
			solidEdicts:   &ring.Ring[int]{},
		}
	}
	an := &areaNode{
		triggerEdicts: &ring.Ring[int]{},
		solidEdicts:   &ring.Ring[int]{},
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
func (v *virtualMachine) UnlinkEdict(e int) {
	r, ok := edictToRing[e]
	if !ok {
		return
	}
	r.Prev().Unlink(1)
}

func triggerEdicts(e int, a *areaNode) []int {
	ret := []int{}
	ev := entvars.Get(e)

	for l := a.triggerEdicts.Next(); l != a.triggerEdicts; l = l.Next() {
		if l == nil {
			// my area got removed out from under me!
			log.Printf("triggerEdicts: encountered NULL link!\n")
			break
		}
		touch := l.Value
		if touch == e {
			continue
		}
		tv := entvars.Get(touch)
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

func (v *virtualMachine) touchLinks(e int, a *areaNode, s *Server) error {
	te := triggerEdicts(e, a)
	ev := entvars.Get(e)

	for _, touch := range te {
		if touch == e {
			continue
		}
		tv := entvars.Get(touch)
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
		progsdat.Globals.Time = s.time
		if err := v.ExecuteProgram(tv.Touch, s); err != nil {
			return err
		}

		progsdat.Globals.Self = oldSelf
		progsdat.Globals.Other = oldOther
	}
	return nil
}

// Needs to be called any time an entity changes origin, mins, max,
// or solid flags ent.v.modified
// sets the related entvar.absmin and entvar.absmax
// if touchTriggers calls prog functions for the intersected triggers
func (v *virtualMachine) LinkEdict(e int, touchTriggers bool, s *Server) error {
	v.UnlinkEdict(e)
	if e == 0 {
		return nil // don't add the world
	}
	ed := &s.edicts[e]
	if ed.Free {
		return nil
	}
	ev := entvars.Get(e)

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
		findTouchedLeafs(e, s.worldModel.Node, s.worldModel, ed)
	}

	if ev.Solid == SOLID_NOT {
		return nil
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

	r := &ring.Ring[int]{Value: e}
	edictToRing[e] = r
	if ev.Solid == SOLID_TRIGGER {
		node.triggerEdicts.Prev().Link(r)
	} else {
		node.solidEdicts.Prev().Link(r)
	}

	if touchTriggers {
		if err := v.touchLinks(e, gArea, s); err != nil {
			return err
		}
	}
	return nil
}

func findTouchedLeafs(e int, node bsp.Node, world *bsp.Model, ed *Edict) {
	if node.Contents() == bsp.CONTENTS_SOLID {
		return
	}
	if node.Contents() < 0 {
		// This is a leaf
		if ed.num_leafs == MAX_ENT_LEAFS {
			return
		}
		leaf := node.(*bsp.MLeaf)
		leafNum := -2
		for i, l := range world.Leafs {
			if l == leaf {
				leafNum = i - 1 // -1 to remove the solid 0 leaf
			}
		}
		if leafNum == -2 {
			log.Printf("Got leafnum -2, len(leafs)= %d", len(world.Leafs))
			debug.PrintStack()
		}

		ed.leafnums[ed.num_leafs] = leafNum
		ed.num_leafs++
		return
	}
	n := node.(*bsp.MNode)
	splitplane := n.Plane
	ev := entvars.Get(e)
	sides := splitplane.BoxOnPlaneSide(vec.VFromA(ev.AbsMin), vec.VFromA(ev.AbsMax))
	if sides&1 != 0 {
		findTouchedLeafs(e, n.Children[0], world, ed)
	}
	if sides&2 != 0 {
		findTouchedLeafs(e, n.Children[1], world, ed)
	}
}

type moveClip struct {
	boxmins, boxmaxs, mins, maxs, mins2, maxs2, start, end vec.Vec3
	trace                                                  bsp.Trace
	typ, edict                                             int
}

func clipToLinks(a *areaNode, clip *moveClip, s *Server) {
	for l := a.solidEdicts.Next(); l != a.solidEdicts; l = l.Next() {
		touch := l.Value
		tv := entvars.Get(touch)
		if tv.Solid == SOLID_NOT {
			continue
		}
		if touch == clip.edict {
			continue
		}
		if tv.Solid == SOLID_TRIGGER {
			debug.PrintStack()
			log.Fatalf("Trigger in clipping list")
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

		if clip.edict >= 0 && entvars.Get(clip.edict).Size[0] != 0 &&
			tv.Size[0] == 0 {
			continue
		}
		if clip.trace.AllSolid {
			return
		}
		if clip.edict >= 0 {
			if tv.Owner == int32(clip.edict) {
				continue
			}
			if entvars.Get(clip.edict).Owner == int32(touch) {
				continue
			}
		}
		t := func() bsp.Trace {
			if (int(tv.Flags) & FL_MONSTER) != 0 {
				// this just makes monstern easier to hit with missiles
				return clipMoveToEntity(touch, clip.start, clip.mins2, clip.maxs2, clip.end, s)
			}
			return clipMoveToEntity(touch, clip.start, clip.mins, clip.maxs, clip.end, s)
		}()
		if t.AllSolid || t.StartSolid || t.Fraction < clip.trace.Fraction {
			t.EntNumber = touch
			t.EntPointer = true
			if clip.trace.StartSolid {
				clip.trace = t
				clip.trace.StartSolid = true
			} else {
				clip.trace = t
			}
		}
	}

	if a.axis == -1 {
		return
	}
	if clip.boxmaxs[a.axis] > a.dist {
		clipToLinks(a.children[0], clip, s)
	}
	if clip.boxmins[a.axis] < a.dist {
		clipToLinks(a.children[1], clip, s)
	}
}

func initBoxHull() {
	boxHull.ClipNodes = make([]*bsp.ClipNode, 6)
	boxHull.Planes = make([]*bsp.Plane, 6)
	boxHull.FirstClipNode = 0
	boxHull.LastClipNode = 5
	for i := 0; i < 6; i++ {
		boxHull.ClipNodes[i] = &bsp.ClipNode{}
		boxHull.Planes[i] = &bsp.Plane{}
		boxHull.ClipNodes[i].Plane = boxHull.Planes[i]
		side := i & 1
		boxHull.ClipNodes[i].Children[side] = bsp.CONTENTS_EMPTY
		if i == 5 {
			boxHull.ClipNodes[i].Children[side^1] = bsp.CONTENTS_SOLID
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

func hullForBox(mins, maxs vec.Vec3) *bsp.Hull {
	boxHull.Planes[0].Dist = maxs[0]
	boxHull.Planes[1].Dist = mins[0]
	boxHull.Planes[2].Dist = maxs[1]
	boxHull.Planes[3].Dist = mins[1]
	boxHull.Planes[4].Dist = maxs[2]
	boxHull.Planes[5].Dist = mins[2]
	return &boxHull
}

func hullForEntity(ent *progs.EntVars, mins, maxs vec.Vec3, m model.Model) (*bsp.Hull, vec.Vec3) {
	if ent.Solid == SOLID_BSP {
		if ent.MoveType != progs.MoveTypePush {
			debug.PrintStack()
			log.Fatalf("SOLID_BSP without MOVETYPE_PUSH")
		}
		switch qm := m.(type) {
		default:
			debug.PrintStack()
			log.Fatalf("MOVETYPE_PUSH with a non bsp model")
		case *bsp.Model:
			s := maxs[0] - mins[0]
			h := func() *bsp.Hull {
				if s < 3 {
					return &qm.Hulls[0]
				} else if s <= 32 {
					return &qm.Hulls[1]
				}
				return &qm.Hulls[2]
			}()
			offset := vec.Add(vec.Sub(h.ClipMins, mins), vec.VFromA(ent.Origin))
			return h, offset
		}
	}
	hullmins := vec.Sub(vec.VFromA(ent.Mins), maxs)
	hullmaxs := vec.Sub(vec.VFromA(ent.Maxs), mins)
	origin := vec.VFromA(ent.Origin)
	return hullForBox(hullmins, hullmaxs), origin
}

func pointContents(p vec.Vec3, m *bsp.Model) int {
	return m.Hulls[0].PointContents(0, p)
}

func clipMoveToEntity(e int, start, mins, maxs, end vec.Vec3, s *Server) bsp.Trace {
	var t bsp.Trace
	t.Fraction = 1
	t.AllSolid = true
	t.EndPos = end
	ent := entvars.Get(e)
	m := s.models[int(ent.ModelIndex)]
	hull, offset := hullForEntity(ent, mins, maxs, m)
	startL := vec.Sub(start, offset)
	endL := vec.Sub(end, offset)
	hull.RecursiveCheck(hull.FirstClipNode, 0, 1, startL, endL, &t)

	if t.Fraction != 1 {
		t.EndPos[0] += offset[0]
		t.EndPos[1] += offset[1]
		t.EndPos[2] += offset[2]
	}
	if t.Fraction < 1 || t.StartSolid {
		t.EntNumber = e
		t.EntPointer = true
	}
	return t
}

func (c *moveClip) moveBounds(s, e vec.Vec3) {
	min, max := vec.MinMax(s, e)
	c.boxmins = vec.Sub(vec.Add(min, c.mins2), vec.Vec3{1, 1, 1})
	c.boxmaxs = vec.Add(vec.Add(max, c.maxs2), vec.Vec3{1, 1, 1})
}

func testEntityPosition(ent int, s *Server) bool {
	ev := entvars.Get(ent)
	t := svMove(ev.Origin, ev.Mins, ev.Maxs, ev.Origin, MOVE_NORMAL, ent, s)
	return t.StartSolid
}

// mins and maxs are relative
// if the entire move stays in a solid volume, trace.allsolid will be set
// if the starting point is in a solid, it will be allowed to move out to
// an open area
// nomonsters is used for line of sight or edge testing where monsters
// shouldn't be considered solid objects
// passedict is explicitly excluded from clipping checks (normally NULL)
func svMove(start, mins, maxs, end vec.Vec3, typ, ed int, s *Server) bsp.Trace {
	clip := moveClip{
		trace: clipMoveToEntity(0, start, mins, maxs, end, s),
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

	clipToLinks(gArea, &clip, s)

	return clip.trace
}

const (
	kStepSize = 18
)

// Returns false if any part of the bottom of the entity is off an edge that
// is not a staircase.
func checkBottom(ent int, s *Server) bool {
	ev := entvars.Get(ent)
	o := ev.Origin
	mins := vec.Add(o, ev.Mins)
	maxs := vec.Add(o, ev.Maxs)

	// if all of the points under the corners are solid world, don't bother
	// with the tougher checks
	d := []vec.Vec3{
		{mins[0], mins[1], mins[2] - 1},
		{mins[0], maxs[1], mins[2] - 1},
		{maxs[0], mins[1], mins[2] - 1},
		{maxs[0], maxs[1], mins[2] - 1},
	}
	for _, start := range d {
		if pointContents(start, s.worldModel) != bsp.CONTENTS_SOLID {
			return expensiveCheckBottom(ent, mins, maxs, s)
		}
	}
	return true
}

func expensiveCheckBottom(ent int, mins, maxs vec.Vec3, s *Server) bool {
	level := mins[2]
	below := mins[2] - 2*kStepSize
	start := vec.Vec3{
		(mins[0] + maxs[0]) * 0.5,
		(mins[1] + maxs[1]) * 0.5,
		level,
	}
	stop := vec.Vec3{start[0], start[1], below}
	t := svMove(start, vec.Vec3{}, vec.Vec3{}, stop, MOVE_NOMONSTERS, ent, s)

	if t.Fraction == 1.0 {
		return false
	}
	mid := t.EndPos[2]
	bottom := t.EndPos[2]

	d := []vec.Vec3{
		{mins[0], mins[1], 0},
		{mins[0], maxs[1], 0},
		{maxs[0], mins[1], 0},
		{maxs[0], maxs[1], 0},
	}

	for _, p := range d {
		start := vec.Vec3{p[0], p[1], level}
		stop := vec.Vec3{p[0], p[1], below}
		t := svMove(start, vec.Vec3{}, vec.Vec3{}, stop, MOVE_NOMONSTERS, ent, s)

		if t.Fraction != 1.0 && t.EndPos[2] > bottom {
			bottom = t.EndPos[2]
		}
		if t.Fraction == 1.0 || mid-t.EndPos[2] > kStepSize {
			return false
		}
	}
	return true
}
