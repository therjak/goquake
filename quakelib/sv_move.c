// sv_move.c -- monster movement

#include "quakedef.h"

/*
================
SV_NewChaseDir

================
*/
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

  if (!SV_CheckBottom(actor)) {
    entvars_t *ent = EVars(actor);
    ent->flags = (int)ent->flags | FL_PARTIALGROUND;
  }
}

/*
======================
SV_CloseEnough

======================
*/
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
  if (EVars(ent)->enemy != 0 && SV_CloseEnough(ent, goal, dist))
    return;

  // bump around...
  if ((rand() & 3) == 1 ||
      !SV_StepDirection(ent, EVars(ent)->ideal_yaw, dist)) {
    SV_NewChaseDir(ent, goal, dist);
  }
}
