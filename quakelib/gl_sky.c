// SPDX-License-Identifier: GPL-2.0-or-later

#include "quakedef.h"

float Fog_GetDensity(void);
float *Fog_GetColor(void);

extern int rs_skypolys;   // for r_speeds readout
extern int rs_skypasses;  // for r_speeds readout
float skymins[2][6], skymaxs[2][6];

char skybox_name[32] = "";  // name of current skybox, or "" if no skybox

uint32_t skybox_textures[6];

extern cvar_t gl_farclip;

int skytexorder[6] = {0, 2, 1, 3, 4, 5};  // for skybox

int st_to_vec[6][3] = {
    {3, -1, 2}, {-3, 1, 2}, {1, 3, 2}, {-1, -3, 2}, {-2, -1, 3},  // straight up
    {2, -1, -3}  // straight down
};

float skyfog;  // ericw

/*
=============
Sky_Init
=============
*/
void Sky_Init(void) {
  int i;

  for (i = 0; i < 6; i++) skybox_textures[i] = 0;
}

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
  // glTexCoord2f(s, t);
  // glVertex3fv(v);
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
