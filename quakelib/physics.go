package quakelib

//#include "trace.h"
//void SV_PushMove(int pusher, float movetime);
//void SV_AddGravity(int ent);
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
	C.SV_AddGravity(C.int(ent))
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

// Try fixing by pushing one pixel in each direction.

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
		ev.Velocity = oldvel.Array()
		ev.Velocity[2] = 0 // TODO: why?
		steptrace := C.trace_t{}
		clip := SV_FlyMove(ent, 0.1, &steptrace)
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
	ev.Velocity[0] = side.X * (1 + d)
	ev.Velocity[1] = side.Y * (1 + d)
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
	clip := SV_FlyMove(ent, time, &steptrace)

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
	downMove := vec.Vec3{0, 0, -STEPSIZE + oldVelocity.Z*time}

	// move up
	pushEntity(ent, upMove) // FIXME: don't link?

	// move forward
	ev.Velocity = oldVelocity.Array()
	ev.Velocity[2] = 0
	clip = SV_FlyMove(ent, time, &steptrace)

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
	// regular thinking
	runThink(ent)
}

//export SV_Physics_Noclip
func SV_Physics_Noclip(ent int) {
	physics.noClip(ent)
}

//A moving object that doesn't obey physics
func (q *qphysics) noClip(ent int) {
	// regular thinking
	if !runThink(ent) {
		return
	}
	time := float32(host.frameTime)

	ev := EntVars(ent)
	av := vec.VFromA(ev.AVelocity)
	av = av.Scale(time)
	angles := vec.VFromA(ev.Angles)
	na := vec.Add(angles, av)
	ev.Angles = na.Array()

	v := vec.VFromA(ev.Velocity)
	v = v.Scale(time)
	origin := vec.VFromA(ev.Origin)
	no := vec.Add(origin, v)
	ev.Origin = no.Array()

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
	// regular thinking
	if !runThink(ent) {
		return
	}

	ev := EntVars(ent)
	// if onground, return without moving
	if int(ev.Flags)&FL_ONGROUND != 0 {
		return
	}
	CheckVelocity(ev)

	// add gravity
	if ev.MoveType != progs.MoveTypeFly &&
		ev.MoveType != progs.MoveTypeFlyMissile {
		q.addGravity(ent)
	}

	time := float32(host.frameTime)
	// move angles
	av := vec.VFromA(ev.AVelocity)
	av = av.Scale(time)
	angles := vec.VFromA(ev.Angles)
	na := vec.Add(angles, av)
	ev.Angles = na.Array()

	// move origin
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
	_, velocity = clipVelocity(velocity, n, backOff)
	ev.Velocity = velocity.Array()

	// stop if on ground
	if trace.plane.normal[2] > 0.7 {
		if ev.Velocity[2] < 60 || ev.MoveType != progs.MoveTypeBounce {
			ev.Flags = float32(int(ev.Flags) | FL_ONGROUND)
			ev.GroundEntity = int32(trace.entn)
			ev.Velocity = [3]float32{0, 0, 0}
			ev.AVelocity = [3]float32{0, 0, 0}
		}
	}

	// check for in water
	q.checkWaterTransition(ent)
}
