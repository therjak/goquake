// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"goquake/bsp"
	"goquake/cvars"
	"goquake/math/vec"
	"goquake/progs"
	"log"
	"log/slog"
	"runtime/debug"

	"github.com/chewxy/math32"
)

/*
pushmove objects do not obey gravity, and do not interact with each other or
trigger fields, but block normal movement and push normal objects when they
move.

onground is set for toss objects when they come to a complete rest.  it is set
for steping or walking objects

doors, plats, etc are SOLID_BSP, and MOVETYPE_PUSH
bonus items are SOLID_TRIGGER touch, and MOVETYPE_TOSS
corpses are SOLID_NOT and MOVETYPE_TOSS
crates are SOLID_BBOX and MOVETYPE_TOSS
walking monsters are SOLID_SLIDEBOX and MOVETYPE_STEP
flying/floating monsters are SOLID_SLIDEBOX and MOVETYPE_FLY

solid_edge items only clip against bsp models.
*/
func (s *Server) pushMove(pusher int, movetime float32) error {
	pev := entvars.Get(pusher)
	if pev.Velocity == [3]float32{} {
		pev.LTime += movetime
		return nil
	}

	move := vec.Scale(movetime, pev.Velocity)
	mins := vec.Add(pev.AbsMin, move)
	maxs := vec.Add(pev.AbsMax, move)
	pushOrigin := vec.Vec3(pev.Origin)

	// move the pusher to it's final position
	pev.Origin = vec.Add(pev.Origin, move)
	pev.LTime += movetime
	if err := vm.LinkEdict(pusher, false, s); err != nil {
		return err
	}

	type moved struct {
		ent    int
		origin vec.Vec3
	}
	movedEnts := []moved{}

	// see if any solid entities are inside the final position
	for c := 1; c < s.numEdicts; c++ {
		if s.edicts[c].Free {
			continue
		}
		cev := entvars.Get(c)
		switch cev.MoveType {
		case progs.MoveTypePush, progs.MoveTypeNone, progs.MoveTypeNoClip:
			continue
		}

		// if the entity is standing on the pusher, it will definitely be moved
		if !(int(cev.Flags)&FL_ONGROUND != 0 && cev.GroundEntity == int32(pusher)) {
			if cev.AbsMin[0] >= maxs[0] ||
				cev.AbsMin[1] >= maxs[1] ||
				cev.AbsMin[2] >= maxs[2] ||
				cev.AbsMax[0] <= mins[0] ||
				cev.AbsMax[1] <= mins[1] ||
				cev.AbsMax[2] <= mins[2] {
				continue
			}
			// see if the ent's bbox is inside the pusher's final position
			if !testEntityPosition(c) {
				continue
			}
		}

		// remove the onground flag for non-players
		if cev.MoveType != progs.MoveTypeWalk {
			cev.Flags = float32(int(cev.Flags) &^ FL_ONGROUND)
		}

		entOrigin := cev.Origin
		movedEnts = append(movedEnts, moved{c, cev.Origin})

		// try moving the contacted entity
		pev.Solid = SOLID_NOT
		if _, err := s.pushEntity(c, move); err != nil {
			return err
		}
		pev.Solid = SOLID_BSP

		// if it is still inside the pusher, block
		if testEntityPosition(c) {
			// fail the move
			if cev.Mins[0] == cev.Maxs[0] {
				continue
			}
			switch cev.Solid {
			case SOLID_NOT, SOLID_TRIGGER:
				// corpse
				cev.Mins[0] = 0
				cev.Mins[1] = 0
				cev.Maxs = cev.Mins
				continue
			}
			cev.Origin = entOrigin
			if err := vm.LinkEdict(c, true, s); err != nil {
				return err
			}

			pev.Origin = pushOrigin
			if err := vm.LinkEdict(pusher, false, s); err != nil {
				return err
			}
			pev.LTime -= movetime

			// if the pusher has a "blocked" function, call it
			// otherwise, just stay in place until the obstacle is gone
			if pev.Blocked != 0 {
				progsdat.Globals.Self = int32(pusher)
				progsdat.Globals.Other = int32(c)
				if err := vm.ExecuteProgram(pev.Blocked, s); err != nil {
					return err
				}
			}

			// move back any entities we already moved
			for _, m := range movedEnts {
				entvars.Get(m.ent).Origin = m.origin
				if err := vm.LinkEdict(m.ent, false, s); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return nil
}

func (s *Server) addGravity(ent int) {
	val, err := entvars.FieldValue(ent, "gravity")
	if err != nil || val == 0 {
		val = 1.0
	}
	entvars.Get(ent).Velocity[2] -= val * cvars.ServerGravity.Value() * float32(host.FrameTime())
}

func (s *Server) pusher(ent int, time float32) error {
	ev := entvars.Get(ent)
	oldltime := float64(ev.LTime)
	thinktime := float64(ev.NextThink)

	movetime := func() float32 {
		if thinktime < oldltime+host.FrameTime() {
			t := thinktime - oldltime
			if t < 0 {
				return 0
			}
			return float32(t)
		}
		return float32(host.FrameTime())
	}()

	if movetime != 0 {
		// advances ent->v.ltime if not blocked
		if err := s.pushMove(ent, movetime); err != nil {
			return err
		}
	}

	if thinktime > oldltime && thinktime <= float64(ev.LTime) {
		ev.NextThink = 0
		progsdat.Globals.Time = time
		progsdat.Globals.Self = int32(ent)
		progsdat.Globals.Other = 0
		if err := vm.ExecuteProgram(ev.Think, s); err != nil {
			return err
		}
	}
	return nil
}

// Player has come to a dead stop, possibly due to the problem with limited
// float precision at some angle joins in the BSP hull.
//
// Try fixing by pushing one pixel in each direction.
//
// This is a hack, but in the interest of good gameplay...
func (s *Server) tryUnstick(ent int, oldvel vec.Vec3) (int, error) {
	ev := entvars.Get(ent)
	oldorg := ev.Origin

	for _, dir := range []vec.Vec3{
		// try pushing a little in an axial direction
		{2, 0, 0},
		{0, 2, 0},
		{-2, 0, 0},
		{0, -2, 0},
		{2, 2, 0},
		{-2, 2, 0},
		{2, -2, 0},
		{-2, -2, 0},
	} {
		if _, err := s.pushEntity(ent, dir); err != nil {
			return 0, err
		}
		// retry the original move
		ev.Velocity = oldvel
		ev.Velocity[2] = 0 // TODO: why?
		steptrace := bsp.Trace{}
		clip, err := s.flyMove(ent, 0.1, &steptrace)
		if err != nil {
			return 0, err
		}
		if math32.Abs(oldorg[1]-ev.Origin[1]) > 4 ||
			math32.Abs(oldorg[0]-ev.Origin[0]) > 4 {
			slog.Debug("unstuck!")
			return clip, nil
		}
		// go back to the original pos and try again
		ev.Origin = oldorg
	}
	ev.Velocity = [3]float32{0, 0, 0}
	// still not moving
	return 7, nil
}

func (s *Server) wallFriction(ent int, planeNormal vec.Vec3) {
	const deg = math32.Pi * 2 / 360

	ev := entvars.Get(ent)
	sp, cp := math32.Sincos(ev.VAngle[0] * deg) // PITCH
	sy, cy := math32.Sincos(ev.VAngle[1] * deg) // YAW
	forward := vec.Vec3{cp * cy, cp * sy, -sp}
	d := vec.Dot(planeNormal, forward)

	d += 0.5
	if d >= 0 {
		return
	}

	// cut the tangential velocity
	v := ev.Velocity
	i := vec.Dot(planeNormal, v)
	into := vec.Scale(i, planeNormal)
	side := vec.Sub(v, into)
	ev.Velocity[0] = side[0] * (1 + d)
	ev.Velocity[1] = side[1] * (1 + d)
}

// Only used by players
func (s *Server) walkMove(ent int) error {
	const STEPSIZE = 18
	ev := entvars.Get(ent)

	// do a regular slide move unless it looks like you ran into a step
	oldOnGround := int(ev.Flags)&FL_ONGROUND != 0
	ev.Flags = float32(int(ev.Flags) &^ FL_ONGROUND)

	oldOrigin := ev.Origin
	oldVelocity := ev.Velocity

	time := float32(host.FrameTime())
	steptrace := bsp.Trace{}
	clip, err := s.flyMove(ent, time, &steptrace)
	if err != nil {
		return err
	}

	if (clip & 2) == 0 {
		// move didn't block on a step
		return nil
	}

	if !oldOnGround && ev.WaterLevel == 0 {
		// don't stair up while jumping
		return nil
	}

	if ev.MoveType != progs.MoveTypeWalk {
		// gibbed by a trigger
		return nil
	}

	if cvars.ServerNoStep.Bool() {
		return nil
	}

	if int(ev.Flags)&FL_WATERJUMP != 0 {
		return nil
	}

	noStepOrigin := ev.Origin
	noStepVelocity := ev.Velocity

	// try moving up and forward to go up a step

	// back to start pos
	ev.Origin = oldOrigin
	upMove := vec.Vec3{0, 0, STEPSIZE}
	downMove := vec.Vec3{0, 0, -STEPSIZE + oldVelocity[2]*time}

	// move up
	if _, err := s.pushEntity(ent, upMove); err != nil { // FIXME: don't link?
		return err
	}

	// move forward
	ev.Velocity = oldVelocity
	ev.Velocity[2] = 0
	clip, err = s.flyMove(ent, time, &steptrace)
	if err != nil {
		return err
	}

	// check for stuckness, possibly due to the limited precision of floats
	// in the clipping hulls
	if clip != 0 {
		if math32.Abs(oldOrigin[1]-ev.Origin[1]) < 0.03125 &&
			math32.Abs(oldOrigin[0]-ev.Origin[0]) < 0.03125 {
			// stepping up didn't make any progress
			var err error
			clip, err = s.tryUnstick(ent, oldVelocity)
			if err != nil {
				return err
			}
		}
	}

	// extra friction based on view angle
	if clip&2 != 0 {
		planeNormal := steptrace.Plane.Normal
		s.wallFriction(ent, planeNormal)
	}

	// move down
	downTrace, err := s.pushEntity(ent, downMove) // FIXME: don't link?
	if err != nil {
		return err
	}

	if downTrace.Plane.Normal[2] > 0.7 {
		if ev.Solid == SOLID_BSP {
			ev.Flags = float32(int(ev.Flags) | FL_ONGROUND)
			ev.GroundEntity = int32(downTrace.EntNumber)
		}
		return nil
	}

	// if the push down didn't end up on good ground, use the move without
	// the step up.  This happens near wall / slope combinations, and can
	// cause the player to hop up higher on a slope too steep to climb
	ev.Origin = noStepOrigin
	ev.Velocity = noStepVelocity
	return nil
}

// Non moving objects can only think
func (s *Server) none(ent int) error {
	if _, err := s.runThink(ent); err != nil {
		return err
	}
	return nil
}

// A moving object that doesn't obey physics
func (s *Server) noClip(ent int) error {
	if ok, err := s.runThink(ent); err != nil {
		return err
	} else if !ok {
		return nil
	}
	time := float32(host.FrameTime())

	ev := entvars.Get(ent)
	av := vec.Vec3(ev.AVelocity)
	av = vec.Scale(time, av)
	angles := ev.Angles
	ev.Angles = vec.Add(angles, av)

	v := vec.Scale(time, ev.Velocity)
	origin := ev.Origin
	ev.Origin = vec.Add(origin, v)

	if err := vm.LinkEdict(ent, false, s); err != nil {
		return err
	}
	return nil
}

func (s *Server) checkWaterTransition(ent int) error {
	ev := entvars.Get(ent)

	cont := pointContents(ev.Origin)

	if ev.WaterType == 0 {
		// just spawned here
		ev.WaterType = float32(cont)
		ev.WaterLevel = 1
		return nil
	}

	if cont <= bsp.CONTENTS_WATER {
		if ev.WaterType == bsp.CONTENTS_EMPTY {
			// just crossed into water
			if err := s.StartSound(ent, 0, 255, "misc/h2ohit1.wav", 1); err != nil {
				return err
			}
		}
		ev.WaterType = float32(cont)
		ev.WaterLevel = 1
		return nil
	}

	if ev.WaterType != bsp.CONTENTS_EMPTY {
		// just crossed into water
		if err := s.StartSound(ent, 0, 255, "misc/h2ohit1.wav", 1); err != nil {
			return err
		}
	}
	ev.WaterType = bsp.CONTENTS_EMPTY
	ev.WaterLevel = float32(cont) // TODO: why?
	return nil
}

// Toss, bounce, and fly movement.  When onground, do nothing.
func (s *Server) toss(ent int) error {
	if ok, err := s.runThink(ent); err != nil {
		return err
	} else if !ok {
		return nil
	}

	ev := entvars.Get(ent)
	if int(ev.Flags)&FL_ONGROUND != 0 {
		return nil
	}
	CheckVelocity(ev)

	if ev.MoveType != progs.MoveTypeFly &&
		ev.MoveType != progs.MoveTypeFlyMissile {
		s.addGravity(ent)
	}

	time := float32(host.FrameTime())

	av := vec.Scale(time, ev.AVelocity)
	ev.Angles = vec.Add(ev.Angles, av)

	velocity := ev.Velocity
	move := vec.Scale(time, velocity)
	t, err := s.pushEntity(ent, move)
	if err != nil {
		return err
	}

	if t.Fraction == 1 {
		return nil
	}
	if s.edicts[ent].Free {
		return nil
	}

	backOff := func() float32 {
		if ev.MoveType == progs.MoveTypeBounce {
			return 1.5
		}
		return 1
	}()

	n := t.Plane.Normal
	_, velocity = clipVelocity(velocity, n, backOff)
	ev.Velocity = velocity

	// stop if on ground
	if t.Plane.Normal[2] > 0.7 {
		if ev.Velocity[2] < 60 || ev.MoveType != progs.MoveTypeBounce {
			ev.Flags = float32(int(ev.Flags) | FL_ONGROUND)
			ev.GroundEntity = int32(t.EntNumber)
			ev.Velocity = [3]float32{0, 0, 0}
			ev.AVelocity = [3]float32{0, 0, 0}
		}
	}

	if err := s.checkWaterTransition(ent); err != nil {
		return err
	}
	return nil
}

// Monsters freefall when they don't have a ground entity, otherwise
// all movement is done with discrete steps.

// This is also used for objects that have become still on the ground, but
// will fall if the floor is pulled out from under them.
func (s *Server) step(ent int) error {
	ev := entvars.Get(ent)

	// freefall if not onground
	if int(ev.Flags)&(FL_ONGROUND|FL_FLY|FL_SWIM) == 0 {
		hitSound := ev.Velocity[2] < cvars.ServerGravity.Value()*-0.1

		time := float32(host.FrameTime())
		s.addGravity(ent)
		CheckVelocity(ev)
		if _, err := s.flyMove(ent, time, nil); err != nil {
			return err
		}
		if err := vm.LinkEdict(ent, true, s); err != nil {
			return err
		}

		if int(ev.Flags)&FL_ONGROUND != 0 {
			// just hit ground
			if hitSound {
				if err := s.StartSound(ent, 0, 255, "demon/dland2.wav", 1); err != nil {
					return err
				}
			}
		}
	}

	if ok, err := s.runThink(ent); err != nil {
		return err
	} else if !ok {
		return nil
	}

	if err := s.checkWaterTransition(ent); err != nil {
		return err
	}
	return nil
}

// This is a big hack to try and fix the rare case of getting stuck in the world
// clipping hull.
func (s *Server) checkStuck(ent int) error {
	ev := entvars.Get(ent)
	if !testEntityPosition(ent) {
		ev.OldOrigin = ev.Origin
		return nil
	}

	org := ev.Origin
	ev.Origin = ev.OldOrigin
	if !testEntityPosition(ent) {
		slog.Debug("Unstuck.") // debug
		if err := vm.LinkEdict(ent, true, s); err != nil {
			return err
		}
		return nil
	}

	for z := float32(0); z < 18; z++ {
		for i := float32(-1); i <= 1; i++ {
			for j := float32(-1); j <= 1; j++ {
				ev.Origin[0] = org[0] + i
				ev.Origin[1] = org[1] + j
				ev.Origin[2] = org[2] + z
				if !testEntityPosition(ent) {
					slog.Debug("Unstuck.")
					if err := vm.LinkEdict(ent, true, s); err != nil {
						return err
					}
					return nil
				}
			}
		}
	}

	ev.Origin = org
	slog.Warn("player is stuck.")
	return nil
}

// The basic solid body movement clip that slides along multiple planes
// Returns the clipflags if the velocity was modified (hit something solid)
// 1 = floor
// 2 = wall / step
// 4 = dead stop
// If steptrace is not NULL, the trace of any vertical wall hit will be stored
func (s *Server) flyMove(ent int, time float32, steptrace *bsp.Trace) (int, error) {
	const MAX_CLIP_PLANES = 5
	planes := [MAX_CLIP_PLANES]vec.Vec3{}

	numbumps := 4

	blocked := 0
	ev := entvars.Get(ent)
	original_velocity := ev.Velocity
	primal_velocity := ev.Velocity
	numplanes := 0

	time_left := time

	for bumpcount := 0; bumpcount < numbumps; bumpcount++ {
		if ev.Velocity == [3]float32{0, 0, 0} {
			break
		}

		origin := ev.Origin
		velocity := ev.Velocity
		end := vec.Vec3{
			origin[0] + time_left*velocity[0],
			origin[1] + time_left*velocity[1],
			origin[2] + time_left*velocity[2],
		}

		t := svMove(origin, ev.Mins, ev.Maxs, end, MOVE_NORMAL, ent)

		if t.AllSolid {
			// entity is trapped in another solid
			ev.Velocity = [3]float32{0, 0, 0}
			return 3, nil
		}

		if t.Fraction > 0 {
			// actually covered some distance
			ev.Origin = t.EndPos
			original_velocity = ev.Velocity
			numplanes = 0
		}
		if t.Fraction == 1 {
			// moved the entire distance
			break
		}
		if !t.EntPointer {
			debug.PrintStack()
			log.Fatalf("SV_FlyMove: !trace.ent")
		}
		if t.Plane.Normal[2] > 0.7 {
			blocked |= 1 // floor
			if entvars.Get(t.EntNumber).Solid == SOLID_BSP {
				ev.Flags = float32(int(ev.Flags) | FL_ONGROUND)
				ev.GroundEntity = int32(t.EntNumber)
			}
		}
		if t.Plane.Normal[2] == 0 {
			blocked |= 2 // step
			if steptrace != nil {
				*steptrace = t // save for player extrafriction
			}
		}
		if err := s.impact(ent, t.EntNumber); err != nil {
			return 0, err
		}
		if s.edicts[ent].Free {
			// removed by the impact function
			break
		}
		time_left -= time_left * t.Fraction

		// cliped to another plane
		if numplanes >= MAX_CLIP_PLANES {
			// this shouldn't really happen
			ev.Velocity = [3]float32{0, 0, 0}
			return 3, nil
		}

		planes[numplanes] = t.Plane.Normal
		numplanes++

		// modify original_velocity so it parallels all of the clip planes
		new_velocity := vec.Vec3{}
		i := 0
		for i = 0; i < numplanes; i++ {
			j := 0
			_, new_velocity = clipVelocity(original_velocity, planes[i], 1)
			for j = 0; j < numplanes; j++ {
				if j != i {
					if vec.Dot(new_velocity, planes[j]) < 0 {
						break // not ok
					}
				}
			}
			if j == numplanes {
				break
			}
		}

		if i != numplanes { // go along this plane
			ev.Velocity = new_velocity
		} else { // go along the crease
			if numplanes != 2 {
				//	fmt.Printf("clip velocity, numplanes == %i\n",numplanes)
				ev.Velocity = [3]float32{0, 0, 0}
				return 7, nil
			}
			dir := vec.Cross(planes[0], planes[1])
			d := vec.Dot(dir, ev.Velocity)
			ev.Velocity = vec.Scale(d, dir)
		}

		// if original velocity is against the original velocity, stop dead
		// to avoid tiny occilations in sloping corners
		if vec.Dot(ev.Velocity, primal_velocity) <= 0 {
			ev.Velocity = [3]float32{0, 0, 0}
			return blocked, nil
		}
	}
	return blocked, nil
}

func (s *Server) checkWater(ent int) bool {
	ev := entvars.Get(ent)
	point := vec.Vec3{
		ev.Origin[0],
		ev.Origin[1],
		ev.Origin[2] + ev.Mins[2] + 1,
	}

	ev.WaterLevel = 0
	ev.WaterType = bsp.CONTENTS_EMPTY

	cont := pointContents(point)
	if cont <= bsp.CONTENTS_WATER {
		ev.WaterType = float32(cont)
		ev.WaterLevel = 1
		point[2] = ev.Origin[2] + (ev.Mins[2]+ev.Maxs[2])*0.5
		cont = pointContents(point)
		if cont <= bsp.CONTENTS_WATER {
			ev.WaterLevel = 2
			point[2] = ev.Origin[2] + ev.ViewOfs[2]
			cont = pointContents(point)
			if cont <= bsp.CONTENTS_WATER {
				ev.WaterLevel = 3
			}
		}
	}

	return ev.WaterLevel > 1
}

// Slide off of the impacting object
// returns the blocked flags (1 = floor, 2 = step / wall) and clipped velocity
func clipVelocity(in, normal vec.Vec3, overbounce float32) (int, vec.Vec3) {
	blocked := func() int {
		switch {
		case normal[2] > 0:
			return 1 // floor
		case normal[2] == 0:
			return 2 // step
		default:
			return 0
		}
	}()

	backoff := vec.Dot(in, normal) * overbounce

	e := func(x float32) float32 {
		const EPSILON = 0.1
		if x > -EPSILON && x < EPSILON {
			return 0
		}
		return x
	}

	out := vec.Vec3{
		e(in[0] - normal[0]*backoff),
		e(in[1] - normal[1]*backoff),
		e(in[2] - normal[2]*backoff),
	}

	return blocked, out
}

// Player character actions
func (s *Server) playerActions(ent, num int, time float32) error {
	if !sv_clients[num-1].active {
		// unconnected slot
		return nil
	}

	progsdat.Globals.Time = time
	progsdat.Globals.Self = int32(ent)
	if err := vm.ExecuteProgram(progsdat.Globals.PlayerPreThink, s); err != nil {
		return err
	}

	ev := entvars.Get(ent)
	CheckVelocity(ev)

	switch int(ev.MoveType) {
	case progs.MoveTypeNone:
		if ok, err := s.runThink(ent); err != nil {
			return err
		} else if !ok {
			return nil
		}

	case progs.MoveTypeWalk:
		if ok, err := s.runThink(ent); err != nil {
			return err
		} else if !ok {
			return nil
		}
		if !s.checkWater(ent) && int(ev.Flags)&FL_WATERJUMP == 0 {
			s.addGravity(ent)
		}
		if err := s.checkStuck(ent); err != nil {
			return err
		}
		if err := s.walkMove(ent); err != nil {
			return err
		}

	case progs.MoveTypeToss, progs.MoveTypeBounce, progs.MoveTypeGib:
		if err := s.toss(ent); err != nil {
			return err
		}

	case progs.MoveTypeFly:
		if ok, err := s.runThink(ent); err != nil {
			return err
		} else if !ok {
			return nil
		}
		time := float32(host.FrameTime())
		if _, err := s.flyMove(ent, time, nil); err != nil {
			return err
		}

	case progs.MoveTypeNoClip:
		if ok, err := s.runThink(ent); err != nil {
			return err
		} else if !ok {
			return nil
		}
		time := float32(host.FrameTime())
		v := vec.Scale(time, ev.Velocity)
		ev.Origin = vec.Add(ev.Origin, v)

	default:
		debug.PrintStack()
		log.Fatalf("SV_Physics_client: bad movetype %v", ev.MoveType)
	}

	if err := vm.LinkEdict(ent, true, s); err != nil {
		return err
	}

	progsdat.Globals.Time = time
	progsdat.Globals.Self = int32(ent)
	return vm.ExecuteProgram(progsdat.Globals.PlayerPostThink, s)
}

func (s *Server) runPhysics() error {
	// let the progs know that a new frame has started
	progsdat.Globals.Time = s.time
	progsdat.Globals.Self = 0
	progsdat.Globals.Other = 0
	if err := vm.ExecuteProgram(progsdat.Globals.PlayerPostThink, s); err != nil {
		return err
	}

	freezeNonClients := cvars.ServerFreezeNonClients.Bool()
	entityCap := func() int {
		if freezeNonClients {
			// Only run physics on clients and the world
			return svs.maxClients + 1
		}
		return s.numEdicts
	}()

	for i := 0; i < entityCap; i++ {
		if s.edicts[i].Free {
			continue
		}
		if progsdat.Globals.ForceRetouch != 0 {
			// force retouch even for stationary
			if err := vm.LinkEdict(i, true, s); err != nil {
				return err
			}
		}
		if i > 0 && i <= svs.maxClients {
			if err := s.playerActions(i, i, s.time); err != nil {
				return err
			}
		} else {
			mt := entvars.Get(i).MoveType
			switch mt {
			case progs.MoveTypePush:
				if err := s.pusher(i, s.time); err != nil {
					return err
				}
			case progs.MoveTypeNone:
				if err := s.none(i); err != nil {
					return err
				}
			case progs.MoveTypeNoClip:
				if err := s.noClip(i); err != nil {
					return err
				}
			case progs.MoveTypeStep:
				if err := s.step(i); err != nil {
					return err
				}
			case progs.MoveTypeToss,
				progs.MoveTypeBounce,
				progs.MoveTypeGib,
				progs.MoveTypeFly,
				progs.MoveTypeFlyMissile:
				if err := s.toss(i); err != nil {
					return err
				}
			default:
				debug.PrintStack()
				log.Fatalf("SV_Physics: bad movetype %v", mt)
			}
		}
	}

	if progsdat.Globals.ForceRetouch != 0 {
		progsdat.Globals.ForceRetouch--
	}

	if !freezeNonClients {
		s.time += float32(host.FrameTime())
	}
	return nil
}
