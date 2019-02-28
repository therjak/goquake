// sv_move.c -- monster movement

#include "quakedef.h"

#define STEPSIZE 18

/*
=============
SV_CheckBottom

Returns false if any part of the bottom of the entity is off an edge that
is not a staircase.

=============
*/
int c_yes, c_no;

qboolean SV_CheckBottom(edict_t *ent) {
  vec3_t mins, maxs, start, stop;
  trace_t trace;
  int x, y;
  float mid, bottom;

  VectorAdd(EdictV(ent)->origin, EdictV(ent)->mins, mins);
  VectorAdd(EdictV(ent)->origin, EdictV(ent)->maxs, maxs);

  // if all of the points under the corners are solid world, don't bother
  // with the tougher checks
  // the corners must be within 16 of the midpoint
  start[2] = mins[2] - 1;
  for (x = 0; x <= 1; x++)
    for (y = 0; y <= 1; y++) {
      start[0] = x ? maxs[0] : mins[0];
      start[1] = y ? maxs[1] : mins[1];
      if (SV_PointContents(start) != CONTENTS_SOLID) goto realcheck;
    }

  c_yes++;
  return true;  // we got out easy

realcheck:
  c_no++;
  //
  // check it for real...
  //
  start[2] = mins[2];

  // the midpoint must be within 16 of the bottom
  start[0] = stop[0] = (mins[0] + maxs[0]) * 0.5;
  start[1] = stop[1] = (mins[1] + maxs[1]) * 0.5;
  stop[2] = start[2] - 2 * STEPSIZE;
  trace =
      SV_Move(start, vec3_origin, vec3_origin, stop, true, NUM_FOR_EDICT(ent));

  if (trace.fraction == 1.0) return false;
  mid = bottom = trace.endpos[2];

  // the corners must be within 16 of the midpoint
  for (x = 0; x <= 1; x++)
    for (y = 0; y <= 1; y++) {
      start[0] = stop[0] = x ? maxs[0] : mins[0];
      start[1] = stop[1] = y ? maxs[1] : mins[1];

      trace = SV_Move(start, vec3_origin, vec3_origin, stop, true,
                      NUM_FOR_EDICT(ent));

      if (trace.fraction != 1.0 && trace.endpos[2] > bottom)
        bottom = trace.endpos[2];
      if (trace.fraction == 1.0 || mid - trace.endpos[2] > STEPSIZE)
        return false;
    }

  c_yes++;
  return true;
}

/*
=============
SV_movestep

Called by monster program code.
The move will be adjusted for slopes and stairs, but if the move isn't
possible, no move is done, false is returned, and
pr_global_struct->trace_normal is set to the normal of the blocking wall
=============
*/
qboolean SV_movestep(edict_t *ent, vec3_t move, qboolean relink) {
  float dz;
  vec3_t oldorg, neworg, end;
  trace_t trace;
  int i;
  edict_t *enemy;

  // try the move
  VectorCopy(EdictV(ent)->origin, oldorg);
  VectorAdd(EdictV(ent)->origin, move, neworg);

  // flying monsters don't step up
  if ((int)EdictV(ent)->flags & (FL_SWIM | FL_FLY)) {
    // try one move with vertical motion, then one without
    for (i = 0; i < 2; i++) {
      VectorAdd(EdictV(ent)->origin, move, neworg);
      enemy = EDICT_NUM(EdictV(ent)->enemy);
      if (i == 0 && enemy != sv.edicts) {
        dz = EdictV(ent)->origin[2] -
             EdictV(EDICT_NUM(EdictV(ent)->enemy))->origin[2];
        if (dz > 40) neworg[2] -= 8;
        if (dz < 30) neworg[2] += 8;
      }
      trace = SV_Move(EdictV(ent)->origin, EdictV(ent)->mins, EdictV(ent)->maxs,
                      neworg, false, NUM_FOR_EDICT(ent));

      if (trace.fraction == 1) {
        if (((int)EdictV(ent)->flags & FL_SWIM) &&
            SV_PointContents(trace.endpos) == CONTENTS_EMPTY)
          return false;  // swim monster left water

        VectorCopy(trace.endpos, EdictV(ent)->origin);
        if (relink) SV_LinkEdict(ent, true);
        return true;
      }

      if (enemy == sv.edicts) break;
    }

    return false;
  }

  // push down from a step height above the wished position
  neworg[2] += STEPSIZE;
  VectorCopy(neworg, end);
  end[2] -= STEPSIZE * 2;

  trace = SV_Move(neworg, EdictV(ent)->mins, EdictV(ent)->maxs, end, false,
                  NUM_FOR_EDICT(ent));

  if (trace.allsolid) return false;

  if (trace.startsolid) {
    neworg[2] -= STEPSIZE;
    trace = SV_Move(neworg, EdictV(ent)->mins, EdictV(ent)->maxs, end, false,
                    NUM_FOR_EDICT(ent));
    if (trace.allsolid || trace.startsolid) return false;
  }
  if (trace.fraction == 1) {
    // if monster had the ground pulled out, go ahead and fall
    if ((int)EdictV(ent)->flags & FL_PARTIALGROUND) {
      VectorAdd(EdictV(ent)->origin, move, EdictV(ent)->origin);
      if (relink) SV_LinkEdict(ent, true);
      EdictV(ent)->flags = (int)EdictV(ent)->flags & ~FL_ONGROUND;
      //	Con_Printf ("fall down\n");
      return true;
    }

    return false;  // walked off an edge
  }

  // check point traces down for dangling corners
  VectorCopy(trace.endpos, EdictV(ent)->origin);

  if (!SV_CheckBottom(ent)) {
    if ((int)EdictV(ent)->flags & FL_PARTIALGROUND) {  // entity had floor
                                                       // mostly pulled out from
                                                       // underneath it
      // and is trying to correct
      if (relink) SV_LinkEdict(ent, true);
      return true;
    }
    VectorCopy(oldorg, EdictV(ent)->origin);
    return false;
  }

  if ((int)EdictV(ent)->flags & FL_PARTIALGROUND) {
    //		Con_Printf ("back on ground\n");
    EdictV(ent)->flags = (int)EdictV(ent)->flags & ~FL_PARTIALGROUND;
  }
  EdictV(ent)->groundentity = NUM_FOR_EDICT(trace.ent);

  // the move is ok
  if (relink) SV_LinkEdict(ent, true);
  return true;
}

//============================================================================

/*
======================
SV_StepDirection

Turns to the movement direction, and walks the current distance if
facing it.

======================
*/
void PF_changeyaw(void);
qboolean SV_StepDirection(edict_t *ent, float yaw, float dist) {
  vec3_t move, oldorigin;
  float delta;

  EdictV(ent)->ideal_yaw = yaw;
  PF_changeyaw();

  yaw = yaw * M_PI * 2 / 360;
  move[0] = cos(yaw) * dist;
  move[1] = sin(yaw) * dist;
  move[2] = 0;

  VectorCopy(EdictV(ent)->origin, oldorigin);
  if (SV_movestep(ent, move, false)) {
    delta = EdictV(ent)->angles[YAW] - EdictV(ent)->ideal_yaw;
    if (delta > 45 &&
        delta < 315) {  // not turned far enough, so don't take the step
      VectorCopy(oldorigin, EdictV(ent)->origin);
    }
    SV_LinkEdict(ent, true);
    return true;
  }
  SV_LinkEdict(ent, true);

  return false;
}

/*
======================
SV_FixCheckBottom

======================
*/
void SV_FixCheckBottom(entvars_t *ent) {
  //	Con_Printf ("SV_FixCheckBottom\n");

  ent->flags = (int)ent->flags | FL_PARTIALGROUND;
}

/*
================
SV_NewChaseDir

================
*/
#define DI_NODIR -1
void SV_NewChaseDir(edict_t *actor, entvars_t *enemy, float dist) {
  float deltax, deltay;
  float d[3];
  float tdir, olddir, turnaround;

  olddir = anglemod((int)(EdictV(actor)->ideal_yaw / 45) * 45);
  turnaround = anglemod(olddir - 180);

  deltax = enemy->origin[0] - EdictV(actor)->origin[0];
  deltay = enemy->origin[1] - EdictV(actor)->origin[1];
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

  /* there is no direct path to the player, so pick another direction */

  if (olddir != DI_NODIR && SV_StepDirection(actor, olddir, dist)) return;

  if (rand() & 1) /*randomly determine direction of search*/
  {
    for (tdir = 0; tdir <= 315; tdir += 45)
      if (tdir != turnaround && SV_StepDirection(actor, tdir, dist)) return;
  } else {
    for (tdir = 315; tdir >= 0; tdir -= 45)
      if (tdir != turnaround && SV_StepDirection(actor, tdir, dist)) return;
  }

  if (turnaround != DI_NODIR && SV_StepDirection(actor, turnaround, dist))
    return;

  EdictV(actor)->ideal_yaw = olddir;  // can't move

  // if a bridge was pulled out from underneath a monster, it may not have
  // a valid standing position at all

  if (!SV_CheckBottom(actor)) SV_FixCheckBottom(EdictV(actor));
}

/*
======================
SV_CloseEnough

======================
*/
qboolean SV_CloseEnough(entvars_t *ent, entvars_t *goal, float dist) {
  int i;

  for (i = 0; i < 3; i++) {
    if (goal->absmin[i] > ent->absmax[i] + dist) return false;
    if (goal->absmax[i] < ent->absmin[i] - dist) return false;
  }
  return true;
}

/*
======================
SV_MoveToGoal

======================
*/
void SV_MoveToGoal(void) {
  edict_t *ent, *goal;
  float dist;

  ent = EDICT_NUM(Pr_global_struct_self());
  goal = EDICT_NUM(EdictV(ent)->goalentity);
  dist = Pr_globalsf(OFS_PARM0);

  if (!((int)EdictV(ent)->flags & (FL_ONGROUND | FL_FLY | FL_SWIM))) {
    Set_Pr_globalsf(OFS_RETURN, 0);
    return;
  }

  // if the next step hits the enemy, return immediately
  if (EDICT_NUM(EdictV(ent)->enemy) != sv.edicts &&
      SV_CloseEnough(EdictV(ent), EdictV(goal), dist))
    return;

  // bump around...
  if ((rand() & 3) == 1 ||
      !SV_StepDirection(ent, EdictV(ent)->ideal_yaw, dist)) {
    SV_NewChaseDir(ent, EdictV(goal), dist);
  }
}
