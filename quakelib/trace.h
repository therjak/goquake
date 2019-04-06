
#ifndef _QUAKE_TRACE_H
#define _QUAKE_TRACE_H

typedef float v3[3];

typedef struct {
  float normal[3];
  float dist;
} plane_t;

typedef struct {
  int allsolid;    // if true, plane is not valid
  int startsolid;  // if true, the initial point was in a solid area
  int inopen, inwater;
  float fraction;   // time completed, 1.0 = didn't hit anything
  float endpos[3];  // final position
  plane_t plane;    // surface normal at impact
  int entp;         // entity the surface is on
  int entn;         // entity the surface is on
} trace_t;
#endif  //  _QUAKE_TRACE_H
