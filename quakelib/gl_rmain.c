// SPDX-License-Identifier: GPL-2.0-or-later
// r_main.c

#include "quakedef.h"

// johnfitz -- rendering statistics
int rs_skypolys;
int rs_skypasses;

//
// view origin
//
vec3_t r_origin;

cvar_t gl_farclip;  // = {"gl_farclip", "16384", CVAR_ARCHIVE};

void R_Init(void) {
  Cvar_FakeRegister(&gl_farclip, "gl_farclip");
}

void R_SetupView(void) {
  // build the transformation matrix for the given view angles
  vec3_t vieworg = {R_Refdef_vieworg(0), R_Refdef_vieworg(1),
                    R_Refdef_vieworg(2)};
  VectorCopy(vieworg, r_origin);
}
