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
====================
R_SetWateralpha_f -- ericw
====================
*/
static void R_SetWateralpha_f(cvar_t *var) {
  map_wateralpha = Cvar_GetValue(var);
}

/*
====================
R_SetLavaalpha_f -- ericw
====================
*/
static void R_SetLavaalpha_f(cvar_t *var) {
  map_lavaalpha = Cvar_GetValue(var);
}

/*
====================
R_SetTelealpha_f -- ericw
====================
*/
static void R_SetTelealpha_f(cvar_t *var) {
  map_telealpha = Cvar_GetValue(var);
}

/*
====================
R_SetSlimealpha_f -- ericw
====================
*/
static void R_SetSlimealpha_f(cvar_t *var) {
  map_slimealpha = Cvar_GetValue(var);
}

/*
====================
GL_WaterAlphaForSurfface -- ericw
====================
*/
float GL_WaterAlphaForSurface(msurface_t *fa) {
  if (fa->flags & SURF_DRAWLAVA)
    return map_lavaalpha > 0 ? map_lavaalpha : map_wateralpha;
  else if (fa->flags & SURF_DRAWTELE)
    return map_telealpha > 0 ? map_telealpha : map_wateralpha;
  else if (fa->flags & SURF_DRAWSLIME)
    return map_slimealpha > 0 ? map_slimealpha : map_wateralpha;
  else
    return map_wateralpha;
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
  Cvar_FakeRegister(&r_wateralpha, "r_wateralpha");
  Cvar_SetCallback(&r_wateralpha, R_SetWateralpha_f);
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
  Cvar_FakeRegister(&r_lavaalpha, "r_lavaalpha");
  Cvar_SetCallback(&r_lavaalpha, R_SetLavaalpha_f);
  Cvar_FakeRegister(&r_telealpha, "r_telealpha");
  Cvar_SetCallback(&r_telealpha, R_SetTelealpha_f);
  Cvar_FakeRegister(&r_slimealpha, "r_slimealpha");
  Cvar_SetCallback(&r_slimealpha, R_SetSlimealpha_f);

  ParticlesInit();
  R_SetClearColor_f(&r_clearcolor);

  SkyInit();
  Fog_Init();
}

/*
=============
R_ParseWorldspawn

called at map load
=============
*/
static void R_ParseWorldspawn(void) {
  char key[128], value[4096];
  const char *data;

  map_wateralpha = Cvar_GetValue(&r_wateralpha);
  map_lavaalpha = Cvar_GetValue(&r_lavaalpha);
  map_telealpha = Cvar_GetValue(&r_telealpha);
  map_slimealpha = Cvar_GetValue(&r_slimealpha);

  data = COM_Parse(cl.worldmodel->entities);
  if (!data)
    return;  // error
  if (com_token[0] != '{')
    return;  // error
  while (1) {
    data = COM_Parse(data);
    if (!data)
      return;  // error
    if (com_token[0] == '}')
      break;  // end of worldspawn
    if (com_token[0] == '_')
      strcpy(key, com_token + 1);
    else
      strcpy(key, com_token);
    while (key[strlen(key) - 1] == ' ')  // remove trailing spaces
      key[strlen(key) - 1] = 0;
    data = COM_Parse(data);
    if (!data)
      return;  // error
    strcpy(value, com_token);

    if (!strcmp("wateralpha", key))
      map_wateralpha = atof(value);

    if (!strcmp("lavaalpha", key))
      map_lavaalpha = atof(value);

    if (!strcmp("telealpha", key))
      map_telealpha = atof(value);

    if (!strcmp("slimealpha", key))
      map_slimealpha = atof(value);
  }
}

/*
===============
R_NewMap
===============
*/
void R_NewMap(void) {
  int i;

  for (i = 0; i < 256; i++) d_lightstylevalue[i] = 264;  // normal light value

  // clear out efrags in case the level hasn't been reloaded
  ClearMapEntityFragments();

  r_viewleaf = NULL;
  ParticlesClear();

  // GL_BuildLightmaps();
  GL_BuildBModelVertexBuffer();
  // ericw -- no longer load alias models into a VBO here, it's done in
  // Mod_LoadAliasModel

  R_framecount_reset();     // johnfitz -- paranoid?
  R_visframecount_reset();  // johnfitz -- paranoid?

  SkyNewMap();          // johnfitz -- skybox in worldspawn
  Fog_NewMap();         // johnfitz -- global fog in worldspawn
  R_ParseWorldspawn();  // ericw -- wateralpha, lavaalpha, telealpha, slimealpha
                        // in worldspawn

  // johnfitz -- is this the right place to set this?
  load_subdivide_size = Cvar_GetValue(&gl_subdivide_size);
}
