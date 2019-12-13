// view.c -- player eye positioning

#include "quakedef.h"

/*

The view is allowed to move slightly from it's true position for bobbing,
but if it exceeds 8 pixels linear distance (spherical, not box), the list of
entities sent from the server may not include everything in the pvs, especially
when crossing a water boudnary.

*/

cvar_t scr_ofsx;
cvar_t scr_ofsy;
cvar_t scr_ofsz;
cvar_t v_gunkick;
cvar_t v_idlescale;
cvar_t crosshair;

extern int in_forward, in_forward2, in_back;

float v_blend[4];  // rgba 0.0 - 1.0

void SetCLWeaponModel(int v) {
  entity_t *view;
  view = &cl_viewent;
  view->model = cl.model_precache[v];
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
  Cvar_FakeRegister(&v_idlescale, "v_idlescale");
  Cvar_FakeRegister(&crosshair, "crosshair");

  Cvar_FakeRegister(&scr_ofsx, "scr_ofsx");
  Cvar_FakeRegister(&scr_ofsy, "scr_ofsy");
  Cvar_FakeRegister(&scr_ofsz, "scr_ofsz");

  Cvar_FakeRegister(&v_gunkick, "v_gunkick");
}
