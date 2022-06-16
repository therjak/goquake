// SPDX-License-Identifier: GPL-2.0-or-later
// r_misc.c

#include "quakedef.h"

// johnfitz -- new cvars
extern cvar_t r_clearcolor;
extern cvar_t r_drawflat;
extern cvar_t r_flatlightstyles;
extern cvar_t gl_fullbrights;
extern cvar_t gl_farclip;
extern cvar_t r_waterquality;
extern cvar_t r_oldwater;
extern cvar_t r_waterwarp;
extern cvar_t r_oldskyleaf;
extern cvar_t r_drawworld;
extern cvar_t r_showtris;
extern cvar_t r_lerpmodels;
extern cvar_t r_lerpmove;
extern cvar_t r_nolerp_list;
extern cvar_t r_noshadow_list;
// johnfitz
extern cvar_t gl_zfix;  // QuakeSpasm z-fighting fix

/*
====================
R_SetClearColor_f -- johnfitz
====================
*/
static void R_SetClearColor_f(cvar_t *var) {
  int s;

  s = (int)Cvar_GetValue(&r_clearcolor) & 0xFF;
  glClearColor(D8To24Table(s, 0) / 255.0, D8To24Table(s, 1) / 255.0,
               D8To24Table(s, 2) / 255.0, 0);
}

/*
====================
R_Novis_f -- johnfitz
====================
*/
static void R_VisChanged(cvar_t *var) {
  extern int vis_changed;
  vis_changed = 1;
}

/*
===============
R_Model_ExtraFlags_List_f -- johnfitz -- called when r_nolerp_list or
r_noshadow_list cvar changes
===============
*/
static void R_Model_ExtraFlags_List_f(cvar_t *var) {
  int i;
  for (i = 0; i < MAX_MODELS; i++) Mod_SetExtraFlags(cl.model_precache[i]);
}

/*
===============
R_Init
===============
*/
void R_Init(void) {
  extern cvar_t gl_finish;

  Cvar_FakeRegister(&r_norefresh, "r_norefresh");
  Cvar_FakeRegister(&r_lightmap, "r_lightmap");
  Cvar_FakeRegister(&r_fullbright, "r_fullbright");
  Cvar_FakeRegister(&r_drawentities, "r_drawentities");
  Cvar_FakeRegister(&r_drawviewmodel, "r_drawviewmodel");
  Cvar_FakeRegister(&r_shadows, "r_shadows");
  Cvar_FakeRegister(&r_dynamic, "r_dynamic");
  Cvar_FakeRegister(&r_novis, "r_novis");
  Cvar_SetCallback(&r_novis, R_VisChanged);

  Cvar_FakeRegister(&gl_finish, "gl_finish");
  Cvar_FakeRegister(&gl_clear, "gl_clear");
  Cvar_FakeRegister(&gl_cull, "gl_cull");
  Cvar_FakeRegister(&gl_smoothmodels, "gl_smoothmodels");
  Cvar_FakeRegister(&gl_affinemodels, "gl_affinemodels");
  Cvar_FakeRegister(&gl_polyblend, "gl_polyblend");
  Cvar_FakeRegister(&gl_flashblend, "gl_flashblend");
  Cvar_FakeRegister(&gl_playermip, "gl_playermip");
  Cvar_FakeRegister(&gl_nocolors, "gl_nocolors");

  Cvar_FakeRegister(&r_clearcolor, "r_clearcolor");
  Cvar_SetCallback(&r_clearcolor, R_SetClearColor_f);
  Cvar_FakeRegister(&r_waterquality, "r_waterquality");
  Cvar_FakeRegister(&r_oldwater, "r_oldwater");
  Cvar_FakeRegister(&r_waterwarp, "r_waterwarp");
  Cvar_FakeRegister(&r_drawflat, "r_drawflat");
  Cvar_FakeRegister(&r_flatlightstyles, "r_flatlightstyles");
  Cvar_FakeRegister(&r_oldskyleaf, "r_oldskyleaf");
  Cvar_SetCallback(&r_oldskyleaf, R_VisChanged);
  Cvar_FakeRegister(&r_drawworld, "r_drawworld");
  Cvar_FakeRegister(&r_showtris, "r_showtris");
  Cvar_FakeRegister(&gl_farclip, "gl_farclip");
  Cvar_FakeRegister(&gl_fullbrights, "gl_fullbrights");
  Cvar_FakeRegister(&r_lerpmodels, "r_lerpmodels");
  Cvar_FakeRegister(&r_lerpmove, "r_lerpmove");
  Cvar_FakeRegister(&r_nolerp_list, "r_nolerp_list");
  Cvar_SetCallback(&r_nolerp_list, R_Model_ExtraFlags_List_f);
  Cvar_FakeRegister(&r_noshadow_list, "r_noshadow_list");

  Cvar_SetCallback(&r_noshadow_list, R_Model_ExtraFlags_List_f);

  Cvar_FakeRegister(&gl_zfix, "gl_zfix");

  ParticlesInit();
  R_SetClearColor_f(&r_clearcolor);

  SkyInit();
  Fog_Init();
}
