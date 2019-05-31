package quakelib

import "C"

import (
	"quake/math/vec"

	"github.com/chewxy/math32"
)

//export SV_movestep
func SV_movestep(ent int, move *C.float, relink int) C.int {
	m := p2v3(move)
	return b2i(monsterMoveStep(ent, m, relink != 0))
}

//Called by monster program code.
//The move will be adjusted for slopes and stairs, but if the move isn't
//possible, no move is done, false is returned, and
//pr_global_struct->trace_normal is set to the normal of the blocking wall
func monsterMoveStep(ent int, move vec.Vec3, relink bool) bool {
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
				dz := origin.Z - EntVars(enemy).Origin[2]
				if dz > 40 {
					neworg.Z -= 8
				}
				if dz < 30 {
					neworg.Z += 8
				}
			}
			trace := svMove(origin, mins, maxs, neworg, MOVE_NORMAL, ent)
			if trace.fraction == 1 {
				endpos := vec.Vec3{
					float32(trace.endpos[0]),
					float32(trace.endpos[1]),
					float32(trace.endpos[2]),
				}
				if flags&FL_SWIM != 0 && pointContents(endpos) == CONTENTS_EMPTY {
					// swim monster left water
					return false
				}

				ev.Origin = endpos.Array()
				if relink {
					LinkEdict(ent, true)
				}
				return true
			}

			if enemy == 0 {
				break
			}
		}
		return false
	}

	oldorg := vec.VFromA(ev.Origin)
	neworg := vec.Add(oldorg, move)

	// push down from a step height above the wished position
	neworg.Z += STEPSIZE
	end := neworg
	end.Z -= STEPSIZE * 2
	trace := svMove(neworg, mins, maxs, end, MOVE_NORMAL, ent)
	if trace.allsolid != 0 {
		return false
	}
	if trace.startsolid != 0 {
		neworg.Z -= STEPSIZE
		trace = svMove(neworg, mins, maxs, end, MOVE_NORMAL, ent)
		if trace.allsolid != 0 || trace.startsolid != 0 {
			return false
		}
	}

	if trace.fraction == 1 {
		// if monster had the ground pulled out, go ahead and fall
		if flags&FL_PARTIALGROUND != 0 {
			neworg = vec.Add(oldorg, move)
			ev.Origin = neworg.Array()
			if relink {
				LinkEdict(ent, true)
			}
			ev.Flags = float32(flags &^ FL_ONGROUND)
			return true
		}
		// walked off an edge
		return false
	}
	endpos := vec.Vec3{
		float32(trace.endpos[0]),
		float32(trace.endpos[1]),
		float32(trace.endpos[2]),
	}
	// check point traces down for dangling corners
	ev.Origin = endpos.Array()

	if !checkBottom(ent) {
		if flags&FL_PARTIALGROUND != 0 {
			// entity had floor mostly pulled out from underneath it
			// and is trying to correct
			if relink {
				LinkEdict(ent, true)
			}
			return true
		}
		ev.Origin = oldorg.Array()
		return false
	}

	if flags&FL_PARTIALGROUND != 0 {
		ev.Flags = float32(flags &^ FL_PARTIALGROUND)
	}

	ev.GroundEntity = int32(trace.entn)
	// the move is ok
	if relink {
		LinkEdict(ent, true)
	}
	return true
}

//export SV_StepDirection
func SV_StepDirection(ent int, yaw, dist float32) C.int {
	return b2i(monsterStepDirection(ent, yaw, dist))
}

// Turns to the movement direction, and walks the current distance if
// facing it.
func monsterStepDirection(ent int, yaw, dist float32) bool {
	ev := EntVars(ent)
	ev.IdealYaw = yaw

	PF_changeyaw() // TODO: probably both should call another function.

	yaw = yaw * math32.Pi * 2 / 360
	s, c := math32.Sincos(yaw)
	move := vec.Vec3{
		c * dist,
		s * dist,
		0,
	}

	oldorigin := ev.Origin
	if monsterMoveStep(ent, move, false) {
		delta := ev.Angles[1] - ev.IdealYaw
		if delta > 45 && delta < 315 {
			// not turned far enough, so don't take the step
			ev.Origin = oldorigin
		}
		LinkEdict(ent, true)
		return true
	}

	LinkEdict(ent, true)
	return false
}

/*
#define DI_NODIR -1
void SV_NewChaseDir(int actor, int e, float dist) {
  float deltax, deltay;
  float d[3];
  float tdir, olddir, turnaround;
  entvars_t *enemy = EVars(e);

  olddir = anglemod((int)(EVars(actor)->ideal_yaw / 45) * 45);
  turnaround = anglemod(olddir - 180);

  deltax = enemy->origin[0] - EVars(actor)->origin[0];
  deltay = enemy->origin[1] - EVars(actor)->origin[1];
  if (deltax > 10)
    d[1] = 0;
  else if (deltax < -10)
    d[1] = 180;
  else
    d[1] = DI_NODIR;
  if (deltay < -10)
    d[2] = 270;
  else if (deltay > 10)
    d[2] = 90;
  else
    d[2] = DI_NODIR;

  // try direct route
  if (d[1] != DI_NODIR && d[2] != DI_NODIR) {
    if (d[1] == 0)
      tdir = d[2] == 90 ? 45 : 315;
    else
      tdir = d[2] == 90 ? 135 : 215;

    if (tdir != turnaround && SV_StepDirection(actor, tdir, dist)) return;
  }

  // try other directions
  if (((rand() & 3) & 1) || abs((int)deltay) > abs((int)deltax)) {
    tdir = d[1];
    d[1] = d[2];
    d[2] = tdir;
  }

  if (d[1] != DI_NODIR && d[1] != turnaround &&
      SV_StepDirection(actor, d[1], dist))
    return;

  if (d[2] != DI_NODIR && d[2] != turnaround &&
      SV_StepDirection(actor, d[2], dist))
    return;

  // there is no direct path to the player, so pick another direction

  if (olddir != DI_NODIR && SV_StepDirection(actor, olddir, dist)) return;

  if (rand() & 1) // randomly determine direction of search
  {
    for (tdir = 0; tdir <= 315; tdir += 45)
      if (tdir != turnaround && SV_StepDirection(actor, tdir, dist)) return;
  } else {
    for (tdir = 315; tdir >= 0; tdir -= 45)
      if (tdir != turnaround && SV_StepDirection(actor, tdir, dist)) return;
  }

  if (turnaround != DI_NODIR && SV_StepDirection(actor, turnaround, dist))
    return;

  EVars(actor)->ideal_yaw = olddir;  // can't move

  // if a bridge was pulled out from underneath a monster, it may not have
  // a valid standing position at all

  if (!SV_CheckBottom(actor)) {
    entvars_t *ent = EVars(actor);
    ent->flags = (int)ent->flags | FL_PARTIALGROUND;
  }
}

qboolean SV_CloseEnough(int e, int g, float dist) {
  int i;
  entvars_t *ent = EVars(e);
  entvars_t *goal = EVars(g);

  for (i = 0; i < 3; i++) {
    if (goal->absmin[i] > ent->absmax[i] + dist) return false;
    if (goal->absmax[i] < ent->absmin[i] - dist) return false;
  }
  return true;
}

void SV_MoveToGoal(void) {
  int ent;
  int goal;
  float dist;

  ent = Pr_global_struct_self();
  goal = EVars(ent)->goalentity;
  dist = Pr_globalsf(OFS_PARM0);

  if (!((int)EVars(ent)->flags & (FL_ONGROUND | FL_FLY | FL_SWIM))) {
    Set_Pr_globalsf(OFS_RETURN, 0);
    return;
  }

  // if the next step hits the enemy, return immediately
  if (EVars(ent)->enemy != 0 && SV_CloseEnough(ent, goal, dist))
    return;

  // bump around...
  if ((rand() & 3) == 1 ||
      !SV_StepDirection(ent, EVars(ent)->ideal_yaw, dist)) {
    SV_NewChaseDir(ent, goal, dist);
  }
}
*/
