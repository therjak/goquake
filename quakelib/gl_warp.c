// gl_warp.c -- warping animation support

#include "quakedef.h"

extern cvar_t r_drawflat;

cvar_t r_oldwater;      // = {"r_oldwater", "0", CVAR_ARCHIVE};
cvar_t r_waterquality;  // = {"r_waterquality", "8", CVAR_NONE};
cvar_t r_waterwarp;     //= {"r_waterwarp", "1", CVAR_NONE};

float load_subdivide_size;  // johnfitz -- remember what subdivide_size value
                            // was when this map was loaded

float turbsin[] = {
#include "gl_warp_sin.h"
};

#define WARPCALC(s, t)                                                  \
  ((s + turbsin[(int)((t * 2) + (CL_Time() * (128.0 / M_PI))) & 255]) * \
   (1.0 / 64))  // johnfitz -- correct warp
#define WARPCALC2(s, t)                                                   \
  ((s + turbsin[(int)((t * 0.125 + CL_Time()) * (128.0 / M_PI)) & 255]) * \
   (1.0 / 64))  // johnfitz -- old warp

//==============================================================================
//
//  OLD-STYLE WATER
//
//==============================================================================

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
      if (*v < mins[j]) mins[j] = *v;
      if (*v > maxs[j]) maxs[j] = *v;
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

  if (numverts > 60) Go_Error_I("numverts = %v", numverts);

  BoundPoly(numverts, verts, mins, maxs);

  for (i = 0; i < 3; i++) {
    m = (mins[i] + maxs[i]) * 0.5;
    m = Cvar_GetValue(&gl_subdivide_size) *
        floor(m / Cvar_GetValue(&gl_subdivide_size) + 0.5);
    if (maxs[i] - m < 8) continue;
    if (m - mins[i] < 8) continue;

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
      if (dist[j] == 0 || dist[j + 1] == 0) continue;
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

/*
================
DrawWaterPoly -- johnfitz
================
*/
void DrawWaterPoly(glpoly_t *p) {
  float *v;
  int i;

  if (load_subdivide_size > 48) {
    glBegin(GL_POLYGON);
    v = p->verts[0];
    for (i = 0; i < p->numverts; i++, v += VERTEXSIZE) {
      glTexCoord2f(WARPCALC2(v[3], v[4]), WARPCALC2(v[4], v[3]));
      glVertex3fv(v);
    }
    glEnd();
  } else {
    glBegin(GL_POLYGON);
    v = p->verts[0];
    for (i = 0; i < p->numverts; i++, v += VERTEXSIZE) {
      glTexCoord2f(WARPCALC(v[3], v[4]), WARPCALC(v[4], v[3]));
      glVertex3fv(v);
    }
    glEnd();
  }
}

//==============================================================================
//
//  RENDER-TO-FRAMEBUFFER WATER
//
//==============================================================================

/*
=============
R_UpdateWarpTextures -- johnfitz -- each frame, update warping textures
=============
*/
void R_UpdateWarpTextures(void) {
  texture_t *tx;
  int i;
  float x, y, x2, warptess;

  if (Cvar_GetValue(&r_oldwater) || CL_Paused())
    return;

  warptess = 128.0 / CLAMP(3.0, floor(Cvar_GetValue(&r_waterquality)), 64.0);

  for (i = 0; i < cl.worldmodel->numtextures; i++) {
    if (!(tx = cl.worldmodel->textures[i])) continue;

    if (!tx->update_warp) continue;

    // render warp
    GLSetCanvas(CANVAS_WARPIMAGE);
    GLBind(tx->gltexture);
    for (x = 0.0; x < 128.0; x = x2) {
      x2 = x + warptess;
      glBegin(GL_TRIANGLE_STRIP);
      for (y = 0.0; y < 128.01; y += warptess)  // .01 for rounding errors
      {
        glTexCoord2f(WARPCALC(x, y), WARPCALC(y, x));
        glVertex2f(x, y);
        glTexCoord2f(WARPCALC(x2, y), WARPCALC(y, x2));
        glVertex2f(x2, y);
      }
      glEnd();
    }

    // copy to texture
    GLBind(tx->warpimage);
    glCopyTexSubImage2D(GL_TEXTURE_2D, 0, 0, 0, 0,
                        GL_Height() - GL_warpimagesize(), GL_warpimagesize(),
                        GL_warpimagesize());

    tx->update_warp = false;
  }

  // ericw -- workaround for osx 10.6 driver bug when using FSAA. R_Clear only
  // clears the warpimage part of the screen.
  GLSetCanvas(CANVAS_DEFAULT);

  // if warp render went down into sbar territory, we need to be sure to refresh
  // it next frame
  if (GL_warpimagesize() + Sbar_Lines() > GL_Height()) Sbar_Changed();

  // if viewsize is less than 100, we need to redraw the frame around the
  // viewport
  SCR_ResetTileClearUpdates();
}
