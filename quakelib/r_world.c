// SPDX-License-Identifier: GPL-2.0-or-later
// r_world.c: world model rendering

#include "quakedef.h"

extern cvar_t gl_fullbrights, r_drawflat, gl_overbright, r_oldwater,
    r_oldskyleaf, r_showtris;  // johnfitz

extern glpoly_t *lightmap_polys[MAX_LIGHTMAPS];

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
R_ClearTextureChains -- ericw

clears texture chains for all textures used by the given model, and also
clears the lightmap chains
================
*/
void R_ClearTextureChains(qmodel_t *mod, texchain_t chain) {
  int i;

  // set all chains to null
  for (i = 0; i < mod->numtextures; i++)
    if (mod->textures[i]) mod->textures[i]->texturechains[chain] = NULL;

  // clear lightmap chains
  memset(lightmap_polys, 0, sizeof(lightmap_polys));
}

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

  // clear lightmap chains
  memset(lightmap_polys, 0, sizeof(lightmap_polys));

  MarkSurfacesAddStaticEntities();
  // check this leaf for water portals
  // TODO: loop through all water surfs and use distance to leaf cullbox
  nearwaterportal = false;
  for (i = 0, mark = r_viewleaf->firstmarksurface;
       i < r_viewleaf->nummarksurfaces; i++, mark++)
    if ((*mark)->flags & SURF_DRAWTURB) nearwaterportal = true;

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
  UpdateOldViewLeafGo();
  r_oldviewleaf = r_viewleaf;

  // iterate through leaves, marking surfaces
  MarkSurfacesAddStaticEntitiesAndMark();
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

  if ((dot < 0) ^ !!(surf->flags & SURF_PLANEBACK)) return true;

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

    if (!t || !t->texturechains[chain_world]) continue;

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

/*
================
R_BuildLightmapChains -- johnfitz -- used for r_lightmap 1

ericw -- now always used at the start of R_DrawTextureChains for the
mh dynamic lighting speedup
================
*/
void R_BuildLightmapChains(qmodel_t *model, texchain_t chain) {
  texture_t *t;
  msurface_t *s;
  int i;

  // clear lightmap chains (already done in r_marksurfaces, but clearing them
  // here to be safe becuase of r_stereo)
  memset(lightmap_polys, 0, sizeof(lightmap_polys));

  // now rebuild them
  for (i = 0; i < model->numtextures; i++) {
    t = model->textures[i];

    if (!t || !t->texturechains[chain]) continue;

    for (s = t->texturechains[chain]; s; s = s->texturechain)
      if (!s->culled) R_RenderDynamicLightmaps(s);
  }
}

//==============================================================================
//
// DRAW CHAINS
//
//==============================================================================

/*
=============
R_BeginTransparentDrawing -- ericw
=============
*/
static void R_BeginTransparentDrawing(float entalpha) {
  if (entalpha < 1.0f) {
    glDepthMask(GL_FALSE);
    glEnable(GL_BLEND);
    glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_MODULATE);
    glColor4f(1, 1, 1, entalpha);
  }
}

/*
=============
R_EndTransparentDrawing -- ericw
=============
*/
static void R_EndTransparentDrawing(float entalpha) {
  if (entalpha < 1.0f) {
    glDepthMask(GL_TRUE);
    glDisable(GL_BLEND);
    glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_REPLACE);
    glColor3f(1, 1, 1);
  }
}

/*
================
R_DrawTextureChains_Drawflat -- johnfitz
================
*/
void R_DrawTextureChains_Drawflat(qmodel_t *model, texchain_t chain) {
  int i;
  msurface_t *s;
  texture_t *t;
  glpoly_t *p;

  for (i = 0; i < model->numtextures; i++) {
    t = model->textures[i];
    if (!t) continue;

    if (Cvar_GetValue(&r_oldwater) && t->texturechains[chain] &&
        (t->texturechains[chain]->flags & SURF_DRAWTURB)) {
      for (s = t->texturechains[chain]; s; s = s->texturechain)
        if (!s->culled)
          for (p = s->polys->next; p; p = p->next) {
            srand((unsigned int)(uintptr_t)p);
            glColor3f(rand() % 256 / 255.0, rand() % 256 / 255.0,
                      rand() % 256 / 255.0);
            DrawGLPoly(p);
            rs_brushpasses++;
          }
    } else {
      for (s = t->texturechains[chain]; s; s = s->texturechain)
        if (!s->culled) {
          srand((unsigned int)(uintptr_t)s->polys);
          glColor3f(rand() % 256 / 255.0, rand() % 256 / 255.0,
                    rand() % 256 / 255.0);
          DrawGLPoly(s->polys);
          rs_brushpasses++;
        }
    }
  }
  glColor3f(1, 1, 1);
  srand((int)(CL_Time() * 1000));
}

/*
================
R_DrawTextureChains_Glow -- johnfitz
================
*/
void R_DrawTextureChains_Glow(qmodel_t *model, entity_t *ent,
                              texchain_t chain) {
  int i;
  msurface_t *s;
  texture_t *t;
  uint32_t glt;
  qboolean bound;

  for (i = 0; i < model->numtextures; i++) {
    t = model->textures[i];

    if (!t || !t->texturechains[chain] ||
        !(glt =
              R_TextureAnimation(t, ent != NULL ? ent->frame : 0)->fullbright))
      continue;

    bound = false;

    for (s = t->texturechains[chain]; s; s = s->texturechain)
      if (!s->culled) {
        if (!bound)  // only bind once we are sure we need this texture
        {
          GLBind(glt);
          bound = true;
        }
        DrawGLPoly(s->polys);
        rs_brushpasses++;
      }
  }
}

//==============================================================================
//
// VBO SUPPORT
//
//==============================================================================

static unsigned int R_NumTriangleIndicesForSurf(msurface_t *s) {
  return 3 * (s->numedges - 2);
}

/*
================
R_TriangleIndicesForSurf

Writes out the triangle indices needed to draw s as a triangle list.
The number of indices it will write is given by R_NumTriangleIndicesForSurf.
================
*/
static void R_TriangleIndicesForSurf(msurface_t *s, unsigned int *dest) {
  int i;
  for (i = 2; i < s->numedges; i++) {
    *dest++ = s->vbo_firstvert;
    *dest++ = s->vbo_firstvert + i - 1;
    *dest++ = s->vbo_firstvert + i;
  }
}

#define MAX_BATCH_SIZE 4096

static unsigned int vbo_indices[MAX_BATCH_SIZE];
static unsigned int num_vbo_indices;

/*
================
R_ClearBatch
================
*/
static void R_ClearBatch() { num_vbo_indices = 0; }

/*
================
R_FlushBatch

Draw the current batch if non-empty and clears it, ready for more R_BatchSurface
calls.
================
*/
static void R_FlushBatch() {
  if (num_vbo_indices > 0) {
    glDrawElements(GL_TRIANGLES, num_vbo_indices, GL_UNSIGNED_INT, vbo_indices);
    num_vbo_indices = 0;
  }
}

/*
================
R_BatchSurface

Add the surface to the current batch, or just draw it immediately if we're not
using VBOs.
================
*/
static void R_BatchSurface(msurface_t *s) {
  int num_surf_indices;

  num_surf_indices = R_NumTriangleIndicesForSurf(s);

  if (num_vbo_indices + num_surf_indices > MAX_BATCH_SIZE) R_FlushBatch();

  R_TriangleIndicesForSurf(s, &vbo_indices[num_vbo_indices]);
  num_vbo_indices += num_surf_indices;
}

/*
================
R_DrawTextureChains_Multitexture -- johnfitz
================
*/
void R_DrawTextureChains_Multitexture(qmodel_t *model, entity_t *ent,
                                      texchain_t chain) {
  int i, j;
  msurface_t *s;
  texture_t *t;
  float *v;
  qboolean bound;

  for (i = 0; i < model->numtextures; i++) {
    t = model->textures[i];

    if (!t || !t->texturechains[chain] ||
        t->texturechains[chain]->flags & (SURF_DRAWTILED | SURF_NOTEXTURE))
      continue;

    bound = false;
    for (s = t->texturechains[chain]; s; s = s->texturechain)
      if (!s->culled) {
        if (!bound)  // only bind once we are sure we need this texture
        {
          GLBind(
              (R_TextureAnimation(t, ent != NULL ? ent->frame : 0))->gltexture);

          if (t->texturechains[chain]->flags & SURF_DRAWFENCE)
            glEnable(GL_ALPHA_TEST);  // Flip alpha test back on

          GLEnableMultitexture();  // selects TEXTURE1
          bound = true;
        }
        GLBind(lightmap_textures[s->lightmaptexturenum]);
        glBegin(GL_POLYGON);
        v = s->polys->verts[0];
        for (j = 0; j < s->polys->numverts; j++, v += VERTEXSIZE) {
          glMultiTexCoord2f(GL_TEXTURE0, v[3], v[4]);
          glMultiTexCoord2f(GL_TEXTURE1, v[5], v[6]);
          glVertex3fv(v);
        }
        glEnd();
        rs_brushpasses++;
      }
    GLDisableMultitexture();  // selects TEXTURE0

    if (bound && t->texturechains[chain]->flags & SURF_DRAWFENCE)
      glDisable(GL_ALPHA_TEST);  // Flip alpha test back off
  }
}

/*
================
R_DrawTextureChains_NoTexture -- johnfitz

draws surfs whose textures were missing from the BSP
================
*/
void R_DrawTextureChains_NoTexture(qmodel_t *model, texchain_t chain) {
  int i;
  msurface_t *s;
  texture_t *t;
  qboolean bound;

  for (i = 0; i < model->numtextures; i++) {
    t = model->textures[i];

    if (!t || !t->texturechains[chain] ||
        !(t->texturechains[chain]->flags & SURF_NOTEXTURE))
      continue;

    bound = false;

    for (s = t->texturechains[chain]; s; s = s->texturechain)
      if (!s->culled) {
        if (!bound)  // only bind once we are sure we need this texture
        {
          GLBind(t->gltexture);
          bound = true;
        }
        DrawGLPoly(s->polys);
        rs_brushpasses++;
      }
  }
}

/*
================
R_DrawTextureChains_TextureOnly -- johnfitz
================
*/
void R_DrawTextureChains_TextureOnly(qmodel_t *model, entity_t *ent,
                                     texchain_t chain) {
  int i;
  msurface_t *s;
  texture_t *t;
  qboolean bound;

  for (i = 0; i < model->numtextures; i++) {
    t = model->textures[i];

    if (!t || !t->texturechains[chain] ||
        t->texturechains[chain]->flags & (SURF_DRAWTURB | SURF_DRAWSKY))
      continue;

    bound = false;

    for (s = t->texturechains[chain]; s; s = s->texturechain)
      if (!s->culled) {
        if (!bound)  // only bind once we are sure we need this texture
        {
          GLBind(
              (R_TextureAnimation(t, ent != NULL ? ent->frame : 0))->gltexture);

          if (t->texturechains[chain]->flags & SURF_DRAWFENCE)
            glEnable(GL_ALPHA_TEST);  // Flip alpha test back on

          bound = true;
        }
        DrawGLPoly(s->polys);
        rs_brushpasses++;
      }

    if (bound && t->texturechains[chain]->flags & SURF_DRAWFENCE)
      glDisable(GL_ALPHA_TEST);  // Flip alpha test back off
  }
}

/*
================
GL_WaterAlphaForEntitySurface -- ericw

Returns the water alpha to use for the entity and surface combination.
================
*/
float GL_WaterAlphaForEntitySurface(entity_t *ent, msurface_t *s) {
  float entalpha;
  if (ent == NULL || ent->alpha2 == ENTALPHA_DEFAULT)
    entalpha = GL_WaterAlphaForSurface(s);
  else
    entalpha = ENTALPHA_DECODE(ent->alpha2);
  return entalpha;
}

/*
================
R_DrawTextureChains_Water -- johnfitz
================
*/
void R_DrawTextureChains_Water(qmodel_t *model, entity_t *ent,
                               texchain_t chain) {
  int i;
  msurface_t *s;
  texture_t *t;
  glpoly_t *p;
  qboolean bound;
  float entalpha;

  if (Cvar_GetValue(&r_oldwater)) {
    for (i = 0; i < model->numtextures; i++) {
      t = model->textures[i];
      if (!t || !t->texturechains[chain] ||
          !(t->texturechains[chain]->flags & SURF_DRAWTURB))
        continue;
      bound = false;
      entalpha = 1.0f;
      for (s = t->texturechains[chain]; s; s = s->texturechain)
        if (!s->culled) {
          if (!bound)  // only bind once we are sure we need this texture
          {
            entalpha = GL_WaterAlphaForEntitySurface(ent, s);
            R_BeginTransparentDrawing(entalpha);
            GLBind(t->gltexture);
            bound = true;
          }
          for (p = s->polys->next; p; p = p->next) {
            DrawWaterPoly(p);
            rs_brushpasses++;
          }
        }
      R_EndTransparentDrawing(entalpha);
    }
  } else {
    for (i = 0; i < model->numtextures; i++) {
      t = model->textures[i];
      if (!t || !t->texturechains[chain] ||
          !(t->texturechains[chain]->flags & SURF_DRAWTURB))
        continue;
      bound = false;
      entalpha = 1.0f;
      for (s = t->texturechains[chain]; s; s = s->texturechain)
        if (!s->culled) {
          if (!bound)  // only bind once we are sure we need this texture
          {
            entalpha = GL_WaterAlphaForEntitySurface(ent, s);
            R_BeginTransparentDrawing(entalpha);
            GLBind(t->warpimage);

            if (model != cl.worldmodel) {
              // ericw -- this is copied from R_DrawSequentialPoly.
              // If the poly is not part of the world we have to
              // set this flag
              t->update_warp = true;  // FIXME: one frame too late!
            }

            bound = true;
          }
          DrawGLPoly(s->polys);
          rs_brushpasses++;
        }
      R_EndTransparentDrawing(entalpha);
    }
  }
}

/*
================
R_DrawTextureChains_White -- johnfitz -- draw sky and water as white polys when
r_lightmap is 1
================
*/
void R_DrawTextureChains_White(qmodel_t *model, texchain_t chain) {
  int i;
  msurface_t *s;
  texture_t *t;

  glDisable(GL_TEXTURE_2D);
  for (i = 0; i < model->numtextures; i++) {
    t = model->textures[i];

    if (!t || !t->texturechains[chain] ||
        !(t->texturechains[chain]->flags & SURF_DRAWTILED))
      continue;

    for (s = t->texturechains[chain]; s; s = s->texturechain)
      if (!s->culled) {
        DrawGLPoly(s->polys);
        rs_brushpasses++;
      }
  }
  glEnable(GL_TEXTURE_2D);
}

/*
================
R_DrawLightmapChains -- johnfitz -- R_BlendLightmaps stripped down to almost
nothing
================
*/
void R_DrawLightmapChains(void) {
  int i, j;
  glpoly_t *p;
  float *v;

  for (i = 0; i < MAX_LIGHTMAPS; i++) {
    if (!lightmap_polys[i]) continue;

    GLBind(lightmap_textures[i]);
    for (p = lightmap_polys[i]; p; p = p->chain) {
      glBegin(GL_POLYGON);
      v = p->verts[0];
      for (j = 0; j < p->numverts; j++, v += VERTEXSIZE) {
        glTexCoord2f(v[5], v[6]);
        glVertex3fv(v);
      }
      glEnd();
      rs_brushpasses++;
    }
  }
}

extern GLuint gl_bmodel_vbo;

/*
================
R_DrawTextureChains_Multitexture_VBO -- ericw

Draw lightmapped surfaces with fulbrights in one pass, using VBO.
Requires 3 TMUs, GL_COMBINE_EXT, and GL_ADD.
================
*/
void R_DrawTextureChains_Multitexture_VBO(qmodel_t *model, entity_t *ent,
                                          texchain_t chain) {
  int i;
  msurface_t *s;
  texture_t *t;
  qboolean bound;
  int lastlightmap;
  uint32_t fullbright = 0;

  // Bind the buffers
  glBindBuffer(GL_ARRAY_BUFFER, gl_bmodel_vbo);
  glBindBuffer(GL_ELEMENT_ARRAY_BUFFER,
               0);  // indices come from client memory!

  // Setup vertex array pointers
  glVertexPointer(3, GL_FLOAT, VERTEXSIZE * sizeof(float), ((float *)0));
  glEnableClientState(GL_VERTEX_ARRAY);

  glClientActiveTexture(GL_TEXTURE0);
  glTexCoordPointer(2, GL_FLOAT, VERTEXSIZE * sizeof(float), ((float *)0) + 3);
  glEnableClientState(GL_TEXTURE_COORD_ARRAY);

  glClientActiveTexture(GL_TEXTURE1);
  glTexCoordPointer(2, GL_FLOAT, VERTEXSIZE * sizeof(float), ((float *)0) + 5);
  glEnableClientState(GL_TEXTURE_COORD_ARRAY);

  // TMU 2 is for fullbrights; same texture coordinates as TMU 0
  glClientActiveTexture(GL_TEXTURE2);
  glTexCoordPointer(2, GL_FLOAT, VERTEXSIZE * sizeof(float), ((float *)0) + 3);
  glEnableClientState(GL_TEXTURE_COORD_ARRAY);

  // Setup TMU 1 (lightmap)
  GLSelectTexture(GL_TEXTURE1);
  glTexEnvi(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_COMBINE);
  glTexEnvi(GL_TEXTURE_ENV, GL_COMBINE_RGB, GL_MODULATE);
  glTexEnvi(GL_TEXTURE_ENV, GL_SOURCE0_RGB, GL_PREVIOUS);
  glTexEnvi(GL_TEXTURE_ENV, GL_SOURCE1_RGB, GL_TEXTURE);
  glTexEnvf(GL_TEXTURE_ENV, GL_RGB_SCALE,
            Cvar_GetValue(&gl_overbright) ? 2.0f : 1.0f);
  glEnable(GL_TEXTURE_2D);

  // Setup TMU 2 (fullbrights)
  GLSelectTexture(GL_TEXTURE2);
  glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_ADD);

  for (i = 0; i < model->numtextures; i++) {
    t = model->textures[i];

    if (!t || !t->texturechains[chain] ||
        t->texturechains[chain]->flags & (SURF_DRAWTILED | SURF_NOTEXTURE))
      continue;

    // Enable/disable TMU 2 (fullbrights)
    GLSelectTexture(GL_TEXTURE2);
    if (Cvar_GetValue(&gl_fullbrights) &&
        (fullbright =
             R_TextureAnimation(t, ent != NULL ? ent->frame : 0)->fullbright)) {
      glEnable(GL_TEXTURE_2D);
      GLBind(fullbright);
    } else
      glDisable(GL_TEXTURE_2D);

    R_ClearBatch();

    bound = false;
    lastlightmap = 0;  // avoid compiler warning
    for (s = t->texturechains[chain]; s; s = s->texturechain)
      if (!s->culled) {
        if (!bound)  // only bind once we are sure we need this texture
        {
          GLSelectTexture(GL_TEXTURE0);
          GLBind(
              (R_TextureAnimation(t, ent != NULL ? ent->frame : 0))->gltexture);

          if (t->texturechains[chain]->flags & SURF_DRAWFENCE)
            glEnable(GL_ALPHA_TEST);  // Flip alpha test back on

          bound = true;
          lastlightmap = s->lightmaptexturenum;
        }

        if (s->lightmaptexturenum != lastlightmap) R_FlushBatch();

        GLSelectTexture(GL_TEXTURE1);
        GLBind(lightmap_textures[s->lightmaptexturenum]);
        lastlightmap = s->lightmaptexturenum;
        R_BatchSurface(s);

        rs_brushpasses++;
      }

    R_FlushBatch();

    if (bound && t->texturechains[chain]->flags & SURF_DRAWFENCE)
      glDisable(GL_ALPHA_TEST);  // Flip alpha test back off
  }

  // Reset TMU states
  GLSelectTexture(GL_TEXTURE2);
  glDisable(GL_TEXTURE_2D);

  GLSelectTexture(GL_TEXTURE1);
  glTexEnvf(GL_TEXTURE_ENV, GL_RGB_SCALE, 1.0f);
  glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_MODULATE);
  glDisable(GL_TEXTURE_2D);

  GLSelectTexture(GL_TEXTURE0);
  glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_REPLACE);

  // Disable client state
  glDisableClientState(GL_VERTEX_ARRAY);

  glClientActiveTexture(GL_TEXTURE0);
  glDisableClientState(GL_TEXTURE_COORD_ARRAY);

  glClientActiveTexture(GL_TEXTURE1);
  glDisableClientState(GL_TEXTURE_COORD_ARRAY);

  glClientActiveTexture(GL_TEXTURE2);
  glDisableClientState(GL_TEXTURE_COORD_ARRAY);
}

/*
=============
R_DrawWorld -- johnfitz -- rewritten
=============
*/
void R_DrawTextureChains(qmodel_t *model, entity_t *ent, texchain_t chain) {
  float entalpha;

  if (ent != NULL)
    entalpha = ENTALPHA_DECODE(ent->alpha2);
  else
    entalpha = 1;

  // ericw -- the mh dynamic lightmap speedup: make a first pass through all
  // surfaces we are going to draw, and rebuild any lightmaps that need it.
  // this also chains surfaces by lightmap which is used by r_lightmap 1.
  // the previous implementation of the speedup uploaded lightmaps one frame
  // late which was visible under some conditions, this method avoids that.
  R_BuildLightmapChains(model, chain);
  R_UploadLightmaps();

  R_BeginTransparentDrawing(entalpha);

  R_DrawTextureChains_NoTexture(model, chain);

  R_DrawTextureChains_Multitexture_VBO(model, ent, chain);
  R_EndTransparentDrawing(entalpha);
}

/*
=============
R_DrawWorld -- ericw -- moved from R_DrawTextureChains, which is no longer
specific to the world.
=============
*/
void R_DrawWorld(void) {
  R_DrawTextureChains(cl.worldmodel, NULL, chain_world);
}

/*
=============
R_DrawWorld_Water -- ericw -- moved from R_DrawTextureChains_Water, which is no
longer specific to the world.
=============
*/
void R_DrawWorld_Water(void) {
  R_DrawTextureChains_Water(cl.worldmodel, NULL, chain_world);
}
