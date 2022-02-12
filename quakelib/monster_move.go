// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"goquake/bsp"
	"goquake/math"
	"goquake/math/vec"
	"goquake/progs"

	"github.com/chewxy/math32"
)

//Called by monster program code.
//The move will be adjusted for slopes and stairs, but if the move isn't
//possible, no move is done, false is returned, and
//pr_global_struct->trace_normal is set to the normal of the blocking wall
func (v *virtualMachine) monsterMoveStep(ent int, move vec.Vec3, relink bool) (bool, error) {
	const STEPSIZE = 18
	ev := EntVars(ent)
	mins := vec.VFromA(ev.Mins)
	maxs := vec.VFromA(ev.Maxs)
	flags := int(ev.Flags)

	// flying monsters don't step up
	if flags&(FL_SWIM|FL_FLY) != 0 {
		// try one move with vertical motion, then one without
		for i := 0; i < 2; i++ {
			origin := vec.VFromA(ev.Origin)
			neworg := vec.Add(origin, move)
			enemy := int(ev.Enemy)
			if i == 0 && enemy != 0 {
				dz := origin[2] - EntVars(enemy).Origin[2]
				if dz > 40 {
					neworg[2] -= 8
				}
				if dz < 30 {
					neworg[2] += 8
				}
			}
			trace := svMove(origin, mins, maxs, neworg, MOVE_NORMAL, ent)
			if trace.Fraction == 1 {
				endpos := trace.EndPos
				if flags&FL_SWIM != 0 && pointContents(endpos) == bsp.CONTENTS_EMPTY {
					// swim monster left water
					return false, nil
				}

				ev.Origin = endpos
				if relink {
					if err := v.LinkEdict(ent, true); err != nil {
						return false, err
					}
				}
				return true, nil
			}

			if enemy == 0 {
				break
			}
		}
		return false, nil
	}

	oldorg := ev.Origin
	neworg := vec.Add(oldorg, move)

	// push down from a step height above the wished position
	neworg[2] += STEPSIZE
	end := neworg
	end[2] -= STEPSIZE * 2
	trace := svMove(neworg, mins, maxs, end, MOVE_NORMAL, ent)
	if trace.AllSolid {
		return false, nil
	}
	if trace.StartSolid {
		neworg[2] -= STEPSIZE
		trace = svMove(neworg, mins, maxs, end, MOVE_NORMAL, ent)
		if trace.AllSolid || trace.StartSolid {
			return false, nil
		}
	}

	if trace.Fraction == 1 {
		// if monster had the ground pulled out, go ahead and fall
		if flags&FL_PARTIALGROUND != 0 {
			neworg = vec.Add(oldorg, move)
			ev.Origin = neworg
			if relink {
				if err := v.LinkEdict(ent, true); err != nil {
					return false, err
				}
			}
			ev.Flags = float32(flags &^ FL_ONGROUND)
			return true, nil
		}
		// walked off an edge
		return false, nil
	}
	// check point traces down for dangling corners
	ev.Origin = trace.EndPos

	if !checkBottom(ent) {
		if flags&FL_PARTIALGROUND != 0 {
			// entity had floor mostly pulled out from underneath it
			// and is trying to correct
			if relink {
				if err := v.LinkEdict(ent, true); err != nil {
					return false, err
				}
			}
			return true, nil
		}
		ev.Origin = oldorg
		return false, nil
	}

	if flags&FL_PARTIALGROUND != 0 {
		ev.Flags = float32(flags &^ FL_PARTIALGROUND)
	}

	ev.GroundEntity = int32(trace.EntNumber)
	// the move is ok
	if relink {
		if err := v.LinkEdict(ent, true); err != nil {
			return false, err
		}
	}
	return true, nil
}

// This was a major timewaster in progs
func changeYaw(ent int) {
	ev := EntVars(ent)
	current := math.AngleMod32(ev.Angles[1])
	ideal := ev.IdealYaw
	speed := ev.YawSpeed

	if current == ideal {
		return
	}
	move := ideal - current
	if ideal > current {
		if move >= 180 {
			move -= 360
		}
	} else {
		if move <= -180 {
			move += 360
		}
	}
	if move > 0 {
		if move > speed {
			move = speed
		}
	} else {
		if move < -speed {
			move = -speed
		}
	}
	ev.Angles[1] = math.AngleMod32(current + move)
}

// Turns to the movement direction, and walks the current distance if
// facing it.
func (v *virtualMachine) monsterStepDirection(ent int, yaw, dist float32) (bool, error) {
	ev := EntVars(ent)
	ev.IdealYaw = yaw

	changeYaw(ent)

	yaw = yaw * math32.Pi * 2 / 360
	s, c := math32.Sincos(yaw)
	move := vec.Vec3{
		c * dist,
		s * dist,
		0,
	}

	oldorigin := ev.Origin
	if ok, err := v.monsterMoveStep(ent, move, false); err != nil {
		return false, err
	} else if ok {
		delta := ev.Angles[1] - ev.IdealYaw
		if delta > 45 && delta < 315 {
			// not turned far enough, so don't take the step
			ev.Origin = oldorigin
		}
		if err := v.LinkEdict(ent, true); err != nil {
			return false, err
		}
		return true, nil
	}

	if err := v.LinkEdict(ent, true); err != nil {
		return false, err
	}
	return false, nil
}

func (v *virtualMachine) monsterNewChaseDir(a, e int, dist float32) error {
	const DI_NODIR = -1
	actor := EntVars(a)
	enemy := EntVars(e)

	olddir := math.AngleMod32(math32.Trunc(actor.IdealYaw/45) * 45)
	turnaround := math.AngleMod32(olddir - 180)

	deltax := enemy.Origin[0] - actor.Origin[0]
	deltay := enemy.Origin[1] - actor.Origin[1]

	d1 := func() float32 {
		if deltax > 10 {
			return 0
		} else if deltax < -10 {
			return 180
		}
		return DI_NODIR
	}()
	d2 := func() float32 {
		if deltay < -10 {
			return 270
		} else if deltay > 10 {
			return 90
		}
		return DI_NODIR
	}()

	// try direct route
	if d1 != DI_NODIR && d2 != DI_NODIR {
		tdir := func() float32 {
			if d1 == 0 {
				if d2 == 90 {
					return 45
				}
				return 315
			}
			if d2 == 90 {
				return 135
			}
			return 215
		}()

		if tdir != turnaround {
			if ok, err := v.monsterStepDirection(a, tdir, dist); err != nil {
				return err
			} else if ok {
				return nil
			}
		}
	}
	// try other directions
	if sRand.Uint32n(2) == 0 ||
		// TODO: Abs(Trunc seems overkill
		math32.Abs(math32.Trunc(deltay)) > math32.Abs(math32.Trunc(deltax)) {
		tdir := d1
		d1 = d2
		d2 = tdir
	}
	if d1 != DI_NODIR && d1 != turnaround {
		if ok, err := v.monsterStepDirection(a, d1, dist); err != nil {
			return err
		} else if ok {
			return nil
		}
	}
	if d2 != DI_NODIR && d2 != turnaround {
		if ok, err := v.monsterStepDirection(a, d2, dist); err != nil {
			return err
		} else if ok {
			return nil
		}
	}
	// there is no direct path to the player, so pick another direction
	if olddir != DI_NODIR {
		if ok, err := v.monsterStepDirection(a, olddir, dist); err != nil {
			return err
		} else if ok {
			return nil
		}
	}

	// randomly determine direction of search
	if sRand.Uint32n(2) == 0 {
		for tdir := float32(0); tdir <= 315; tdir += 45 {
			if tdir != turnaround {
				if ok, err := v.monsterStepDirection(a, tdir, dist); err != nil {
					return err
				} else if ok {
					return nil
				}
			}
		}
	} else {
		for tdir := float32(315); tdir >= 0; tdir -= 45 {
			if tdir != turnaround {
				if ok, err := v.monsterStepDirection(a, tdir, dist); err != nil {
					return err
				} else if ok {
					return nil
				}
			}
		}
	}

	if turnaround != DI_NODIR {
		if ok, err := v.monsterStepDirection(a, turnaround, dist); err != nil {
			return err
		} else if ok {
			return nil
		}
	}

	// can't move
	actor.IdealYaw = olddir

	// if a bridge was pulled out from underneath a monster, it may not have
	// a valid standing position at all
	if !checkBottom(a) {
		actor.Flags = float32(int(actor.Flags) | FL_PARTIALGROUND)
	}
	return nil
}

func monsterCloseEnough(e, g int, dist float32) bool {
	eev := EntVars(e)
	gev := EntVars(g)

	for i := 0; i < 3; i++ {
		if (gev.AbsMin[i] > eev.AbsMax[i]+dist) ||
			(gev.AbsMax[i] < eev.AbsMin[i]-dist) {
			return false
		}
	}
	return true
}

// this is part of vm_functions
func (v *virtualMachine) monsterMoveToGoal() error {
	ent := int(progsdat.Globals.Self)
	ev := EntVars(ent)

	if int(ev.Flags)&(FL_ONGROUND|FL_FLY|FL_SWIM) == 0 {
		progsdat.Globals.Returnf()[0] = 0
		return nil
	}
	goal := int(ev.GoalEntity)
	dist := progsdat.RawGlobalsF[progs.OffsetParm0]

	// if the next step hits the enemy, return immediately
	if ev.Enemy != 0 && monsterCloseEnough(ent, goal, dist) {
		return nil
	}

	// bump around...
	if sRand.Uint32n(3) == 0 {
		if err := v.monsterNewChaseDir(ent, goal, dist); err != nil {
			return err
		}
		return nil
	}

	if ok, err := v.monsterStepDirection(ent, ev.IdealYaw, dist); err != nil {
		return err
	} else if !ok {
		if err := v.monsterNewChaseDir(ent, goal, dist); err != nil {
			return err
		}
	}
	return nil
}
