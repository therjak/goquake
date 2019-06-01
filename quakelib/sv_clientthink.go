package quakelib

import (
	"quake/cvars"
	"quake/math/vec"
	"quake/progs"

	"github.com/chewxy/math32"
)

func (c *SVClient) accelerate(wishspeed float32, wishdir vec.Vec3) {
	ev := EntVars(c.edictId)
	velocity := vec.VFromA(ev.Velocity)
	currentspeed := vec.Dot(velocity, wishdir)
	addspeed := wishspeed - currentspeed
	if addspeed <= 0 {
		return
	}
	accelspeed := cvars.ServerAccelerate.Value() * float32(host.frameTime) * wishspeed
	if accelspeed > addspeed {
		accelspeed = addspeed
	}
	velocity = vec.Add(velocity, wishdir.Scale(accelspeed))
	ev.Velocity = velocity.Array()
}

func (c *SVClient) airAccelerate(wishspeed float32, wishveloc vec.Vec3) {
	ev := EntVars(c.edictId)
	velocity := vec.VFromA(ev.Velocity)

	wishspd := wishveloc.Length()
	if wishspd <= 0 {
		return
	}
	wishveloc = wishveloc.Scale(1 / wishspd)
	if wishspd > 30 {
		wishspd = 30
	}
	addspeed := wishspd - vec.Dot(velocity, wishveloc)
	if addspeed <= 0 {
		return
	}
	accelspeed := cvars.ServerAccelerate.Value() * float32(host.frameTime) * wishspeed
	if accelspeed > addspeed {
		accelspeed = addspeed
	}
	velocity = vec.Add(velocity, wishveloc.Scale(accelspeed))
	ev.Velocity = velocity.Array()
}

func (c *SVClient) noclipMove() {
	ev := EntVars(c.edictId)
	vangle := vec.VFromA(ev.VAngle)
	forward, right, _ := vec.AngleVectors(vangle)

	fmove := float32(c.cmd.forwardmove)
	smove := float32(c.cmd.sidemove)
	umove := float32(c.cmd.upmove)

	velocity := vec.Vec3{
		forward.X*fmove + right.X*smove,
		forward.Y*fmove + right.Y*smove,
		forward.Z*fmove + right.Z*smove,
	}
	// doubled to match running speed
	velocity.Z += umove * 2

	max := cvars.ServerMaxSpeed.Value()
	if velocity.Length() > max {
		velocity = velocity.Normalize()
		velocity = velocity.Scale(max)
	}
	ev.Velocity = velocity.Array()
}

func (c *SVClient) waterMove() {
	ev := EntVars(c.edictId)
	// user intentions
	vangle := vec.VFromA(ev.VAngle)
	forward, right, _ := vec.AngleVectors(vangle)

	fmove := float32(c.cmd.forwardmove)
	smove := float32(c.cmd.sidemove)
	umove := float32(c.cmd.upmove)

	wishvel := vec.Vec3{
		forward.X*fmove + right.X*smove,
		forward.Y*fmove + right.Y*smove,
		forward.Z*fmove + right.Z*smove,
	}

	if fmove == 0 && smove == 0 && umove == 0 {
		// drift towards bottom
		wishvel.Z -= 60
	} else {
		wishvel.Z += umove
	}

	wishspeed := wishvel.Length()
	max := cvars.ServerMaxSpeed.Value()
	if wishspeed > max {
		wishvel = wishvel.Scale(max / wishspeed)
		wishspeed = max
	}
	wishspeed *= 0.7

	// water friction
	velocity := vec.VFromA(ev.Velocity)
	speed := velocity.Length()
	newspeed := float32(0)
	if speed != 0 {
		newspeed = speed - float32(host.frameTime)*speed*cvars.ServerFriction.Value()
		if newspeed < 0 {
			newspeed = 0
		}
		velocity = velocity.Scale(newspeed / speed)
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
	accelspeed := cvars.ServerAccelerate.Value() * wishspeed * float32(host.frameTime)
	if accelspeed > addspeed {
		accelspeed = addspeed
	}
	velocity = vec.Add(velocity, wishvel.Scale(accelspeed))

	ev.Velocity = velocity.Array()
}

func (c *SVClient) userFriction() {
	ev := EntVars(c.edictId)
	velocity := vec.VFromA(ev.Velocity)
	origin := vec.VFromA(ev.Origin)
	speed := math32.Sqrt(velocity.X*velocity.X + velocity.Y*velocity.Y)
	if speed == 0 {
		return
	}

	// if the leading edge is over a dropoff, increase friction
	start := vec.Vec3{
		origin.X + velocity.X/speed*16,
		origin.Y + velocity.Y/speed*16,
		origin.Z + ev.Mins[2],
	}
	stop := start
	stop.Z -= 34

	trace := svMove(start, vec.Vec3{}, vec.Vec3{}, stop, 1, c.edictId)

	friction := cvars.ServerFriction.Value()
	if trace.fraction == 1.0 {
		friction *= cvars.ServerEdgeFriction.Value()
	}

	control := func() float32 {
		stopspeed := cvars.ServerStopSpeed.Value()
		if speed < stopspeed {
			return stopspeed
		}
		return speed
	}()
	newspeed := speed - float32(host.frameTime)*control*friction

	if newspeed <= 0 {
		ev.Velocity = [3]float32{0, 0, 0}
		return
	}
	newspeed /= speed
	velocity = velocity.Scale(newspeed)
	ev.Velocity = velocity.Array()
}

func (c *SVClient) airMove() {
	ev := EntVars(c.edictId)
	forward, right, _ := vec.AngleVectors(vec.VFromA(ev.Angles))
	fmove := float32(c.cmd.forwardmove)
	smove := float32(c.cmd.sidemove)
	umove := float32(c.cmd.upmove)

	// hack to not let you back into teleporter
	if sv.time < ev.TeleportTime && fmove < 0 {
		fmove = 0
	}

	wishvel := vec.Vec3{
		forward.X*fmove + right.X*smove,
		forward.Y*fmove + right.Y*smove,
		0,
	}

	if ev.MoveType != progs.MoveTypeWalk {
		wishvel.Z = umove
	}

	wishspeed := wishvel.Length()
	wishdir := func() vec.Vec3 {
		if wishspeed != 0 {
			return wishvel.Scale(1 / wishspeed)
		}
		return wishvel
	}()

	max := cvars.ServerMaxSpeed.Value()
	if wishspeed > max {
		wishvel = wishvel.Scale(max / wishspeed)
		wishspeed = max
	}

	onground := int(ev.Flags)&FL_ONGROUND != 0
	if ev.MoveType == progs.MoveTypeNoClip {
		ev.Velocity = wishvel.Array()
	} else if onground {
		c.userFriction()
		c.accelerate(wishspeed, wishdir)
	} else {
		// not on ground, so little effect on velocity
		c.airAccelerate(wishspeed, wishvel)
	}
}

func (c *SVClient) DropPunchAngle() {
	ev := EntVars(c.edictId)
	pa := vec.VFromA(ev.PunchAngle)
	len := pa.Length()
	if len == 0 {
		len = 1
	}
	len2 := 1 - (10 * float32(host.frameTime) / len)
	if len2 < 0 {
		len2 = 0
	}
	newPa := pa.Scale(len2)
	ev.PunchAngle = newPa.Array()
}

func (c *SVClient) WaterJump() {
	ev := EntVars(c.edictId)
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
	ev := EntVars(c.edictId)

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
	angles.Z = CalcRoll(angles, vec.VFromA(ev.Velocity)) * 4 // ROLL
	if ev.FixAngle == 0 {
		angles.X = -vAngle.X / 3 // PITCH
		angles.Y = vAngle.Y      // YAW
	}
	ev.Angles = angles.Array()

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