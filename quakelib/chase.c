// chase.c -- chase camera code

#include "quakedef.h"

cvar_t chase_back;
cvar_t chase_up;
cvar_t chase_right;
cvar_t chase_active;

/*
==============
Chase_Init
==============
*/
void Chase_Init(void) {
  Cvar_FakeRegister(&chase_back,"chase_back");
  Cvar_FakeRegister(&chase_up,"chase_up");
  Cvar_FakeRegister(&chase_right,"chase_right");
  Cvar_FakeRegister(&chase_active,"chase_active");
}

/*
==============
TraceLine

TODO: impact on bmodels, monsters
==============
*/
void TraceLine(vec3_t start, vec3_t end, vec3_t impact) {
  trace_t trace;

  memset(&trace, 0, sizeof(trace));
  SV_RecursiveHullCheck(cl.worldmodel->hulls, 0, 0, 1, start, end, &trace);

  VectorCopy(trace.endpos, impact);
}

/*
==============
Chase_UpdateForClient -- johnfitz -- orient client based on camera. called after
input
==============
*/
void Chase_UpdateForClient(void) {
  // place camera

  // assign client angles to camera

  // see where camera points

  // adjust client angles to point at the same place
}

/*
==============
Chase_UpdateForDrawing -- johnfitz -- orient camera based on client. called
before drawing

TODO: stay at least 8 units away from all walls in this leaf
==============
*/
void Chase_UpdateForDrawing(void) {
  int i;
  vec3_t forward, up, right;
  vec3_t ideal, crosshair, temp;
  vec3_t clviewangles;
  clviewangles[PITCH]=CLPitch();
  clviewangles[YAW]=CLYaw();
  clviewangles[ROLL]=CLRoll();

  AngleVectors(clviewangles, forward, right, up);

  // calc ideal camera location before checking for walls
  for (i = 0; i < 3; i++)
    ideal[i] = cl.viewent.origin[i] - forward[i] * Cvar_GetValue(&chase_back) +
               right[i] * Cvar_GetValue(&chase_right);
  //+ up[i]*Cvar_GetValue(&chase_up);
  ideal[2] = cl.viewent.origin[2] + Cvar_GetValue(&chase_up);

  // make sure camera is not in or behind a wall
  TraceLine(r_refdef.vieworg, ideal, temp);
  if (VectorLength(temp) != 0) VectorCopy(temp, ideal);

  // place camera
  VectorCopy(ideal, r_refdef.vieworg);

  // find the spot the player is looking at
  VectorMA(cl.viewent.origin, 4096, forward, temp);
  TraceLine(cl.viewent.origin, temp, crosshair);

  // calculate camera angles to look at the same spot
  VectorSubtract(crosshair, r_refdef.vieworg, temp);
  VectorAngles(temp, r_refdef.viewangles);
  if (r_refdef.viewangles[PITCH] == 90 || r_refdef.viewangles[PITCH] == -90)
    r_refdef.viewangles[YAW] = CLYaw();
}
