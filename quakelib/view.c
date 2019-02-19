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
cvar_t gl_cshiftpercent;

float v_dmg_time, v_dmg_roll, v_dmg_pitch;

extern int in_forward, in_forward2, in_back;

vec3_t v_punchangles[2];  // johnfitz -- copied from cl.punchangle.  0 is
                          // current, 1 is previous value. never the same unless
                          // map just loaded

/*
===============
V_CalcRoll

Used by view and sv_user
===============
*/
float V_CalcRoll(vec3_t angles, vec3_t velocity) {
  vec3_t forward, right, up;
  float sign;
  float side;
  float value;

  AngleVectors(angles, forward, right, up);
  side = DotProduct(velocity, right);
  sign = side < 0 ? -1 : 1;
  side = fabs(side);

  value = Cvar_GetValue(&cl_rollangle);

  if (side < Cvar_GetValue(&cl_rollspeed))
    side = side * value / Cvar_GetValue(&cl_rollspeed);
  else
    side = value;

  return side * sign;
}

/*
===============
V_CalcBob

===============
*/
float V_CalcBob(void) {
  float bob;
  float cycle;

  cycle = CL_Time() - (int)(CL_Time() / Cvar_GetValue(&cl_bobcycle)) *
                          Cvar_GetValue(&cl_bobcycle);
  cycle /= Cvar_GetValue(&cl_bobcycle);
  if (cycle < Cvar_GetValue(&cl_bobup))
    cycle = M_PI * cycle / Cvar_GetValue(&cl_bobup);
  else
    cycle = M_PI + M_PI * (cycle - Cvar_GetValue(&cl_bobup)) /
                       (1.0 - Cvar_GetValue(&cl_bobup));

  // bob is proportional to velocity in the xy plane
  // (don't count Z, or jumping messes it up)

  bob =
      sqrt(cl.velocity[0] * cl.velocity[0] + cl.velocity[1] * cl.velocity[1]) *
      Cvar_GetValue(&cl_bob);
  bob = bob * 0.3 + bob * 0.7 * sin(cycle);
  if (bob > 4)
    bob = 4;
  else if (bob < -7)
    bob = -7;
  return bob;
}

//=============================================================================

void V_StartPitchDrift(void) {
#if 1
  if (cl.laststop == CL_Time()) {
    return;  // something else is keeping it from drifting
  }
#endif
  if (cl.nodrift || !cl.pitchvel) {
    cl.pitchvel = Cvar_GetValue(&v_centerspeed);
    cl.nodrift = false;
    cl.driftmove = 0;
  }
}

void V_StopPitchDrift(void) {
  cl.laststop = CL_Time();
  cl.nodrift = true;
  cl.pitchvel = 0;
}

/*
===============
V_DriftPitch

Moves the client pitch angle towards cl.idealpitch sent by the server.

If the user is adjusting pitch manually, either with lookup/lookdown,
mlook and mouse, or klook and keyboard, pitch drifting is constantly stopped.

Drifting is enabled when the center view key is hit, mlook is released and
lookspring is non 0, or when
===============
*/
void V_DriftPitch(void) {
  float delta, move;

  if (noclip_anglehack || !CL_OnGround() || CLS_IsDemoPlayback())
  // FIXME: noclip_anglehack is set on the server, so in a nonlocal game this
  // won't work.
  {
    cl.driftmove = 0;
    cl.pitchvel = 0;
    return;
  }

  // don't count small mouse motion
  if (cl.nodrift) {
    if (fabs(CL_CmdForwardMove()) < Cvar_GetValue(&cl_forwardspeed))
      cl.driftmove = 0;
    else
      cl.driftmove += HostFrameTime();

    if (cl.driftmove > Cvar_GetValue(&v_centermove)) {
      if (Cvar_GetValue(&lookspring)) V_StartPitchDrift();
    }
    return;
  }

  delta = cl.idealpitch - CLPitch();

  if (!delta) {
    cl.pitchvel = 0;
    return;
  }

  move = HostFrameTime() * cl.pitchvel;
  cl.pitchvel += HostFrameTime() * Cvar_GetValue(&v_centerspeed);

  if (delta > 0) {
    if (move > delta) {
      cl.pitchvel = 0;
      move = delta;
    }
    IncCLPitch(move);
  } else if (delta < 0) {
    if (move > -delta) {
      cl.pitchvel = 0;
      move = -delta;
    }
    DecCLPitch(move);
  }
}

/*
==============================================================================

        VIEW BLENDING

==============================================================================
*/

cshift_t cshift_empty = {{130, 80, 50}, 0};
cshift_t cshift_water = {{130, 80, 50}, 128};
cshift_t cshift_slime = {{0, 25, 5}, 150};
cshift_t cshift_lava = {{255, 80, 0}, 150};

float v_blend[4];  // rgba 0.0 - 1.0

// johnfitz -- deleted BuildGammaTable(), V_CheckGamma(), gammatable[], and
// ramps[][]

/*
===============
V_ParseDamage
===============
*/
void V_ParseDamage(int armor, int blood, float fromx, float fromy,
                   float fromz) {
  vec3_t from;
  int i;
  vec3_t forward, right, up;
  entity_t *ent;
  float side;
  float count;

  from[0] = fromx;
  from[1] = fromy;
  from[2] = fromz;

  count = blood * 0.5 + armor * 0.5;
  if (count < 10) count = 10;

  cl.faceanimtime = CL_Time() + 0.2;  // but sbar face into pain frame

  cl.cshifts[CSHIFT_DAMAGE].percent += 3 * count;
  if (cl.cshifts[CSHIFT_DAMAGE].percent < 0)
    cl.cshifts[CSHIFT_DAMAGE].percent = 0;
  if (cl.cshifts[CSHIFT_DAMAGE].percent > 150)
    cl.cshifts[CSHIFT_DAMAGE].percent = 150;

  if (armor > blood) {
    cl.cshifts[CSHIFT_DAMAGE].destcolor[0] = 200;
    cl.cshifts[CSHIFT_DAMAGE].destcolor[1] = 100;
    cl.cshifts[CSHIFT_DAMAGE].destcolor[2] = 100;
  } else if (armor) {
    cl.cshifts[CSHIFT_DAMAGE].destcolor[0] = 220;
    cl.cshifts[CSHIFT_DAMAGE].destcolor[1] = 50;
    cl.cshifts[CSHIFT_DAMAGE].destcolor[2] = 50;
  } else {
    cl.cshifts[CSHIFT_DAMAGE].destcolor[0] = 255;
    cl.cshifts[CSHIFT_DAMAGE].destcolor[1] = 0;
    cl.cshifts[CSHIFT_DAMAGE].destcolor[2] = 0;
  }

  //
  // calculate view angle kicks
  //
  ent = &cl_entities[CL_Viewentity()];

  VectorSubtract(from, ent->origin, from);
  VectorNormalize(from);

  AngleVectors(ent->angles, forward, right, up);

  side = DotProduct(from, right);
  v_dmg_roll = count * side * Cvar_GetValue(&v_kickroll);

  side = DotProduct(from, forward);
  v_dmg_pitch = count * side * Cvar_GetValue(&v_kickpitch);

  v_dmg_time = Cvar_GetValue(&v_kicktime);
}

/*
==================
V_cshift_f
==================
*/
void V_cshift_f(void) {
  cshift_empty.destcolor[0] = Cmd_ArgvAsInt(1);
  cshift_empty.destcolor[1] = Cmd_ArgvAsInt(2);
  cshift_empty.destcolor[2] = Cmd_ArgvAsInt(3);
  cshift_empty.percent = Cmd_ArgvAsInt(4);
}

/*
==================
V_BonusFlash_f

When you run over an item, the server sends this command
==================
*/
void V_BonusFlash_f(void) {
  cl.cshifts[CSHIFT_BONUS].destcolor[0] = 215;
  cl.cshifts[CSHIFT_BONUS].destcolor[1] = 186;
  cl.cshifts[CSHIFT_BONUS].destcolor[2] = 69;
  cl.cshifts[CSHIFT_BONUS].percent = 50;
}

/*
=============
V_SetContentsColor

Underwater, lava, etc each has a color shift
=============
*/
void V_SetContentsColor(int contents) {
  switch (contents) {
    case CONTENTS_EMPTY:
    case CONTENTS_SOLID:
    case CONTENTS_SKY:  // johnfitz -- no blend in sky
      cl.cshifts[CSHIFT_CONTENTS] = cshift_empty;
      break;
    case CONTENTS_LAVA:
      cl.cshifts[CSHIFT_CONTENTS] = cshift_lava;
      break;
    case CONTENTS_SLIME:
      cl.cshifts[CSHIFT_CONTENTS] = cshift_slime;
      break;
    default:
      cl.cshifts[CSHIFT_CONTENTS] = cshift_water;
  }
}

/*
=============
V_CalcPowerupCshift
=============
*/
void V_CalcPowerupCshift(void) {
  if (cl.items & IT_QUAD) {
    cl.cshifts[CSHIFT_POWERUP].destcolor[0] = 0;
    cl.cshifts[CSHIFT_POWERUP].destcolor[1] = 0;
    cl.cshifts[CSHIFT_POWERUP].destcolor[2] = 255;
    cl.cshifts[CSHIFT_POWERUP].percent = 30;
  } else if (cl.items & IT_SUIT) {
    cl.cshifts[CSHIFT_POWERUP].destcolor[0] = 0;
    cl.cshifts[CSHIFT_POWERUP].destcolor[1] = 255;
    cl.cshifts[CSHIFT_POWERUP].destcolor[2] = 0;
    cl.cshifts[CSHIFT_POWERUP].percent = 20;
  } else if (cl.items & IT_INVISIBILITY) {
    cl.cshifts[CSHIFT_POWERUP].destcolor[0] = 100;
    cl.cshifts[CSHIFT_POWERUP].destcolor[1] = 100;
    cl.cshifts[CSHIFT_POWERUP].destcolor[2] = 100;
    cl.cshifts[CSHIFT_POWERUP].percent = 100;
  } else if (cl.items & IT_INVULNERABILITY) {
    cl.cshifts[CSHIFT_POWERUP].destcolor[0] = 255;
    cl.cshifts[CSHIFT_POWERUP].destcolor[1] = 255;
    cl.cshifts[CSHIFT_POWERUP].destcolor[2] = 0;
    cl.cshifts[CSHIFT_POWERUP].percent = 30;
  } else
    cl.cshifts[CSHIFT_POWERUP].percent = 0;
}

/*
=============
V_CalcBlend
=============
*/
void V_CalcBlend(void) {
  float r, g, b, a, a2;
  int j;

  r = 0;
  g = 0;
  b = 0;
  a = 0;

  for (j = 0; j < NUM_CSHIFTS; j++) {
    if (!Cvar_GetValue(&gl_cshiftpercent)) continue;

    // johnfitz -- only apply leaf contents color shifts during intermission
    if (CL_Intermission() && j != CSHIFT_CONTENTS) continue;
    // johnfitz

    a2 = ((cl.cshifts[j].percent * Cvar_GetValue(&gl_cshiftpercent)) / 100.0) /
         255.0;
    if (!a2) continue;
    a = a + a2 * (1 - a);
    a2 = a2 / a;
    r = r * (1 - a2) + cl.cshifts[j].destcolor[0] * a2;
    g = g * (1 - a2) + cl.cshifts[j].destcolor[1] * a2;
    b = b * (1 - a2) + cl.cshifts[j].destcolor[2] * a2;
  }

  v_blend[0] = r / 255.0;
  v_blend[1] = g / 255.0;
  v_blend[2] = b / 255.0;
  v_blend[3] = a;
  if (v_blend[3] > 1) v_blend[3] = 1;
  if (v_blend[3] < 0) v_blend[3] = 0;
}

/*
=============
V_UpdateBlend -- johnfitz -- V_UpdatePalette cleaned up and renamed
=============
*/
void V_UpdateBlend(void) {
  int i, j;
  qboolean blend_changed;

  V_CalcPowerupCshift();

  blend_changed = false;

  for (i = 0; i < NUM_CSHIFTS; i++) {
    if (cl.cshifts[i].percent != cl.prev_cshifts[i].percent) {
      blend_changed = true;
      cl.prev_cshifts[i].percent = cl.cshifts[i].percent;
    }
    for (j = 0; j < 3; j++)
      if (cl.cshifts[i].destcolor[j] != cl.prev_cshifts[i].destcolor[j]) {
        blend_changed = true;
        cl.prev_cshifts[i].destcolor[j] = cl.cshifts[i].destcolor[j];
      }
  }

  // drop the damage value
  cl.cshifts[CSHIFT_DAMAGE].percent -= HostFrameTime() * 150;
  if (cl.cshifts[CSHIFT_DAMAGE].percent <= 0)
    cl.cshifts[CSHIFT_DAMAGE].percent = 0;

  // drop the bonus value
  cl.cshifts[CSHIFT_BONUS].percent -= HostFrameTime() * 100;
  if (cl.cshifts[CSHIFT_BONUS].percent <= 0)
    cl.cshifts[CSHIFT_BONUS].percent = 0;

  if (blend_changed) V_CalcBlend();
}

/*
============
V_PolyBlend -- johnfitz -- moved here from gl_rmain.c, and rewritten to use
glOrtho
============
*/
void V_PolyBlend(void) {
  if (!Cvar_GetValue(&gl_polyblend) || !v_blend[3]) return;

  GL_DisableMultitexture();

  glDisable(GL_ALPHA_TEST);
  glDisable(GL_TEXTURE_2D);
  glDisable(GL_DEPTH_TEST);
  glEnable(GL_BLEND);

  glMatrixMode(GL_PROJECTION);
  glLoadIdentity();
  glOrtho(0, 1, 1, 0, -99999, 99999);
  glMatrixMode(GL_MODELVIEW);
  glLoadIdentity();

  glColor4fv(v_blend);

  glBegin(GL_QUADS);
  glVertex2f(0, 0);
  glVertex2f(1, 0);
  glVertex2f(1, 1);
  glVertex2f(0, 1);
  glEnd();

  glDisable(GL_BLEND);
  glEnable(GL_DEPTH_TEST);
  glEnable(GL_TEXTURE_2D);
  glEnable(GL_ALPHA_TEST);
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
==============
V_CalcViewRoll

Roll is induced by movement and damage
==============
*/
void V_CalcViewRoll(void) {
  float side;

  side = V_CalcRoll(cl_entities[CL_Viewentity()].angles, cl.velocity);
  r_refdef.viewangles[ROLL] += side;

  if (v_dmg_time > 0) {
    r_refdef.viewangles[ROLL] +=
        v_dmg_time / Cvar_GetValue(&v_kicktime) * v_dmg_roll;
    r_refdef.viewangles[PITCH] +=
        v_dmg_time / Cvar_GetValue(&v_kicktime) * v_dmg_pitch;
    v_dmg_time -= HostFrameTime();
  }

  if (CL_Stats(STAT_HEALTH) <= 0) {
    r_refdef.viewangles[ROLL] = 80;  // dead view angle
    return;
  }
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
  r_refdef.vieworg[2] += cl.viewheight + bob;

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
  if (cl.maxclients <= 1)
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
  view->origin[2] += cl.viewheight;

  for (i = 0; i < 3; i++) view->origin[i] += forward[i] * bob * 0.4;
  view->origin[2] += bob;

  // johnfitz -- removed all gun position fudging code (was used to keep gun
  // from getting covered by sbar)

  view->model = cl.model_precache[CL_Stats(STAT_WEAPON)];
  view->frame = CL_Stats(STAT_WEAPONFRAME);
  view->colormap = host_colormap;

  // johnfitz -- v_gunkick
  if (Cvar_GetValue(&v_gunkick) == 1)  // original quake kick
    VectorAdd(r_refdef.viewangles, cl.punchangle, r_refdef.viewangles);
  if (Cvar_GetValue(&v_gunkick) == 2)  // lerped kick
  {
    for (i = 0; i < 3; i++)
      if (punch[i] != v_punchangles[0][i]) {
        // speed determined by how far we need to lerp in 1/10th of a second
        delta =
            (v_punchangles[0][i] - v_punchangles[1][i]) * HostFrameTime() * 10;

        if (delta > 0)
          punch[i] = q_min(punch[i] + delta, v_punchangles[0][i]);
        else if (delta < 0)
          punch[i] = q_max(punch[i] + delta, v_punchangles[0][i]);
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
extern vrect_t scr_vrect;

void V_RenderView(void) {
  if (con_forcedup) return;

  if (CL_Intermission())
    V_CalcIntermissionRefdef();
  else if (
      !CL_Paused() /* && (cl.maxclients > 1 || GetKeyDest() == key_game) */)
    V_CalcRefdef();

  // johnfitz -- removed lcd code

  R_RenderView();

  V_PolyBlend();  // johnfitz -- moved here from R_Renderview ();
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
  Cmd_AddCommand("v_cshift", V_cshift_f);
  Cmd_AddCommand("bf", V_BonusFlash_f);
  Cmd_AddCommand("centerview", V_StartPitchDrift);

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
  Cvar_FakeRegister(&gl_cshiftpercent, "gl_cshiftpercent");

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
