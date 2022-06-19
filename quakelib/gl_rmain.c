// SPDX-License-Identifier: GPL-2.0-or-later
// r_main.c

#include "quakedef.h"

// johnfitz -- rendering statistics
int rs_brushpolys, rs_aliaspolys, rs_skypolys, rs_particles, rs_fogpolys;
int rs_dynamiclightmaps, rs_brushpasses, rs_aliaspasses, rs_skypasses;
float rs_megatexels;

//
// view origin
//
vec3_t vup;
vec3_t vpn;
vec3_t vright;
vec3_t r_origin;

mleaf_t *r_viewleaf, *r_oldviewleaf;

cvar_t r_norefresh;      // = {"r_norefresh", "0", CVAR_NONE};
cvar_t r_drawentities;   // = {"r_drawentities", "1", CVAR_NONE};
cvar_t r_drawviewmodel;  // = {"r_drawviewmodel", "1", CVAR_NONE};
cvar_t r_fullbright;     // = {"r_fullbright", "0", CVAR_NONE};
cvar_t r_lightmap;       // = {"r_lightmap", "0", CVAR_NONE};
cvar_t r_shadows;        // = {"r_shadows", "0", CVAR_ARCHIVE};
cvar_t r_dynamic;        // = {"r_dynamic", "1", CVAR_ARCHIVE};
cvar_t r_novis;          // = {"r_novis", "0", CVAR_ARCHIVE};

cvar_t gl_finish;        // = {"gl_finish", "0", CVAR_NONE};
cvar_t gl_clear;         // = {"gl_clear", "1", CVAR_NONE};
cvar_t gl_cull;          // = {"gl_cull", "1", CVAR_NONE};
cvar_t gl_smoothmodels;  // = {"gl_smoothmodels", "1", CVAR_NONE};
cvar_t gl_affinemodels;  // = {"gl_affinemodels", "0", CVAR_NONE};
cvar_t gl_polyblend;     // = {"gl_polyblend", "1", CVAR_NONE};
cvar_t gl_flashblend;    // = {"gl_flashblend", "0", CVAR_ARCHIVE};
cvar_t gl_playermip;     // = {"gl_playermip", "0", CVAR_NONE};
cvar_t gl_nocolors;      // = {"gl_nocolors", "0", CVAR_NONE};

// johnfitz -- new cvars
cvar_t r_clearcolor;          // = {"r_clearcolor", "2", CVAR_ARCHIVE};
cvar_t r_drawflat;            // = {"r_drawflat", "0", CVAR_NONE};
cvar_t r_flatlightstyles;     // = {"r_flatlightstyles", "0", CVAR_NONE};
cvar_t gl_fullbrights;        // = {"gl_fullbrights", "1", CVAR_ARCHIVE};
cvar_t gl_farclip;            // = {"gl_farclip", "16384", CVAR_ARCHIVE};
cvar_t gl_overbright;         // = {"gl_overbright", "1", CVAR_ARCHIVE};
cvar_t gl_overbright_models;  // = {"gl_overbright_models", "1", CVAR_ARCHIVE};
cvar_t r_oldskyleaf;          // = {"r_oldskyleaf", "0", CVAR_NONE};
cvar_t r_drawworld;           // = {"r_drawworld", "1", CVAR_NONE};
cvar_t r_showtris;            // = {"r_showtris", "0", CVAR_NONE};
cvar_t r_lerpmodels;          // = {"r_lerpmodels", "1", CVAR_NONE};
cvar_t r_lerpmove;            // = {"r_lerpmove", "1", CVAR_NONE};
cvar_t r_nolerp_list;         // = {"r_nolerp_list",
                              //   "progs/flame.mdl,progs/flame2.mdl,progs/"
                              //   "braztall.mdl,progs/brazshrt.mdl,progs/"
                              //  "longtrch.mdl,progs/flame_pyre.mdl,progs/"
//       "v_saw.mdl,progs/v_xfist.mdl,progs/h2stuff/newfire.mdl",
//     CVAR_NONE};
cvar_t r_noshadow_list;  // = {"r_noshadow_list",
                         //  "progs/flame2.mdl,progs/flame.mdl,progs/"
                         //  "bolt1.mdl,progs/bolt2.mdl,progs/bolt3.mdl,progs/"
                         //  "laser.mdl",
                         //  CVAR_NONE};

// johnfitz

cvar_t gl_zfix;  // = {"gl_zfix", "0", CVAR_NONE};  // QuakeSpasm z-fighting fix

#define DEG2RAD(a) ((a)*M_PI_DIV_180)

void R_Clear(void) {
  unsigned int clearbits;

  clearbits = GL_DEPTH_BUFFER_BIT;
  // if we get a stencil buffer, we should clear it, even though we
  // don't use it
  if (gl_stencilbits)
    clearbits |= GL_STENCIL_BUFFER_BIT;
  if (Cvar_GetValue(&gl_clear))
    clearbits |= GL_COLOR_BUFFER_BIT;
  glClear(clearbits);
}

void R_SetupView(void) {
  // build the transformation matrix for the given view angles
  vec3_t vieworg = {R_Refdef_vieworg(0), R_Refdef_vieworg(1),
                    R_Refdef_vieworg(2)};
  VectorCopy(vieworg, r_origin);
  vec3_t viewangles = {R_Refdef_viewangles(0), R_Refdef_viewangles(1),
                       R_Refdef_viewangles(2)};
  AngleVectors(viewangles, vpn, vright, vup);
  UpdateVpnGo();

  // current viewleaf
  UpdateViewLeafGo();
  r_oldviewleaf = r_viewleaf;
  r_viewleaf = Mod_PointInLeaf(r_origin, cl.worldmodel);

  V_SetContentsColor(r_viewleaf->contents);
  V_CalcBlend();

  MarkSurfaces();  // create texture chains from PVS
  R_CullSurfaces();
  GLSetCanvas(CANVAS_DEFAULT);
  Sbar_Changed();
  SCR_ResetTileClearUpdates();
  R_Clear();
}
