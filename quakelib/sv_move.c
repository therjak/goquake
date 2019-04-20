// sv_move.c -- monster movement

#include "quakedef.h"

#define STEPSIZE 18

/*
=============
SV_movestep

Called by monster program code.
The move will be adjusted for slopes and stairs, but if the move isn't
possible, no move is done, false is returned, and
pr_global_struct->trace_normal is set to the normal of the blocking wall
=============
*/
qboolean SV_movestep(int ent, vec3_t move, qboolean relink) {
  float dz;
  vec3_t oldorg, neworg, end;
  trace_t trace;
  int i;
  int enemy;

  // try the move
  VectorCopy(EVars(ent)->origin, oldorg);
  VectorAdd(EVars(ent)->origin, move, neworg);

  // flying monsters don't step up
  if ((int)EVars(ent)->flags & (FL_SWIM | FL_FLY)) {
    // try one move with vertical motion, then one without
    for (i = 0; i < 2; i++) {
      VectorAdd(EVars(ent)->origin, move, neworg);
      enemy = EVars(ent)->enemy;
      if (i == 0 && enemy != 0) {
        dz = EVars(ent)->origin[2] -
             EVars(EVars(ent)->enemy)->origin[2];
        if (dz > 40) neworg[2] -= 8;
        if (dz < 30) neworg[2] += 8;
      }
      trace = SV_Move(EVars(ent)->origin, EVars(ent)->mins, EVars(ent)->maxs,
                      neworg, false, ent);

      if (trace.fraction == 1) {
        if (((int)EVars(ent)->flags & FL_SWIM) &&
            SV_PointContents(trace.endpos) == CONTENTS_EMPTY)
          return false;  // swim monster left water

        VectorCopy(trace.endpos, EVars(ent)->origin);
        if (relink) SV_LinkEdict(ent, true);
        return true;
      }

      if (enemy == 0) break;
    }

    return false;
  }

  // push down from a step height above the wished position
  neworg[2] += STEPSIZE;
  VectorCopy(neworg, end);
  end[2] -= STEPSIZE * 2;

  trace = SV_Move(neworg, EVars(ent)->mins, EVars(ent)->maxs, end, false,
                  ent);

  if (trace.allsolid) return false;

  if (trace.startsolid) {
    neworg[2] -= STEPSIZE;
    trace = SV_Move(neworg, EVars(ent)->mins, EVars(ent)->maxs, end, false,
                    ent);
    if (trace.allsolid || trace.startsolid) return false;
  }
  if (trace.fraction == 1) {
    // if monster had the ground pulled out, go ahead and fall
    if ((int)EVars(ent)->flags & FL_PARTIALGROUND) {
      VectorAdd(EVars(ent)->origin, move, EVars(ent)->origin);
      if (relink) SV_LinkEdict(ent, true);
      EVars(ent)->flags = (int)EVars(ent)->flags & ~FL_ONGROUND;
      //	Con_Printf ("fall down\n");
      return true;
    }

    return false;  // walked off an edge
  }

  // check point traces down for dangling corners
  VectorCopy(trace.endpos, EVars(ent)->origin);

  if (!SV_CheckBottom(ent)) {
    if ((int)EVars(ent)->flags & FL_PARTIALGROUND) {  // entity had floor
                                                       // mostly pulled out from
                                                       // underneath it
      // and is trying to correct
      if (relink) SV_LinkEdict(ent, true);
      return true;
    }
    VectorCopy(oldorg, EVars(ent)->origin);
    return false;
  }

  if ((int)EVars(ent)->flags & FL_PARTIALGROUND) {
    //		Con_Printf ("back on ground\n");
    EVars(ent)->flags = (int)EVars(ent)->flags & ~FL_PARTIALGROUND;
  }
  EVars(ent)->groundentity = trace.entn;

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
qboolean SV_StepDirection(int ent, float yaw, float dist) {
  vec3_t move, oldorigin;
  float delta;

  EVars(ent)->ideal_yaw = yaw;
  PF_changeyaw();

  yaw = yaw * M_PI * 2 / 360;
  move[0] = cos(yaw) * dist;
  move[1] = sin(yaw) * dist;
  move[2] = 0;

  VectorCopy(EVars(ent)->origin, oldorigin);
  if (SV_movestep(ent, move, false)) {
    delta = EVars(ent)->angles[YAW] - EVars(ent)->ideal_yaw;
    if (delta > 45 &&
        delta < 315) {  // not turned far enough, so don't take the step
      VectorCopy(oldorigin, EVars(ent)->origin);
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
void SV_NewChaseDir(int actor, entvars_t *enemy, float dist) {
  float deltax, deltay;
  float d[3];
  float tdir, olddir, turnaround;

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

  EVars(actor)->ideal_yaw = olddir;  // can't move

  // if a bridge was pulled out from underneath a monster, it may not have
  // a valid standing position at all

  if (!SV_CheckBottom(actor)) SV_FixCheckBottom(EVars(actor));
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
  if (EVars(ent)->enemy != 0 &&
      SV_CloseEnough(EVars(ent), EVars(goal), dist))
    return;

  // bump around...
  if ((rand() & 3) == 1 ||
      !SV_StepDirection(ent, EVars(ent)->ideal_yaw, dist)) {
    SV_NewChaseDir(ent, EVars(goal), dist);
  }
}
