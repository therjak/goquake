// sv_phys.c

#include "quakedef.h"

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

cvar_t sv_friction;
cvar_t sv_stopspeed;
cvar_t sv_gravity;
cvar_t sv_maxvelocity;
cvar_t sv_nostep;
cvar_t sv_freezenonclients;

#define MOVE_EPSILON 0.01

void SV_Physics_Toss(int ent);

/*
================
SV_CheckAllEnts
================
*/
void SV_CheckAllEnts(void) {
  int e;
  int check;

  // see if any solid entities are inside the final position
  check = 1;
  for (e = 1; e < SV_NumEdicts(); e++, check++) {
    if (EDICT_NUM(check)->free) continue;
    if (EVars(check)->movetype == MOVETYPE_PUSH ||
        EVars(check)->movetype == MOVETYPE_NONE ||
        EVars(check)->movetype == MOVETYPE_NOCLIP)
      continue;

    if (SV_TestEntityPosition(check))
      Con_Printf("entity in invalid position\n");
  }
}

/*
================
SV_CheckVelocity
================
*/
void SV_CheckVelocity(entvars_t *ent) {
  int i;

  //
  // bound velocity
  //
  for (i = 0; i < 3; i++) {
    if (IS_NAN(ent->velocity[i])) {
      Con_Printf("Got a NaN velocity on %s\n", PR_GetString(ent->classname));
      ent->velocity[i] = 0;
    }
    if (IS_NAN(ent->origin[i])) {
      Con_Printf("Got a NaN origin on %s\n", PR_GetString(ent->classname));
      ent->origin[i] = 0;
    }
    if (ent->velocity[i] > Cvar_GetValue(&sv_maxvelocity))
      ent->velocity[i] = Cvar_GetValue(&sv_maxvelocity);
    else if (ent->velocity[i] < -Cvar_GetValue(&sv_maxvelocity))
      ent->velocity[i] = -Cvar_GetValue(&sv_maxvelocity);
  }
}

/*
=============
SV_RunThink

Runs thinking code if time.  There is some play in the exact time the think
function will be called, because it is called before any movement is done
in a frame.  Not used for pushmove objects, because they must be exact.
Returns false if the entity removed itself.
=============
*/
qboolean SV_RunThink(int e) {
  float thinktime;
  float oldframe;  // johnfitz
  int i;           // johnfitz

  thinktime = EVars(e)->nextthink;
  if (thinktime <= 0 || thinktime > SV_Time() + HostFrameTime()) return true;

  if (thinktime < SV_Time())
    thinktime = SV_Time();  // don't let things stay in the past.
                            // it is possible to start that way
                            // by a trigger with a local time.

  oldframe = EVars(e)->frame;  // johnfitz

  EVars(e)->nextthink = 0;
  Set_pr_global_struct_time(thinktime);
  Set_pr_global_struct_self(e);
  Set_pr_global_struct_other(0);
  PR_ExecuteProgram(EVars(e)->think);

  // johnfitz -- PROTOCOL_FITZQUAKE
  // capture interval to nextthink here and send it to client for better
  // lerp timing, but only if interval is not 0.1 (which client assumes)
  EDICT_NUM(e)->sendinterval = false;
  if (!EDICT_NUM(e)->free && EVars(e)->nextthink &&
      (EVars(e)->movetype == MOVETYPE_STEP ||
       EVars(e)->frame != oldframe)) {
    i = Q_rint((EVars(e)->nextthink - thinktime) * 255);
    if (i >= 0 && i < 256 && i != 25 &&
        i != 26)  // 25 and 26 are close enough to 0.1 to not send
      EDICT_NUM(e)->sendinterval = true;
  }
  // johnfitz

  return !EDICT_NUM(e)->free;
}

/*
==================
SV_Impact

Two entities have touched, so run their touch functions
==================
*/
void SV_Impact(int e1, int e2) {
  int old_self, old_other;

  old_self = Pr_global_struct_self();
  old_other = Pr_global_struct_other();

  Set_pr_global_struct_time(SV_Time());
  if (EVars(e1)->touch && EVars(e1)->solid != SOLID_NOT) {
    Set_pr_global_struct_self(e1);
    Set_pr_global_struct_other(e2);
    PR_ExecuteProgram(EVars(e1)->touch);
  }

  if (EVars(e2)->touch && EVars(e2)->solid != SOLID_NOT) {
    Set_pr_global_struct_self(e2);
    Set_pr_global_struct_other(e1);
    PR_ExecuteProgram(EVars(e2)->touch);
  }

  Set_pr_global_struct_self(old_self);
  Set_pr_global_struct_other(old_other);
}

/*
==================
ClipVelocity

Slide off of the impacting object
returns the blocked flags (1 = floor, 2 = step / wall)
==================
*/
#define STOP_EPSILON 0.1

int ClipVelocity(vec3_t in, vec3_t normal, vec3_t out, float overbounce) {
  float backoff;
  float change;
  int i, blocked;

  blocked = 0;
  if (normal[2] > 0) blocked |= 1;  // floor
  if (!normal[2]) blocked |= 2;     // step

  backoff = DotProduct(in, normal) * overbounce;

  for (i = 0; i < 3; i++) {
    change = normal[i] * backoff;
    out[i] = in[i] - change;
    if (out[i] > -STOP_EPSILON && out[i] < STOP_EPSILON) out[i] = 0;
  }

  return blocked;
}

/*
============
SV_FlyMove

The basic solid body movement clip that slides along multiple planes
Returns the clipflags if the velocity was modified (hit something solid)
1 = floor
2 = wall / step
4 = dead stop
If steptrace is not NULL, the trace of any vertical wall hit will be stored
============
*/
#define MAX_CLIP_PLANES 5
int SV_FlyMove(int ent, float time, trace_t *steptrace) {
  int bumpcount, numbumps;
  vec3_t dir;
  float d;
  int numplanes;
  vec3_t planes[MAX_CLIP_PLANES];
  vec3_t primal_velocity, original_velocity, new_velocity;
  int i, j;
  trace_t trace;
  vec3_t end;
  float time_left;
  int blocked;

  numbumps = 4;

  blocked = 0;
  VectorCopy(EVars(ent)->velocity, original_velocity);
  VectorCopy(EVars(ent)->velocity, primal_velocity);
  numplanes = 0;

  time_left = time;

  for (bumpcount = 0; bumpcount < numbumps; bumpcount++) {
    if (!EVars(ent)->velocity[0] && !EVars(ent)->velocity[1] &&
        !EVars(ent)->velocity[2])
      break;

    for (i = 0; i < 3; i++)
      end[i] = EVars(ent)->origin[i] + time_left * EVars(ent)->velocity[i];

    trace = SV_Move(EVars(ent)->origin, EVars(ent)->mins, EVars(ent)->maxs,
                    end, false, ent);

    if (trace.allsolid) {  // entity is trapped in another solid
      VectorCopy(vec3_origin, EVars(ent)->velocity);
      return 3;
    }

    if (trace.fraction > 0) {  // actually covered some distance
      VectorCopy(trace.endpos, EVars(ent)->origin);
      VectorCopy(EVars(ent)->velocity, original_velocity);
      numplanes = 0;
    }

    if (trace.fraction == 1) break;  // moved the entire distance

    if (!trace.entp) Go_Error("SV_FlyMove: !trace.ent");

    if (trace.plane.normal[2] > 0.7) {
      blocked |= 1;  // floor
      if (EVars(trace.entn)->solid == SOLID_BSP) {
        EVars(ent)->flags = (int)EVars(ent)->flags | FL_ONGROUND;
        EVars(ent)->groundentity = trace.entn;
      }
    }
    if (!trace.plane.normal[2]) {
      blocked |= 2;                       // step
      if (steptrace) *steptrace = trace;  // save for player extrafriction
    }

    //
    // run the impact function
    //
    SV_Impact(ent, trace.entn);
    if (EDICT_NUM(ent)->free) break;  // removed by the impact function

    time_left -= time_left * trace.fraction;

    // cliped to another plane
    if (numplanes >= MAX_CLIP_PLANES) {  // this shouldn't really happen
      VectorCopy(vec3_origin, EVars(ent)->velocity);
      return 3;
    }

    VectorCopy(trace.plane.normal, planes[numplanes]);
    numplanes++;

    //
    // modify original_velocity so it parallels all of the clip planes
    //
    for (i = 0; i < numplanes; i++) {
      ClipVelocity(original_velocity, planes[i], new_velocity, 1);
      for (j = 0; j < numplanes; j++)
        if (j != i) {
          if (DotProduct(new_velocity, planes[j]) < 0) break;  // not ok
        }
      if (j == numplanes) break;
    }

    if (i != numplanes) {  // go along this plane
      VectorCopy(new_velocity, EVars(ent)->velocity);
    } else {  // go along the crease
      if (numplanes != 2) {
        //				Con_Printf ("clip velocity, numplanes ==
        //%i\n",numplanes);
        VectorCopy(vec3_origin, EVars(ent)->velocity);
        return 7;
      }
      CrossProduct(planes[0], planes[1], dir);
      d = DotProduct(dir, EVars(ent)->velocity);
      VectorScale(dir, d, EVars(ent)->velocity);
    }

    //
    // if original velocity is against the original velocity, stop dead
    // to avoid tiny occilations in sloping corners
    //
    if (DotProduct(EVars(ent)->velocity, primal_velocity) <= 0) {
      VectorCopy(vec3_origin, EVars(ent)->velocity);
      return blocked;
    }
  }

  return blocked;
}

/*
============
SV_AddGravity

============
*/
void SV_AddGravity(int ent) {
  float ent_gravity;
  eval_t *val;

  val = GetEdictFieldValue(EVars(ent), "gravity");
  if (val && val->_float)
    ent_gravity = val->_float;
  else
    ent_gravity = 1.0;

  EVars(ent)->velocity[2] -=
      ent_gravity * Cvar_GetValue(&sv_gravity) * HostFrameTime();
}

/*
===============================================================================

PUSHMOVE

===============================================================================
*/

/*
============
SV_PushEntity

Does not change the entities velocity at all
============
*/
trace_t SV_PushEntity(int ent, vec3_t push) {
  trace_t trace;
  vec3_t end;

  VectorAdd(EVars(ent)->origin, push, end);

  if (EVars(ent)->movetype == MOVETYPE_FLYMISSILE)
    trace = SV_Move(EVars(ent)->origin, EVars(ent)->mins, EVars(ent)->maxs,
                    end, MOVE_MISSILE, ent);
  else if (EVars(ent)->solid == SOLID_TRIGGER ||
           EVars(ent)->solid == SOLID_NOT)
    // only clip against bmodels
    trace = SV_Move(EVars(ent)->origin, EVars(ent)->mins, EVars(ent)->maxs,
                    end, MOVE_NOMONSTERS, ent);
  else
    trace = SV_Move(EVars(ent)->origin, EVars(ent)->mins, EVars(ent)->maxs,
                    end, MOVE_NORMAL, ent);

  VectorCopy(trace.endpos, EVars(ent)->origin);
  SV_LinkEdict(ent, true);

  if (trace.entp) SV_Impact(ent, trace.entn);

  return trace;
}

/*
============
SV_PushMove
============
*/
void SV_PushMove(int pusher, float movetime) {
  int i, e;
  int check;
  qboolean block;
  vec3_t mins, maxs, move;
  vec3_t entorig, pushorig;
  int num_moved;
  int *moved_edict;  // johnfitz -- dynamically allocate
  vec3_t *moved_from;     // johnfitz -- dynamically allocate
  int mark;               // johnfitz

  if (!EVars(pusher)->velocity[0] && !EVars(pusher)->velocity[1] &&
      !EVars(pusher)->velocity[2]) {
    EVars(pusher)->ltime += movetime;
    return;
  }

  for (i = 0; i < 3; i++) {
    move[i] = EVars(pusher)->velocity[i] * movetime;
    mins[i] = EVars(pusher)->absmin[i] + move[i];
    maxs[i] = EVars(pusher)->absmax[i] + move[i];
  }

  VectorCopy(EVars(pusher)->origin, pushorig);

  // move the pusher to it's final position

  VectorAdd(EVars(pusher)->origin, move, EVars(pusher)->origin);
  EVars(pusher)->ltime += movetime;
  SV_LinkEdict(pusher, false);

  // johnfitz -- dynamically allocate
  mark = Hunk_LowMark();
  moved_edict = (int *)Hunk_Alloc(SV_NumEdicts() * sizeof(int));
  moved_from = (vec3_t *)Hunk_Alloc(SV_NumEdicts() * sizeof(vec3_t));
  // johnfitz

  // see if any solid entities are inside the final position
  num_moved = 0;
  check = 1;
  for (e = 1; e < SV_NumEdicts(); e++, check++) {
    if (EDICT_NUM(check)->free) continue;
    if (EVars(check)->movetype == MOVETYPE_PUSH ||
        EVars(check)->movetype == MOVETYPE_NONE ||
        EVars(check)->movetype == MOVETYPE_NOCLIP)
      continue;

    // if the entity is standing on the pusher, it will definately be moved
    if (!(((int)EVars(check)->flags & FL_ONGROUND) &&
          EVars(check)->groundentity == pusher)) {
      if (EVars(check)->absmin[0] >= maxs[0] ||
          EVars(check)->absmin[1] >= maxs[1] ||
          EVars(check)->absmin[2] >= maxs[2] ||
          EVars(check)->absmax[0] <= mins[0] ||
          EVars(check)->absmax[1] <= mins[1] ||
          EVars(check)->absmax[2] <= mins[2])
        continue;

      // see if the ent's bbox is inside the pusher's final position
      if (!SV_TestEntityPosition(check)) continue;
    }

    // remove the onground flag for non-players
    if (EVars(check)->movetype != MOVETYPE_WALK)
      EVars(check)->flags = (int)EVars(check)->flags & ~FL_ONGROUND;

    VectorCopy(EVars(check)->origin, entorig);
    VectorCopy(EVars(check)->origin, moved_from[num_moved]);
    moved_edict[num_moved] = check;
    num_moved++;

    // try moving the contacted entity
    EVars(pusher)->solid = SOLID_NOT;
    SV_PushEntity(check, move);
    EVars(pusher)->solid = SOLID_BSP;

    // if it is still inside the pusher, block
    block = SV_TestEntityPosition(check);
    if (block) {  // fail the move
      if (EVars(check)->mins[0] == EVars(check)->maxs[0]) continue;
      if (EVars(check)->solid == SOLID_NOT ||
          EVars(check)->solid == SOLID_TRIGGER) {  // corpse
        EVars(check)->mins[0] = EVars(check)->mins[1] = 0;
        VectorCopy(EVars(check)->mins, EVars(check)->maxs);
        continue;
      }

      VectorCopy(entorig, EVars(check)->origin);
      SV_LinkEdict(check, true);

      VectorCopy(pushorig, EVars(pusher)->origin);
      SV_LinkEdict(pusher, false);
      EVars(pusher)->ltime -= movetime;

      // if the pusher has a "blocked" function, call it
      // otherwise, just stay in place until the obstacle is gone
      if (EVars(pusher)->blocked) {
        Set_pr_global_struct_self(pusher);
        Set_pr_global_struct_other(check);
        PR_ExecuteProgram(EVars(pusher)->blocked);
      }

      // move back any entities we already moved
      for (i = 0; i < num_moved; i++) {
        VectorCopy(moved_from[i], EVars(moved_edict[i])->origin);
        SV_LinkEdict(moved_edict[i], false);
      }
      Hunk_FreeToLowMark(mark);  // johnfitz
      return;
    }
  }

  Hunk_FreeToLowMark(mark);  // johnfitz
}

/*
================
SV_Physics_Pusher

================
*/
void SV_Physics_Pusher(int ent) {
  float thinktime;
  float oldltime;
  float movetime;

  oldltime = EVars(ent)->ltime;

  thinktime = EVars(ent)->nextthink;
  if (thinktime < EVars(ent)->ltime + HostFrameTime()) {
    movetime = thinktime - EVars(ent)->ltime;
    if (movetime < 0) movetime = 0;
  } else
    movetime = HostFrameTime();

  if (movetime) {
    SV_PushMove(ent, movetime);  // advances ent->v.ltime if not blocked
  }

  if (thinktime > oldltime && thinktime <= EVars(ent)->ltime) {
    EVars(ent)->nextthink = 0;
    Set_pr_global_struct_time(SV_Time());
    Set_pr_global_struct_self(ent);
    Set_pr_global_struct_other(0);
    PR_ExecuteProgram(EVars(ent)->think);
    if (EDICT_NUM(ent)->free) return;
  }
}

/*
===============================================================================

CLIENT MOVEMENT

===============================================================================
*/

/*
=============
SV_CheckStuck

This is a big hack to try and fix the rare case of getting stuck in the world
clipping hull.
=============
*/
void SV_CheckStuck(int ent) {
  int i, j;
  int z;
  vec3_t org;

  if (!SV_TestEntityPosition(ent)) {
    VectorCopy(EVars(ent)->origin, EVars(ent)->oldorigin);
    return;
  }

  VectorCopy(EVars(ent)->origin, org);
  VectorCopy(EVars(ent)->oldorigin, EVars(ent)->origin);
  if (!SV_TestEntityPosition(ent)) {
    Con_DPrintf("Unstuck.\n");
    SV_LinkEdict(ent, true);
    return;
  }

  for (z = 0; z < 18; z++)
    for (i = -1; i <= 1; i++)
      for (j = -1; j <= 1; j++) {
        EVars(ent)->origin[0] = org[0] + i;
        EVars(ent)->origin[1] = org[1] + j;
        EVars(ent)->origin[2] = org[2] + z;
        if (!SV_TestEntityPosition(ent)) {
          Con_DPrintf("Unstuck.\n");
          SV_LinkEdict(ent, true);
          return;
        }
      }

  VectorCopy(org, EVars(ent)->origin);
  Con_DPrintf("player is stuck.\n");
}

/*
=============
SV_CheckWater
=============
*/
qboolean SV_CheckWater(int ent) {
  vec3_t point;
  int cont;

  point[0] = EVars(ent)->origin[0];
  point[1] = EVars(ent)->origin[1];
  point[2] = EVars(ent)->origin[2] + EVars(ent)->mins[2] + 1;

  EVars(ent)->waterlevel = 0;
  EVars(ent)->watertype = CONTENTS_EMPTY;
  cont = SV_PointContents(point);
  if (cont <= CONTENTS_WATER) {
    EVars(ent)->watertype = cont;
    EVars(ent)->waterlevel = 1;
    point[2] = EVars(ent)->origin[2] +
               (EVars(ent)->mins[2] + EVars(ent)->maxs[2]) * 0.5;
    cont = SV_PointContents(point);
    if (cont <= CONTENTS_WATER) {
      EVars(ent)->waterlevel = 2;
      point[2] = EVars(ent)->origin[2] + EVars(ent)->view_ofs[2];
      cont = SV_PointContents(point);
      if (cont <= CONTENTS_WATER) EVars(ent)->waterlevel = 3;
    }
  }

  return EVars(ent)->waterlevel > 1;
}

/*
============
SV_WallFriction

============
*/
void SV_WallFriction(int ent, trace_t *trace) {
  vec3_t forward, right, up;
  float d, i;
  vec3_t into, side;

  AngleVectors(EVars(ent)->v_angle, forward, right, up);
  d = DotProduct(trace->plane.normal, forward);

  d += 0.5;
  if (d >= 0) return;

  // cut the tangential velocity
  i = DotProduct(trace->plane.normal, EVars(ent)->velocity);
  VectorScale(trace->plane.normal, i, into);
  VectorSubtract(EVars(ent)->velocity, into, side);

  EVars(ent)->velocity[0] = side[0] * (1 + d);
  EVars(ent)->velocity[1] = side[1] * (1 + d);
}

/*
=====================
SV_TryUnstick

Player has come to a dead stop, possibly due to the problem with limited
float precision at some angle joins in the BSP hull.

Try fixing by pushing one pixel in each direction.

This is a hack, but in the interest of good gameplay...
======================
*/
int SV_TryUnstick(int ent, vec3_t oldvel) {
  int i;
  vec3_t oldorg;
  vec3_t dir;
  int clip;
  trace_t steptrace;

  VectorCopy(EVars(ent)->origin, oldorg);
  VectorCopy(vec3_origin, dir);

  for (i = 0; i < 8; i++) {
    // try pushing a little in an axial direction
    switch (i) {
      case 0:
        dir[0] = 2;
        dir[1] = 0;
        break;
      case 1:
        dir[0] = 0;
        dir[1] = 2;
        break;
      case 2:
        dir[0] = -2;
        dir[1] = 0;
        break;
      case 3:
        dir[0] = 0;
        dir[1] = -2;
        break;
      case 4:
        dir[0] = 2;
        dir[1] = 2;
        break;
      case 5:
        dir[0] = -2;
        dir[1] = 2;
        break;
      case 6:
        dir[0] = 2;
        dir[1] = -2;
        break;
      case 7:
        dir[0] = -2;
        dir[1] = -2;
        break;
    }

    SV_PushEntity(ent, dir);

    // retry the original move
    EVars(ent)->velocity[0] = oldvel[0];
    EVars(ent)->velocity[1] = oldvel[1];
    EVars(ent)->velocity[2] = 0;
    clip = SV_FlyMove(ent, 0.1, &steptrace);

    if (fabs(oldorg[1] - EVars(ent)->origin[1]) > 4 ||
        fabs(oldorg[0] - EVars(ent)->origin[0]) > 4) {
      // Con_DPrintf ("unstuck!\n");
      return clip;
    }

    // go back to the original pos and try again
    VectorCopy(oldorg, EVars(ent)->origin);
  }

  VectorCopy(vec3_origin, EVars(ent)->velocity);
  return 7;  // still not moving
}

/*
=====================
SV_WalkMove

Only used by players
======================
*/
#define STEPSIZE 18
void SV_WalkMove(int ent) {
  vec3_t upmove, downmove;
  vec3_t oldorg, oldvel;
  vec3_t nosteporg, nostepvel;
  int clip;
  int oldonground;
  trace_t steptrace, downtrace;

  //
  // do a regular slide move unless it looks like you ran into a step
  //
  oldonground = (int)EVars(ent)->flags & FL_ONGROUND;
  EVars(ent)->flags = (int)EVars(ent)->flags & ~FL_ONGROUND;

  VectorCopy(EVars(ent)->origin, oldorg);
  VectorCopy(EVars(ent)->velocity, oldvel);

  clip = SV_FlyMove(ent, HostFrameTime(), &steptrace);

  if (!(clip & 2)) return;  // move didn't block on a step

  if (!oldonground && EVars(ent)->waterlevel == 0)
    return;  // don't stair up while jumping

  if (EVars(ent)->movetype != MOVETYPE_WALK) return;  // gibbed by a trigger

  if (Cvar_GetValue(&sv_nostep)) return;

  if ((int)EVars(SV_Player())->flags & FL_WATERJUMP) return;

  VectorCopy(EVars(ent)->origin, nosteporg);
  VectorCopy(EVars(ent)->velocity, nostepvel);

  //
  // try moving up and forward to go up a step
  //
  VectorCopy(oldorg, EVars(ent)->origin);  // back to start pos

  VectorCopy(vec3_origin, upmove);
  VectorCopy(vec3_origin, downmove);
  upmove[2] = STEPSIZE;
  downmove[2] = -STEPSIZE + oldvel[2] * HostFrameTime();

  // move up
  SV_PushEntity(ent, upmove);  // FIXME: don't link?

  // move forward
  EVars(ent)->velocity[0] = oldvel[0];
  EVars(ent)->velocity[1] = oldvel[1];
  EVars(ent)->velocity[2] = 0;
  clip = SV_FlyMove(ent, HostFrameTime(), &steptrace);

  // check for stuckness, possibly due to the limited precision of floats
  // in the clipping hulls
  if (clip) {
    if (fabs(oldorg[1] - EVars(ent)->origin[1]) < 0.03125 &&
        fabs(oldorg[0] - EVars(ent)->origin[0]) <
            0.03125) {  // stepping up didn't make any progress
      clip = SV_TryUnstick(ent, oldvel);
    }
  }

  // extra friction based on view angle
  if (clip & 2) SV_WallFriction(ent, &steptrace);

  // move down
  downtrace = SV_PushEntity(ent, downmove);  // FIXME: don't link?

  if (downtrace.plane.normal[2] > 0.7) {
    if (EVars(ent)->solid == SOLID_BSP) {
      EVars(ent)->flags = (int)EVars(ent)->flags | FL_ONGROUND;
      EVars(ent)->groundentity = downtrace.entn;
    }
  } else {
    // if the push down didn't end up on good ground, use the move without
    // the step up.  This happens near wall / slope combinations, and can
    // cause the player to hop up higher on a slope too steep to climb
    VectorCopy(nosteporg, EVars(ent)->origin);
    VectorCopy(nostepvel, EVars(ent)->velocity);
  }
}

/*
================
SV_Physics_Client

Player character actions
================
*/
void SV_Physics_Client(int ent, int num) {
  if (!GetClientActive(num - 1)) return;  // unconnected slot

  //
  // call standard client pre-think
  //
  Set_pr_global_struct_time(SV_Time());
  Set_pr_global_struct_self(ent);
  PR_ExecuteProgram(Pr_global_struct_PlayerPreThink());

  //
  // do a move
  //
  SV_CheckVelocity(EVars(ent));

  //
  // decide which move function to call
  //
  switch ((int)EVars(ent)->movetype) {
    case MOVETYPE_NONE:
      if (!SV_RunThink(ent)) return;
      break;

    case MOVETYPE_WALK:
      if (!SV_RunThink(ent)) return;
      if (!SV_CheckWater(ent) && !((int)EVars(ent)->flags & FL_WATERJUMP))
        SV_AddGravity(ent);
      SV_CheckStuck(ent);
      SV_WalkMove(ent);
      break;

    case MOVETYPE_TOSS:
    case MOVETYPE_BOUNCE:
      SV_Physics_Toss(ent);
      break;

    case MOVETYPE_FLY:
      if (!SV_RunThink(ent)) return;
      SV_FlyMove(ent, HostFrameTime(), NULL);
      break;

    case MOVETYPE_NOCLIP:
      if (!SV_RunThink(ent)) return;
      VectorMA(EVars(ent)->origin, HostFrameTime(), EVars(ent)->velocity,
               EVars(ent)->origin);
      break;

    default:
      Go_Error_I("SV_Physics_client: bad movetype %v",
                 (int)EVars(ent)->movetype);
  }

  //
  // call standard player post-think
  //
  SV_LinkEdict(ent, true);

  Set_pr_global_struct_time(SV_Time());
  Set_pr_global_struct_self(ent);
  PR_ExecuteProgram(Pr_global_struct_PlayerPostThink());
}

//============================================================================

/*
=============
SV_Physics_None

Non moving objects can only think
=============
*/
void SV_Physics_None(int ent) {
  // regular thinking
  SV_RunThink(ent);
}

/*
=============
SV_Physics_Noclip

A moving object that doesn't obey physics
=============
*/
void SV_Physics_Noclip(int ent) {
  // regular thinking
  if (!SV_RunThink(ent)) return;

  VectorMA(EVars(ent)->angles, HostFrameTime(), EVars(ent)->avelocity,
           EVars(ent)->angles);
  VectorMA(EVars(ent)->origin, HostFrameTime(), EVars(ent)->velocity,
           EVars(ent)->origin);

  SV_LinkEdict(ent, false);
}

/*
==============================================================================

TOSS / BOUNCE

==============================================================================
*/

/*
=============
SV_CheckWaterTransition

=============
*/
void SV_CheckWaterTransition(int ent) {
  int cont;

  cont = SV_PointContents(EVars(ent)->origin);

  if (!EVars(ent)->watertype) {  // just spawned here
    EVars(ent)->watertype = cont;
    EVars(ent)->waterlevel = 1;
    return;
  }

  if (cont <= CONTENTS_WATER) {
    if (EVars(ent)->watertype == CONTENTS_EMPTY) {  // just crossed into water
      SV_StartSound(ent, 0, "misc/h2ohit1.wav", 255, 1);
    }
    EVars(ent)->watertype = cont;
    EVars(ent)->waterlevel = 1;
  } else {
    if (EVars(ent)->watertype != CONTENTS_EMPTY) {  // just crossed into water
      SV_StartSound(ent, 0, "misc/h2ohit1.wav", 255, 1);
    }
    EVars(ent)->watertype = CONTENTS_EMPTY;
    EVars(ent)->waterlevel = cont;
  }
}

/*
=============
SV_Physics_Toss

Toss, bounce, and fly movement.  When onground, do nothing.
=============
*/
void SV_Physics_Toss(int ent) {
  trace_t trace;
  vec3_t move;
  float backoff;

  // regular thinking
  if (!SV_RunThink(ent)) return;

  // if onground, return without moving
  if (((int)EVars(ent)->flags & FL_ONGROUND)) return;

  SV_CheckVelocity(EVars(ent));

  // add gravity
  if (EVars(ent)->movetype != MOVETYPE_FLY &&
      EVars(ent)->movetype != MOVETYPE_FLYMISSILE)
    SV_AddGravity(ent);

  // move angles
  VectorMA(EVars(ent)->angles, HostFrameTime(), EVars(ent)->avelocity,
           EVars(ent)->angles);

  // move origin
  VectorScale(EVars(ent)->velocity, HostFrameTime(), move);
  trace = SV_PushEntity(ent, move);
  if (trace.fraction == 1) return;
  if (EDICT_NUM(ent)->free) return;

  if (EVars(ent)->movetype == MOVETYPE_BOUNCE)
    backoff = 1.5;
  else
    backoff = 1;

  ClipVelocity(EVars(ent)->velocity, trace.plane.normal, EVars(ent)->velocity,
               backoff);

  // stop if on ground
  if (trace.plane.normal[2] > 0.7) {
    if (EVars(ent)->velocity[2] < 60 ||
        EVars(ent)->movetype != MOVETYPE_BOUNCE) {
      EVars(ent)->flags = (int)EVars(ent)->flags | FL_ONGROUND;
      EVars(ent)->groundentity = trace.entn;
      VectorCopy(vec3_origin, EVars(ent)->velocity);
      VectorCopy(vec3_origin, EVars(ent)->avelocity);
    }
  }

  // check for in water
  SV_CheckWaterTransition(ent);
}

/*
===============================================================================

STEPPING MOVEMENT

===============================================================================
*/

/*
=============
SV_Physics_Step

Monsters freefall when they don't have a ground entity, otherwise
all movement is done with discrete steps.

This is also used for objects that have become still on the ground, but
will fall if the floor is pulled out from under them.
=============
*/
void SV_Physics_Step(int ent) {
  qboolean hitsound;

  // freefall if not onground
  if (!((int)EVars(ent)->flags & (FL_ONGROUND | FL_FLY | FL_SWIM))) {
    if (EVars(ent)->velocity[2] < Cvar_GetValue(&sv_gravity) * -0.1)
      hitsound = true;
    else
      hitsound = false;

    SV_AddGravity(ent);
    SV_CheckVelocity(EVars(ent));
    SV_FlyMove(ent, HostFrameTime(), NULL);
    SV_LinkEdict(ent, true);

    if ((int)EVars(ent)->flags & FL_ONGROUND)  // just hit ground
    {
      if (hitsound) SV_StartSound(ent, 0, "demon/dland2.wav", 255, 1);
    }
  }

  // regular thinking
  SV_RunThink(ent);

  SV_CheckWaterTransition(ent);
}

//============================================================================

/*
================
SV_Physics

================
*/
void SV_Physics(void) {
  int i;
  int entity_cap;  // For sv_freezenonclients
  int ent;

  // let the progs know that a new frame has started
  Set_pr_global_struct_self(0);
  Set_pr_global_struct_other(0);
  Set_pr_global_struct_time(SV_Time());
  PR_ExecuteProgram(Pr_global_struct_StartFrame());

  // SV_CheckAllEnts ();

  //
  // treat each object in turn
  //
  ent = 0;

  if (Cvar_GetValue(&sv_freezenonclients))
    entity_cap =
        SVS_GetMaxClients() + 1;  // Only run physics on clients and the world
  else
    entity_cap = SV_NumEdicts();

  for (i = 0; i < entity_cap; i++, ent++) {
    if (EDICT_NUM(ent)->free) continue;

    if (Pr_global_struct_force_retouch()) {
      SV_LinkEdict(ent,
                   true);  // force retouch even for stationary
    }

    if (i > 0 && i <= SVS_GetMaxClients())
      SV_Physics_Client(ent, i);
    else if (EVars(ent)->movetype == MOVETYPE_PUSH)
      SV_Physics_Pusher(ent);
    else if (EVars(ent)->movetype == MOVETYPE_NONE)
      SV_Physics_None(ent);
    else if (EVars(ent)->movetype == MOVETYPE_NOCLIP)
      SV_Physics_Noclip(ent);
    else if (EVars(ent)->movetype == MOVETYPE_STEP)
      SV_Physics_Step(ent);
    else if (EVars(ent)->movetype == MOVETYPE_TOSS ||
             EVars(ent)->movetype == MOVETYPE_BOUNCE ||
             EVars(ent)->movetype == MOVETYPE_FLY ||
             EVars(ent)->movetype == MOVETYPE_FLYMISSILE)
      SV_Physics_Toss(ent);
    else
      Go_Error_I("SV_Physics: bad movetype %v", (int)EVars(ent)->movetype);
  }

  if (Pr_global_struct_force_retouch()) Dec_pr_global_struct_force_retouch();

  if (!Cvar_GetValue(&sv_freezenonclients)) {
    SV_SetTime(SV_Time() + HostFrameTime());
  }
}
