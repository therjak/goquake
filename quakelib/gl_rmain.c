// r_main.c

#include "quakedef.h"

extern cvar_t vid_gamma;
extern cvar_t vid_contrast;

vec3_t modelorg, r_entorigin;
entity_t *currententity;

int r_visframecount;  // bumped when going to a new PVS
int r_framecount;     // used for dlight push checking

int GetRFrameCount() { return r_framecount; }

mplane_t frustum[4];

// johnfitz -- rendering statistics
int rs_brushpolys, rs_aliaspolys, rs_skypolys, rs_particles, rs_fogpolys;
int rs_dynamiclightmaps, rs_brushpasses, rs_aliaspasses, rs_skypasses;
float rs_megatexels;

//
// view origin
//
vec3_t vup;
vec3_t vpn;
vec3_t vright;
vec3_t r_origin;

float r_fovx, r_fovy;  // johnfitz -- rendering fov may be different becuase of
                       // r_waterwarp and r_stereo

//
// screen size info
//
refdef_t r_refdef;

mleaf_t *r_viewleaf, *r_oldviewleaf;

int d_lightstylevalue[256];  // 8.8 fraction of base light value

cvar_t r_norefresh;      // = {"r_norefresh", "0", CVAR_NONE};
cvar_t r_drawentities;   // = {"r_drawentities", "1", CVAR_NONE};
cvar_t r_drawviewmodel;  // = {"r_drawviewmodel", "1", CVAR_NONE};
cvar_t r_speeds;         // = {"r_speeds", "0", CVAR_NONE};
cvar_t r_pos;            // = {"r_pos", "0", CVAR_NONE};
cvar_t r_fullbright;     // = {"r_fullbright", "0", CVAR_NONE};
cvar_t r_lightmap;       // = {"r_lightmap", "0", CVAR_NONE};
cvar_t r_shadows;        // = {"r_shadows", "0", CVAR_ARCHIVE};
cvar_t r_wateralpha;     // = {"r_wateralpha", "1", CVAR_ARCHIVE};
cvar_t r_dynamic;        // = {"r_dynamic", "1", CVAR_ARCHIVE};
cvar_t r_novis;          // = {"r_novis", "0", CVAR_ARCHIVE};

cvar_t gl_finish;        // = {"gl_finish", "0", CVAR_NONE};
cvar_t gl_clear;         // = {"gl_clear", "1", CVAR_NONE};
cvar_t gl_cull;          // = {"gl_cull", "1", CVAR_NONE};
cvar_t gl_smoothmodels;  // = {"gl_smoothmodels", "1", CVAR_NONE};
cvar_t gl_affinemodels;  // = {"gl_affinemodels", "0", CVAR_NONE};
cvar_t gl_polyblend;     // = {"gl_polyblend", "1", CVAR_NONE};
cvar_t gl_flashblend;    // = {"gl_flashblend", "0", CVAR_ARCHIVE};
cvar_t gl_playermip;     // = {"gl_playermip", "0", CVAR_NONE};
cvar_t gl_nocolors;      // = {"gl_nocolors", "0", CVAR_NONE};

// johnfitz -- new cvars
cvar_t r_stereo;              // = {"r_stereo", "0", CVAR_NONE};
cvar_t r_stereodepth;         // = {"r_stereodepth", "128", CVAR_NONE};
cvar_t r_clearcolor;          // = {"r_clearcolor", "2", CVAR_ARCHIVE};
cvar_t r_drawflat;            // = {"r_drawflat", "0", CVAR_NONE};
cvar_t r_flatlightstyles;     // = {"r_flatlightstyles", "0", CVAR_NONE};
cvar_t gl_fullbrights;        // = {"gl_fullbrights", "1", CVAR_ARCHIVE};
cvar_t gl_farclip;            // = {"gl_farclip", "16384", CVAR_ARCHIVE};
cvar_t gl_overbright;         // = {"gl_overbright", "1", CVAR_ARCHIVE};
cvar_t gl_overbright_models;  // = {"gl_overbright_models", "1", CVAR_ARCHIVE};
cvar_t r_oldskyleaf;          // = {"r_oldskyleaf", "0", CVAR_NONE};
cvar_t r_drawworld;           // = {"r_drawworld", "1", CVAR_NONE};
cvar_t r_showtris;            // = {"r_showtris", "0", CVAR_NONE};
cvar_t r_lerpmodels;          // = {"r_lerpmodels", "1", CVAR_NONE};
cvar_t r_lerpmove;            // = {"r_lerpmove", "1", CVAR_NONE};
cvar_t r_nolerp_list;         // = {"r_nolerp_list",
                              //   "progs/flame.mdl,progs/flame2.mdl,progs/"
                              //   "braztall.mdl,progs/brazshrt.mdl,progs/"
                              //  "longtrch.mdl,progs/flame_pyre.mdl,progs/"
//       "v_saw.mdl,progs/v_xfist.mdl,progs/h2stuff/newfire.mdl",
//     CVAR_NONE};
cvar_t r_noshadow_list;  // = {"r_noshadow_list",
                         //  "progs/flame2.mdl,progs/flame.mdl,progs/"
                         //  "bolt1.mdl,progs/bolt2.mdl,progs/bolt3.mdl,progs/"
                         //  "laser.mdl",
                         //  CVAR_NONE};

// johnfitz

cvar_t gl_zfix;  // = {"gl_zfix", "0", CVAR_NONE};  // QuakeSpasm z-fighting fix

cvar_t r_lavaalpha;   // = {"r_lavaalpha", "0", CVAR_NONE};
cvar_t r_telealpha;   // = {"r_telealpha", "0", CVAR_NONE};
cvar_t r_slimealpha;  // = {"r_slimealpha", "0", CVAR_NONE};

float map_wateralpha, map_lavaalpha, map_telealpha, map_slimealpha;

qboolean r_drawflat_cheatsafe, r_fullbright_cheatsafe, r_lightmap_cheatsafe,
    r_drawworld_cheatsafe;  // johnfitz

//==============================================================================
//
// GLSL GAMMA CORRECTION
//
//==============================================================================

static GLuint r_gamma_texture;
static GLuint r_gamma_program;
static int r_gamma_texture_width, r_gamma_texture_height;

// uniforms used in gamma shader
static GLuint gammaLoc;
static GLuint contrastLoc;
static GLuint textureLoc;

/*
=============
GLSLGamma_DeleteTexture
=============
*/
void GLSLGamma_DeleteTexture(void) {
  glDeleteTextures(1, &r_gamma_texture);
  r_gamma_texture = 0;
  r_gamma_program = 0;  // deleted in R_DeleteShaders
}

/*
=============
GLSLGamma_CreateShaders
=============
*/
static void GLSLGamma_CreateShaders(void) {
  const GLchar *vertSource =
      "#version 110\n"
      "\n"
      "void main(void) {\n"
      "	gl_Position = vec4(gl_Vertex.xy, 0.0, 1.0);\n"
      "	gl_TexCoord[0] = gl_MultiTexCoord0;\n"
      "}\n";

  const GLchar *fragSource =
      "#version 110\n"
      "\n"
      "uniform sampler2D GammaTexture;\n"
      "uniform float GammaValue;\n"
      "uniform float ContrastValue;\n"
      "\n"
      "void main(void) {\n"
      "	  vec4 frag = texture2D(GammaTexture, gl_TexCoord[0].xy);\n"
      "	  frag.rgb = frag.rgb * ContrastValue;\n"
      "	  gl_FragColor = vec4(pow(frag.rgb, vec3(GammaValue)), 1.0);\n"
      "}\n";

  r_gamma_program = GL_CreateProgram(vertSource, fragSource, 0, NULL);

  // get uniform locations
  gammaLoc = GL_GetUniformLocation(&r_gamma_program, "GammaValue");
  contrastLoc = GL_GetUniformLocation(&r_gamma_program, "ContrastValue");
  textureLoc = GL_GetUniformLocation(&r_gamma_program, "GammaTexture");
}

/*
=============
GLSLGamma_GammaCorrect
=============
*/
void GLSLGamma_GammaCorrect(void) {
  //THERJAK
  float smax, tmax;

  if (Cvar_GetValue(&vid_gamma) == 1 && Cvar_GetValue(&vid_contrast) == 1)
    return;

  // create render-to-texture texture if needed
  if (!r_gamma_texture) {
    glGenTextures(1, &r_gamma_texture);
    glBindTexture(GL_TEXTURE_2D, r_gamma_texture);

    r_gamma_texture_width = GL_Width();
    r_gamma_texture_height = GL_Height();

    glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA8, r_gamma_texture_width,
                 r_gamma_texture_height, 0, GL_BGRA,
                 GL_UNSIGNED_INT_8_8_8_8_REV, NULL);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_NEAREST);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_NEAREST);
  }

  // create shader if needed
  if (!r_gamma_program) {
    GLSLGamma_CreateShaders();
    if (!r_gamma_program) {
      Go_Error("GLSLGamma_CreateShaders failed");
    }
  }

  // copy the framebuffer to the texture
  GLDisableMultitexture();
  glBindTexture(GL_TEXTURE_2D, r_gamma_texture);
  glCopyTexSubImage2D(GL_TEXTURE_2D, 0, 0, 0, 0, 0, GL_Width(), GL_Height());

  // draw the texture back to the framebuffer with a fragment shader
  GL_UseProgramFunc(r_gamma_program);
  GL_Uniform1fFunc(gammaLoc, Cvar_GetValue(&vid_gamma));
  GL_Uniform1fFunc(contrastLoc,
                   q_min(2.0, q_max(1.0, Cvar_GetValue(&vid_contrast))));
  GL_Uniform1iFunc(textureLoc, 0);  // use texture unit 0

  glDisable(GL_ALPHA_TEST);
  glDisable(GL_DEPTH_TEST);

  glViewport(0, 0, GL_Width(), GL_Height());

  smax = GL_Width() / (float)r_gamma_texture_width;
  tmax = GL_Height() / (float)r_gamma_texture_height;

  glBegin(GL_QUADS);
  glTexCoord2f(0, 0);
  glVertex2f(-1, -1);
  glTexCoord2f(smax, 0);
  glVertex2f(1, -1);
  glTexCoord2f(smax, tmax);
  glVertex2f(1, 1);
  glTexCoord2f(0, tmax);
  glVertex2f(-1, 1);
  glEnd();

  GL_UseProgramFunc(0);

  // clear cached binding
  GLClearBindings();
}

/*
=================
R_CullBox -- johnfitz -- replaced with new function from lordhavoc

Returns true if the box is completely outside the frustum
=================
*/
qboolean R_CullBox(vec3_t emins, vec3_t emaxs) {
  int i;
  mplane_t *p;
  for (i = 0; i < 4; i++) {
    p = frustum + i;
    switch (p->signbits) {
      default:
      case 0:
        if (p->normal[0] * emaxs[0] + p->normal[1] * emaxs[1] +
                p->normal[2] * emaxs[2] <
            p->dist)
          return true;
        break;
      case 1:
        if (p->normal[0] * emins[0] + p->normal[1] * emaxs[1] +
                p->normal[2] * emaxs[2] <
            p->dist)
          return true;
        break;
      case 2:
        if (p->normal[0] * emaxs[0] + p->normal[1] * emins[1] +
                p->normal[2] * emaxs[2] <
            p->dist)
          return true;
        break;
      case 3:
        if (p->normal[0] * emins[0] + p->normal[1] * emins[1] +
                p->normal[2] * emaxs[2] <
            p->dist)
          return true;
        break;
      case 4:
        if (p->normal[0] * emaxs[0] + p->normal[1] * emaxs[1] +
                p->normal[2] * emins[2] <
            p->dist)
          return true;
        break;
      case 5:
        if (p->normal[0] * emins[0] + p->normal[1] * emaxs[1] +
                p->normal[2] * emins[2] <
            p->dist)
          return true;
        break;
      case 6:
        if (p->normal[0] * emaxs[0] + p->normal[1] * emins[1] +
                p->normal[2] * emins[2] <
            p->dist)
          return true;
        break;
      case 7:
        if (p->normal[0] * emins[0] + p->normal[1] * emins[1] +
                p->normal[2] * emins[2] <
            p->dist)
          return true;
        break;
    }
  }
  return false;
}
/*
===============
R_CullModelForEntity -- johnfitz -- uses correct bounds based on rotation
===============
*/
qboolean R_CullModelForEntity(entity_t *e) {
  vec3_t mins, maxs;

  if (e->angles[0] || e->angles[2])  // pitch or roll
  {
    VectorAdd(e->origin, e->model->rmins, mins);
    VectorAdd(e->origin, e->model->rmaxs, maxs);
  } else if (e->angles[1])  // yaw
  {
    VectorAdd(e->origin, e->model->ymins, mins);
    VectorAdd(e->origin, e->model->ymaxs, maxs);
  } else  // no rotation
  {
    VectorAdd(e->origin, e->model->mins, mins);
    VectorAdd(e->origin, e->model->maxs, maxs);
  }

  return R_CullBox(mins, maxs);
}

/*
===============
R_RotateForEntity -- johnfitz -- modified to take origin and angles instead of
pointer to entity
===============
*/
void R_RotateForEntity(vec3_t origin, vec3_t angles) {
  glTranslatef(origin[0], origin[1], origin[2]);
  glRotatef(angles[1], 0, 0, 1);
  glRotatef(-angles[0], 0, 1, 0);
  glRotatef(angles[2], 1, 0, 0);
}

/*
=============
GL_PolygonOffset -- johnfitz

negative offset moves polygon closer to camera
=============
*/
void GL_PolygonOffset(int offset) {
  if (offset > 0) {
    glEnable(GL_POLYGON_OFFSET_FILL);
    glEnable(GL_POLYGON_OFFSET_LINE);
    glPolygonOffset(1, offset);
  } else if (offset < 0) {
    glEnable(GL_POLYGON_OFFSET_FILL);
    glEnable(GL_POLYGON_OFFSET_LINE);
    glPolygonOffset(-1, offset);
  } else {
    glDisable(GL_POLYGON_OFFSET_FILL);
    glDisable(GL_POLYGON_OFFSET_LINE);
  }
}

//==============================================================================
//
// SETUP FRAME
//
//==============================================================================

int SignbitsForPlane(mplane_t *out) {
  int bits, j;

  // for fast box on planeside test

  bits = 0;
  for (j = 0; j < 3; j++) {
    if (out->normal[j] < 0) bits |= 1 << j;
  }
  return bits;
}

/*
===============
TurnVector -- johnfitz

turn forward towards side on the plane defined by forward and side
if angle = 90, the result will be equal to side
assumes side and forward are perpendicular, and normalized
to turn away from side, use a negative angle
===============
*/
#define DEG2RAD(a) ((a)*M_PI_DIV_180)
void TurnVector(vec3_t out, const vec3_t forward, const vec3_t side,
                float angle) {
  float scale_forward, scale_side;

  scale_forward = cos(DEG2RAD(angle));
  scale_side = sin(DEG2RAD(angle));

  out[0] = scale_forward * forward[0] + scale_side * side[0];
  out[1] = scale_forward * forward[1] + scale_side * side[1];
  out[2] = scale_forward * forward[2] + scale_side * side[2];
}

/*
===============
R_SetFrustum -- johnfitz -- rewritten
===============
*/
void R_SetFrustum(float fovx, float fovy) {
  int i;

  if (Cvar_GetValue(&r_stereo))
    fovx += 10;  // silly hack so that polygons don't drop out becuase of stereo
                 // skew

  TurnVector(frustum[0].normal, vpn, vright, fovx / 2 - 90);  // left plane
  TurnVector(frustum[1].normal, vpn, vright, 90 - fovx / 2);  // right plane
  TurnVector(frustum[2].normal, vpn, vup, 90 - fovy / 2);     // bottom plane
  TurnVector(frustum[3].normal, vpn, vup, fovy / 2 - 90);     // top plane

  for (i = 0; i < 4; i++) {
    frustum[i].Type = PLANE_ANYZ;
    frustum[i].dist = DotProduct(
        r_origin, frustum[i].normal);  // FIXME: shouldn't this always be zero?
    frustum[i].signbits = SignbitsForPlane(&frustum[i]);
  }
}

/*
=============
GL_SetFrustum -- johnfitz -- written to replace MYgluPerspective
=============
*/
#define NEARCLIP 4
float frustum_skew = 0.0;  // used by r_stereo
void GL_SetFrustum(float fovx, float fovy) {
  float xmax, ymax;
  xmax = NEARCLIP * tan(fovx * M_PI / 360.0);
  ymax = NEARCLIP * tan(fovy * M_PI / 360.0);
  glFrustum(-xmax + frustum_skew, xmax + frustum_skew, -ymax, ymax, NEARCLIP,
            Cvar_GetValue(&gl_farclip));
}

/*
=============
R_SetupGL
=============
*/
void R_SetupGL(void) {
  // johnfitz -- rewrote this section
  glMatrixMode(GL_PROJECTION);
  glLoadIdentity();
  glViewport(R_Refdef_vrect_x(),
             GL_Height() - R_Refdef_vrect_y() - R_Refdef_vrect_height(),
             R_Refdef_vrect_width(), R_Refdef_vrect_height());
  // johnfitz

  GL_SetFrustum(r_fovx, r_fovy);  // johnfitz -- use r_fov* vars

  //	glCullFace(GL_BACK); //johnfitz -- glquake used CCW with backwards
  // culling -- let's do it right

  glMatrixMode(GL_MODELVIEW);
  glLoadIdentity();

  glRotatef(-90, 1, 0, 0);  // put Z going up
  glRotatef(90, 0, 0, 1);   // put Z going up
  vec3_t viewangles = {R_Refdef_viewangles(0), R_Refdef_viewangles(1),
                       R_Refdef_viewangles(2)};
  glRotatef(-viewangles[2], 1, 0, 0);
  glRotatef(-viewangles[0], 0, 1, 0);
  glRotatef(-viewangles[1], 0, 0, 1);
  vec3_t vieworg = {R_Refdef_vieworg(0), R_Refdef_vieworg(1),
                    R_Refdef_vieworg(2)};
  glTranslatef(-vieworg[0], -vieworg[1], -vieworg[2]);

  //
  // set drawing parms
  //
  if (Cvar_GetValue(&gl_cull)) {
    glEnable(GL_CULL_FACE);
  } else {
    glDisable(GL_CULL_FACE);
  }

  glDisable(GL_BLEND);
  glDisable(GL_ALPHA_TEST);
  glEnable(GL_DEPTH_TEST);
}

/*
=============
R_Clear -- johnfitz -- rewritten and gutted
=============
*/
void R_Clear(void) {
  unsigned int clearbits;

  clearbits = GL_DEPTH_BUFFER_BIT;
  // from mh -- if we get a stencil buffer, we should clear it, even though we
  // don't use it
  if (gl_stencilbits) clearbits |= GL_STENCIL_BUFFER_BIT;
  if (Cvar_GetValue(&gl_clear)) clearbits |= GL_COLOR_BUFFER_BIT;
  glClear(clearbits);
}

/*
===============
R_SetupScene -- johnfitz -- this is the stuff that needs to be done once per eye
in stereo mode
===============
*/
void R_SetupScene(void) {
  R_PushDlights();
  R_AnimateLight();
  r_framecount++;
  R_SetupGL();
}

/*
===============
R_SetupView -- johnfitz -- this is the stuff that needs to be done once per
frame, even in stereo mode
===============
*/
void R_SetupView(void) {
  Fog_SetupFrame();  // johnfitz

  // build the transformation matrix for the given view angles
  vec3_t vieworg = {R_Refdef_vieworg(0), R_Refdef_vieworg(1),
                    R_Refdef_vieworg(2)};
  VectorCopy(vieworg, r_origin);
  vec3_t viewangles = {R_Refdef_viewangles(0), R_Refdef_viewangles(1),
                       R_Refdef_viewangles(2)};
  AngleVectors(viewangles, vpn, vright, vup);
  UpdateVpnGo();

  // current viewleaf
  r_oldviewleaf = r_viewleaf;
  r_viewleaf = Mod_PointInLeaf(r_origin, cl.worldmodel);

  V_SetContentsColor(r_viewleaf->contents);
  V_CalcBlend();

  // johnfitz -- calculate r_fovx and r_fovy here
  r_fovx = R_Refdef_fov_x();
  r_fovy = R_Refdef_fov_y();
  if (Cvar_GetValue(&r_waterwarp)) {
    int contents = Mod_PointInLeaf(r_origin, cl.worldmodel)->contents;
    if (contents == CONTENTS_WATER || contents == CONTENTS_SLIME ||
        contents == CONTENTS_LAVA) {
      // variance is a percentage of width, where width = 2 * tan(fov / 2)
      // otherwise the effect is too dramatic at high FOV and too subtle at low
      // FOV.  what a mess!
      r_fovx = atan(tan(DEG2RAD(R_Refdef_fov_x()) / 2) *
                    (0.97 + sin(CL_Time() * 1.5) * 0.03)) *
               2 / M_PI_DIV_180;
      r_fovy = atan(tan(DEG2RAD(R_Refdef_fov_y()) / 2) *
                    (1.03 - sin(CL_Time() * 1.5) * 0.03)) *
               2 / M_PI_DIV_180;
    }
  }
  // johnfitz

  R_SetFrustum(r_fovx, r_fovy);  // johnfitz -- use r_fov* vars

  R_MarkSurfaces();  // johnfitz -- create texture chains from PVS

  R_CullSurfaces();  // johnfitz -- do after R_SetFrustum and R_MarkSurfaces

  R_UpdateWarpTextures();  // johnfitz -- do this before R_Clear

  R_Clear();

  // johnfitz -- cheat-protect some draw modes
  r_drawflat_cheatsafe = r_fullbright_cheatsafe = r_lightmap_cheatsafe = false;
  r_drawworld_cheatsafe = true;
  if (CL_MaxClients() == 1) {
    if (!Cvar_GetValue(&r_drawworld)) r_drawworld_cheatsafe = false;

    if (Cvar_GetValue(&r_drawflat))
      r_drawflat_cheatsafe = true;
    else if (Cvar_GetValue(&r_fullbright) || !cl.worldmodel->lightdata)
      r_fullbright_cheatsafe = true;
    else if (Cvar_GetValue(&r_lightmap))
      r_lightmap_cheatsafe = true;
  }
  // johnfitz
}

//==============================================================================
//
// RENDER VIEW
//
//==============================================================================

/*
=============
R_DrawEntitiesOnList
=============
*/
void R_DrawEntitiesOnList(qboolean alphapass)  // johnfitz -- added parameter
{
  int i;

  if (!Cvar_GetValue(&r_drawentities)) return;

  // johnfitz -- sprites are not a special case
  for (i = 0; i < cl_numvisedicts; i++) {
    currententity = cl_visedicts[i];

    // johnfitz -- if alphapass is true, draw only alpha entites this time
    // if alphapass is false, draw only nonalpha entities this time
    if ((ENTALPHA_DECODE(currententity->alpha) < 1 && !alphapass) ||
        (ENTALPHA_DECODE(currententity->alpha) == 1 && alphapass))
      continue;

    // johnfitz -- chasecam
    if (currententity == &cl_entities[CL_Viewentity()])
      currententity->angles[0] *= 0.3;
    // johnfitz

    switch (currententity->model->Type) {
      case mod_alias:
        R_DrawAliasModel(currententity);
        break;
      case mod_brush:
        R_DrawBrushModel(currententity);
        break;
      case mod_sprite:
        //THERJAK
        R_DrawSpriteModel(currententity);
        break;
    }
  }
}

/*
=============
R_DrawViewModel -- johnfitz -- gutted
=============
*/
void R_DrawViewModel(void) {
  if (!Cvar_GetValue(&r_drawviewmodel) || !Cvar_GetValue(&r_drawentities) ||
      Cvar_GetValue(&chase_active))
    return;

  if (CL_HasItem(IT_INVISIBILITY) || CL_Stats(STAT_HEALTH) <= 0) return;

  currententity = &cl_viewent;
  if (!currententity->model) return;

  // johnfitz -- this fixes a crash
  if (currententity->model->Type != mod_alias) return;
  // johnfitz

  // hack the depth range to prevent view model from poking into walls
  glDepthRange(0, 0.3);
  R_DrawAliasModel(currententity);
  glDepthRange(0, 1);
}

/*
================
R_EmitWirePoint -- johnfitz -- draws a wireframe cross shape for point entities
================
*/
void R_EmitWirePoint(vec3_t origin) {
  int size = 8;

  glBegin(GL_LINES);
  glVertex3f(origin[0] - size, origin[1], origin[2]);
  glVertex3f(origin[0] + size, origin[1], origin[2]);
  glVertex3f(origin[0], origin[1] - size, origin[2]);
  glVertex3f(origin[0], origin[1] + size, origin[2]);
  glVertex3f(origin[0], origin[1], origin[2] - size);
  glVertex3f(origin[0], origin[1], origin[2] + size);
  glEnd();
}

/*
================
R_EmitWireBox -- johnfitz -- draws one axis aligned bounding box
================
*/
void R_EmitWireBox(vec3_t mins, vec3_t maxs) {
  glBegin(GL_QUAD_STRIP);
  glVertex3f(mins[0], mins[1], mins[2]);
  glVertex3f(mins[0], mins[1], maxs[2]);
  glVertex3f(maxs[0], mins[1], mins[2]);
  glVertex3f(maxs[0], mins[1], maxs[2]);
  glVertex3f(maxs[0], maxs[1], mins[2]);
  glVertex3f(maxs[0], maxs[1], maxs[2]);
  glVertex3f(mins[0], maxs[1], mins[2]);
  glVertex3f(mins[0], maxs[1], maxs[2]);
  glVertex3f(mins[0], mins[1], mins[2]);
  glVertex3f(mins[0], mins[1], maxs[2]);
  glEnd();
}

/*
================
R_DrawShadows
================
*/
void R_DrawShadows(void) {
  int i;

  if (!Cvar_GetValue(&r_shadows) || !Cvar_GetValue(&r_drawentities) ||
      r_drawflat_cheatsafe || r_lightmap_cheatsafe)
    return;

  // Use stencil buffer to prevent self-intersecting shadows, from Baker (MarkV)
  if (gl_stencilbits) {
    glClear(GL_STENCIL_BUFFER_BIT);
    glStencilFunc(GL_EQUAL, 0, ~0);
    glStencilOp(GL_KEEP, GL_KEEP, GL_INCR);
    glEnable(GL_STENCIL_TEST);
  }

  for (i = 0; i < cl_numvisedicts; i++) {
    currententity = cl_visedicts[i];

    if (currententity->model->Type != mod_alias) continue;

    if (currententity == &cl_viewent) return;

    GL_DrawAliasShadow(currententity);
  }

  if (gl_stencilbits) {
    glDisable(GL_STENCIL_TEST);
  }
}

/*
================
R_RenderScene
================
*/
void R_RenderScene(void) {
  R_SetupScene();  // johnfitz -- this does everything that should be done once
                   // per call to RenderScene

  Fog_EnableGFog();  // johnfitz

  Sky_DrawSky();  // johnfitz

  R_DrawWorld();

  S_ExtraUpdate();  // don't let sound get messed up if going slow

  R_DrawShadows();  // johnfitz -- render entity shadows

  R_DrawEntitiesOnList(
      false);  // johnfitz -- false means this is the pass for nonalpha entities

  R_DrawWorld_Water();  // johnfitz -- drawn here since they might have
                        // transparency

  R_DrawEntitiesOnList(
      true);  // johnfitz -- true means this is the pass for alpha entities

  R_RenderDlights();  // triangle fan dlights -- johnfitz -- moved after water

  ParticlesDraw();

  Fog_DisableGFog();  // johnfitz

  R_DrawViewModel();  // johnfitz -- moved here from R_RenderView
}

/*
================
R_RenderView
================
*/
void R_RenderView(void) {
  double time1, time2;

  if (Cvar_GetValue(&r_norefresh)) return;

  if (!cl.worldmodel) Go_Error("R_RenderView: NULL worldmodel");

  time1 = 0; /* avoid compiler warning */
  if (Cvar_GetValue(&r_speeds)) {
    glFinish();
    time1 = Sys_DoubleTime();

    // johnfitz -- rendering statistics
    rs_brushpolys = rs_aliaspolys = rs_skypolys = rs_particles = rs_fogpolys =
        rs_megatexels = rs_dynamiclightmaps = rs_aliaspasses = rs_skypasses =
            rs_brushpasses = 0;
  } else if (Cvar_GetValue(&gl_finish)) {
    glFinish();
  }

  R_SetupView();  // johnfitz -- this does everything that should be done once
                  // per frame

  // johnfitz -- stereo rendering -- full of hacky goodness
  if (Cvar_GetValue(&r_stereo)) {
    float eyesep = CLAMP(-8.0f, Cvar_GetValue(&r_stereo), 8.0f);
    float fdepth = CLAMP(32.0f, Cvar_GetValue(&r_stereodepth), 1024.0f);

    vec3_t viewangles = {R_Refdef_viewangles(0), R_Refdef_viewangles(1),
                         R_Refdef_viewangles(2)};
    AngleVectors(viewangles, vpn, vright, vup);
    UpdateVpnGo();

    // render left eye (red)
    glColorMask(1, 0, 0, 1);
    VectorMA(r_refdef.vieworg, -0.5f * eyesep, vright, r_refdef.vieworg);
    frustum_skew = 0.5 * eyesep * NEARCLIP / fdepth;
    srand((int)(CL_Time() * 1000));  // sync random stuff between eyes

    R_RenderScene();

    // render right eye (cyan)
    glClear(GL_DEPTH_BUFFER_BIT);
    glColorMask(0, 1, 1, 1);
    VectorMA(r_refdef.vieworg, 1.0f * eyesep, vright, r_refdef.vieworg);
    frustum_skew = -frustum_skew;
    srand((int)(CL_Time() * 1000));  // sync random stuff between eyes

    R_RenderScene();

    // restore
    glColorMask(1, 1, 1, 1);
    VectorMA(r_refdef.vieworg, -0.5f * eyesep, vright, r_refdef.vieworg);
    frustum_skew = 0.0f;
  } else {
    R_RenderScene();
  }
  // johnfitz

  // johnfitz -- modified r_speeds output
  time2 = Sys_DoubleTime();
  if (Cvar_GetValue(&r_pos)) {
    Con_Printf("x %i y %i z %i (pitch %i yaw %i roll %i)\n",
               (int)cl_entities[CL_Viewentity()].origin[0],
               (int)cl_entities[CL_Viewentity()].origin[1],
               (int)cl_entities[CL_Viewentity()].origin[2], (int)CLPitch(),
               (int)CLYaw(), (int)CLRoll());
  } else if (Cvar_GetValue(&r_speeds) == 2) {
    Con_Printf(
        "%3i ms  %4i/%4i wpoly %4i/%4i epoly %3i lmap %4i/%4i sky %1.1f mtex\n",
        (int)((time2 - time1) * 1000), rs_brushpolys, rs_brushpasses,
        rs_aliaspolys, rs_aliaspasses, rs_dynamiclightmaps, rs_skypolys,
        rs_skypasses, TexMgrFrameUsage());
  } else if (Cvar_GetValue(&r_speeds)) {
    Con_Printf("%3i ms  %4i wpoly %4i epoly %3i lmap\n",
               (int)((time2 - time1) * 1000), rs_brushpolys, rs_aliaspolys,
               rs_dynamiclightmaps);
  }
  // johnfitz
}
