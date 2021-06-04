// SPDX-License-Identifier: GPL-2.0-or-later
// r_light.c

#include "quakedef.h"

extern cvar_t r_flatlightstyles;  // johnfitz

/*
=============================================================================

DYNAMIC LIGHTS

=============================================================================
*/

/*
=============
R_MarkLights -- johnfitz -- rewritten to use LordHavoc's lighting speedup
=============
*/
void R_MarkLight(dlight_t *light, int num, mnode_t *node) {
  mplane_t *splitplane;
  msurface_t *surf;
  vec3_t impact;
  float dist, l, maxdist;
  int i, j, s, t;
start:

  if (node->contents < 0) return;

  splitplane = node->plane;
  if (splitplane->Type < 3)
    dist = light->origin[splitplane->Type] - splitplane->dist;
  else
    dist = DotProduct(light->origin, splitplane->normal) - splitplane->dist;

  if (dist > light->radius) {
    node = node->children[0];
    goto start;
  }
  if (dist < -light->radius) {
    node = node->children[1];
    goto start;
  }

  maxdist = light->radius * light->radius;
  // mark the polygons
  surf = cl.worldmodel->surfaces + node->firstsurface;
  for (i = 0; i < node->numsurfaces; i++, surf++) {
    for (j = 0; j < 3; j++)
      impact[j] = light->origin[j] - surf->plane->normal[j] * dist;
    // clamp center of light to corner and check brightness
    l = DotProduct(impact, surf->texinfo->vecs[0]) + surf->texinfo->vecs[0][3] -
        surf->texturemins[0];
    s = l + 0.5;
    if (s < 0)
      s = 0;
    else if (s > surf->extents[0])
      s = surf->extents[0];
    s = l - s;
    l = DotProduct(impact, surf->texinfo->vecs[1]) + surf->texinfo->vecs[1][3] -
        surf->texturemins[1];
    t = l + 0.5;
    if (t < 0)
      t = 0;
    else if (t > surf->extents[1])
      t = surf->extents[1];
    t = l - t;
    // compare to minimum light
    if ((s * s + t * t + dist * dist) < maxdist) {
      if (surf->dlightframe != R_dlightframecount())  // not dynamic until now
      {
        surf->dlightbits[num >> 5] = 1U << (num & 31);
        surf->dlightframe = R_dlightframecount();
      } else  // already dynamic
        surf->dlightbits[num >> 5] |= 1U << (num & 31);
    }
  }

  if (node->children[0]->contents >= 0)
    R_MarkLight(light, num, node->children[0]);
  if (node->children[1]->contents >= 0)
    R_MarkLight(light, num, node->children[1]);
}

/*
=============
R_PushDlights
=============
*/
void R_PushDlights(void) {
  if (Cvar_GetValue(&gl_flashblend)) return;
  // TODO(therjak): disabling flashblend is broken since transparent console

  R_dlightframecount_up();

  R_MarkLights(cl.worldmodel->nodes);
}




