// view.c -- player eye positioning

#include "quakedef.h"

/*

The view is allowed to move slightly from it's true position for bobbing,
but if it exceeds 8 pixels linear distance (spherical, not box), the list of
entities sent from the server may not include everything in the pvs, especially
when crossing a water boudnary.

*/

cvar_t v_centermove;
cvar_t v_centerspeed;
cvar_t scr_ofsx;
cvar_t scr_ofsy;
cvar_t scr_ofsz;
cvar_t cl_rollspeed;
cvar_t cl_rollangle;
cvar_t cl_bob;
cvar_t cl_bobcycle;
cvar_t cl_bobup;
cvar_t v_kicktime;
cvar_t v_kickroll;
cvar_t v_kickpitch;
cvar_t v_gunkick;
cvar_t v_iyaw_cycle;
cvar_t v_iroll_cycle;
cvar_t v_ipitch_cycle;
cvar_t v_iyaw_level;
cvar_t v_iroll_level;
cvar_t v_ipitch_level;
cvar_t v_idlescale;
cvar_t crosshair;

extern int in_forward, in_forward2, in_back;

float v_blend[4];  // rgba 0.0 - 1.0

/*
=============
V_CalcBlend
=============
*/
void V_CalcBlend(void) {
  V_CalcBlendGo(&v_blend[0],&v_blend[1],&v_blend[2],&v_blend[3]);
}

/*
==============================================================================

        VIEW RENDERING

==============================================================================
*/

float angledelta(float a) {
  a = anglemod(a);
  if (a > 180) a -= 360;
  return a;
}

/*
==================
CalcGunAngle
==================
*/
void CalcGunAngle(void) {
  float yaw, pitch, move;
  static float oldyaw = 0;
  static float oldpitch = 0;

  yaw = r_refdef.viewangles[YAW];
  pitch = -r_refdef.viewangles[PITCH];

  yaw = angledelta(yaw - r_refdef.viewangles[YAW]) * 0.4;
  if (yaw > 10) yaw = 10;
  if (yaw < -10) yaw = -10;
  pitch = angledelta(-pitch - r_refdef.viewangles[PITCH]) * 0.4;
  if (pitch > 10) pitch = 10;
  if (pitch < -10) pitch = -10;
  move = HostFrameTime() * 20;
  if (yaw > oldyaw) {
    if (oldyaw + move < yaw) yaw = oldyaw + move;
  } else {
    if (oldyaw - move > yaw) yaw = oldyaw - move;
  }

  if (pitch > oldpitch) {
    if (oldpitch + move < pitch) pitch = oldpitch + move;
  } else {
    if (oldpitch - move > pitch) pitch = oldpitch - move;
  }

  oldyaw = yaw;
  oldpitch = pitch;

  cl.viewent.angles[YAW] = r_refdef.viewangles[YAW] + yaw;
  cl.viewent.angles[PITCH] = -(r_refdef.viewangles[PITCH] + pitch);

  cl.viewent.angles[ROLL] -= Cvar_GetValue(&v_idlescale) *
                             sin(CL_Time() * Cvar_GetValue(&v_iroll_cycle)) *
                             Cvar_GetValue(&v_iroll_level);
  cl.viewent.angles[PITCH] -= Cvar_GetValue(&v_idlescale) *
                              sin(CL_Time() * Cvar_GetValue(&v_ipitch_cycle)) *
                              Cvar_GetValue(&v_ipitch_level);
  cl.viewent.angles[YAW] -= Cvar_GetValue(&v_idlescale) *
                            sin(CL_Time() * Cvar_GetValue(&v_iyaw_cycle)) *
                            Cvar_GetValue(&v_iyaw_level);
}

/*
==============
V_BoundOffsets
==============
*/
void V_BoundOffsets(void) {
  entity_t *ent;

  ent = &cl_entities[CL_Viewentity()];

  // absolutely bound refresh reletive to entity clipping hull
  // so the view can never be inside a solid wall

  if (r_refdef.vieworg[0] < ent->origin[0] - 14)
    r_refdef.vieworg[0] = ent->origin[0] - 14;
  else if (r_refdef.vieworg[0] > ent->origin[0] + 14)
    r_refdef.vieworg[0] = ent->origin[0] + 14;
  if (r_refdef.vieworg[1] < ent->origin[1] - 14)
    r_refdef.vieworg[1] = ent->origin[1] - 14;
  else if (r_refdef.vieworg[1] > ent->origin[1] + 14)
    r_refdef.vieworg[1] = ent->origin[1] + 14;
  if (r_refdef.vieworg[2] < ent->origin[2] - 22)
    r_refdef.vieworg[2] = ent->origin[2] - 22;
  else if (r_refdef.vieworg[2] > ent->origin[2] + 30)
    r_refdef.vieworg[2] = ent->origin[2] + 30;
}

/*
==============
V_AddIdle

Idle swaying
==============
*/
void V_AddIdle(float idlescale) {
  r_refdef.viewangles[ROLL] += idlescale *
                               sin(CL_Time() * Cvar_GetValue(&v_iroll_cycle)) *
                               Cvar_GetValue(&v_iroll_level);
  r_refdef.viewangles[PITCH] +=
      idlescale * sin(CL_Time() * Cvar_GetValue(&v_ipitch_cycle)) *
      Cvar_GetValue(&v_ipitch_level);
  r_refdef.viewangles[YAW] += idlescale *
                              sin(CL_Time() * Cvar_GetValue(&v_iyaw_cycle)) *
                              Cvar_GetValue(&v_iyaw_level);
}

/*
==================
V_CalcIntermissionRefdef

==================
*/
void V_CalcIntermissionRefdef(void) {
  entity_t *ent, *view;
  float old;

  // ent is the player model (visible when out of body)
  ent = &cl_entities[CL_Viewentity()];
  // view is the weapon model (only visible from inside body)
  view = &cl.viewent;

  VectorCopy(ent->origin, r_refdef.vieworg);
  VectorCopy(ent->angles, r_refdef.viewangles);
  view->model = NULL;

  // allways idle in intermission
  V_AddIdle(1);
}

/*
==================
V_CalcRefdef
==================
*/
void V_CalcRefdef(void) {
  entity_t *ent, *view;
  int i;
  vec3_t forward, right, up;
  vec3_t angles;
  float bob;
  static float oldz = 0;
  static vec3_t punch = {0, 0, 0};  // johnfitz -- v_gunkick
  float delta;                      // johnfitz -- v_gunkick

  V_DriftPitch();

  // ent is the player model (visible when out of body)
  ent = &cl_entities[CL_Viewentity()];
  // view is the weapon model (only visible from inside body)
  view = &cl.viewent;

  // transform the view offset by the model's matrix to get the offset from
  // model origin for the view
  ent->angles[YAW] = CLYaw();  // the model should face the view dir
  // the model should face the view dir
  ent->angles[PITCH] = -CLPitch();

  bob = V_CalcBob();

  // refresh position
  VectorCopy(ent->origin, r_refdef.vieworg);
  r_refdef.vieworg[2] += CL_ViewHeight() + bob;

  // never let it sit exactly on a node line, because a water plane can
  // dissapear when viewed with the eye exactly on it.
  // the server protocol only specifies to 1/16 pixel, so add 1/32 in each axis
  r_refdef.vieworg[0] += 1.0 / 32;
  r_refdef.vieworg[1] += 1.0 / 32;
  r_refdef.vieworg[2] += 1.0 / 32;

  r_refdef.viewangles[ROLL] = CLRoll();
  r_refdef.viewangles[PITCH] = CLPitch();
  r_refdef.viewangles[YAW] = CLYaw();

  V_CalcViewRoll();
  V_AddIdle(Cvar_GetValue(&v_idlescale));

  // offsets
  // because entity pitches are actually backward
  angles[PITCH] = -ent->angles[PITCH];
  angles[YAW] = ent->angles[YAW];
  angles[ROLL] = ent->angles[ROLL];

  AngleVectors(angles, forward, right, up);

  // johnfitz -- moved cheat-protection here from V_RenderView
  if (CL_MaxClients() <= 1)
    for (i = 0; i < 3; i++)
      r_refdef.vieworg[i] += Cvar_GetValue(&scr_ofsx) * forward[i] +
                             Cvar_GetValue(&scr_ofsy) * right[i] +
                             Cvar_GetValue(&scr_ofsz) * up[i];

  V_BoundOffsets();

  // set up gun position
  view->angles[ROLL] = CLRoll();
  view->angles[PITCH] = CLPitch();
  view->angles[YAW] = CLYaw();

  CalcGunAngle();

  VectorCopy(ent->origin, view->origin);
  view->origin[2] += CL_ViewHeight();

  for (i = 0; i < 3; i++) view->origin[i] += forward[i] * bob * 0.4;
  view->origin[2] += bob;

  // johnfitz -- removed all gun position fudging code (was used to keep gun
  // from getting covered by sbar)

  view->model = cl.model_precache[CL_Stats(STAT_WEAPON)];
  view->frame = CL_Stats(STAT_WEAPONFRAME);

  // johnfitz -- v_gunkick
  if (Cvar_GetValue(&v_gunkick) == 1) { // original quake kick
    r_refdef.viewangles[0] += CL_PunchAngle(0,0);
    r_refdef.viewangles[1] += CL_PunchAngle(0,1);
    r_refdef.viewangles[2] += CL_PunchAngle(0,2);
  }
  if (Cvar_GetValue(&v_gunkick) == 2) { // lerped kick
    for (i = 0; i < 3; i++)
      if (punch[i] != CL_PunchAngle(0,i)) {
        // speed determined by how far we need to lerp in 1/10th of a second
        delta =
            (CL_PunchAngle(0,i) - CL_PunchAngle(1,i)) * HostFrameTime() * 10;

        if (delta > 0)
          punch[i] = q_min(punch[i] + delta, CL_PunchAngle(0,i));
        else if (delta < 0)
          punch[i] = q_max(punch[i] + delta, CL_PunchAngle(0,i));
      }

    VectorAdd(r_refdef.viewangles, punch, r_refdef.viewangles);
  }
  // johnfitz

  // smooth out stair step ups
  if (!noclip_anglehack && CL_OnGround() &&
      ent->origin[2] - oldz > 0)  // johnfitz -- added exception for noclip
  // FIXME: noclip_anglehack is set on the server, so in a nonlocal game this
  // won't work.
  {
    float steptime;

    steptime = CL_Time() - CL_OldTime();
    if (steptime < 0)
      // FIXME	I_Error ("steptime < 0");
      steptime = 0;

    oldz += steptime * 80;
    if (oldz > ent->origin[2]) oldz = ent->origin[2];
    if (ent->origin[2] - oldz > 12) oldz = ent->origin[2] - 12;
    r_refdef.vieworg[2] += oldz - ent->origin[2];
    view->origin[2] += oldz - ent->origin[2];
  } else
    oldz = ent->origin[2];

  if (Cvar_GetValue(&chase_active)) Chase_UpdateForDrawing();  // johnfitz
}

/*
==================
V_RenderView

The player's clipping box goes from (-16 -16 -24) to (16 16 32) from
the entity origin, so any view position inside that will be valid
==================
*/
void V_RenderView(void) {
  if (Con_ForceDup()) return;

  if (CL_Intermission())
    V_CalcIntermissionRefdef();
  else if (
      !CL_Paused() /* && (CL_MaxClients() > 1 || GetKeyDest() == key_game) */)
    V_CalcRefdef();

  // johnfitz -- removed lcd code

  R_RenderView();

  V_PolyBlend(v_blend);  // johnfitz -- moved here from R_Renderview ();
}

/*
==============================================================================

        INIT

==============================================================================
*/

/*
=============
V_Init
=============
*/
void V_Init(void) {
  Cvar_FakeRegister(&v_centermove, "v_centermove");
  Cvar_FakeRegister(&v_centerspeed, "v_centerspeed");

  Cvar_FakeRegister(&v_iyaw_cycle, "v_iyaw_cycle");
  Cvar_FakeRegister(&v_iroll_cycle, "v_iroll_cycle");
  Cvar_FakeRegister(&v_ipitch_cycle, "v_ipitch_cycle");
  Cvar_FakeRegister(&v_iyaw_level, "v_iyaw_level");
  Cvar_FakeRegister(&v_iroll_level, "v_iroll_level");
  Cvar_FakeRegister(&v_ipitch_level, "v_ipitch_level");

  Cvar_FakeRegister(&v_idlescale, "v_idlescale");
  Cvar_FakeRegister(&crosshair, "crosshair");

  Cvar_FakeRegister(&scr_ofsx, "scr_ofsx");
  Cvar_FakeRegister(&scr_ofsy, "scr_ofsy");
  Cvar_FakeRegister(&scr_ofsz, "scr_ofsz");
  Cvar_FakeRegister(&cl_rollspeed, "cl_rollspeed");
  Cvar_FakeRegister(&cl_rollangle, "cl_rollangle");
  Cvar_FakeRegister(&cl_bob, "cl_bob");
  Cvar_FakeRegister(&cl_bobcycle, "cl_bobcycle");
  Cvar_FakeRegister(&cl_bobup, "cl_bobup");

  Cvar_FakeRegister(&v_kicktime, "v_kicktime");
  Cvar_FakeRegister(&v_kickroll, "v_kickroll");
  Cvar_FakeRegister(&v_kickpitch, "v_kickpitch");
  Cvar_FakeRegister(&v_gunkick, "v_gunkick");
}
