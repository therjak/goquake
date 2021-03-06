// gl_sky.c

#include "quakedef.h"

#define MAX_CLIP_VERTS 64

float Fog_GetDensity(void);
float *Fog_GetColor(void);

extern int rs_skypolys;   // for r_speeds readout
extern int rs_skypasses;  // for r_speeds readout
float skyflatcolor[3];
float skymins[2][6], skymaxs[2][6];

char skybox_name[32] = "";  // name of current skybox, or "" if no skybox

uint32_t skybox_textures[6];
uint32_t solidskytexture2, alphaskytexture2;

extern cvar_t gl_farclip;
cvar_t r_fastsky;
cvar_t r_sky_quality;
cvar_t r_skyalpha;

int skytexorder[6] = {0, 2, 1, 3, 4, 5};  // for skybox

vec3_t skyclip[6] = {{1, 1, 0}, {1, -1, 0}, {0, -1, 1},
                     {0, 1, 1}, {1, 0, 1},  {-1, 0, 1}};

int st_to_vec[6][3] = {
    {3, -1, 2}, {-3, 1, 2}, {1, 3, 2}, {-1, -3, 2}, {-2, -1, 3},  // straight up
    {2, -1, -3}  // straight down
};

int vec_to_st[6][3] = {{-2, 3, 1},  {2, 3, -1},  {1, 3, 2},
                       {-1, 3, -2}, {-2, -1, 3}, {-2, 1, -3}};

float skyfog;  // ericw

/*
=============
Sky_Init
=============
*/
void Sky_Init(void) {
  int i;

  Cvar_FakeRegister(&r_fastsky, "r_fastsky");
  Cvar_FakeRegister(&r_sky_quality, "r_sky_quality");
  Cvar_FakeRegister(&r_skyalpha, "r_skyalpha");

  for (i = 0; i < 6; i++) skybox_textures[i] = 0;
}

//==============================================================================
//
//  PROCESS SKY SURFS
//
//==============================================================================

/*
=================
Sky_ProjectPoly

update sky bounds
=================
*/
void Sky_ProjectPoly(int nump, vec3_t vecs) {
  int i, j;
  vec3_t v, av;
  float s, t, dv;
  int axis;
  float *vp;

  // decide which face it maps to
  VectorCopy(vec3_origin, v);
  for (i = 0, vp = vecs; i < nump; i++, vp += 3) {
    VectorAdd(vp, v, v);
  }
  av[0] = fabs(v[0]);
  av[1] = fabs(v[1]);
  av[2] = fabs(v[2]);
  if (av[0] > av[1] && av[0] > av[2]) {
    if (v[0] < 0)
      axis = 1;
    else
      axis = 0;
  } else if (av[1] > av[2] && av[1] > av[0]) {
    if (v[1] < 0)
      axis = 3;
    else
      axis = 2;
  } else {
    if (v[2] < 0)
      axis = 5;
    else
      axis = 4;
  }

  // project new texture coords
  for (i = 0; i < nump; i++, vecs += 3) {
    j = vec_to_st[axis][2];
    if (j > 0)
      dv = vecs[j - 1];
    else
      dv = -vecs[-j - 1];

    j = vec_to_st[axis][0];
    if (j < 0)
      s = -vecs[-j - 1] / dv;
    else
      s = vecs[j - 1] / dv;
    j = vec_to_st[axis][1];
    if (j < 0)
      t = -vecs[-j - 1] / dv;
    else
      t = vecs[j - 1] / dv;

    if (s < skymins[0][axis]) skymins[0][axis] = s;
    if (t < skymins[1][axis]) skymins[1][axis] = t;
    if (s > skymaxs[0][axis]) skymaxs[0][axis] = s;
    if (t > skymaxs[1][axis]) skymaxs[1][axis] = t;
  }
}

/*
=================
Sky_ClipPoly
=================
*/
void Sky_ClipPoly(int nump, vec3_t vecs, int stage) {
  float *norm;
  float *v;
  qboolean front, back;
  float d, e;
  float dists[MAX_CLIP_VERTS];
  int sides[MAX_CLIP_VERTS];
  vec3_t newv[2][MAX_CLIP_VERTS];
  int newc[2];
  int i, j;

  if (nump > MAX_CLIP_VERTS - 2) Go_Error("Sky_ClipPoly: MAX_CLIP_VERTS");
  if (stage == 6)  // fully clipped
  {
    Sky_ProjectPoly(nump, vecs);
    return;
  }

  front = back = false;
  norm = skyclip[stage];
  for (i = 0, v = vecs; i < nump; i++, v += 3) {
    d = DotProduct(v, norm);
    if (d > ON_EPSILON) {
      front = true;
      sides[i] = SIDE_FRONT;
    } else if (d < ON_EPSILON) {
      back = true;
      sides[i] = SIDE_BACK;
    } else
      sides[i] = SIDE_ON;
    dists[i] = d;
  }

  if (!front || !back) {  // not clipped
    Sky_ClipPoly(nump, vecs, stage + 1);
    return;
  }

  // clip it
  sides[i] = sides[0];
  dists[i] = dists[0];
  VectorCopy(vecs, (vecs + (i * 3)));
  newc[0] = newc[1] = 0;

  for (i = 0, v = vecs; i < nump; i++, v += 3) {
    switch (sides[i]) {
      case SIDE_FRONT:
        VectorCopy(v, newv[0][newc[0]]);
        newc[0]++;
        break;
      case SIDE_BACK:
        VectorCopy(v, newv[1][newc[1]]);
        newc[1]++;
        break;
      case SIDE_ON:
        VectorCopy(v, newv[0][newc[0]]);
        newc[0]++;
        VectorCopy(v, newv[1][newc[1]]);
        newc[1]++;
        break;
    }

    if (sides[i] == SIDE_ON || sides[i + 1] == SIDE_ON ||
        sides[i + 1] == sides[i])
      continue;

    d = dists[i] / (dists[i] - dists[i + 1]);
    for (j = 0; j < 3; j++) {
      e = v[j] + d * (v[j + 3] - v[j]);
      newv[0][newc[0]][j] = e;
      newv[1][newc[1]][j] = e;
    }
    newc[0]++;
    newc[1]++;
  }

  // continue
  Sky_ClipPoly(newc[0], newv[0][0], stage + 1);
  Sky_ClipPoly(newc[1], newv[1][0], stage + 1);
}

/*
================
Sky_ProcessPoly
================
*/
void Sky_ProcessPoly(glpoly_t *p) {
  int i;
  vec3_t verts[MAX_CLIP_VERTS];

  // draw it
  DrawGLPoly(p);
  rs_brushpasses++;

  // update sky bounds
  if (!Cvar_GetValue(&r_fastsky)) {
    for (i = 0; i < p->numverts; i++)
      VectorSubtract(p->verts[i], r_origin, verts[i]);
    Sky_ClipPoly(p->numverts, verts[0], 0);
  }
}

/*
================
Sky_ProcessTextureChains -- handles sky polys in world model
================
*/
void Sky_ProcessTextureChains(void) {
  int i;
  msurface_t *s;
  texture_t *t;

  for (i = 0; i < cl.worldmodel->numtextures; i++) {
    t = cl.worldmodel->textures[i];

    if (!t || !t->texturechains[chain_world] ||
        !(t->texturechains[chain_world]->flags & SURF_DRAWSKY))
      continue;

    for (s = t->texturechains[chain_world]; s; s = s->texturechain)
      if (!s->culled) Sky_ProcessPoly(s->polys);
  }
}

/*
================
Sky_ProcessEntities -- handles sky polys on brush models
================
*/
void Sky_ProcessEntities(void) {
  entity_t *e;
  msurface_t *s;
  glpoly_t *p;
  int i, j, k, mark;
  float dot;
  qboolean rotated;
  vec3_t temp, forward, right, up, modelorg;

  if (!Cvar_GetValue(&r_drawentities)) return;

  vec3_t vieworg = {R_Refdef_vieworg(0), R_Refdef_vieworg(1),
                    R_Refdef_vieworg(2)};

  for (i = 0; i < VisibleEntitiesNum(); i++) {
    e = VisibleEntity(i);

    if (e->model->Type != mod_brush) continue;

    if (R_CullModelForEntity(e)) continue;

    if (e->alpha2 == ENTALPHA_ZERO) continue;

    VectorSubtract(vieworg, e->origin, modelorg);
    if (e->angles[0] || e->angles[1] || e->angles[2]) {
      rotated = true;
      AngleVectors(e->angles, forward, right, up);
      VectorCopy(modelorg, temp);
      modelorg[0] = DotProduct(temp, forward);
      modelorg[1] = -DotProduct(temp, right);
      modelorg[2] = DotProduct(temp, up);
    } else
      rotated = false;

    s = &e->model->surfaces[e->model->firstmodelsurface];

    for (j = 0; j < e->model->nummodelsurfaces; j++, s++) {
      if (s->flags & SURF_DRAWSKY) {
        dot = DotProduct(modelorg, s->plane->normal) - s->plane->dist;
        if (((s->flags & SURF_PLANEBACK) && (dot < -BACKFACE_EPSILON)) ||
            (!(s->flags & SURF_PLANEBACK) && (dot > BACKFACE_EPSILON))) {
          // copy the polygon and translate manually, since Sky_ProcessPoly
          // needs it to be in world space
          mark = Hunk_LowMark();
          p = (glpoly_t *)Hunk_Alloc(
              sizeof(*s->polys));  // FIXME: don't allocate for each poly
          p->numverts = s->polys->numverts;
          for (k = 0; k < p->numverts; k++) {
            if (rotated) {
              p->verts[k][0] = e->origin[0] +
                               s->polys->verts[k][0] * forward[0] -
                               s->polys->verts[k][1] * right[0] +
                               s->polys->verts[k][2] * up[0];
              p->verts[k][1] = e->origin[1] +
                               s->polys->verts[k][0] * forward[1] -
                               s->polys->verts[k][1] * right[1] +
                               s->polys->verts[k][2] * up[1];
              p->verts[k][2] = e->origin[2] +
                               s->polys->verts[k][0] * forward[2] -
                               s->polys->verts[k][1] * right[2] +
                               s->polys->verts[k][2] * up[2];
            } else
              VectorAdd(s->polys->verts[k], e->origin, p->verts[k]);
          }
          Sky_ProcessPoly(p);
          Hunk_FreeToLowMark(mark);
        }
      }
    }
  }
}

//==============================================================================
//
//  RENDER SKYBOX
//
//==============================================================================

/*
==============
Sky_EmitSkyBoxVertex
==============
*/
void Sky_EmitSkyBoxVertex(float s, float t, int axis) {
  vec3_t v, b;
  int j, k;
  float w, h;

  b[0] = s * Cvar_GetValue(&gl_farclip) / sqrt(3.0);
  b[1] = t * Cvar_GetValue(&gl_farclip) / sqrt(3.0);
  b[2] = Cvar_GetValue(&gl_farclip) / sqrt(3.0);

  for (j = 0; j < 3; j++) {
    k = st_to_vec[axis][j];
    if (k < 0)
      v[j] = -b[-k - 1];
    else
      v[j] = b[k - 1];
    v[j] += r_origin[j];
  }

  // convert from range [-1,1] to [0,1]
  s = (s + 1) * 0.5;
  t = (t + 1) * 0.5;

  // avoid bilerp seam
  w = GetTextureWidth(skybox_textures[skytexorder[axis]]);   // ->width;
  h = GetTextureHeight(skybox_textures[skytexorder[axis]]);  //->height;
  s = s * (w - 1) / w + 0.5 / w;
  t = t * (h - 1) / h + 0.5 / h;

  t = 1.0 - t;
  glTexCoord2f(s, t);
  glVertex3fv(v);
}

/*
==============
Sky_DrawSkyBox

FIXME: eliminate cracks by adding an extra vert on tjuncs
==============
*/
void Sky_DrawSkyBox(void) {
  int i;

  for (i = 0; i < 6; i++) {
    if (skymins[0][i] >= skymaxs[0][i] || skymins[1][i] >= skymaxs[1][i])
      continue;

    GLBind(skybox_textures[skytexorder[i]]);

#if 1  // FIXME: this is to avoid tjunctions until i can do it the right way
    skymins[0][i] = -1;
    skymins[1][i] = -1;
    skymaxs[0][i] = 1;
    skymaxs[1][i] = 1;
#endif
    glBegin(GL_QUADS);
    Sky_EmitSkyBoxVertex(skymins[0][i], skymins[1][i], i);
    Sky_EmitSkyBoxVertex(skymins[0][i], skymaxs[1][i], i);
    Sky_EmitSkyBoxVertex(skymaxs[0][i], skymaxs[1][i], i);
    Sky_EmitSkyBoxVertex(skymaxs[0][i], skymins[1][i], i);
    glEnd();

    rs_skypolys++;
    rs_skypasses++;

    if (Fog_GetDensity() > 0 && skyfog > 0) {
      float *c;

      c = Fog_GetColor();
      glEnable(GL_BLEND);
      glDisable(GL_TEXTURE_2D);
      glColor4f(c[0], c[1], c[2], CLAMP(0.0, skyfog, 1.0));

      glBegin(GL_QUADS);
      Sky_EmitSkyBoxVertex(skymins[0][i], skymins[1][i], i);
      Sky_EmitSkyBoxVertex(skymins[0][i], skymaxs[1][i], i);
      Sky_EmitSkyBoxVertex(skymaxs[0][i], skymaxs[1][i], i);
      Sky_EmitSkyBoxVertex(skymaxs[0][i], skymins[1][i], i);
      glEnd();

      glColor3f(1, 1, 1);
      glEnable(GL_TEXTURE_2D);
      glDisable(GL_BLEND);

      rs_skypasses++;
    }
  }
}
