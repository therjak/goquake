package quakelib

//#include "trace.h"
//void SV_PushMove(int pusher, float movetime);
import "C"

import (
	"quake/conlog"
	"quake/cvars"
	"quake/math/vec"
	"quake/progs"

	"github.com/chewxy/math32"
)

func pushMove(pusher int, movetime float32) {
	C.SV_PushMove(C.int(pusher), C.float(movetime))
}

type qphysics struct {
}

var (
	physics qphysics
)

func (q *qphysics) addGravity(ent int) {
	val, err := EntVarsFieldValue(ent, "gravity")
	if err != nil || val == 0 {
		val = 1.0
	}
	EntVars(ent).Velocity[2] -= val * cvars.ServerGravity.Value() * float32(host.frameTime)
}

//export SV_Physics_Pusher
func SV_Physics_Pusher(ent int) {
	physics.pusher(ent)
}

func (q *qphysics) pusher(ent int) {
	ev := EntVars(ent)
	oldltime := float64(ev.LTime)
	thinktime := float64(ev.NextThink)

	movetime := func() float32 {
		if thinktime < oldltime+host.frameTime {
			t := thinktime - oldltime
			if t < 0 {
				return 0
			}
			return float32(t)
		}
		return float32(host.frameTime)
	}()

	if movetime != 0 {
		// advances ent->v.ltime if not blocked
		pushMove(ent, movetime)
	}

	if thinktime > oldltime && thinktime <= float64(ev.LTime) {
		ev.NextThink = 0
		progsdat.Globals.Time = sv.time
		progsdat.Globals.Self = int32(ent)
		progsdat.Globals.Other = 0
		PRExecuteProgram(ev.Think)
	}
}

// Player has come to a dead stop, possibly due to the problem with limited
// float precision at some angle joins in the BSP hull.
//
// Try fixing by pushing one pixel in each direction.
//
// This is a hack, but in the interest of good gameplay...
func (q *qphysics) tryUnstick(ent int, oldvel vec.Vec3) int {
	ev := EntVars(ent)
	oldorg := ev.Origin

	for _, dir := range []vec.Vec3{
		// try pushing a little in an axial direction
		vec.Vec3{2, 0, 0},
		vec.Vec3{0, 2, 0},
		vec.Vec3{-2, 0, 0},
		vec.Vec3{0, -2, 0},
		vec.Vec3{2, 2, 0},
		vec.Vec3{-2, 2, 0},
		vec.Vec3{2, -2, 0},
		vec.Vec3{-2, -2, 0},
	} {
		pushEntity(ent, dir)
		// retry the original move
		ev.Velocity = oldvel
		ev.Velocity[2] = 0 // TODO: why?
		steptrace := C.trace_t{}
		clip := q.flyMove(ent, 0.1, &steptrace)
		if math32.Abs(oldorg[1]-ev.Origin[1]) > 4 ||
			math32.Abs(oldorg[0]-ev.Origin[0]) > 4 {
			conlog.DPrintf("unstuck!\n")
			return clip
		}
		// go back to the original pos and try again
		ev.Origin = oldorg
	}
	ev.Velocity = [3]float32{0, 0, 0}
	// still not moving
	return 7
}

func (q *qphysics) wallFriction(ent int, planeNormal vec.Vec3) {
	const deg = math32.Pi * 2 / 360

	ev := EntVars(ent)
	sp, cp := math32.Sincos(ev.VAngle[0] * deg) // PITCH
	sy, cy := math32.Sincos(ev.VAngle[1] * deg) // YAW
	forward := vec.Vec3{cp * cy, cp * sy, -sp}
	d := vec.Dot(planeNormal, forward)

	d += 0.5
	if d >= 0 {
		return
	}

	// cut the tangential velocity
	v := vec.VFromA(ev.Velocity)
	i := vec.Dot(planeNormal, v)
	into := planeNormal.Scale(i)
	side := vec.Sub(v, into)
	ev.Velocity[0] = side[0] * (1 + d)
	ev.Velocity[1] = side[1] * (1 + d)
}

//export SV_WalkMove
func SV_WalkMove(ent int) {
	physics.walkMove(ent)
}

// Only used by players
func (q *qphysics) walkMove(ent int) {
	const STEPSIZE = 18
	ev := EntVars(ent)

	// do a regular slide move unless it looks like you ran into a step
	oldOnGround := int(ev.Flags)&FL_ONGROUND != 0
	ev.Flags = float32(int(ev.Flags) &^ FL_ONGROUND)

	oldOrigin := ev.Origin
	oldVelocity := vec.VFromA(ev.Velocity)

	time := float32(host.frameTime)
	steptrace := C.trace_t{}
	clip := q.flyMove(ent, time, &steptrace)

	if (clip & 2) == 0 {
		// move didn't block on a step
		return
	}

	if !oldOnGround && ev.WaterLevel == 0 {
		// don't stair up while jumping
		return
	}

	if ev.MoveType != progs.MoveTypeWalk {
		// gibbed by a trigger
		return
	}

	if cvars.ServerNoStep.Bool() {
		return
	}

	if sv_player != ent {
		conlog.Printf("walkMove: sv_player != ent")
		// the following was not done on EntVars(ent) but EntVars(sv_player)
	}
	if int(ev.Flags)&FL_WATERJUMP != 0 {
		return
	}

	noStepOrigin := ev.Origin
	noStepVelocity := ev.Velocity

	// try moving up and forward to go up a step

	// back to start pos
	ev.Origin = oldOrigin
	upMove := vec.Vec3{0, 0, STEPSIZE}
	downMove := vec.Vec3{0, 0, -STEPSIZE + oldVelocity[2]*time}

	// move up
	pushEntity(ent, upMove) // FIXME: don't link?

	// move forward
	ev.Velocity = oldVelocity
	ev.Velocity[2] = 0
	clip = q.flyMove(ent, time, &steptrace)

	// check for stuckness, possibly due to the limited precision of floats
	// in the clipping hulls
	if clip != 0 {
		if math32.Abs(oldOrigin[1]-ev.Origin[1]) < 0.03125 &&
			math32.Abs(oldOrigin[0]-ev.Origin[0]) < 0.03125 {
			// stepping up didn't make any progress
			clip = q.tryUnstick(ent, oldVelocity)
		}
	}

	// extra friction based on view angle
	if clip&2 != 0 {
		planeNormal := vec.Vec3{
			float32(steptrace.plane.normal[0]),
			float32(steptrace.plane.normal[1]),
			float32(steptrace.plane.normal[2]),
		}
		q.wallFriction(ent, planeNormal)
	}

	// move down
	downTrace := pushEntity(ent, downMove) // FIXME: don't link?

	if downTrace.plane.normal[2] > 0.7 {
		if ev.Solid == SOLID_BSP {
			ev.Flags = float32(int(ev.Flags) | FL_ONGROUND)
			ev.GroundEntity = int32(downTrace.entn)
		}
		return
	}

	// if the push down didn't end up on good ground, use the move without
	// the step up.  This happens near wall / slope combinations, and can
	// cause the player to hop up higher on a slope too steep to climb
	ev.Origin = noStepOrigin
	ev.Velocity = noStepVelocity
}

//export SV_Physics_None
func SV_Physics_None(ent int) {
	physics.none(ent)
}

// Non moving objects can only think
func (q *qphysics) none(ent int) {
	runThink(ent)
}

//export SV_Physics_Noclip
func SV_Physics_Noclip(ent int) {
	physics.noClip(ent)
}

//A moving object that doesn't obey physics
func (q *qphysics) noClip(ent int) {
	if !runThink(ent) {
		return
	}
	time := float32(host.frameTime)

	ev := EntVars(ent)
	av := vec.VFromA(ev.AVelocity)
	av = av.Scale(time)
	angles := vec.VFromA(ev.Angles)
	ev.Angles = vec.Add(angles, av)

	v := vec.VFromA(ev.Velocity)
	v = v.Scale(time)
	origin := vec.VFromA(ev.Origin)
	ev.Origin = vec.Add(origin, v)

	LinkEdict(ent, false)
}

//export SV_CheckWaterTransition
func SV_CheckWaterTransition(ent int) {
	physics.checkWaterTransition(ent)
}

func (q *qphysics) checkWaterTransition(ent int) {
	ev := EntVars(ent)

	origin := vec.VFromA(ev.Origin)
	cont := pointContents(origin)

	if ev.WaterType == 0 {
		// just spawned here
		ev.WaterType = float32(cont)
		ev.WaterLevel = 1
		return
	}

	if cont <= CONTENTS_WATER {
		if ev.WaterType == CONTENTS_EMPTY {
			// just crossed into water
			sv.StartSound(ent, 0, 255, "misc/h2ohit1.wav", 1)
		}
		ev.WaterType = float32(cont)
		ev.WaterLevel = 1
		return
	}

	if ev.WaterType != CONTENTS_EMPTY {
		// just crossed into water
		sv.StartSound(ent, 0, 255, "misc/h2ohit1.wav", 1)
	}
	ev.WaterType = CONTENTS_EMPTY
	ev.WaterLevel = float32(cont) // TODO: why?
}

//export SV_Physics_Toss
func SV_Physics_Toss(ent int) {
	physics.toss(ent)
}

// Toss, bounce, and fly movement.  When onground, do nothing.
func (q *qphysics) toss(ent int) {
	if !runThink(ent) {
		return
	}

	ev := EntVars(ent)
	if int(ev.Flags)&FL_ONGROUND != 0 {
		return
	}
	CheckVelocity(ev)

	if ev.MoveType != progs.MoveTypeFly &&
		ev.MoveType != progs.MoveTypeFlyMissile {
		q.addGravity(ent)
	}

	time := float32(host.frameTime)

	av := vec.VFromA(ev.AVelocity)
	av = av.Scale(time)
	angles := vec.VFromA(ev.Angles)
	ev.Angles = vec.Add(angles, av)

	velocity := vec.VFromA(ev.Velocity)
	move := velocity.Scale(time)
	trace := pushEntity(ent, move)

	if trace.fraction == 1 {
		return
	}
	if edictNum(ent).free != 0 {
		return
	}

	backOff := func() float32 {
		if ev.MoveType == progs.MoveTypeBounce {
			return 1.5
		}
		return 1
	}()

	n := vec.Vec3{
		float32(trace.plane.normal[0]),
		float32(trace.plane.normal[1]),
		float32(trace.plane.normal[2]),
	}
	_, velocity = q.clipVelocity(velocity, n, backOff)
	ev.Velocity = velocity

	// stop if on ground
	if trace.plane.normal[2] > 0.7 {
		if ev.Velocity[2] < 60 || ev.MoveType != progs.MoveTypeBounce {
			ev.Flags = float32(int(ev.Flags) | FL_ONGROUND)
			ev.GroundEntity = int32(trace.entn)
			ev.Velocity = [3]float32{0, 0, 0}
			ev.AVelocity = [3]float32{0, 0, 0}
		}
	}

	q.checkWaterTransition(ent)
}

//export SV_Physics_Step
func SV_Physics_Step(ent int) {
	physics.step(ent)
}

// Monsters freefall when they don't have a ground entity, otherwise
// all movement is done with discrete steps.

// This is also used for objects that have become still on the ground, but
// will fall if the floor is pulled out from under them.
func (q *qphysics) step(ent int) {
	ev := EntVars(ent)

	// freefall if not onground
	if int(ev.Flags)&(FL_ONGROUND|FL_FLY|FL_SWIM) == 0 {
		hitSound := ev.Velocity[2] < cvars.ServerGravity.Value()*-0.1

		time := float32(host.frameTime)
		q.addGravity(ent)
		CheckVelocity(ev)
		q.flyMove(ent, time, nil)
		LinkEdict(ent, true)

		if int(ev.Flags)&FL_ONGROUND != 0 {
			// just hit ground
			if hitSound {
				sv.StartSound(ent, 0, 255, "demon/dland2.wav", 1)
			}
		}
	}

	if !runThink(ent) {
		return
	}

	q.checkWaterTransition(ent)
}

// This is a big hack to try and fix the rare case of getting stuck in the world
// clipping hull.
func (q *qphysics) checkStuck(ent int) {
	ev := EntVars(ent)
	if !testEntityPosition(ent) {
		ev.OldOrigin = ev.Origin
		return
	}

	org := ev.Origin
	ev.Origin = ev.OldOrigin
	if !testEntityPosition(ent) {
		conlog.Printf("Unstuck.\n") // debug
		LinkEdict(ent, true)
		return
	}

	for z := float32(0); z < 18; z++ {
		for i := float32(-1); i <= 1; i++ {
			for j := float32(-1); j <= 1; j++ {
				ev.Origin[0] = org[0] + i
				ev.Origin[1] = org[1] + j
				ev.Origin[2] = org[2] + z
				if !testEntityPosition(ent) {
					conlog.Printf("Unstuck.\n")
					LinkEdict(ent, true)
					return
				}
			}
		}
	}

	ev.Origin = org
	conlog.Printf("player is stuck.\n")
}

//The basic solid body movement clip that slides along multiple planes
//Returns the clipflags if the velocity was modified (hit something solid)
//1 = floor
//2 = wall / step
//4 = dead stop
//If steptrace is not NULL, the trace of any vertical wall hit will be stored
func (q *qphysics) flyMove(ent int, time float32, steptrace *C.trace_t) int {
	const MAX_CLIP_PLANES = 5
	planes := [MAX_CLIP_PLANES]vec.Vec3{}

	numbumps := 4

	blocked := 0
	ev := EntVars(ent)
	original_velocity := vec.VFromA(ev.Velocity)
	primal_velocity := vec.VFromA(ev.Velocity)
	numplanes := 0

	time_left := time

	for bumpcount := 0; bumpcount < numbumps; bumpcount++ {
		if ev.Velocity == [3]float32{0, 0, 0} {
			break
		}

		origin := vec.VFromA(ev.Origin)
		mins := vec.VFromA(ev.Mins)
		maxs := vec.VFromA(ev.Maxs)
		velocity := vec.VFromA(ev.Velocity)
		end := vec.Vec3{
			origin[0] + time_left*velocity[0],
			origin[1] + time_left*velocity[1],
			origin[2] + time_left*velocity[2],
		}

		trace := svMove(origin, mins, maxs, end, MOVE_NORMAL, ent)

		if trace.allsolid != 0 {
			// entity is trapped in another solid
			ev.Velocity = [3]float32{0, 0, 0}
			return 3
		}

		if trace.fraction > 0 {
			// actually covered some distance
			ev.Origin[0] = float32(trace.endpos[0])
			ev.Origin[1] = float32(trace.endpos[1])
			ev.Origin[2] = float32(trace.endpos[2])
			original_velocity = vec.VFromA(ev.Velocity)
			numplanes = 0
		}
		if trace.fraction == 1 {
			// moved the entire distance
			break
		}
		if trace.entp == 0 {
			Error("SV_FlyMove: !trace.ent")
		}
		if trace.plane.normal[2] > 0.7 {
			blocked |= 1 // floor
			if EntVars(int(trace.entn)).Solid == SOLID_BSP {
				ev.Flags = float32(int(ev.Flags) | FL_ONGROUND)
				ev.GroundEntity = int32(trace.entn)
			}
		}
		if trace.plane.normal[2] == 0 {
			blocked |= 2 // step
			if steptrace != nil {
				*steptrace = trace // save for player extrafriction
			}
		}
		sv.Impact(ent, int(trace.entn))
		if edictNum(ent).free != 0 {
			// removed by the impact function
			break
		}
		time_left -= time_left * float32(trace.fraction)

		// cliped to another plane
		if numplanes >= MAX_CLIP_PLANES {
			// this shouldn't really happen
			ev.Velocity = [3]float32{0, 0, 0}
			return 3
		}

		planes[numplanes] = vec.Vec3{
			float32(trace.plane.normal[0]),
			float32(trace.plane.normal[1]),
			float32(trace.plane.normal[2]),
		}
		numplanes++

		// modify original_velocity so it parallels all of the clip planes
		new_velocity := vec.Vec3{}
		i := 0
		for i = 0; i < numplanes; i++ {
			j := 0
			_, new_velocity = q.clipVelocity(original_velocity, planes[i], 1)
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
				//	conlog.Printf ("clip velocity, numplanes == %i\n",numplanes)
				ev.Velocity = [3]float32{0, 0, 0}
				return 7
			}
			dir := vec.Cross(planes[0], planes[1])
			d := vec.Dot(dir, vec.VFromA(ev.Velocity))
			ev.Velocity = dir.Scale(d)
		}

		// if original velocity is against the original velocity, stop dead
		// to avoid tiny occilations in sloping corners
		if vec.Dot(vec.VFromA(ev.Velocity), primal_velocity) <= 0 {
			ev.Velocity = [3]float32{0, 0, 0}
			return blocked
		}
	}
	return blocked
}

func (q *qphysics) checkWater(ent int) bool {
	ev := EntVars(ent)
	point := vec.Vec3{
		ev.Origin[0],
		ev.Origin[1],
		ev.Origin[2] + ev.Mins[2] + 1,
	}

	ev.WaterLevel = 0
	ev.WaterType = CONTENTS_EMPTY

	cont := pointContents(point)
	if cont <= CONTENTS_WATER {
		ev.WaterType = float32(cont)
		ev.WaterLevel = 1
		point[2] = ev.Origin[2] + (ev.Mins[2]+ev.Maxs[2])*0.5
		cont = pointContents(point)
		if cont <= CONTENTS_WATER {
			ev.WaterLevel = 2
			point[2] = ev.Origin[2] + ev.ViewOfs[2]
			cont = pointContents(point)
			if cont <= CONTENTS_WATER {
				ev.WaterLevel = 3
			}
		}
	}

	return ev.WaterLevel > 1
}

// Slide off of the impacting object
// returns the blocked flags (1 = floor, 2 = step / wall) and clipped velocity
func (q *qphysics) clipVelocity(in, normal vec.Vec3, overbounce float32) (int, vec.Vec3) {
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

//export SV_Physics_Client
func SV_Physics_Client(ent, num int) {
	physics.playerActions(ent, num)
}

// Player character actions
func (q *qphysics) playerActions(ent, num int) {
	if !sv_clients[num-1].active {
		// unconnected slot
		return
	}

	progsdat.Globals.Time = sv.time
	progsdat.Globals.Self = int32(ent)
	PRExecuteProgram(progsdat.Globals.PlayerPreThink)

	ev := EntVars(ent)
	CheckVelocity(ev)

	switch int(ev.MoveType) {
	case progs.MoveTypeNone:
		if !runThink(ent) {
			return
		}

	case progs.MoveTypeWalk:
		if !runThink(ent) {
			return
		}
		if !q.checkWater(ent) && int(ev.Flags)&FL_WATERJUMP == 0 {
			q.addGravity(ent)
		}
		q.checkStuck(ent)
		q.walkMove(ent)

	case progs.MoveTypeToss, progs.MoveTypeBounce:
		q.toss(ent)

	case progs.MoveTypeFly:
		if !runThink(ent) {
			return
		}
		time := float32(host.frameTime)
		q.flyMove(ent, time, nil)

	case progs.MoveTypeNoClip:
		if !runThink(ent) {
			return
		}
		time := float32(host.frameTime)
		v := vec.VFromA(ev.Velocity)
		v = v.Scale(time)
		origin := vec.VFromA(ev.Origin)
		ev.Origin = vec.Add(origin, v)

	default:
		Error("SV_Physics_client: bad movetype %v", ev.MoveType)
	}

	LinkEdict(ent, true)

	progsdat.Globals.Time = sv.time
	progsdat.Globals.Self = int32(ent)
	PRExecuteProgram(progsdat.Globals.PlayerPostThink)
}

//export SV_Physics
func SV_Physics() {
	RunPhysics()
}

func RunPhysics() {
	// let the progs know that a new frame has started
	progsdat.Globals.Time = sv.time
	progsdat.Globals.Self = 0
	progsdat.Globals.Other = 0
	PRExecuteProgram(progsdat.Globals.PlayerPostThink)

	freezeNonClients := cvars.ServerFreezeNonClients.Bool()
	entityCap := func() int {
		if freezeNonClients {
			// Only run physics on clients and the world
			return svs.maxClients + 1
		}
		return sv.numEdicts
	}()

	for i := 0; i < entityCap; i++ {
		if edictNum(i).free != 0 {
			continue
		}
		if progsdat.Globals.ForceRetouch != 0 {
			// force retouch even for stationary
			LinkEdict(i, true)
		}
		q := qphysics{}
		if i > 0 && i <= svs.maxClients {
			q.playerActions(i, i)
		} else {
			mt := EntVars(i).MoveType
			switch mt {
			case progs.MoveTypePush:
				q.pusher(i)
			case progs.MoveTypeNone:
				q.none(i)
			case progs.MoveTypeNoClip:
				q.noClip(i)
			case progs.MoveTypeStep:
				q.step(i)
			case progs.MoveTypeToss,
				progs.MoveTypeBounce,
				progs.MoveTypeFly,
				progs.MoveTypeFlyMissile:
				q.toss(i)
			default:
				Error("SV_Physics: bad movetype %v", mt)
			}
		}
	}

	if progsdat.Globals.ForceRetouch != 0 {
		progsdat.Globals.ForceRetouch--
	}

	if !freezeNonClients {
		sv.time += float32(host.frameTime)
	}
}
