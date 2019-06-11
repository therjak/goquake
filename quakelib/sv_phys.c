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

cvar_t sv_gravity;
cvar_t sv_nostep;
cvar_t sv_freezenonclients;

#define MOVE_EPSILON 0.01

/*
============
SV_AddGravity

============
*/
// THERJAK
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
  int *moved_edict;    // johnfitz -- dynamically allocate
  vec3_t *moved_from;  // johnfitz -- dynamically allocate
  int mark;            // johnfitz

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
SV_Physics

================
*/
// THERJAK
/*
void SV_Physics(void) {
  int i;
  int entity_cap;  // For sv_freezenonclients
  int ent;

  // let the progs know that a new frame has started
  Set_pr_global_struct_self(0);
  Set_pr_global_struct_other(0);
  Set_pr_global_struct_time(SV_Time());
  PR_ExecuteProgram(Pr_global_struct_StartFrame());

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
}*/
