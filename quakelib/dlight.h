// SPDX-License-Identifier: GPL-2.0-or-later
#ifndef DLIGHT_H
#define DLIGHT_H

#include "q_stdinc.h"

#define MAX_DLIGHTS 64  // johnfitz -- was 32
typedef struct {
  vec3_t origin;
  float radius;
  float die;       // stop lighting after this time
  float minlight;  // don't add when contributing less
  vec3_t color;  // johnfitz -- lit support via lordhavoc
} dlight_t;

extern dlight_t cl_dlights[MAX_DLIGHTS];

#endif // DLIGHT_H
