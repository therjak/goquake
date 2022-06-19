// SPDX-License-Identifier: GPL-2.0-or-later
// r_world.c: world model rendering

#include "quakedef.h"

extern cvar_t r_oldskyleaf;  // johnfitz

extern byte mod_novis[MAX_MAP_LEAFS / 8];
int vis_changed;  // if true, force pvs to be refreshed

//==============================================================================
//
// SETUP CHAINS
//
//==============================================================================

/*
================
R_BackFaceCull -- johnfitz -- returns true if the surface is facing away from
vieworg
================
*/
qboolean R_BackFaceCull(msurface_t *surf) {
  double dot;
  vec3_t vieworg = {R_Refdef_vieworg(0), R_Refdef_vieworg(1),
                    R_Refdef_vieworg(2)};

  switch (surf->plane->Type) {
    case PLANE_X:
      dot = vieworg[0] - surf->plane->dist;
      break;
    case PLANE_Y:
      dot = vieworg[1] - surf->plane->dist;
      break;
    case PLANE_Z:
      dot = vieworg[2] - surf->plane->dist;
      break;
    default:
      dot = DotProduct(vieworg, surf->plane->normal) - surf->plane->dist;
      break;
  }

  if ((dot < 0) ^ !!(surf->flags & SURF_PLANEBACK))
    return true;

  return false;
}

/*
================
R_CullSurfaces -- johnfitz
================
*/
void R_CullSurfaces(void) {
  msurface_t *s;
  int i;
  texture_t *t;

  // ericw -- instead of testing (s->visframe == r_visframecount) on all world
  // surfaces, use the chained surfaces, which is exactly the same set of
  // sufaces
  for (i = 0; i < cl.worldmodel->numtextures; i++) {
    t = cl.worldmodel->textures[i];

    if (!t || !t->texturechains[chain_world])
      continue;

    for (s = t->texturechains[chain_world]; s; s = s->texturechain) {
      if (R_CullBox(s->mins, s->maxs) || R_BackFaceCull(s))
        s->culled = true;
      else {
        s->culled = false;
        rs_brushpolys++;  // count wpolys here
      }
    }
  }
}
