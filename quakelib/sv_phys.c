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

void SV_Physics_Toss(edict_t *ent);

/*
================
SV_CheckAllEnts
================
*/
void SV_CheckAllEnts(void) {
  int e;
  edict_t *check;

  // see if any solid entities are inside the final position
  check = NEXT_EDICT(sv.edicts);
  for (e = 1; e < SV_NumEdicts(); e++, check = NEXT_EDICT(check)) {
    if (check->free) continue;
    if (EdictV(check)->movetype == MOVETYPE_PUSH ||
        EdictV(check)->movetype == MOVETYPE_NONE ||
        EdictV(check)->movetype == MOVETYPE_NOCLIP)
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
void SV_CheckVelocity(edict_t *ent) {
  int i;

  //
  // bound velocity
  //
  for (i = 0; i < 3; i++) {
    if (IS_NAN(EdictV(ent)->velocity[i])) {
      Con_Printf("Got a NaN velocity on %s\n", PR_GetString(EdictV(ent)->classname));
      EdictV(ent)->velocity[i] = 0;
    }
    if (IS_NAN(EdictV(ent)->origin[i])) {
      Con_Printf("Got a NaN origin on %s\n", PR_GetString(EdictV(ent)->classname));
      EdictV(ent)->origin[i] = 0;
    }
    if (EdictV(ent)->velocity[i] > Cvar_GetValue(&sv_maxvelocity))
      EdictV(ent)->velocity[i] = Cvar_GetValue(&sv_maxvelocity);
    else if (EdictV(ent)->velocity[i] < -Cvar_GetValue(&sv_maxvelocity))
      EdictV(ent)->velocity[i] = -Cvar_GetValue(&sv_maxvelocity);
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
qboolean SV_RunThink(edict_t *ent) {
  float thinktime;
  float oldframe;  // johnfitz
  int i;           // johnfitz

  thinktime = EdictV(ent)->nextthink;
  if (thinktime <= 0 || thinktime > SV_Time() + HostFrameTime()) return true;

  if (thinktime < SV_Time())
    thinktime = SV_Time();  // don't let things stay in the past.
                          // it is possible to start that way
                          // by a trigger with a local time.

  oldframe = EdictV(ent)->frame;  // johnfitz

  EdictV(ent)->nextthink = 0;
  pr_global_struct->time = thinktime;
  pr_global_struct->self = NUM_FOR_EDICT(ent);
  pr_global_struct->other = NUM_FOR_EDICT(sv.edicts);
  PR_ExecuteProgram(EdictV(ent)->think);

  // johnfitz -- PROTOCOL_FITZQUAKE
  // capture interval to nextthink here and send it to client for better
  // lerp timing, but only if interval is not 0.1 (which client assumes)
  ent->sendinterval = false;
  if (!ent->free && EdictV(ent)->nextthink &&
      (EdictV(ent)->movetype == MOVETYPE_STEP || EdictV(ent)->frame != oldframe)) {
    i = Q_rint((EdictV(ent)->nextthink - thinktime) * 255);
    if (i >= 0 && i < 256 && i != 25 &&
        i != 26)  // 25 and 26 are close enough to 0.1 to not send
      ent->sendinterval = true;
  }
  // johnfitz

  return !ent->free;
}

/*
==================
SV_Impact

Two entities have touched, so run their touch functions
==================
*/
void SV_Impact(edict_t *e1, edict_t *e2) {
  int old_self, old_other;

  old_self = pr_global_struct->self;
  old_other = pr_global_struct->other;

  pr_global_struct->time = SV_Time();
  if (EdictV(e1)->touch && EdictV(e1)->solid != SOLID_NOT) {
    pr_global_struct->self = NUM_FOR_EDICT(e1);
    pr_global_struct->other = NUM_FOR_EDICT(e2);
    PR_ExecuteProgram(EdictV(e1)->touch);
  }

  if (EdictV(e2)->touch && EdictV(e2)->solid != SOLID_NOT) {
    pr_global_struct->self = NUM_FOR_EDICT(e2);
    pr_global_struct->other = NUM_FOR_EDICT(e1);
    PR_ExecuteProgram(EdictV(e2)->touch);
  }

  pr_global_struct->self = old_self;
  pr_global_struct->other = old_other;
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
int SV_FlyMove(edict_t *ent, float time, trace_t *steptrace) {
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
  VectorCopy(EdictV(ent)->velocity, original_velocity);
  VectorCopy(EdictV(ent)->velocity, primal_velocity);
  numplanes = 0;

  time_left = time;

  for (bumpcount = 0; bumpcount < numbumps; bumpcount++) {
    if (!EdictV(ent)->velocity[0] &&
        !EdictV(ent)->velocity[1] &&
        !EdictV(ent)->velocity[2])
      break;

    for (i = 0; i < 3; i++)
      end[i] = EdictV(ent)->origin[i] + time_left * EdictV(ent)->velocity[i];

    trace = SV_Move(EdictV(ent)->origin, EdictV(ent)->mins, EdictV(ent)->maxs, end, false, ent);

    if (trace.allsolid) {  // entity is trapped in another solid
      VectorCopy(vec3_origin, EdictV(ent)->velocity);
      return 3;
    }

    if (trace.fraction > 0) {  // actually covered some distance
      VectorCopy(trace.endpos, EdictV(ent)->origin);
      VectorCopy(EdictV(ent)->velocity, original_velocity);
      numplanes = 0;
    }

    if (trace.fraction == 1) break;  // moved the entire distance

    if (!trace.ent) Go_Error("SV_FlyMove: !trace.ent");

    if (trace.plane.normal[2] > 0.7) {
      blocked |= 1;  // floor
      if (EdictV(trace.ent)->solid == SOLID_BSP) {
        EdictV(ent)->flags = (int)EdictV(ent)->flags | FL_ONGROUND;
        EdictV(ent)->groundentity = NUM_FOR_EDICT(trace.ent);
      }
    }
    if (!trace.plane.normal[2]) {
      blocked |= 2;                       // step
      if (steptrace) *steptrace = trace;  // save for player extrafriction
    }

    //
    // run the impact function
    //
    SV_Impact(ent, trace.ent);
    if (ent->free) break;  // removed by the impact function

    time_left -= time_left * trace.fraction;

    // cliped to another plane
    if (numplanes >= MAX_CLIP_PLANES) {  // this shouldn't really happen
      VectorCopy(vec3_origin, EdictV(ent)->velocity);
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
      VectorCopy(new_velocity, EdictV(ent)->velocity);
    } else {  // go along the crease
      if (numplanes != 2) {
        //				Con_Printf ("clip velocity, numplanes ==
        //%i\n",numplanes);
        VectorCopy(vec3_origin, EdictV(ent)->velocity);
        return 7;
      }
      CrossProduct(planes[0], planes[1], dir);
      d = DotProduct(dir, EdictV(ent)->velocity);
      VectorScale(dir, d, EdictV(ent)->velocity);
    }

    //
    // if original velocity is against the original velocity, stop dead
    // to avoid tiny occilations in sloping corners
    //
    if (DotProduct(EdictV(ent)->velocity, primal_velocity) <= 0) {
      VectorCopy(vec3_origin, EdictV(ent)->velocity);
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
void SV_AddGravity(edict_t *ent) {
  float ent_gravity;
  eval_t *val;

  val = GetEdictFieldValue(EdictV(ent), "gravity");
  if (val && val->_float)
    ent_gravity = val->_float;
  else
    ent_gravity = 1.0;

  EdictV(ent)->velocity[2] -=
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
trace_t SV_PushEntity(edict_t *ent, vec3_t push) {
  trace_t trace;
  vec3_t end;

  VectorAdd(EdictV(ent)->origin, push, end);

  if (EdictV(ent)->movetype == MOVETYPE_FLYMISSILE)
    trace = SV_Move(EdictV(ent)->origin, 
        EdictV(ent)->mins, EdictV(ent)->maxs, end, MOVE_MISSILE, ent);
  else if (EdictV(ent)->solid == SOLID_TRIGGER || EdictV(ent)->solid == SOLID_NOT)
    // only clip against bmodels
    trace = SV_Move(EdictV(ent)->origin, EdictV(ent)->mins, EdictV(ent)->maxs, end,
                    MOVE_NOMONSTERS, ent);
  else
    trace =
        SV_Move(EdictV(ent)->origin, EdictV(ent)->mins, EdictV(ent)->maxs, end, MOVE_NORMAL, ent);

  VectorCopy(trace.endpos, EdictV(ent)->origin);
  SV_LinkEdict(ent, true);

  if (trace.ent) SV_Impact(ent, trace.ent);

  return trace;
}

/*
============
SV_PushMove
============
*/
void SV_PushMove(edict_t *pusher, float movetime) {
  int i, e;
  edict_t *check, *block;
  vec3_t mins, maxs, move;
  vec3_t entorig, pushorig;
  int num_moved;
  edict_t **moved_edict;  // johnfitz -- dynamically allocate
  vec3_t *moved_from;     // johnfitz -- dynamically allocate
  int mark;               // johnfitz

  if (!EdictV(pusher)->velocity[0] && !EdictV(pusher)->velocity[1] &&
      !EdictV(pusher)->velocity[2]) {
    EdictV(pusher)->ltime += movetime;
    return;
  }

  for (i = 0; i < 3; i++) {
    move[i] = EdictV(pusher)->velocity[i] * movetime;
    mins[i] = EdictV(pusher)->absmin[i] + move[i];
    maxs[i] = EdictV(pusher)->absmax[i] + move[i];
  }

  VectorCopy(EdictV(pusher)->origin, pushorig);

  // move the pusher to it's final position

  VectorAdd(EdictV(pusher)->origin, move, EdictV(pusher)->origin);
  EdictV(pusher)->ltime += movetime;
  SV_LinkEdict(pusher, false);

  // johnfitz -- dynamically allocate
  mark = Hunk_LowMark();
  moved_edict = (edict_t **)Hunk_Alloc(SV_NumEdicts() * sizeof(edict_t *));
  moved_from = (vec3_t *)Hunk_Alloc(SV_NumEdicts() * sizeof(vec3_t));
  // johnfitz

  // see if any solid entities are inside the final position
  num_moved = 0;
  check = NEXT_EDICT(sv.edicts);
  for (e = 1; e < SV_NumEdicts(); e++, check = NEXT_EDICT(check)) {
    if (check->free) continue;
    if (EdictV(check)->movetype == MOVETYPE_PUSH ||
        EdictV(check)->movetype == MOVETYPE_NONE ||
        EdictV(check)->movetype == MOVETYPE_NOCLIP)
      continue;

    // if the entity is standing on the pusher, it will definately be moved
    if (!(((int)EdictV(check)->flags & FL_ONGROUND) &&
          EDICT_NUM(EdictV(check)->groundentity) == pusher)) {
      if (EdictV(check)->absmin[0] >= maxs[0] || EdictV(check)->absmin[1] >= maxs[1] ||
          EdictV(check)->absmin[2] >= maxs[2] || EdictV(check)->absmax[0] <= mins[0] ||
          EdictV(check)->absmax[1] <= mins[1] || EdictV(check)->absmax[2] <= mins[2])
        continue;

      // see if the ent's bbox is inside the pusher's final position
      if (!SV_TestEntityPosition(check)) continue;
    }

    // remove the onground flag for non-players
    if (EdictV(check)->movetype != MOVETYPE_WALK)
      EdictV(check)->flags = (int)EdictV(check)->flags & ~FL_ONGROUND;

    VectorCopy(EdictV(check)->origin, entorig);
    VectorCopy(EdictV(check)->origin, moved_from[num_moved]);
    moved_edict[num_moved] = check;
    num_moved++;

    // try moving the contacted entity
    EdictV(pusher)->solid = SOLID_NOT;
    SV_PushEntity(check, move);
    EdictV(pusher)->solid = SOLID_BSP;

    // if it is still inside the pusher, block
    block = SV_TestEntityPosition(check);
    if (block) {  // fail the move
      if (EdictV(check)->mins[0] == EdictV(check)->maxs[0]) continue;
      if (EdictV(check)->solid == SOLID_NOT ||
          EdictV(check)->solid == SOLID_TRIGGER) {  // corpse
        EdictV(check)->mins[0] = EdictV(check)->mins[1] = 0;
        VectorCopy(EdictV(check)->mins, EdictV(check)->maxs);
        continue;
      }

      VectorCopy(entorig, EdictV(check)->origin);
      SV_LinkEdict(check, true);

      VectorCopy(pushorig, EdictV(pusher)->origin);
      SV_LinkEdict(pusher, false);
      EdictV(pusher)->ltime -= movetime;

      // if the pusher has a "blocked" function, call it
      // otherwise, just stay in place until the obstacle is gone
      if (EdictV(pusher)->blocked) {
        pr_global_struct->self = NUM_FOR_EDICT(pusher);
        pr_global_struct->other = NUM_FOR_EDICT(check);
        PR_ExecuteProgram(EdictV(pusher)->blocked);
      }

      // move back any entities we already moved
      for (i = 0; i < num_moved; i++) {
        VectorCopy(moved_from[i], EdictV(moved_edict[i])->origin);
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
void SV_Physics_Pusher(edict_t *ent) {
  float thinktime;
  float oldltime;
  float movetime;

  oldltime = EdictV(ent)->ltime;

  thinktime = EdictV(ent)->nextthink;
  if (thinktime < EdictV(ent)->ltime + HostFrameTime()) {
    movetime = thinktime - EdictV(ent)->ltime;
    if (movetime < 0) movetime = 0;
  } else
    movetime = HostFrameTime();

  if (movetime) {
    SV_PushMove(ent, movetime);  // advances ent->v.ltime if not blocked
  }

  if (thinktime > oldltime && thinktime <= EdictV(ent)->ltime) {
    EdictV(ent)->nextthink = 0;
    pr_global_struct->time = SV_Time();
    pr_global_struct->self = NUM_FOR_EDICT(ent);
    pr_global_struct->other = NUM_FOR_EDICT(sv.edicts);
    PR_ExecuteProgram(EdictV(ent)->think);
    if (ent->free) return;
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
void SV_CheckStuck(edict_t *ent) {
  int i, j;
  int z;
  vec3_t org;

  if (!SV_TestEntityPosition(ent)) {
    VectorCopy(EdictV(ent)->origin, EdictV(ent)->oldorigin);
    return;
  }

  VectorCopy(EdictV(ent)->origin, org);
  VectorCopy(EdictV(ent)->oldorigin, EdictV(ent)->origin);
  if (!SV_TestEntityPosition(ent)) {
    Con_DPrintf("Unstuck.\n");
    SV_LinkEdict(ent, true);
    return;
  }

  for (z = 0; z < 18; z++)
    for (i = -1; i <= 1; i++)
      for (j = -1; j <= 1; j++) {
        EdictV(ent)->origin[0] = org[0] + i;
        EdictV(ent)->origin[1] = org[1] + j;
        EdictV(ent)->origin[2] = org[2] + z;
        if (!SV_TestEntityPosition(ent)) {
          Con_DPrintf("Unstuck.\n");
          SV_LinkEdict(ent, true);
          return;
        }
      }

  VectorCopy(org, EdictV(ent)->origin);
  Con_DPrintf("player is stuck.\n");
}

/*
=============
SV_CheckWater
=============
*/
qboolean SV_CheckWater(edict_t *ent) {
  vec3_t point;
  int cont;

  point[0] = EdictV(ent)->origin[0];
  point[1] = EdictV(ent)->origin[1];
  point[2] = EdictV(ent)->origin[2] + EdictV(ent)->mins[2] + 1;

  EdictV(ent)->waterlevel = 0;
  EdictV(ent)->watertype = CONTENTS_EMPTY;
  cont = SV_PointContents(point);
  if (cont <= CONTENTS_WATER) {
    EdictV(ent)->watertype = cont;
    EdictV(ent)->waterlevel = 1;
    point[2] = EdictV(ent)->origin[2] + 
      (EdictV(ent)->mins[2] + EdictV(ent)->maxs[2]) * 0.5;
    cont = SV_PointContents(point);
    if (cont <= CONTENTS_WATER) {
      EdictV(ent)->waterlevel = 2;
      point[2] = EdictV(ent)->origin[2] + EdictV(ent)->view_ofs[2];
      cont = SV_PointContents(point);
      if (cont <= CONTENTS_WATER) EdictV(ent)->waterlevel = 3;
    }
  }

  return EdictV(ent)->waterlevel > 1;
}

/*
============
SV_WallFriction

============
*/
void SV_WallFriction(edict_t *ent, trace_t *trace) {
  vec3_t forward, right, up;
  float d, i;
  vec3_t into, side;

  AngleVectors(EdictV(ent)->v_angle, forward, right, up);
  d = DotProduct(trace->plane.normal, forward);

  d += 0.5;
  if (d >= 0) return;

  // cut the tangential velocity
  i = DotProduct(trace->plane.normal, EdictV(ent)->velocity);
  VectorScale(trace->plane.normal, i, into);
  VectorSubtract(EdictV(ent)->velocity, into, side);

  EdictV(ent)->velocity[0] = side[0] * (1 + d);
  EdictV(ent)->velocity[1] = side[1] * (1 + d);
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
int SV_TryUnstick(edict_t *ent, vec3_t oldvel) {
  int i;
  vec3_t oldorg;
  vec3_t dir;
  int clip;
  trace_t steptrace;

  VectorCopy(EdictV(ent)->origin, oldorg);
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
    EdictV(ent)->velocity[0] = oldvel[0];
    EdictV(ent)->velocity[1] = oldvel[1];
    EdictV(ent)->velocity[2] = 0;
    clip = SV_FlyMove(ent, 0.1, &steptrace);

    if (fabs(oldorg[1] - EdictV(ent)->origin[1]) > 4 ||
        fabs(oldorg[0] - EdictV(ent)->origin[0]) > 4) {
      // Con_DPrintf ("unstuck!\n");
      return clip;
    }

    // go back to the original pos and try again
    VectorCopy(oldorg, EdictV(ent)->origin);
  }

  VectorCopy(vec3_origin, EdictV(ent)->velocity);
  return 7;  // still not moving
}

/*
=====================
SV_WalkMove

Only used by players
======================
*/
#define STEPSIZE 18
void SV_WalkMove(edict_t *ent) {
  vec3_t upmove, downmove;
  vec3_t oldorg, oldvel;
  vec3_t nosteporg, nostepvel;
  int clip;
  int oldonground;
  trace_t steptrace, downtrace;

  //
  // do a regular slide move unless it looks like you ran into a step
  //
  oldonground = (int)EdictV(ent)->flags & FL_ONGROUND;
  EdictV(ent)->flags = (int)EdictV(ent)->flags & ~FL_ONGROUND;

  VectorCopy(EdictV(ent)->origin, oldorg);
  VectorCopy(EdictV(ent)->velocity, oldvel);

  clip = SV_FlyMove(ent, HostFrameTime(), &steptrace);

  if (!(clip & 2)) return;  // move didn't block on a step

  if (!oldonground && EdictV(ent)->waterlevel == 0)
    return;  // don't stair up while jumping

  if (EdictV(ent)->movetype != MOVETYPE_WALK) return;  // gibbed by a trigger

  if (Cvar_GetValue(&sv_nostep)) return;

  if ((int)EdictV(sv_player)->flags & FL_WATERJUMP) return;

  VectorCopy(EdictV(ent)->origin, nosteporg);
  VectorCopy(EdictV(ent)->velocity, nostepvel);

  //
  // try moving up and forward to go up a step
  //
  VectorCopy(oldorg, EdictV(ent)->origin);  // back to start pos

  VectorCopy(vec3_origin, upmove);
  VectorCopy(vec3_origin, downmove);
  upmove[2] = STEPSIZE;
  downmove[2] = -STEPSIZE + oldvel[2] * HostFrameTime();

  // move up
  SV_PushEntity(ent, upmove);  // FIXME: don't link?

  // move forward
  EdictV(ent)->velocity[0] = oldvel[0];
  EdictV(ent)->velocity[1] = oldvel[1];
  EdictV(ent)->velocity[2] = 0;
  clip = SV_FlyMove(ent, HostFrameTime(), &steptrace);

  // check for stuckness, possibly due to the limited precision of floats
  // in the clipping hulls
  if (clip) {
    if (fabs(oldorg[1] - EdictV(ent)->origin[1]) < 0.03125 &&
        fabs(oldorg[0] - EdictV(ent)->origin[0]) <
            0.03125) {  // stepping up didn't make any progress
      clip = SV_TryUnstick(ent, oldvel);
    }
  }

  // extra friction based on view angle
  if (clip & 2) SV_WallFriction(ent, &steptrace);

  // move down
  downtrace = SV_PushEntity(ent, downmove);  // FIXME: don't link?

  if (downtrace.plane.normal[2] > 0.7) {
    if (EdictV(ent)->solid == SOLID_BSP) {
      EdictV(ent)->flags = (int)EdictV(ent)->flags | FL_ONGROUND;
      EdictV(ent)->groundentity = NUM_FOR_EDICT(downtrace.ent);
    }
  } else {
    // if the push down didn't end up on good ground, use the move without
    // the step up.  This happens near wall / slope combinations, and can
    // cause the player to hop up higher on a slope too steep to climb
    VectorCopy(nosteporg, EdictV(ent)->origin);
    VectorCopy(nostepvel, EdictV(ent)->velocity);
  }
}

/*
================
SV_Physics_Client

Player character actions
================
*/
void SV_Physics_Client(edict_t *ent, int num) {
  if (!GetClientActive(num - 1)) return;  // unconnected slot

  //
  // call standard client pre-think
  //
  pr_global_struct->time = SV_Time();
  pr_global_struct->self = NUM_FOR_EDICT(ent);
  PR_ExecuteProgram(pr_global_struct->PlayerPreThink);

  //
  // do a move
  //
  SV_CheckVelocity(ent);

  //
  // decide which move function to call
  //
  switch ((int)EdictV(ent)->movetype) {
    case MOVETYPE_NONE:
      if (!SV_RunThink(ent)) return;
      break;

    case MOVETYPE_WALK:
      if (!SV_RunThink(ent)) return;
      if (!SV_CheckWater(ent) && !((int)EdictV(ent)->flags & FL_WATERJUMP))
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
      VectorMA(EdictV(ent)->origin, HostFrameTime(), 
          EdictV(ent)->velocity, EdictV(ent)->origin);
      break;

    default:
      Go_Error_I("SV_Physics_client: bad movetype %v", (int)EdictV(ent)->movetype);
  }

  //
  // call standard player post-think
  //
  SV_LinkEdict(ent, true);

  pr_global_struct->time = SV_Time();
  pr_global_struct->self = NUM_FOR_EDICT(ent);
  PR_ExecuteProgram(pr_global_struct->PlayerPostThink);
}

//============================================================================

/*
=============
SV_Physics_None

Non moving objects can only think
=============
*/
void SV_Physics_None(edict_t *ent) {
  // regular thinking
  SV_RunThink(ent);
}

/*
=============
SV_Physics_Noclip

A moving object that doesn't obey physics
=============
*/
void SV_Physics_Noclip(edict_t *ent) {
  // regular thinking
  if (!SV_RunThink(ent)) return;

  VectorMA(EdictV(ent)->angles, HostFrameTime(), 
      EdictV(ent)->avelocity, EdictV(ent)->angles);
  VectorMA(EdictV(ent)->origin, HostFrameTime(), 
      EdictV(ent)->velocity, EdictV(ent)->origin);

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
void SV_CheckWaterTransition(edict_t *ent) {
  int cont;

  cont = SV_PointContents(EdictV(ent)->origin);

  if (!EdictV(ent)->watertype) {  // just spawned here
    EdictV(ent)->watertype = cont;
    EdictV(ent)->waterlevel = 1;
    return;
  }

  if (cont <= CONTENTS_WATER) {
    if (EdictV(ent)->watertype == CONTENTS_EMPTY) {  // just crossed into water
      SV_StartSound(ent, 0, "misc/h2ohit1.wav", 255, 1);
    }
    EdictV(ent)->watertype = cont;
    EdictV(ent)->waterlevel = 1;
  } else {
    if (EdictV(ent)->watertype != CONTENTS_EMPTY) {  // just crossed into water
      SV_StartSound(ent, 0, "misc/h2ohit1.wav", 255, 1);
    }
    EdictV(ent)->watertype = CONTENTS_EMPTY;
    EdictV(ent)->waterlevel = cont;
  }
}

/*
=============
SV_Physics_Toss

Toss, bounce, and fly movement.  When onground, do nothing.
=============
*/
void SV_Physics_Toss(edict_t *ent) {
  trace_t trace;
  vec3_t move;
  float backoff;

  // regular thinking
  if (!SV_RunThink(ent)) return;

  // if onground, return without moving
  if (((int)EdictV(ent)->flags & FL_ONGROUND)) return;

  SV_CheckVelocity(ent);

  // add gravity
  if (EdictV(ent)->movetype != MOVETYPE_FLY && 
      EdictV(ent)->movetype != MOVETYPE_FLYMISSILE)
    SV_AddGravity(ent);

  // move angles
  VectorMA(EdictV(ent)->angles, HostFrameTime(), 
      EdictV(ent)->avelocity, EdictV(ent)->angles);

  // move origin
  VectorScale(EdictV(ent)->velocity, HostFrameTime(), move);
  trace = SV_PushEntity(ent, move);
  if (trace.fraction == 1) return;
  if (ent->free) return;

  if (EdictV(ent)->movetype == MOVETYPE_BOUNCE)
    backoff = 1.5;
  else
    backoff = 1;

  ClipVelocity(EdictV(ent)->velocity, trace.plane.normal, 
      EdictV(ent)->velocity, backoff);

  // stop if on ground
  if (trace.plane.normal[2] > 0.7) {
    if (EdictV(ent)->velocity[2] < 60 || EdictV(ent)->movetype != MOVETYPE_BOUNCE) {
      EdictV(ent)->flags = (int)EdictV(ent)->flags | FL_ONGROUND;
      EdictV(ent)->groundentity = NUM_FOR_EDICT(trace.ent);
      VectorCopy(vec3_origin, EdictV(ent)->velocity);
      VectorCopy(vec3_origin, EdictV(ent)->avelocity);
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
void SV_Physics_Step(edict_t *ent) {
  qboolean hitsound;

  // freefall if not onground
  if (!((int)EdictV(ent)->flags & (FL_ONGROUND | FL_FLY | FL_SWIM))) {
    if (EdictV(ent)->velocity[2] < Cvar_GetValue(&sv_gravity) * -0.1)
      hitsound = true;
    else
      hitsound = false;

    SV_AddGravity(ent);
    SV_CheckVelocity(ent);
    SV_FlyMove(ent, HostFrameTime(), NULL);
    SV_LinkEdict(ent, true);

    if ((int)EdictV(ent)->flags & FL_ONGROUND)  // just hit ground
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
  edict_t *ent;

  // let the progs know that a new frame has started
  pr_global_struct->self = 0;
  pr_global_struct->other = 0;
  pr_global_struct->time = SV_Time();
  PR_ExecuteProgram(pr_global_struct->StartFrame);

  // SV_CheckAllEnts ();

  //
  // treat each object in turn
  //
  ent = sv.edicts;

  if (Cvar_GetValue(&sv_freezenonclients))
    entity_cap =
        SVS_GetMaxClients() + 1;  // Only run physics on clients and the world
  else
    entity_cap = SV_NumEdicts();

  // for (i=0 ; i<sv.num_edicts ; i++, ent = NEXT_EDICT(ent))
  for (i = 0; i < entity_cap; i++, ent = NEXT_EDICT(ent)) {
    if (ent->free) continue;

    if (pr_global_struct->force_retouch) {
      SV_LinkEdict(ent, true);  // force retouch even for stationary
    }

    if (i > 0 && i <= SVS_GetMaxClients())
      SV_Physics_Client(ent, i);
    else if (EdictV(ent)->movetype == MOVETYPE_PUSH)
      SV_Physics_Pusher(ent);
    else if (EdictV(ent)->movetype == MOVETYPE_NONE)
      SV_Physics_None(ent);
    else if (EdictV(ent)->movetype == MOVETYPE_NOCLIP)
      SV_Physics_Noclip(ent);
    else if (EdictV(ent)->movetype == MOVETYPE_STEP)
      SV_Physics_Step(ent);
    else if (EdictV(ent)->movetype == MOVETYPE_TOSS ||
             EdictV(ent)->movetype == MOVETYPE_BOUNCE ||
             EdictV(ent)->movetype == MOVETYPE_FLY ||
             EdictV(ent)->movetype == MOVETYPE_FLYMISSILE)
      SV_Physics_Toss(ent);
    else
      Go_Error_I("SV_Physics: bad movetype %v", (int)EdictV(ent)->movetype);
  }

  if (pr_global_struct->force_retouch) pr_global_struct->force_retouch--;

  if (!Cvar_GetValue(&sv_freezenonclients)) {
    SV_SetTime(SV_Time() + HostFrameTime());
  }
}
