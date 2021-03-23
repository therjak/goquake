// SPDX-License-Identifier: GPL-2.0-or-later
// mathlib.c -- math primitives

#include "quakedef.h"

vec3_t vec3_origin = {0, 0, 0};

//#define DEG2RAD( a ) ( a * M_PI ) / 180.0F
#define DEG2RAD(a) ((a)*M_PI_DIV_180)  // johnfitz

// johnfitz -- the opposite of AngleVectors.  this takes forward and generates
// pitch yaw roll
// TODO: take right and up vectors to properly set yaw and roll
void VectorAngles(const vec3_t forward, vec3_t angles) {
  vec3_t temp;

  temp[0] = forward[0];
  temp[1] = forward[1];
  temp[2] = 0;
  angles[PITCH] = -atan2(forward[2], VectorLength(temp)) / M_PI_DIV_180;
  angles[YAW] = atan2(forward[1], forward[0]) / M_PI_DIV_180;
  angles[ROLL] = 0;
}

void AngleVectors(vec3_t angles, vec3_t forward, vec3_t right, vec3_t up) {
  float angle;
  float sr, sp, sy, cr, cp, cy;

  angle = angles[YAW] * (M_PI * 2 / 360);
  sy = sin(angle);
  cy = cos(angle);
  angle = angles[PITCH] * (M_PI * 2 / 360);
  sp = sin(angle);
  cp = cos(angle);
  angle = angles[ROLL] * (M_PI * 2 / 360);
  sr = sin(angle);
  cr = cos(angle);

  forward[0] = cp * cy;
  forward[1] = cp * sy;
  forward[2] = -sp;
  right[0] = (-1 * sr * sp * cy + -1 * cr * -sy);
  right[1] = (-1 * sr * sp * sy + -1 * cr * cy);
  right[2] = -1 * sr * cp;
  up[0] = (cr * sp * cy + -sr * -sy);
  up[1] = (cr * sp * sy + -sr * cy);
  up[2] = cr * cp;
}

int VectorCompare(vec3_t v1, vec3_t v2) {
  int i;

  for (i = 0; i < 3; i++)
    if (v1[i] != v2[i]) return 0;

  return 1;
}

void VectorMA(vec3_t veca, float scale, vec3_t vecb, vec3_t vecc) {
  vecc[0] = veca[0] + scale * vecb[0];
  vecc[1] = veca[1] + scale * vecb[1];
  vecc[2] = veca[2] + scale * vecb[2];
}

vec_t VectorLength(vec3_t v) { return sqrt(DotProduct(v, v)); }

float VectorNormalize(vec3_t v) {
  float length, ilength;

  length = sqrt(DotProduct(v, v));

  if (length) {
    ilength = 1 / length;
    v[0] *= ilength;
    v[1] *= ilength;
    v[2] *= ilength;
  }

  return length;
}

void VectorScale(vec3_t in, vec_t scale, vec3_t out) {
  out[0] = in[0] * scale;
  out[1] = in[1] * scale;
  out[2] = in[2] * scale;
}

