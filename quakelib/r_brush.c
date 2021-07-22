// SPDX-License-Identifier: GPL-2.0-or-later
// r_brush.c: brush model rendering. renamed from r_surf.c

#include "quakedef.h"

extern cvar_t gl_fullbrights, r_drawflat, gl_overbright,
    r_oldwater;         // johnfitz
extern cvar_t gl_zfix;  // QuakeSpasm z-fighting fix

int gl_lightmap_format;
int lightmap_bytes = 4;

#define BLOCK_WIDTH 128
#define BLOCK_HEIGHT 128

uint32_t lightmap_textures[MAX_LIGHTMAPS];  // johnfitz -- changed to an array

unsigned blocklights[BLOCK_WIDTH * BLOCK_HEIGHT *
                     3];  // johnfitz -- was 18*18, added lit support (*3) and
                          // loosened surface extents maximum
                          // (BLOCK_WIDTH*BLOCK_HEIGHT)

typedef struct glRect_s {
  unsigned char l, t, w, h;
} glRect_t;

glpoly_t *lightmap_polys[MAX_LIGHTMAPS];  // THERJAK -- extern use
qboolean lightmap_modified[MAX_LIGHTMAPS];
glRect_t lightmap_rectchange[MAX_LIGHTMAPS];

int allocated[MAX_LIGHTMAPS][BLOCK_WIDTH];
int last_lightmap_allocated;  // ericw -- optimization: remember the index of
                              // the last lightmap AllocBlock stored a surf in

// the lightmap texture data needs to be kept in
// main memory so texsubimage can update properly
byte lightmaps[4 * MAX_LIGHTMAPS * BLOCK_WIDTH * BLOCK_HEIGHT];

/*
===============
R_TextureAnimation

Returns the proper texture for a given time and base texture
===============
*/
texture_t *R_TextureAnimation(texture_t *base, int frame) {
  int relative;
  int count;

  if (frame)
    if (base->alternate_anims) base = base->alternate_anims;

  if (!base->anim_total) return base;

  relative = (int)(CL_Time() * 10) % base->anim_total;

  count = 0;
  while (base->anim_min > relative || base->anim_max <= relative) {
    base = base->anim_next;
    if (!base) Go_Error("R_TextureAnimation: broken cycle");
    if (++count > 100) Go_Error("R_TextureAnimation: infinite cycle");
  }

  return base;
}

/*
================
DrawGLPoly
================
*/
void DrawGLPoly(glpoly_t *p) {
  float *v;
  int i;

  glBegin(GL_TRIANGLE_FAN);
  v = p->verts[0];
  for (i = 0; i < p->numverts; i++, v += VERTEXSIZE) {
    glTexCoord2f(v[3], v[4]);
    glVertex3fv(v);
  }
  glEnd();
}

/*
=============================================================

        BRUSH MODELS

=============================================================
*/

/*
=================
R_DrawBrushModel
=================
*/
void R_DrawBrushModel(entity_t *e) {
  int i, k;
  msurface_t *psurf;
  float dot;
  mplane_t *pplane;
  qmodel_t *clmodel;
  vec3_t modelorg;

  if (R_CullModelForEntity(e)) return;

  clmodel = e->model;

  vec3_t vieworg = {R_Refdef_vieworg(0), R_Refdef_vieworg(1),
                    R_Refdef_vieworg(2)};
  VectorSubtract(vieworg, e->origin, modelorg);
  if (e->angles[0] || e->angles[1] || e->angles[2]) {
    vec3_t temp;
    vec3_t forward, right, up;

    VectorCopy(modelorg, temp);
    AngleVectors(e->angles, forward, right, up);
    modelorg[0] = DotProduct(temp, forward);
    modelorg[1] = -DotProduct(temp, right);
    modelorg[2] = DotProduct(temp, up);
  }

  psurf = &clmodel->surfaces[clmodel->firstmodelsurface];

  // calculate dynamic lighting for bmodel if it's not an
  // instanced model
  if (clmodel->firstmodelsurface != 0 && !Cvar_GetValue(&gl_flashblend)) {
    R_MarkLights(clmodel->nodes + clmodel->hulls[0].firstclipnode);
  }

  glPushMatrix();
  if (Cvar_GetValue(&gl_zfix)) {
    e->origin[0] -= DIST_EPSILON;
    e->origin[1] -= DIST_EPSILON;
    e->origin[2] -= DIST_EPSILON;
  }

  glTranslatef(e->origin[0], e->origin[1], e->origin[2]);
  glRotatef(e->angles[1], 0, 0, 1);
  // stupid quake bug, it should be -angles[0]
  glRotatef(e->angles[0], 0, 1, 0);
  glRotatef(e->angles[2], 1, 0, 0);

  if (Cvar_GetValue(&gl_zfix)) {
    e->origin[0] += DIST_EPSILON;
    e->origin[1] += DIST_EPSILON;
    e->origin[2] += DIST_EPSILON;
  }

  R_ClearTextureChains(clmodel, chain_model);
  for (i = 0; i < clmodel->nummodelsurfaces; i++, psurf++) {
    pplane = psurf->plane;
    dot = DotProduct(modelorg, pplane->normal) - pplane->dist;
    if (((psurf->flags & SURF_PLANEBACK) && (dot < -BACKFACE_EPSILON)) ||
        (!(psurf->flags & SURF_PLANEBACK) && (dot > BACKFACE_EPSILON))) {
      R_ChainSurface(psurf, chain_model);
      rs_brushpolys++;
    }
  }

  R_DrawTextureChains(clmodel, e, chain_model);
  R_DrawTextureChains_Water(clmodel, e, chain_model);

  glPopMatrix();
}

/*
=============================================================

        LIGHTMAPS

=============================================================
*/

/*
================
R_RenderDynamicLightmaps
called during rendering
================
*/
void R_RenderDynamicLightmaps(msurface_t *fa) {
  byte *base;
  int maps;
  glRect_t *theRect;
  int smax, tmax;

  if (fa->flags & SURF_DRAWTILED)  // johnfitz -- not a lightmapped surface
    return;

  // add to lightmap chain
  fa->polys->chain = lightmap_polys[fa->lightmaptexturenum];
  lightmap_polys[fa->lightmaptexturenum] = fa->polys;

  // check for lightmap modification
  for (maps = 0; maps < MAXLIGHTMAPS && fa->styles[maps] != 255; maps++)
    if (d_lightstylevalue[fa->styles[maps]] != fa->cached_light[maps])
      goto dynamic;

  if (fa->dlightframe == R_framecount()  // dynamic this frame
      || fa->cached_dlight)              // dynamic previously
  {
  dynamic:
    if (Cvar_GetValue(&r_dynamic)) {
      lightmap_modified[fa->lightmaptexturenum] = true;
      theRect = &lightmap_rectchange[fa->lightmaptexturenum];
      if (fa->light_t < theRect->t) {
        if (theRect->h) theRect->h += theRect->t - fa->light_t;
        theRect->t = fa->light_t;
      }
      if (fa->light_s < theRect->l) {
        if (theRect->w) theRect->w += theRect->l - fa->light_s;
        theRect->l = fa->light_s;
      }
      smax = (fa->extents[0] >> 4) + 1;
      tmax = (fa->extents[1] >> 4) + 1;
      if ((theRect->w + theRect->l) < (fa->light_s + smax))
        theRect->w = (fa->light_s - theRect->l) + smax;
      if ((theRect->h + theRect->t) < (fa->light_t + tmax))
        theRect->h = (fa->light_t - theRect->t) + tmax;
      base = lightmaps + fa->lightmaptexturenum * lightmap_bytes * BLOCK_WIDTH *
                             BLOCK_HEIGHT;
      base += fa->light_t * BLOCK_WIDTH * lightmap_bytes +
              fa->light_s * lightmap_bytes;
      R_BuildLightMap(fa, base, BLOCK_WIDTH * lightmap_bytes);
    }
  }
}

/*
========================
AllocBlock -- returns a texture number and the position inside it
========================
*/
// THERJAK -- dep
int AllocBlock(int w, int h, int *x, int *y) {
  int i, j;
  int best, best2;
  int texnum;

  // ericw -- rather than searching starting at lightmap 0 every time,
  // start at the last lightmap we allocated a surface in.
  // This makes AllocBlock much faster on large levels (can shave off 3+ seconds
  // of load time on a level with 180 lightmaps), at a cost of not quite packing
  // lightmaps as tightly vs. not doing this (uses ~5% more lightmaps)
  for (texnum = last_lightmap_allocated; texnum < MAX_LIGHTMAPS;
       texnum++, last_lightmap_allocated++) {
    best = BLOCK_HEIGHT;

    for (i = 0; i < BLOCK_WIDTH - w; i++) {
      best2 = 0;

      for (j = 0; j < w; j++) {
        if (allocated[texnum][i + j] >= best) break;
        if (allocated[texnum][i + j] > best2) best2 = allocated[texnum][i + j];
      }
      if (j == w) {  // this is a valid spot
        *x = i;
        *y = best = best2;
      }
    }

    if (best + h > BLOCK_HEIGHT) continue;

    for (i = 0; i < w; i++) allocated[texnum][*x + i] = best + h;

    return texnum;
  }

  Go_Error("AllocBlock: full");
  return 0;  // johnfitz -- shut up compiler
}

/*
========================
GL_CreateSurfaceLightmap
========================
*/
// THERJAK -- dep
void GL_CreateSurfaceLightmap(msurface_t *surf) {
  int smax, tmax;
  byte *base;

  smax = (surf->extents[0] >> 4) + 1;
  tmax = (surf->extents[1] >> 4) + 1;

  surf->lightmaptexturenum =
      AllocBlock(smax, tmax, &surf->light_s, &surf->light_t);
  base = lightmaps +
         surf->lightmaptexturenum * lightmap_bytes * BLOCK_WIDTH * BLOCK_HEIGHT;
  base += (surf->light_t * BLOCK_WIDTH + surf->light_s) * lightmap_bytes;
  R_BuildLightMap(surf, base, BLOCK_WIDTH * lightmap_bytes);
}

/*
================
BuildSurfaceDisplayList -- called at level load time
================
*/
// THERJAK -- dep
void BuildSurfaceDisplayList(msurface_t *fa, qmodel_t *currentmodel) {
  int i, lindex, lnumverts;
  medge_t *pedges, *r_pedge;
  float *vec;
  float s, t;
  glpoly_t *poly;

  // reconstruct the polygon
  pedges = currentmodel->edges;
  mvertex_t *r_pcurrentvertbase = currentmodel->vertexes;
  lnumverts = fa->numedges;

  //
  // draw texture
  //
  poly = (glpoly_t *)Hunk_Alloc(sizeof(glpoly_t) +
                                (lnumverts - 4) * VERTEXSIZE * sizeof(float));
  poly->next = fa->polys;
  fa->polys = poly;
  poly->numverts = lnumverts;

  for (i = 0; i < lnumverts; i++) {
    lindex = currentmodel->surfedges[fa->firstedge + i];

    if (lindex > 0) {
      r_pedge = &pedges[lindex];
      vec = r_pcurrentvertbase[r_pedge->v[0]].position;
    } else {
      r_pedge = &pedges[-lindex];
      vec = r_pcurrentvertbase[r_pedge->v[1]].position;
    }
    s = DotProduct(vec, fa->texinfo->vecs[0]) + fa->texinfo->vecs[0][3];
    s /= fa->texinfo->texture->width;

    t = DotProduct(vec, fa->texinfo->vecs[1]) + fa->texinfo->vecs[1][3];
    t /= fa->texinfo->texture->height;

    VectorCopy(vec, poly->verts[i]);
    poly->verts[i][3] = s;
    poly->verts[i][4] = t;

    //
    // lightmap texture coordinates
    //
    s = DotProduct(vec, fa->texinfo->vecs[0]) + fa->texinfo->vecs[0][3];
    s -= fa->texturemins[0];
    s += fa->light_s * 16;
    s += 8;
    s /= BLOCK_WIDTH * 16;  // fa->texinfo->texture->width;

    t = DotProduct(vec, fa->texinfo->vecs[1]) + fa->texinfo->vecs[1][3];
    t -= fa->texturemins[1];
    t += fa->light_t * 16;
    t += 8;
    t /= BLOCK_HEIGHT * 16;  // fa->texinfo->texture->height;

    poly->verts[i][5] = s;
    poly->verts[i][6] = t;
  }

  // johnfitz -- removed gl_keeptjunctions code
}

/*
==================
GL_BuildLightmaps -- called at level load time

Builds the lightmap texture
with all the surfaces from all brush models
==================
*/
// THERJAK -- this is next
void GL_BuildLightmaps(void) {
  char name[16];
  byte *data;
  int i, j;
  qmodel_t *m;

  memset(allocated, 0, sizeof(allocated));
  last_lightmap_allocated = 0;

  R_framecount_reset();
  R_framecount_inc();  // no dlightcache

  // johnfitz -- null out array (the gltexture objects themselves were already
  // freed by Mod_ClearAll)
  for (i = 0; i < MAX_LIGHTMAPS; i++) lightmap_textures[i] = GetNoTexture();
  // johnfitz

  gl_lightmap_format = GL_RGBA;  // FIXME: hardcoded for now!

  for (j = 1; j < MAX_MODELS; j++) {
    m = cl.model_precache[j];
    if (!m) break;
    if (m->name[0] == '*') continue;
    for (i = 0; i < m->numsurfaces; i++) {
      // johnfitz -- rewritten to use SURF_DRAWTILED instead of the sky/water
      // flags
      if (m->surfaces[i].flags & SURF_DRAWTILED) continue;
      GL_CreateSurfaceLightmap(m->surfaces + i);
      BuildSurfaceDisplayList(m->surfaces + i, m);
      // johnfitz
    }
  }

  //
  // upload all lightmaps that were filled
  //
  for (i = 0; i < MAX_LIGHTMAPS; i++) {
    if (!allocated[i][0]) break;  // no more used
    lightmap_modified[i] = false;
    lightmap_rectchange[i].l = BLOCK_WIDTH;
    lightmap_rectchange[i].t = BLOCK_HEIGHT;
    lightmap_rectchange[i].w = 0;
    lightmap_rectchange[i].h = 0;

    // johnfitz -- use texture manager
    sprintf(name, "lightmap%03i", i);
    data = lightmaps + i * BLOCK_WIDTH * BLOCK_HEIGHT * lightmap_bytes;
    lightmap_textures[i] =
        TexMgrLoadLightMapImage(cl.worldmodel, name, BLOCK_WIDTH, BLOCK_HEIGHT,
                                data, TEXPREF_LINEAR | TEXPREF_NOPICMIP);
    // johnfitz
  }

  // johnfitz -- warn about exceeding old limits
  if (i >= 64) Con_DWarning("%i lightmaps exceeds standard limit of 64.\n", i);
  // johnfitz
}

/*
=============================================================

        VBO support

=============================================================
*/

extern GLuint gl_bmodel_vbo;

/*
==================
GL_BuildBModelVertexBuffer

Deletes gl_bmodel_vbo if it already exists, then rebuilds it with all
surfaces from world + all brush models
==================
*/
void GL_BuildBModelVertexBufferOld(void) {
  unsigned int numverts, varray_bytes, varray_index;
  int i, j;
  qmodel_t *m;
  float *varray;
  /*
    // ask GL for a name for our VBO
    glDeleteBuffers(1, &gl_bmodel_vbo);
    glGenBuffers(1, &gl_bmodel_vbo);

    // count all verts in all models
    numverts = 0;
    for (j = 1; j < MAX_MODELS; j++) {
      m = cl.model_precache[j];
      if (!m || m->name[0] == '*' || m->Type != mod_brush) continue;

      for (i = 0; i < m->numsurfaces; i++) {
        numverts += m->surfaces[i].numedges;
      }
    }
    // build vertex array
    varray_bytes = VERTEXSIZE * sizeof(float) * numverts;
    varray = (float *)malloc(varray_bytes);
  */
  varray_index = 0;

  for (j = 1; j < MAX_MODELS; j++) {
    m = cl.model_precache[j];
    if (!m || m->name[0] == '*' || m->Type != mod_brush) continue;

    for (i = 0; i < m->numsurfaces; i++) {
      msurface_t *s = &m->surfaces[i];
      s->vbo_firstvert = varray_index;
      //      memcpy(&varray[VERTEXSIZE * varray_index], s->polys->verts,
      //             VERTEXSIZE * sizeof(float) * s->numedges);
      varray_index += s->numedges;
    }
  }
  /*
    // upload to GPU
    glBindBuffer(GL_ARRAY_BUFFER, gl_bmodel_vbo);
    glBufferData(GL_ARRAY_BUFFER, varray_bytes, varray, GL_STATIC_DRAW);
    free(varray);

  */
}

/*
===============
R_AddDynamicLights
===============
*/
// THERJAK -- dep
void R_AddDynamicLights(msurface_t *surf) {
  int lnum;
  int sd, td;
  float dist, rad, minlight;
  vec3_t impact, local;
  int s, t;
  int i;
  int smax, tmax;
  mtexinfo_t *tex;
  // johnfitz -- lit support via lordhavoc
  float cred, cgreen, cblue, brightness;
  unsigned *bl;
  // johnfitz

  smax = (surf->extents[0] >> 4) + 1;
  tmax = (surf->extents[1] >> 4) + 1;
  tex = surf->texinfo;

  for (lnum = 0; lnum < MAX_DLIGHTS; lnum++) {
    if (!(surf->dlightbits[lnum >> 5] & (1U << (lnum & 31))))
      continue;  // not lit by this light
    dlight_t *l = CL_Dlight(lnum);
    rad = l->radius;
    dist = DotProduct(l->origin, surf->plane->normal) - surf->plane->dist;
    rad -= fabs(dist);
    minlight = l->minlight;
    if (rad < minlight) continue;
    minlight = rad - minlight;

    for (i = 0; i < 3; i++) {
      impact[i] = l->origin[i] - surf->plane->normal[i] * dist;
    }

    local[0] = DotProduct(impact, tex->vecs[0]) + tex->vecs[0][3];
    local[1] = DotProduct(impact, tex->vecs[1]) + tex->vecs[1][3];

    local[0] -= surf->texturemins[0];
    local[1] -= surf->texturemins[1];

    // johnfitz -- lit support via lordhavoc
    bl = blocklights;
    cred = l->color[0] * 256.0f;
    cgreen = l->color[1] * 256.0f;
    cblue = l->color[2] * 256.0f;
    // johnfitz
    for (t = 0; t < tmax; t++) {
      td = local[1] - t * 16;
      if (td < 0) td = -td;
      for (s = 0; s < smax; s++) {
        sd = local[0] - s * 16;
        if (sd < 0) sd = -sd;
        if (sd > td)
          dist = sd + (td >> 1);
        else
          dist = td + (sd >> 1);
        if (dist < minlight)
        // johnfitz -- lit support via lordhavoc
        {
          brightness = rad - dist;
          bl[0] += (int)(brightness * cred);
          bl[1] += (int)(brightness * cgreen);
          bl[2] += (int)(brightness * cblue);
        }
        bl += 3;
        // johnfitz
      }
    }
  }
}

/*
===============
R_BuildLightMap -- johnfitz -- revised for lit support via lordhavoc

Combine and scale multiple lightmaps into the 8.8 format in blocklights
===============
*/
// THERJAK -- dep
void R_BuildLightMap(msurface_t *surf, byte *dest, int stride) {
  int smax, tmax;
  int r, g, b;
  int i, j, size;
  byte *lightmap;
  unsigned scale;
  int maps;
  unsigned *bl;

  surf->cached_dlight = (surf->dlightframe == R_framecount());

  smax = (surf->extents[0] >> 4) + 1;
  tmax = (surf->extents[1] >> 4) + 1;
  size = smax * tmax;
  lightmap = surf->samples;

  if (cl.worldmodel->lightdata) {
    // clear to no light
    memset(&blocklights[0], 0,
           size * 3 *
               sizeof(unsigned int));  // johnfitz -- lit support via lordhavoc

    // add all the lightmaps
    if (lightmap) {
      for (maps = 0; maps < MAXLIGHTMAPS && surf->styles[maps] != 255; maps++) {
        scale = d_lightstylevalue[surf->styles[maps]];
        surf->cached_light[maps] = scale;  // 8.8 fraction
        // johnfitz -- lit support via lordhavoc
        bl = blocklights;
        for (i = 0; i < size; i++) {
          *bl++ += *lightmap++ * scale;
          *bl++ += *lightmap++ * scale;
          *bl++ += *lightmap++ * scale;
        }
        // johnfitz
      }
    }

    // add all the dynamic lights
    if (surf->dlightframe == R_framecount()) R_AddDynamicLights(surf);
  } else {
    // set to full bright if no light data
    memset(&blocklights[0], 255,
           size * 3 *
               sizeof(unsigned int));  // johnfitz -- lit support via lordhavoc
  }

  // bound, invert, and shift
  // store:
  stride -= smax * 4;
  bl = blocklights;
  for (i = 0; i < tmax; i++, dest += stride) {
    for (j = 0; j < smax; j++) {
      if (Cvar_GetValue(&gl_overbright)) {
        r = *bl++ >> 8;
        g = *bl++ >> 8;
        b = *bl++ >> 8;
      } else {
        r = *bl++ >> 7;
        g = *bl++ >> 7;
        b = *bl++ >> 7;
      }
      *dest++ = (r > 255) ? 255 : r;
      *dest++ = (g > 255) ? 255 : g;
      *dest++ = (b > 255) ? 255 : b;
      *dest++ = 255;
    }
  }
}

/*
===============
R_UploadLightmap -- johnfitz -- uploads the modified lightmap to opengl if
necessary

assumes lightmap texture is already bound
===============
*/
static void R_UploadLightmap(int lmap) {
  glRect_t *theRect;

  if (!lightmap_modified[lmap]) return;

  lightmap_modified[lmap] = false;

  theRect = &lightmap_rectchange[lmap];
  glTexSubImage2D(GL_TEXTURE_2D, 0, 0, theRect->t, BLOCK_WIDTH, theRect->h,
                  gl_lightmap_format, GL_UNSIGNED_BYTE,
                  lightmaps + (lmap * BLOCK_HEIGHT + theRect->t) * BLOCK_WIDTH *
                                  lightmap_bytes);
  theRect->l = BLOCK_WIDTH;
  theRect->t = BLOCK_HEIGHT;
  theRect->h = 0;
  theRect->w = 0;

  rs_dynamiclightmaps++;
}

void R_UploadLightmaps(void) {
  int lmap;

  for (lmap = 0; lmap < MAX_LIGHTMAPS; lmap++) {
    if (!lightmap_modified[lmap]) continue;

    GLBind(lightmap_textures[lmap]);
    R_UploadLightmap(lmap);
  }
}

/*
================
R_RebuildAllLightmaps -- johnfitz -- called when gl_overbright gets toggled
================
*/
void R_RebuildAllLightmaps(void) {
  int i, j;
  qmodel_t *mod;
  msurface_t *fa;
  byte *base;

  if (!cl.worldmodel)  // is this the correct test?
    return;

  // for each surface in each model, rebuild lightmap with new scale
  for (i = 1; i < MAX_MODELS; i++) {
    if (!(mod = cl.model_precache[i])) continue;
    fa = &mod->surfaces[mod->firstmodelsurface];
    for (j = 0; j < mod->nummodelsurfaces; j++, fa++) {
      if (fa->flags & SURF_DRAWTILED) continue;
      base = lightmaps + fa->lightmaptexturenum * lightmap_bytes * BLOCK_WIDTH *
                             BLOCK_HEIGHT;
      base += fa->light_t * BLOCK_WIDTH * lightmap_bytes +
              fa->light_s * lightmap_bytes;
      R_BuildLightMap(fa, base, BLOCK_WIDTH * lightmap_bytes);
    }
  }

  // for each lightmap, upload it
  for (i = 0; i < MAX_LIGHTMAPS; i++) {
    if (!allocated[i][0]) break;
    GLBind(lightmap_textures[i]);
    glTexSubImage2D(
        GL_TEXTURE_2D, 0, 0, 0, BLOCK_WIDTH, BLOCK_HEIGHT, gl_lightmap_format,
        GL_UNSIGNED_BYTE,
        lightmaps + i * BLOCK_WIDTH * BLOCK_HEIGHT * lightmap_bytes);
  }
}
