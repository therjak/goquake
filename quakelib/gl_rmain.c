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
vec3_t r_origin;

// johnfitz -- new cvars
cvar_t r_clearcolor;   // = {"r_clearcolor", "2", CVAR_ARCHIVE};
cvar_t gl_farclip;     // = {"gl_farclip", "16384", CVAR_ARCHIVE};
cvar_t r_oldskyleaf;   // = {"r_oldskyleaf", "0", CVAR_NONE};
cvar_t r_nolerp_list;  // = {"r_nolerp_list",
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

#define DEG2RAD(a) ((a)*M_PI_DIV_180)

void R_Init(void) {
  Cvar_FakeRegister(&gl_farclip, "gl_farclip");
}

void R_SetupView(void) {
  // build the transformation matrix for the given view angles
  vec3_t vieworg = {R_Refdef_vieworg(0), R_Refdef_vieworg(1),
                    R_Refdef_vieworg(2)};
  VectorCopy(vieworg, r_origin);
}
