// SPDX-License-Identifier: GPL-2.0-or-later
// r_world.c: world model rendering

#include "quakedef.h"

extern cvar_t gl_fullbrights, r_drawflat, gl_overbright, r_oldwater,
    r_oldskyleaf, r_showtris;  // johnfitz

byte *SV_FatPVS(vec3_t org, qmodel_t *worldmodel);
extern byte mod_novis[MAX_MAP_LEAFS / 8];
int vis_changed;  // if true, force pvs to be refreshed

//==============================================================================
//
// SETUP CHAINS
//
//==============================================================================

/*
================
R_ChainSurface -- ericw -- adds the given surface to its texture chain
================
*/
void R_ChainSurface(msurface_t *surf, texchain_t chain) {
  surf->texturechain = surf->texinfo->texture->texturechains[chain];
  surf->texinfo->texture->texturechains[chain] = surf;
}

/*
===============
R_MarkSurfaces -- johnfitz -- mark surfaces based on PVS and rebuild texture
chains
===============
*/
void R_MarkSurfaces(void) {
  byte *vis;
  mleaf_t *leaf;
  mnode_t *node;
  msurface_t *surf, **mark;
  int i, j;
  qboolean nearwaterportal;

  // check this leaf for water portals
  // TODO: loop through all water surfs and use distance to leaf cullbox
  nearwaterportal = false;
  for (i = 0, mark = r_viewleaf->firstmarksurface;
       i < r_viewleaf->nummarksurfaces; i++, mark++)
    if ((*mark)->flags & SURF_DRAWTURB)
      nearwaterportal = true;

  // choose vis data
  if (Cvar_GetValue(&r_novis) || r_viewleaf->contents == CONTENTS_SOLID ||
      r_viewleaf->contents == CONTENTS_SKY)
    vis = &mod_novis[0];
  else if (nearwaterportal)
    vis = SV_FatPVS(r_origin, cl.worldmodel);
  else
    vis = Mod_LeafPVS(r_viewleaf, cl.worldmodel);

  // if surface chains don't need regenerating, just add static entities and
  // return
  if (r_oldviewleaf == r_viewleaf && !vis_changed && !nearwaterportal) {
    // DONE IN GO
    // leaf = &cl.worldmodel->leafs[1];
    // for (i = 0; i < cl.worldmodel->numleafs; i++, leaf++)
    //  if (vis[i >> 3] & (1 << (i & 7)))
    //    if (leaf->efrags) R_StoreEfrags(&leaf->efrags);
    return;
  }

  vis_changed = false;
  R_visframecount_inc();
  r_oldviewleaf = r_viewleaf;

  // iterate through leaves, marking surfaces
  leaf = &cl.worldmodel->leafs[1];
  for (i = 0; i < cl.worldmodel->numleafs; i++, leaf++) {
    if (vis[i >> 3] & (1 << (i & 7))) {
      if (Cvar_GetValue(&r_oldskyleaf) || leaf->contents != CONTENTS_SKY)
        for (j = 0, mark = leaf->firstmarksurface; j < leaf->nummarksurfaces;
             j++, mark++)
          (*mark)->visframe = R_visframecount();

      // add static models
      // DONE IN GO
      // if (leaf->efrags) R_StoreEfrags(&leaf->efrags);
    }
  }

  // set all chains to null
  for (i = 0; i < cl.worldmodel->numtextures; i++)
    if (cl.worldmodel->textures[i])
      cl.worldmodel->textures[i]->texturechains[chain_world] = NULL;

  // rebuild chains

  // iterate through surfaces one node at a time to rebuild chains
  // need to do it this way if we want to work with tyrann's skip removal tool
  // becuase his tool doesn't actually remove the surfaces from the bsp surfaces
  // lump
  // nor does it remove references to them in each leaf's marksurfaces list
  for (i = 0, node = cl.worldmodel->nodes; i < cl.worldmodel->numnodes;
       i++, node++) {
    for (j = 0, surf = &cl.worldmodel->surfaces[node->firstsurface];
         j < node->numsurfaces; j++, surf++)
      if (surf->visframe == R_visframecount()) {
        R_ChainSurface(surf, chain_world);
      }
  }
}

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
        if (s->texinfo->texture->warpimage)
          s->texinfo->texture->update_warp = true;
      }
    }
  }
}
