// SPDX-License-Identifier: GPL-2.0-or-later
// mathlib.c -- math primitives

#include "quakedef.h"

vec3_t vec3_origin = {0, 0, 0};

//#define DEG2RAD( a ) ( a * M_PI ) / 180.0F
#define DEG2RAD(a) ((a)*M_PI_DIV_180)  // johnfitz

vec_t VectorLength(vec3_t v) {
  return sqrt(DotProduct(v, v));
}
