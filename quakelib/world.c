// world.c -- world query functions

#include "quakedef.h"

#include "_cgo_export.h"

/*

entities never clip against themselves, or their owner

line of sight checks trace->crosscontent, but bullets don't

*/

int SV_HullPointContents(hull_t *hull, int num, vec3_t p);

/*
===============================================================================

POINT TESTING IN HULLS

===============================================================================
*/

/*
==================
SV_HullPointContents

==================
*/
int SV_HullPointContents(hull_t *hull, int num, vec3_t p) {
  float d;
  mclipnode_t *node;  // johnfitz -- was dclipnode_t
  mplane_t *plane;

  while (num >= 0) {
    if (num < hull->firstclipnode || num > hull->lastclipnode)
      Go_Error("SV_HullPointContents: bad node number");

    node = hull->clipnodes + num;
    plane = hull->planes + node->planenum;

    if (plane->Type < 3)
      d = p[plane->Type] - plane->dist;
    else
      d = DoublePrecisionDotProduct(plane->normal, p) - plane->dist;
    if (d < 0)
      num = node->children[1];
    else
      num = node->children[0];
  }

  return num;
}

/*
===============================================================================

LINE TESTING IN HULLS

===============================================================================
*/

/*
==================
SV_RecursiveHullCheck

==================
*/
qboolean SV_RecursiveHullCheck(hull_t *hull, int num, float p1f, float p2f,
                               vec3_t p1, vec3_t p2, trace_t *trace) {
  mclipnode_t *node;  // johnfitz -- was dclipnode_t
  mplane_t *plane;
  float t1, t2;
  float frac;
  int i;
  vec3_t mid;
  int side;
  float midf;

  // check for empty
  if (num < 0) {
    if (num != CONTENTS_SOLID) {
      trace->allsolid = false;
      if (num == CONTENTS_EMPTY)
        trace->inopen = true;
      else
        trace->inwater = true;
    } else
      trace->startsolid = true;
    return true;  // empty
  }

  if (num < hull->firstclipnode || num > hull->lastclipnode)
    Go_Error("SV_RecursiveHullCheck: bad node number");

  //
  // find the point distances
  //
  node = hull->clipnodes + num;
  plane = hull->planes + node->planenum;

  if (plane->Type < 3) {
    t1 = p1[plane->Type] - plane->dist;
    t2 = p2[plane->Type] - plane->dist;
  } else {
    t1 = DoublePrecisionDotProduct(plane->normal, p1) - plane->dist;
    t2 = DoublePrecisionDotProduct(plane->normal, p2) - plane->dist;
  }

  if (t1 >= 0 && t2 >= 0)
    return SV_RecursiveHullCheck(hull, node->children[0], p1f, p2f, p1, p2,
                                 trace);
  if (t1 < 0 && t2 < 0)
    return SV_RecursiveHullCheck(hull, node->children[1], p1f, p2f, p1, p2,
                                 trace);

  // put the crosspoint DIST_EPSILON pixels on the near side
  if (t1 < 0)
    frac = (t1 + DIST_EPSILON) / (t1 - t2);
  else
    frac = (t1 - DIST_EPSILON) / (t1 - t2);
  if (frac < 0) frac = 0;
  if (frac > 1) frac = 1;

  midf = p1f + (p2f - p1f) * frac;
  for (i = 0; i < 3; i++) mid[i] = p1[i] + frac * (p2[i] - p1[i]);

  side = (t1 < 0);

  // move up to the node
  if (!SV_RecursiveHullCheck(hull, node->children[side], p1f, midf, p1, mid,
                             trace))
    return false;

  if (SV_HullPointContents(hull, node->children[side ^ 1], mid) !=
      CONTENTS_SOLID)
    // go past the node
    return SV_RecursiveHullCheck(hull, node->children[side ^ 1], midf, p2f, mid,
                                 p2, trace);

  if (trace->allsolid) return false;  // never got out of the solid area

  //==================
  // the other side of the node is solid, this is the impact point
  //==================
  if (!side) {
    VectorCopy(plane->normal, trace->plane.normal);
    trace->plane.dist = plane->dist;
  } else {
    VectorSubtract(vec3_origin, plane->normal, trace->plane.normal);
    trace->plane.dist = -plane->dist;
  }

  while (SV_HullPointContents(hull, hull->firstclipnode, mid) ==
         CONTENTS_SOLID) {  // shouldn't really happen, but does occasionally
    frac -= 0.1;
    if (frac < 0) {
      trace->fraction = midf;
      VectorCopy(mid, trace->endpos);
      Con_DPrintf("backup past 0\n");
      return false;
    }
    midf = p1f + (p2f - p1f) * frac;
    for (i = 0; i < 3; i++) mid[i] = p1[i] + frac * (p2[i] - p1[i]);
  }

  trace->fraction = midf;
  VectorCopy(mid, trace->endpos);

  return false;
}
