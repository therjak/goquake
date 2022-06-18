// SPDX-License-Identifier: GPL-2.0-or-later
// gl_warp.c -- warping animation support

#include "quakedef.h"

extern cvar_t r_drawflat;

// should be read from gl_subdivide_size cvar
float load_subdivide_size;  // johnfitz -- remember what subdivide_size value
                            // was when this map was loaded

extern qmodel_t *loadmodel;

msurface_t *warpface;

cvar_t gl_subdivide_size;  // = {"gl_subdivide_size", "128", CVAR_ARCHIVE};

void BoundPoly(int numverts, float *verts, vec3_t mins, vec3_t maxs) {
  int i, j;
  float *v;

  mins[0] = mins[1] = mins[2] = FLT_MAX;
  maxs[0] = maxs[1] = maxs[2] = -FLT_MAX;
  v = verts;
  for (i = 0; i < numverts; i++)
    for (j = 0; j < 3; j++, v++) {
      if (*v < mins[j])
        mins[j] = *v;
      if (*v > maxs[j])
        maxs[j] = *v;
    }
}

void SubdividePolygon(int numverts, float *verts) {
  int i, j, k;
  vec3_t mins, maxs;
  float m;
  float *v;
  vec3_t front[64], back[64];
  int f, b;
  float dist[64];
  float frac;
  glpoly_t *poly;
  float s, t;

  if (numverts > 60)
    Go_Error_I("numverts = %v", numverts);

  BoundPoly(numverts, verts, mins, maxs);

  for (i = 0; i < 3; i++) {
    m = (mins[i] + maxs[i]) * 0.5;
    m = Cvar_GetValue(&gl_subdivide_size) *
        floor(m / Cvar_GetValue(&gl_subdivide_size) + 0.5);
    if (maxs[i] - m < 8)
      continue;
    if (m - mins[i] < 8)
      continue;

    // cut it
    v = verts + i;
    for (j = 0; j < numverts; j++, v += 3) dist[j] = *v - m;

    // wrap cases
    dist[j] = dist[0];
    v -= i;
    VectorCopy(verts, v);

    f = b = 0;
    v = verts;
    for (j = 0; j < numverts; j++, v += 3) {
      if (dist[j] >= 0) {
        VectorCopy(v, front[f]);
        f++;
      }
      if (dist[j] <= 0) {
        VectorCopy(v, back[b]);
        b++;
      }
      if (dist[j] == 0 || dist[j + 1] == 0)
        continue;
      if ((dist[j] > 0) != (dist[j + 1] > 0)) {
        // clip point
        frac = dist[j] / (dist[j] - dist[j + 1]);
        for (k = 0; k < 3; k++)
          front[f][k] = back[b][k] = v[k] + frac * (v[3 + k] - v[k]);
        f++;
        b++;
      }
    }

    SubdividePolygon(f, front[0]);
    SubdividePolygon(b, back[0]);
    return;
  }

  poly = (glpoly_t *)Hunk_Alloc(sizeof(glpoly_t) +
                                (numverts - 4) * VERTEXSIZE * sizeof(float));
  poly->next = warpface->polys->next;
  warpface->polys->next = poly;
  poly->numverts = numverts;
  for (i = 0; i < numverts; i++, verts += 3) {
    VectorCopy(verts, poly->verts[i]);
    s = DotProduct(verts, warpface->texinfo->vecs[0]);
    t = DotProduct(verts, warpface->texinfo->vecs[1]);
    poly->verts[i][3] = s;
    poly->verts[i][4] = t;
  }
}

/*
================
GL_SubdivideSurface
================
*/
void GL_SubdivideSurface(msurface_t *fa) {
  vec3_t verts[64];
  int i;

  warpface = fa;

  // the first poly in the chain is the undivided poly for newwater rendering.
  // grab the verts from that.
  for (i = 0; i < fa->polys->numverts; i++)
    VectorCopy(fa->polys->verts[i], verts[i]);

  SubdividePolygon(fa->polys->numverts, verts[0]);
}
