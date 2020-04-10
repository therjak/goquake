#ifndef BEAM_H
#define BEAM_H

#include "q_stdinc.h"

#define MAX_BEAMS 32  // johnfitz -- was 24
typedef struct {
  int entity;
  struct qmodel_s *model;
  float endtime;
  vec3_t start, end;
} beam_t;
extern beam_t cl_beams[MAX_BEAMS];

#endif // BEAM_H
