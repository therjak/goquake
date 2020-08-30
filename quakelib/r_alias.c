// r_alias.c -- alias model rendering

#include "quakedef.h"

extern cvar_t r_drawflat, gl_overbright_models, gl_fullbrights, r_lerpmodels,
       r_lerpmove;  // johnfitz

#define NUMVERTEXNORMALS 162

extern vec3_t
lightcolor;  // johnfitz -- replaces "float shadelight" for lit support

// precalculated dot products for quantized angles
#define SHADEDOT_QUANT 16

extern vec3_t lightspot;

int shadeDots = 0;
vec3_t shadevector;

float entalpha;  // johnfitz

qboolean overbright;  // johnfitz

qboolean shading = true;  // johnfitz -- if false, disable vertex shading for
// various reasons (fullbright, r_lightmap, showtris,
// etc)

// johnfitz -- struct for passing lerp information to drawing functions
typedef struct {
  short pose1;
  short pose2;
  float blend;
  vec3_t origin;
  vec3_t angles;
} lerpdata_t;
// johnfitz

static GLuint r_alias_program;

// uniforms used in vert shader
static GLuint blendLoc;
static GLuint shadevectorLoc;
static GLuint lightColorLoc;

// uniforms used in frag shader
static GLuint texLoc;
static GLuint fullbrightTexLoc;
static GLuint useFullbrightTexLoc;
static GLuint useOverbrightLoc;
static GLuint alFogDensity;
static GLuint alFogColor;

static const GLint pose1VertexAttrIndex = 0;
static const GLint pose1NormalAttrIndex = 1;
static const GLint pose2VertexAttrIndex = 2;
static const GLint pose2NormalAttrIndex = 3;
static const GLint texCoordsAttrIndex = 4;

/*
   =============
   GLARB_GetXYZOffset

   Returns the offset of the first vertex's meshxyz_t.xyz in the vbo for the given
   model and pose.
   =============
   */
static void *GLARB_GetXYZOffset(aliashdr_t *hdr, int pose, entity_t* e) {
  meshxyz_t dummy;
  int xyzoffs = ((char *)&dummy.xyz - (char *)&dummy);
  return (void *)(e->model->vboxyzofs +
      (hdr->numverts_vbo * pose * sizeof(meshxyz_t)) + xyzoffs);
}

/*
   =============
   GLARB_GetNormalOffset

   Returns the offset of the first vertex's meshxyz_t.normal in the vbo for the
   given model and pose.
   =============
   */
static void *GLARB_GetNormalOffset(aliashdr_t *hdr, int pose, entity_t* e) {
  meshxyz_t dummy;
  int normaloffs = ((char *)&dummy.normal - (char *)&dummy);
  return (void *)(e->model->vboxyzofs +
      (hdr->numverts_vbo * pose * sizeof(meshxyz_t)) + normaloffs);
}

/*
   =============
   GLAlias_CreateShaders
   =============
   */
// THERJAK: extern
void GLAlias_CreateShaders(void) {
  const glsl_attrib_binding_t bindings[] = {
    {"TexCoords", texCoordsAttrIndex},
    {"Pose1Vert", pose1VertexAttrIndex},
    {"Pose1Normal", pose1NormalAttrIndex},
    {"Pose2Vert", pose2VertexAttrIndex},
    {"Pose2Normal", pose2NormalAttrIndex}};

  const GLchar *vertSource =
    "#version 110\n"
    "\n"
    "uniform float Blend;\n"
    "uniform vec3 ShadeVector;\n"
    "uniform vec4 LightColor;\n"
    "attribute vec4 TexCoords; // only xy are used \n"
    "attribute vec4 Pose1Vert;\n"
    "attribute vec3 Pose1Normal;\n"
    "attribute vec4 Pose2Vert;\n"
    "attribute vec3 Pose2Normal;\n"
    "varying float FogFragCoord;\n"
    "varying vec4 glTexCoord;\n"
    "varying vec4 frontColor;\n"
    "float r_avertexnormal_dot(vec3 vertexnormal) // from MH \n"
    "{\n"
    "        float dot = dot(vertexnormal, ShadeVector);\n"
    "        // wtf - this reproduces anorm_dots within as reasonable a "
    "degree of tolerance as the >= 0 case\n"
    "        if (dot < 0.0)\n"
    "            return 1.0 + dot * (13.0 / 44.0);\n"
    "        else\n"
    "            return 1.0 + dot;\n"
    "}\n"
    "void main()\n"
    "{\n"
    "	glTexCoord = TexCoords;\n"
    "	vec4 lerpedVert = mix(vec4(Pose1Vert.xyz, 1.0), vec4(Pose2Vert.xyz, 1.0), Blend);\n"
    "	gl_Position = gl_ModelViewProjectionMatrix * lerpedVert;\n"
    " FogFragCoord = gl_Position.w;\n"
    "	float dot1 = r_avertexnormal_dot(Pose1Normal);\n"
    "	float dot2 = r_avertexnormal_dot(Pose2Normal);\n"
    "	frontColor = LightColor * vec4(vec3(mix(dot1, dot2, Blend)), 1.0);\n"
    "}\n";

  const GLchar *fragSource =
    "#version 330\n"
    "\n"
    "uniform sampler2D Tex;\n"
    "uniform sampler2D FullbrightTex;\n"
    "uniform bool UseFullbrightTex;\n"
    "uniform bool UseOverbright;\n"
    "uniform float FogDensity;\n"
    "uniform vec4 FogColor;\n"
    "varying float FogFragCoord;\n"
    "varying vec4 glTexCoord;\n"
    "varying vec4 frontColor;\n"
    "void main()\n"
    "{\n"
    "	vec4 result = texture2D(Tex, glTexCoord.xy);\n"
    "	result *= frontColor;\n"
    "	if (UseOverbright)\n"
    "		result.rgb *= 2.0;\n"
    "	if (UseFullbrightTex)\n"
    "		result += texture2D(FullbrightTex, glTexCoord.xy);\n"
    "	result = clamp(result, 0.0, 1.0);\n"
    "	float fog = exp(-FogDensity * FogDensity * FogFragCoord * FogFragCoord);\n"
    "	fog = clamp(fog, 0.0, 1.0);\n"
    "	result = mix(FogColor, result, fog);\n"
    "	result.a = frontColor.a;\n"
    "	gl_FragColor = result;\n"
    "}\n";

  r_alias_program = GL_CreateProgram(
      vertSource, fragSource, sizeof(bindings) / sizeof(bindings[0]), bindings);

  if (r_alias_program != 0) {
    // get uniform locations
    blendLoc = GL_GetUniformLocation(&r_alias_program, "Blend");
    shadevectorLoc = GL_GetUniformLocation(&r_alias_program, "ShadeVector");
    lightColorLoc = GL_GetUniformLocation(&r_alias_program, "LightColor");
    texLoc = GL_GetUniformLocation(&r_alias_program, "Tex");
    fullbrightTexLoc = GL_GetUniformLocation(&r_alias_program, "FullbrightTex");
    useFullbrightTexLoc =
      GL_GetUniformLocation(&r_alias_program, "UseFullbrightTex");
    useOverbrightLoc = GL_GetUniformLocation(&r_alias_program, "UseOverbright");
    alFogDensity = GL_GetUniformLocation(&r_alias_program, "FogDensity");
    alFogColor = GL_GetUniformLocation(&r_alias_program, "FogColor");
  }
}

/*
   =============
   GL_DrawAliasFrame_GLSL -- ericw

   Optimized alias model drawing codepath.
   Compared to the original GL_DrawAliasFrame, this makes 1 draw call,
   no vertex data is uploaded (it's already in the r_meshvbo and r_meshindexesvbo
   static VBOs), and lerping and lighting is done in the vertex shader.

   Supports optional overbright, optional fullbright pixels.

   Based on code by MH from RMQEngine
   =============
   */
void GL_DrawAliasFrame_GLSL(aliashdr_t *paliashdr, lerpdata_t lerpdata,
    uint32_t tx, uint32_t fb, entity_t* e) {
  float blend;

  if (lerpdata.pose1 != lerpdata.pose2) {
    blend = lerpdata.blend;
  } else  // poses the same means either 1. the entity has paused its animation,
    // or 2. r_lerpmodels is disabled
    {
      blend = 0;
    }

  glUseProgram(r_alias_program);

  GL_BindBuffer(GL_ARRAY_BUFFER, e->model->meshvbo);
  GL_BindBuffer(GL_ELEMENT_ARRAY_BUFFER, e->model->meshindexesvbo);

  glEnableVertexAttribArray(texCoordsAttrIndex);
  glEnableVertexAttribArray(pose1VertexAttrIndex);
  glEnableVertexAttribArray(pose2VertexAttrIndex);
  glEnableVertexAttribArray(pose1NormalAttrIndex);
  glEnableVertexAttribArray(pose2NormalAttrIndex);

  glVertexAttribPointer(texCoordsAttrIndex, 2, GL_FLOAT, GL_FALSE, 0,
      (void *)(intptr_t)e->model->vbostofs);
  glVertexAttribPointer(pose1VertexAttrIndex, 4, GL_UNSIGNED_BYTE, GL_FALSE,
      sizeof(meshxyz_t),
      GLARB_GetXYZOffset(paliashdr, lerpdata.pose1, e));
  glVertexAttribPointer(pose2VertexAttrIndex, 4, GL_UNSIGNED_BYTE, GL_FALSE,
      sizeof(meshxyz_t),
      GLARB_GetXYZOffset(paliashdr, lerpdata.pose2, e));
  // GL_TRUE to normalize the signed bytes to [-1 .. 1]
  glVertexAttribPointer(pose1NormalAttrIndex, 4, GL_BYTE, GL_TRUE,
      sizeof(meshxyz_t),
      GLARB_GetNormalOffset(paliashdr, lerpdata.pose1, e));
  glVertexAttribPointer(pose2NormalAttrIndex, 4, GL_BYTE, GL_TRUE,
      sizeof(meshxyz_t),
      GLARB_GetNormalOffset(paliashdr, lerpdata.pose2, e));

  // set uniforms
  glUniform1f(blendLoc, blend);
  glUniform3f(shadevectorLoc, shadevector[0], shadevector[1], shadevector[2]);
  glUniform4f(lightColorLoc, lightcolor[0], lightcolor[1], lightcolor[2],
      entalpha);
  glUniform1i(texLoc, 0);
  glUniform1i(fullbrightTexLoc, 1);
  glUniform1i(useFullbrightTexLoc, (fb != 0) ? 1 : 0);
  glUniform1f(useOverbrightLoc, overbright ? 1 : 0);
  glUniform1f(alFogDensity, 0);
  float currentFogColor[4];
  glGetFloatv(GL_FOG_COLOR, currentFogColor);
  glUniform4f(alFogColor, currentFogColor[0], currentFogColor[1], currentFogColor[2], currentFogColor[3]);

  // set textures
  GLSelectTexture(GL_TEXTURE0);
  GLBind(tx);

  if (fb) {
    GLSelectTexture(GL_TEXTURE1);
    GLBind(fb);
  }

  // draw
  glDrawElements(GL_TRIANGLES, paliashdr->numindexes, GL_UNSIGNED_SHORT,
      (void *)(intptr_t)e->model->vboindexofs);

  // clean up
  glDisableVertexAttribArray(texCoordsAttrIndex);
  glDisableVertexAttribArray(pose1VertexAttrIndex);
  glDisableVertexAttribArray(pose2VertexAttrIndex);
  glDisableVertexAttribArray(pose1NormalAttrIndex);
  glDisableVertexAttribArray(pose2NormalAttrIndex);

  glUseProgram(0);
  GLSelectTexture(GL_TEXTURE0);

  rs_aliaspasses += paliashdr->numtris;
}

/*
   =============
   GL_DrawAliasFrame -- johnfitz -- rewritten to support colored light, lerping,
   entalpha, multitexture, and r_drawflat
   =============
   */
void GL_DrawAliasFrame(aliashdr_t *paliashdr, lerpdata_t lerpdata) {
  float vertcolor[4];
  trivertx_t *verts1, *verts2;
  int *commands;
  int count;
  float u, v;
  float blend, iblend;
  qboolean lerping;

  if (lerpdata.pose1 != lerpdata.pose2) {
    lerping = true;
    verts1 = (trivertx_t *)((byte *)paliashdr + paliashdr->posedata);
    verts2 = verts1;
    verts1 += lerpdata.pose1 * paliashdr->poseverts;
    verts2 += lerpdata.pose2 * paliashdr->poseverts;
    blend = lerpdata.blend;
    iblend = 1.0f - blend;
  } else  // poses the same means either 1. the entity has paused its animation,
    // or 2. r_lerpmodels is disabled
    {
      lerping = false;
      verts1 = (trivertx_t *)((byte *)paliashdr + paliashdr->posedata);
      verts2 = verts1;  // avoid bogus compiler warning
      verts1 += lerpdata.pose1 * paliashdr->poseverts;
      blend = iblend = 0;  // avoid bogus compiler warning
    }

  commands = (int *)((byte *)paliashdr + paliashdr->commands);

  vertcolor[3] = entalpha;  // never changes, so there's no need to put this
  // inside the loop

  while (1) {
    // get the vertex count and primitive type
    count = *commands++;
    if (!count) break;  // done

    if (count < 0) {
      count = -count;
      glBegin(GL_TRIANGLE_FAN);
    } else
      glBegin(GL_TRIANGLE_STRIP);

    do {
      u = ((float *)commands)[0];
      v = ((float *)commands)[1];
      if (GetMTexEnabled()) {
        glMultiTexCoord2f(GL_TEXTURE0, u, v);
        glMultiTexCoord2f(GL_TEXTURE1, u, v);
      } else
        glTexCoord2f(u, v);

      commands += 2;

      if (shading) {
        if (lerping) {
          vertcolor[0] = (R_and(shadeDots, verts1->lightnormalindex) * iblend +
              R_and(shadeDots, verts2->lightnormalindex) * blend) *
            lightcolor[0];
          vertcolor[1] = (R_and(shadeDots, verts1->lightnormalindex) * iblend +
              R_and(shadeDots, verts2->lightnormalindex) * blend) *
            lightcolor[1];
          vertcolor[2] = (R_and(shadeDots, verts1->lightnormalindex) * iblend +
              R_and(shadeDots, verts2->lightnormalindex) * blend) *
            lightcolor[2];
          glColor4fv(vertcolor);
        } else {
          vertcolor[0] =
            R_and(shadeDots, verts1->lightnormalindex) * lightcolor[0];
          vertcolor[1] =
            R_and(shadeDots, verts1->lightnormalindex) * lightcolor[1];
          vertcolor[2] =
            R_and(shadeDots, verts1->lightnormalindex) * lightcolor[2];
          glColor4fv(vertcolor);
        }
      }

      if (lerping) {
        glVertex3f(verts1->v[0] * iblend + verts2->v[0] * blend,
            verts1->v[1] * iblend + verts2->v[1] * blend,
            verts1->v[2] * iblend + verts2->v[2] * blend);
        verts1++;
        verts2++;
      } else {
        glVertex3f(verts1->v[0], verts1->v[1], verts1->v[2]);
        verts1++;
      }
    } while (--count);

    glEnd();
  }

  rs_aliaspasses += paliashdr->numtris;
}

/*
   =================
   R_SetupAliasFrame -- johnfitz -- rewritten to support lerping
   =================
   */
void R_SetupAliasFrame(aliashdr_t *paliashdr, int frame, lerpdata_t *lerpdata, entity_t* e) {
  int posenum, numposes;

  if ((frame >= paliashdr->numframes) || (frame < 0)) {
    Con_DPrintf("R_AliasSetupFrame: no such frame %d for '%s'\n", frame,
        e->model->name);
    frame = 0;
  }

  posenum = paliashdr->frames[frame].firstpose;
  numposes = paliashdr->frames[frame].numposes;

  if (numposes > 1) {
    e->lerptime = paliashdr->frames[frame].interval;
    posenum += (int)(CL_Time() / e->lerptime) % numposes;
  } else
    e->lerptime = 0.1;

  if (e->lerpflags & LERP_RESETANIM)  // kill any lerp in progress
  {
    e->lerpstart = 0;
    e->previouspose = posenum;
    e->currentpose = posenum;
    e->lerpflags -= LERP_RESETANIM;
  } else if (e->currentpose != posenum)  // pose changed, start new lerp
  {
    if (e->lerpflags & LERP_RESETANIM2)  // defer lerping one more time
    {
      e->lerpstart = 0;
      e->previouspose = posenum;
      e->currentpose = posenum;
      e->lerpflags -= LERP_RESETANIM2;
    } else {
      e->lerpstart = CL_Time();
      e->previouspose = e->currentpose;
      e->currentpose = posenum;
    }
  }

  // set up values
  if (Cvar_GetValue(&r_lerpmodels) &&
      !(e->model->flags & MOD_NOLERP && Cvar_GetValue(&r_lerpmodels) != 2)) {
    if (e->lerpflags & LERP_FINISH && numposes == 1)
      lerpdata->blend = CLAMP(
          0, (CL_Time() - e->lerpstart) / (e->lerpfinish - e->lerpstart), 1);
    else
      lerpdata->blend = CLAMP(0, (CL_Time() - e->lerpstart) / e->lerptime, 1);
    lerpdata->pose1 = e->previouspose;
    lerpdata->pose2 = e->currentpose;
  } else  // don't lerp
  {
    lerpdata->blend = 1;
    lerpdata->pose1 = posenum;
    lerpdata->pose2 = posenum;
  }
}

/*
   =================
   R_SetupEntityTransform -- johnfitz -- set up transform part of lerpdata
   =================
   */
void R_SetupEntityTransform(entity_t *e, lerpdata_t *lerpdata) {
  float blend;
  vec3_t d;
  int i;

  // if LERP_RESETMOVE, kill any lerps in progress
  if (e->lerpflags & LERP_RESETMOVE) {
    e->movelerpstart = 0;
    VectorCopy(e->origin, e->previousorigin);
    VectorCopy(e->origin, e->currentorigin);
    VectorCopy(e->angles, e->previousangles);
    VectorCopy(e->angles, e->currentangles);
    e->lerpflags -= LERP_RESETMOVE;
  } else if (!VectorCompare(e->origin, e->currentorigin) ||
      !VectorCompare(
        e->angles,
        e->currentangles))  // origin/angles changed, start new lerp
  {
    e->movelerpstart = CL_Time();
    VectorCopy(e->currentorigin, e->previousorigin);
    VectorCopy(e->origin, e->currentorigin);
    VectorCopy(e->currentangles, e->previousangles);
    VectorCopy(e->angles, e->currentangles);
  }

  // set up values
  if (Cvar_GetValue(&r_lerpmove) && e->lerpflags & LERP_MOVESTEP) {
    if (e->lerpflags & LERP_FINISH)
      blend = CLAMP(
          0,
          (CL_Time() - e->movelerpstart) / (e->lerpfinish - e->movelerpstart),
          1);
    else
      blend = CLAMP(0, (CL_Time() - e->movelerpstart) / 0.1, 1);

    // translation
    VectorSubtract(e->currentorigin, e->previousorigin, d);
    lerpdata->origin[0] = e->previousorigin[0] + d[0] * blend;
    lerpdata->origin[1] = e->previousorigin[1] + d[1] * blend;
    lerpdata->origin[2] = e->previousorigin[2] + d[2] * blend;

    // rotation
    VectorSubtract(e->currentangles, e->previousangles, d);
    for (i = 0; i < 3; i++) {
      if (d[i] > 180) d[i] -= 360;
      if (d[i] < -180) d[i] += 360;
    }
    lerpdata->angles[0] = e->previousangles[0] + d[0] * blend;
    lerpdata->angles[1] = e->previousangles[1] + d[1] * blend;
    lerpdata->angles[2] = e->previousangles[2] + d[2] * blend;
  } else  // don't lerp
  {
    VectorCopy(e->origin, lerpdata->origin);
    VectorCopy(e->angles, lerpdata->angles);
  }
}

/*
   =================
   R_SetupAliasLighting -- johnfitz -- broken out from R_DrawAliasModel and
   rewritten
   =================
   */
void R_SetupAliasLighting(entity_t *e) {
  vec3_t dist;
  float add;
  int i;
  int quantizedangle;
  float radiansangle;

  R_LightPoint(e->origin);

  // add dlights
  for (i = 0; i < MAX_DLIGHTS; i++) {
    if (cl_dlights[i].die >= CL_Time()) {
      VectorSubtract(e->origin, cl_dlights[i].origin, dist);
      add = cl_dlights[i].radius - VectorLength(dist);
      if (add > 0) VectorMA(lightcolor, add, cl_dlights[i].color, lightcolor);
    }
  }

  // minimum light value on gun (24)
  if (EntityIsPlayerWeapon(e)) {
    add = 72.0f - (lightcolor[0] + lightcolor[1] + lightcolor[2]);
    if (add > 0.0f) {
      lightcolor[0] += add / 3.0f;
      lightcolor[1] += add / 3.0f;
      lightcolor[2] += add / 3.0f;
    }
  }

  // minimum light value on players (8)
  if (EntityIsPlayer(e)) {
    add = 24.0f - (lightcolor[0] + lightcolor[1] + lightcolor[2]);
    if (add > 0.0f) {
      lightcolor[0] += add / 3.0f;
      lightcolor[1] += add / 3.0f;
      lightcolor[2] += add / 3.0f;
    }
  }

  // clamp lighting so it doesn't overbright as much (96)
  if (overbright) {
    add = 288.0f / (lightcolor[0] + lightcolor[1] + lightcolor[2]);
    if (add < 1.0f) VectorScale(lightcolor, add, lightcolor);
  }

  // hack up the brightness when fullbrights but no overbrights (256)
  if (Cvar_GetValue(&gl_fullbrights) && !Cvar_GetValue(&gl_overbright_models))
    if (e->model->flags & MOD_FBRIGHTHACK) {
      lightcolor[0] = 256.0f;
      lightcolor[1] = 256.0f;
      lightcolor[2] = 256.0f;
    }

  quantizedangle =
    ((int)(e->angles[1] * (SHADEDOT_QUANT / 360.0))) & (SHADEDOT_QUANT - 1);

  // shadevector is passed to the shader to compute shadedots inside the
  // shader, see GLAlias_CreateShaders()
  radiansangle = (quantizedangle / 16.0) * 2.0 * 3.14159;
  shadevector[0] = cos(-radiansangle);
  shadevector[1] = sin(-radiansangle);
  shadevector[2] = 1;
  VectorNormalize(shadevector);

  shadeDots = quantizedangle;
  VectorScale(lightcolor, 1.0f / 200.0f, lightcolor);
}

/*
   =================
   R_DrawAliasModel -- johnfitz -- almost completely rewritten
   =================
   */
// THERJAK: extern
void R_DrawAliasModel(entity_t *e) {
  aliashdr_t *paliashdr;
  int i, anim;
  uint32_t tx, fb;
  lerpdata_t lerpdata;

  // setup pose/lerp data -- do it first so we don't miss updates due to culling
  paliashdr = (aliashdr_t *)Mod_Extradata(e->model);
  R_SetupAliasFrame(paliashdr, e->frame, &lerpdata, e);
  R_SetupEntityTransform(e, &lerpdata);

  // cull it
  if (R_CullModelForEntity(e)) return;

  // transform it
  glPushMatrix();
  R_RotateForEntity(lerpdata.origin, lerpdata.angles);
  glTranslatef(paliashdr->scale_origin[0], paliashdr->scale_origin[1],
      paliashdr->scale_origin[2]);
  glScalef(paliashdr->scale[0], paliashdr->scale[1], paliashdr->scale[2]);

  // random stuff
  if (Cvar_GetValue(&gl_smoothmodels)) glShadeModel(GL_SMOOTH);
  if (Cvar_GetValue(&gl_affinemodels))
    glHint(GL_PERSPECTIVE_CORRECTION_HINT, GL_FASTEST);
  overbright = Cvar_GetValue(&gl_overbright_models);
  shading = true;

  // set up for alpha blending
  entalpha = ENTALPHA_DECODE(e->alpha);
  if (entalpha == 0) goto cleanup;
  if (entalpha < 1) {
    glDepthMask(GL_FALSE);
    glEnable(GL_BLEND);
  }

  // set up lighting
  rs_aliaspolys += paliashdr->numtris;
  R_SetupAliasLighting(e);

  // set up textures
  GLDisableMultitexture();
  anim = (int)(CL_Time() * 10) & 3;
  if ((e->skinnum >= paliashdr->numskins) || (e->skinnum < 0)) {
    Con_DPrintf("R_DrawAliasModel: no such skin # %d for '%s'\n", e->skinnum,
        e->model->name);
    tx = 0;  // NULL will give the checkerboard texture
    fb = 0;
  } else {
    tx = paliashdr->gltextures[e->skinnum][anim];
    fb = paliashdr->fbtextures[e->skinnum][anim];
  }
  if (!Cvar_GetValue(&gl_nocolors)) {
    uint32_t t = PlayerTexture(e);
    if (t != 0) {
      tx = t;
    }
  }
  if (!Cvar_GetValue(&gl_fullbrights)) fb = 0;

  GL_DrawAliasFrame_GLSL(paliashdr, lerpdata, tx, fb, e);

cleanup:
  glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_REPLACE);
  glHint(GL_PERSPECTIVE_CORRECTION_HINT, GL_NICEST);
  glShadeModel(GL_FLAT);
  glDepthMask(GL_TRUE);
  glDisable(GL_BLEND);
  glColor3f(1, 1, 1);
  glPopMatrix();
}

// johnfitz -- values for shadow matrix
#define SHADOW_SKEW_X -0.7  // skew along x axis. -0.7 to mimic glquake shadows
#define SHADOW_SKEW_Y 0  // skew along y axis. 0 to mimic glquake shadows
#define SHADOW_VSCALE 0  // 0=completely flat
#define SHADOW_HEIGHT 0.1  // how far above the floor to render the shadow
// johnfitz

/*
=============
GL_DrawAliasShadow -- johnfitz -- rewritten

TODO: orient shadow onto "lightplane" (a global mplane_t*)
=============
*/
// THERJAK: extern
void GL_DrawAliasShadow(entity_t *e) {
  float shadowmatrix[16] = {1,
                            0,
                            0,
                            0,
                            0,
                            1,
                            0,
                            0,
                            SHADOW_SKEW_X,
                            SHADOW_SKEW_Y,
                            SHADOW_VSCALE,
                            0,
                            0,
                            0,
                            SHADOW_HEIGHT,
                            1};
  float lheight;
  aliashdr_t *paliashdr;
  lerpdata_t lerpdata;

  if (R_CullModelForEntity(e)) return;

  if (e->model->flags & MOD_NOSHADOW) return;

  entalpha = ENTALPHA_DECODE(e->alpha);
  if (entalpha == 0) return;

  paliashdr = (aliashdr_t *)Mod_Extradata(e->model);
  R_SetupAliasFrame(paliashdr, e->frame, &lerpdata, e);
  R_SetupEntityTransform(e, &lerpdata);
  R_LightPoint(e->origin);
  lheight = e->origin[2] - lightspot[2];

  // set up matrix
  glPushMatrix();
  glTranslatef(lerpdata.origin[0], lerpdata.origin[1], lerpdata.origin[2]);
  glTranslatef(0, 0, -lheight);
  glMultMatrixf(shadowmatrix);
  glTranslatef(0, 0, lheight);
  glRotatef(lerpdata.angles[1], 0, 0, 1);
  glRotatef(-lerpdata.angles[0], 0, 1, 0);
  glRotatef(lerpdata.angles[2], 1, 0, 0);
  glTranslatef(paliashdr->scale_origin[0], paliashdr->scale_origin[1],
               paliashdr->scale_origin[2]);
  glScalef(paliashdr->scale[0], paliashdr->scale[1], paliashdr->scale[2]);

  // draw it
  glDepthMask(GL_FALSE);
  glEnable(GL_BLEND);
  GLDisableMultitexture();
  glDisable(GL_TEXTURE_2D);
  shading = false;
  glColor4f(0, 0, 0, entalpha * 0.5);
  GL_DrawAliasFrame(paliashdr, lerpdata);
  glEnable(GL_TEXTURE_2D);
  glDisable(GL_BLEND);
  glDepthMask(GL_TRUE);

  // clean up
  glPopMatrix();
}
