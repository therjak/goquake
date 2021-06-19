/*
Copyright (C) 1996-2001 Id Software, Inc.
Copyright (C) 2002-2009 John Fitzgibbons and others
Copyright (C) 2007-2008 Kristian Duske
Copyright (C) 2010-2014 QuakeSpasm developers

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 2
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.

See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software
Foundation, Inc., 59 Temple Place - Suite 330, Boston, MA  02111-1307, USA.

*/

#ifndef __GLQUAKE_H
#define __GLQUAKE_H

#define GL_GLEXT_PROTOTYPES 1
#include <GL/gl.h>

// r_local.h -- private refresh defs

#define ALIAS_BASE_SIZE_RATIO (1.0 / 11.0)
// normalizing factor so player model works out to about
//  1 pixel per triangle
#define MAX_LBM_HEIGHT 480

#define TILE_SIZE 128  // size of textures generated by R_GenTiledSurf

#define SKYSHIFT 7
#define SKYSIZE (1 << SKYSHIFT)
#define SKYMASK (SKYSIZE - 1)

#define BACKFACE_EPSILON 0.01

void R_ReadPointFile_f(void);
texture_t *R_TextureAnimation(texture_t *base, int frame);

typedef struct surfcache_s {
  struct surfcache_s *next;
  struct surfcache_s **owner;  // NULL is an empty chunk of memory
  int lightadj[MAXLIGHTMAPS];  // checked for strobe flush
  int dlight;
  int size;  // including header
  unsigned width;
  unsigned height;  // DEBUG only needed for debug
  float mipscale;
  struct texture_s *texture;  // checked for animating textures
  byte data[4];               // width*height elements
} surfcache_t;

typedef struct {
  byte *surfdat;     // destination for generated surface
  int rowbytes;      // destination logical width in bytes
  msurface_t *surf;  // description for surface to generate
  fixed8_t lightadj[MAXLIGHTMAPS];
  // adjust for lightmap levels for dynamic lighting
  texture_t *texture;  // corrected for animating textures
  int surfmip;         // mipmapped ratio of surface texels / world pixels
  int surfwidth;       // in mipmapped texels
  int surfheight;      // in mipmapped texels
} drawsurf_t;

//====================================================

//
// view origin
//
extern vec3_t vup;
extern vec3_t vpn;
extern vec3_t vright;
extern vec3_t r_origin;

//
// screen size info
//
extern mleaf_t *r_viewleaf, *r_oldviewleaf;
extern int d_lightstylevalue[256];  // 8.8 fraction of base light value

extern cvar_t r_norefresh;
extern cvar_t r_drawentities;
extern cvar_t r_drawworld;
extern cvar_t r_drawviewmodel;
extern cvar_t r_waterwarp;
extern cvar_t r_fullbright;
extern cvar_t r_lightmap;
extern cvar_t r_shadows;
extern cvar_t r_wateralpha;
extern cvar_t r_lavaalpha;
extern cvar_t r_telealpha;
extern cvar_t r_slimealpha;
extern cvar_t r_dynamic;
extern cvar_t r_novis;

extern cvar_t gl_clear;
extern cvar_t gl_cull;
extern cvar_t gl_smoothmodels;
extern cvar_t gl_affinemodels;
extern cvar_t gl_polyblend;
extern cvar_t gl_flashblend;
extern cvar_t gl_nocolors;

extern cvar_t gl_playermip;

extern cvar_t gl_subdivide_size;
extern float load_subdivide_size;  // johnfitz -- remember what subdivide_size
                                   // value was when this map was loaded

extern int gl_stencilbits;

// johnfitz -- rendering statistics
extern int rs_brushpolys, rs_aliaspolys, rs_skypolys, rs_particles, rs_fogpolys;
extern int rs_dynamiclightmaps, rs_brushpasses, rs_aliaspasses, rs_skypasses;
extern float rs_megatexels;

#define CONSOLE_RESPAM_TIME 3  // seconds between repeated warning messages

// johnfitz -- moved here from r_brush.c
extern int gl_lightmap_format, lightmap_bytes;
#define MAX_LIGHTMAPS 512  // johnfitz -- was 64
extern uint32_t
    lightmap_textures[MAX_LIGHTMAPS];  // johnfitz -- changed to an array

typedef struct glsl_attrib_binding_s {
  const char *name;
  GLuint attrib;
} glsl_attrib_binding_t;

extern float map_wateralpha, map_lavaalpha, map_telealpha,
    map_slimealpha;  // ericw

// johnfitz -- fog functions called from outside gl_fog.c
void Fog_EnableGFog(void);
void Fog_DisableGFog(void);
void Fog_SetupFrame(void);
void Fog_Init(void);

void R_AnimateLight(void);
void R_CullSurfaces(void);
qboolean R_CullModelForEntity(entity_t *e);
void R_RotateForEntity(vec3_t origin, vec3_t angles);

void R_DrawWorld(void);
void R_DrawAliasModel(entity_t *e);
void R_DrawBrushModel(entity_t *e);
void R_DrawSpriteModel(entity_t *e);

void R_DrawTextureChains_Water(qmodel_t *model, entity_t *ent,
                               texchain_t chain);

void R_RenderDlights(void);
void GL_BuildLightmaps(void);
void GL_BuildBModelVertexBuffer(void);
void R_RebuildAllLightmaps(void);

void R_LightPoint(vec3_t p);

void GL_SubdivideSurface(msurface_t *fa);
void R_BuildLightMap(msurface_t *surf, byte *dest, int stride);
void R_RenderDynamicLightmaps(msurface_t *fa);
void R_UploadLightmaps(void);

void GL_DrawAliasShadow(entity_t *e);
void DrawGLTriangleFan(glpoly_t *p);
void DrawGLPoly(glpoly_t *p);
void DrawWaterPoly(glpoly_t *p);
void GL_MakeAliasModelDisplayLists(qmodel_t *m, aliashdr_t *hdr);

void R_ClearTextureChains(qmodel_t *mod, texchain_t chain);
void R_ChainSurface(msurface_t *surf, texchain_t chain);
void R_DrawTextureChains(qmodel_t *model, entity_t *ent, texchain_t chain);
void R_DrawWorld_Water(void);

float GL_WaterAlphaForSurface(msurface_t *fa);

#endif /* __GLQUAKE_H */
