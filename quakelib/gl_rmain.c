// r_main.c

#include "quakedef.h"

vec3_t modelorg, r_entorigin;

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
                       // r_waterwarp

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

void R_RotateForEntity(vec3_t origin, vec3_t angles) {
  glTranslatef(origin[0], origin[1], origin[2]);
  glRotatef(angles[1], 0, 0, 1);
  glRotatef(-angles[0], 0, 1, 0);
  glRotatef(angles[2], 1, 0, 0);
}

void GL_PolygonOffset(int offset) {
  if (offset == OFFSET_DECAL) {
    glEnable(GL_POLYGON_OFFSET_FILL);
    glEnable(GL_POLYGON_OFFSET_LINE);
    glPolygonOffset(-1, offset);
  } else { // OFFSET_NONE
    glDisable(GL_POLYGON_OFFSET_FILL);
    glDisable(GL_POLYGON_OFFSET_LINE);
  }
}

#define DEG2RAD(a) ((a)*M_PI_DIV_180)

// THERJAK
#define NEARCLIP 4
void GL_SetFrustum(float fovx, float fovy) {
  float xmax, ymax;
  xmax = NEARCLIP * tan(fovx * M_PI / 360.0);
  ymax = NEARCLIP * tan(fovy * M_PI / 360.0);
  glFrustum(-xmax, xmax, -ymax, ymax, NEARCLIP, Cvar_GetValue(&gl_farclip));
}

void R_SetupGL(void) {
  glMatrixMode(GL_PROJECTION);
  glLoadIdentity();
  glViewport(R_Refdef_vrect_x(),
             GL_Height() - R_Refdef_vrect_y() - R_Refdef_vrect_height(),
             R_Refdef_vrect_width(), R_Refdef_vrect_height());

  GL_SetFrustum(r_fovx, r_fovy);

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

  if (Cvar_GetValue(&gl_cull)) {
    glEnable(GL_CULL_FACE);
  } else {
    glDisable(GL_CULL_FACE);
  }

  glDisable(GL_BLEND);
  glDisable(GL_ALPHA_TEST);
  glEnable(GL_DEPTH_TEST);
}

void R_Clear(void) {
  unsigned int clearbits;

  clearbits = GL_DEPTH_BUFFER_BIT;
  // if we get a stencil buffer, we should clear it, even though we
  // don't use it
  if (gl_stencilbits) clearbits |= GL_STENCIL_BUFFER_BIT;
  if (Cvar_GetValue(&gl_clear)) clearbits |= GL_COLOR_BUFFER_BIT;
  glClear(clearbits);
}

void R_SetupScene(void) {
  R_PushDlights();
  R_AnimateLight();
  r_framecount++;
  R_SetupGL();
}

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

  R_SetFrustum(r_fovx, r_fovy);
  R_MarkSurfaces();  // create texture chains from PVS
  R_CullSurfaces();
  R_UpdateWarpTextures();
  R_Clear();
}

void R_DrawEntitiesOnList(qboolean alphapass)
{
  int i;
  entity_t* ce;

  if (!Cvar_GetValue(&r_drawentities)) return;

  for (i = 0; i < cl_numvisedicts; i++) {
    ce = cl_visedicts[i];

    if ((ENTALPHA_DECODE(ce->alpha) < 1 && !alphapass) ||
        (ENTALPHA_DECODE(ce->alpha) == 1 && alphapass))
      continue;

    if (ce == CLViewEntity())
      ce->angles[0] *= 0.3;

    switch (ce->model->Type) {
      case mod_alias:
        R_DrawAliasModel(ce);
        break;
      case mod_brush:
        R_DrawBrushModel(ce);
        break;
      case mod_sprite:
        // THERJAK
        R_DrawSpriteModel(ce);
        break;
    }
  }
}

void R_DrawShadows(void) {
  int i;
  entity_t* ce;

  if (!Cvar_GetValue(&r_shadows) || !Cvar_GetValue(&r_drawentities)) return;

  // Use stencil buffer to prevent self-intersecting shadows
  if (gl_stencilbits) {
    glClear(GL_STENCIL_BUFFER_BIT);
    glStencilFunc(GL_EQUAL, 0, ~0);
    glStencilOp(GL_KEEP, GL_KEEP, GL_INCR);
    glEnable(GL_STENCIL_TEST);
  }

  for (i = 0; i < cl_numvisedicts; i++) {
    ce = cl_visedicts[i];

    if (ce->model->Type != mod_alias) continue;

    if (ce == &cl_viewent) return;

    GL_DrawAliasShadow(ce);
  }

  if (gl_stencilbits) {
    glDisable(GL_STENCIL_TEST);
  }
}

void R_RenderScene(void) {
  R_SetupScene();
  Fog_EnableGFog();
  SkyDrawSky();
  R_DrawWorld();
  S_ExtraUpdate();  // don't let sound get messed up if going slow
  R_DrawShadows();
  // false means this is the pass for nonalpha entities
  R_DrawEntitiesOnList(false);
  R_DrawWorld_Water();  // drawn here since they might have transparency
  // true means this is the pass for alpha entities
  R_DrawEntitiesOnList(true);
  R_RenderDlights();  // triangle fan dlights
  ParticlesDraw();
  Fog_DisableGFog();
  R_DrawViewModel();
}

void R_RenderView(void) {
  double time1, time2;

  if (Cvar_GetValue(&r_norefresh)) return;

  if (!cl.worldmodel) Go_Error("R_RenderView: NULL worldmodel");

  time1 = 0; /* avoid compiler warning */
  if (Cvar_GetValue(&r_speeds)) {
    glFinish();
    time1 = Sys_DoubleTime();

    // rendering statistics
    rs_brushpolys = rs_aliaspolys = rs_skypolys = rs_particles = rs_fogpolys =
        rs_megatexels = rs_dynamiclightmaps = rs_aliaspasses = rs_skypasses =
            rs_brushpasses = 0;
  } else if (Cvar_GetValue(&gl_finish)) {
    glFinish();
  }

  R_SetupView();
  R_RenderScene();

  time2 = Sys_DoubleTime();
  if (Cvar_GetValue(&r_pos)) {
    printPosition();
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
}
