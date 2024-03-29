// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"goquake/cvars"
	"goquake/math/vec"
	"goquake/progs"

	"github.com/chewxy/math32"
)

func (c *SVClient) accelerate(wishspeed float32, wishdir vec.Vec3) {
	ev := entvars.Get(c.edictId)
	velocity := vec.VFromA(ev.Velocity)
	currentspeed := vec.Dot(velocity, wishdir)
	addspeed := wishspeed - currentspeed
	if addspeed <= 0 {
		return
	}
	accelspeed := cvars.ServerAccelerate.Value() * float32(host.FrameTime()) * wishspeed
	if accelspeed > addspeed {
		accelspeed = addspeed
	}
	ev.Velocity = vec.Add(velocity, vec.Scale(accelspeed, wishdir))
}

func (c *SVClient) airAccelerate(wishspeed float32, wishveloc vec.Vec3) {
	ev := entvars.Get(c.edictId)
	velocity := vec.VFromA(ev.Velocity)

	wishspd := wishveloc.Length()
	if wishspd <= 0 {
		return
	}
	wishveloc = vec.Scale(1/wishspd, wishveloc)
	if wishspd > 30 {
		wishspd = 30
	}
	addspeed := wishspd - vec.Dot(velocity, wishveloc)
	if addspeed <= 0 {
		return
	}
	accelspeed := cvars.ServerAccelerate.Value() * float32(host.FrameTime()) * wishspeed
	if accelspeed > addspeed {
		accelspeed = addspeed
	}
	ev.Velocity = vec.Add(velocity, vec.Scale(accelspeed, wishveloc))
}

func (c *SVClient) noclipMove() {
	ev := entvars.Get(c.edictId)
	vangle := vec.VFromA(ev.VAngle)
	forward, right, _ := vec.AngleVectors(vangle)

	fmove := float32(c.cmd.forwardmove)
	smove := float32(c.cmd.sidemove)
	umove := float32(c.cmd.upmove)

	velocity := vec.Vec3{
		forward[0]*fmove + right[0]*smove,
		forward[1]*fmove + right[1]*smove,
		forward[2]*fmove + right[2]*smove,
	}
	// doubled to match running speed
	velocity[2] += umove * 2

	max := cvars.ServerMaxSpeed.Value()
	if velocity.Length() > max {
		velocity = vec.Scale(max, velocity.Normalize())
	}
	ev.Velocity = velocity
}

func (c *SVClient) waterMove() {
	ev := entvars.Get(c.edictId)
	// user intentions
	vangle := vec.VFromA(ev.VAngle)
	forward, right, _ := vec.AngleVectors(vangle)

	fmove := float32(c.cmd.forwardmove)
	smove := float32(c.cmd.sidemove)
	umove := float32(c.cmd.upmove)

	wishvel := vec.Vec3{
		forward[0]*fmove + right[0]*smove,
		forward[1]*fmove + right[1]*smove,
		forward[2]*fmove + right[2]*smove,
	}

	if fmove == 0 && smove == 0 && umove == 0 {
		// drift towards bottom
		wishvel[2] -= 60
	} else {
		wishvel[2] += umove
	}

	wishspeed := wishvel.Length()
	max := cvars.ServerMaxSpeed.Value()
	if wishspeed > max {
		wishvel = vec.Scale(max/wishspeed, wishvel)
		wishspeed = max
	}
	wishspeed *= 0.7

	// water friction
	velocity := vec.VFromA(ev.Velocity)
	speed := velocity.Length()
	newspeed := float32(0)
	if speed != 0 {
		newspeed = speed - float32(host.FrameTime())*speed*cvars.ServerFriction.Value()
		if newspeed < 0 {
			newspeed = 0
		}
		velocity = vec.Scale(newspeed/speed, velocity)
	}
	// water acceleration
	if wishspeed == 0 {
		return
	}

	addspeed := wishspeed - newspeed
	if addspeed <= 0 {
		return
	}

	wishvel = wishvel.Normalize()
	accelspeed := cvars.ServerAccelerate.Value() * wishspeed * float32(host.FrameTime())
	if accelspeed > addspeed {
		accelspeed = addspeed
	}
	ev.Velocity = vec.Add(velocity, vec.Scale(accelspeed, wishvel))
}

func (c *SVClient) userFriction() {
	ev := entvars.Get(c.edictId)
	velocity := vec.VFromA(ev.Velocity)
	speed2 := velocity[0]*velocity[0] + velocity[1]*velocity[1]
	if speed2 == 0 {
		return
	}
	origin := vec.VFromA(ev.Origin)
	speed := math32.Sqrt(speed2)

	// if the leading edge is over a dropoff, increase friction
	start := vec.Vec3{
		origin[0] + velocity[0]/speed*16,
		origin[1] + velocity[1]/speed*16,
		origin[2] + ev.Mins[2],
	}
	stop := start
	stop[2] -= 34

	t := svMove(start, vec.Vec3{}, vec.Vec3{}, stop, 1, c.edictId)

	friction := cvars.ServerFriction.Value()
	if t.Fraction == 1.0 {
		friction *= cvars.ServerEdgeFriction.Value()
	}

	control := func() float32 {
		stopspeed := cvars.ServerStopSpeed.Value()
		if speed < stopspeed {
			return stopspeed
		}
		return speed
	}()
	newspeed := speed - float32(host.FrameTime())*control*friction

	if newspeed <= 0 {
		ev.Velocity = [3]float32{0, 0, 0}
		return
	}
	newspeed /= speed
	ev.Velocity = vec.Scale(newspeed, velocity)
}

func (c *SVClient) airMove() {
	ev := entvars.Get(c.edictId)
	forward, right, _ := vec.AngleVectors(vec.VFromA(ev.Angles))
	fmove := float32(c.cmd.forwardmove)
	smove := float32(c.cmd.sidemove)
	umove := float32(c.cmd.upmove)

	// hack to not let you back into teleporter
	if sv.time < ev.TeleportTime && fmove < 0 {
		fmove = 0
	}

	wishvel := vec.Vec3{
		forward[0]*fmove + right[0]*smove,
		forward[1]*fmove + right[1]*smove,
		0,
	}

	if ev.MoveType != progs.MoveTypeWalk {
		wishvel[2] = umove
	}

	wishspeed := wishvel.Length()
	wishdir := func() vec.Vec3 {
		if wishspeed != 0 {
			return vec.Scale(1/wishspeed, wishvel)
		}
		return wishvel
	}()

	max := cvars.ServerMaxSpeed.Value()
	if wishspeed > max {
		wishvel = vec.Scale(max/wishspeed, wishvel)
		wishspeed = max
	}

	onground := int(ev.Flags)&FL_ONGROUND != 0
	if ev.MoveType == progs.MoveTypeNoClip {
		ev.Velocity = wishvel
	} else if onground {
		c.userFriction()
		c.accelerate(wishspeed, wishdir)
	} else {
		// not on ground, so little effect on velocity
		c.airAccelerate(wishspeed, wishvel)
	}
}

func (c *SVClient) DropPunchAngle() {
	ev := entvars.Get(c.edictId)
	pa := vec.VFromA(ev.PunchAngle)
	len := pa.Length()
	if len == 0 {
		len = 1
	}
	len2 := 1 - (10 * float32(host.FrameTime()) / len)
	if len2 < 0 {
		len2 = 0
	}
	ev.PunchAngle = vec.Scale(len2, pa)
}

func (c *SVClient) WaterJump() {
	ev := entvars.Get(c.edictId)
	if sv.time > ev.TeleportTime || ev.WaterLevel == 0 {
		ev.Flags = float32(int(ev.Flags) &^ FL_WATERJUMP)
		ev.TeleportTime = 0
	}
	ev.Velocity[0] = ev.MoveDir[0]
	ev.Velocity[1] = ev.MoveDir[1]
}

// the move fields specify an intended velocity in pix/sec
// the angle fields specify an exact angular motion in degrees
func (c *SVClient) Think() {
	ev := entvars.Get(c.edictId)

	if ev.MoveType == progs.MoveTypeNone {
		return
	}
	c.DropPunchAngle()
	if ev.Health <= 0 {
		// if dead, behave differently
		return
	}

	// onground := int(ev.Flags)&FL_ONGROUND != 0
	// origin := vec.VFromA(ev.Origin)
	// velocity := vec.VFromA(ev.Velocity)
	angles := vec.VFromA(ev.Angles)

	// show 1/3 the pitch angle and all the roll angle
	vAngle := vec.Add(vec.VFromA(ev.VAngle), vec.VFromA(ev.PunchAngle))
	angles[2] = cvars.CalcRoll(angles, vec.VFromA(ev.Velocity)) * 4 // ROLL
	if ev.FixAngle == 0 {
		angles[0] = -vAngle[0] / 3 // PITCH
		angles[1] = vAngle[1]      // YAW
	}
	ev.Angles = angles

	if int(ev.Flags)&FL_WATERJUMP != 0 {
		c.WaterJump()
		return
	}
	// walk
	if ev.MoveType == progs.MoveTypeNoClip && cvars.ServerAltNoClip.Bool() {
		c.noclipMove()
	} else if ev.WaterLevel >= 2 && ev.MoveType != progs.MoveTypeNoClip {
		c.waterMove()
	} else {
		c.airMove()
	}
}
