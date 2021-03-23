// SPDX-License-Identifier: GPL-2.0-or-later
// r_misc.c

#include "quakedef.h"

// johnfitz -- new cvars
extern cvar_t r_clearcolor;
extern cvar_t r_drawflat;
extern cvar_t r_flatlightstyles;
extern cvar_t gl_fullbrights;
extern cvar_t gl_farclip;
extern cvar_t gl_overbright;
extern cvar_t gl_overbright_models;
extern cvar_t r_waterquality;
extern cvar_t r_oldwater;
extern cvar_t r_waterwarp;
extern cvar_t r_oldskyleaf;
extern cvar_t r_drawworld;
extern cvar_t r_showtris;
extern cvar_t r_lerpmodels;
extern cvar_t r_lerpmove;
extern cvar_t r_nolerp_list;
extern cvar_t r_noshadow_list;
// johnfitz
extern cvar_t gl_zfix;  // QuakeSpasm z-fighting fix

/*
====================
GL_Overbright_f -- johnfitz
====================
*/
static void GL_Overbright_f(cvar_t *var) { R_RebuildAllLightmaps(); }

/*
====================
R_SetClearColor_f -- johnfitz
====================
*/
static void R_SetClearColor_f(cvar_t *var) {
  int s;

  s = (int)Cvar_GetValue(&r_clearcolor) & 0xFF;
  glClearColor(D8To24Table(s, 0) / 255.0, D8To24Table(s, 1) / 255.0,
               D8To24Table(s, 2) / 255.0, 0);
}

/*
====================
R_Novis_f -- johnfitz
====================
*/
static void R_VisChanged(cvar_t *var) {
  extern int vis_changed;
  vis_changed = 1;
}

/*
===============
R_Model_ExtraFlags_List_f -- johnfitz -- called when r_nolerp_list or
r_noshadow_list cvar changes
===============
*/
static void R_Model_ExtraFlags_List_f(cvar_t *var) {
  int i;
  for (i = 0; i < MAX_MODELS; i++) Mod_SetExtraFlags(cl.model_precache[i]);
}

/*
====================
R_SetWateralpha_f -- ericw
====================
*/
static void R_SetWateralpha_f(cvar_t *var) {
  map_wateralpha = Cvar_GetValue(var);
}

/*
====================
R_SetLavaalpha_f -- ericw
====================
*/
static void R_SetLavaalpha_f(cvar_t *var) {
  map_lavaalpha = Cvar_GetValue(var);
}

/*
====================
R_SetTelealpha_f -- ericw
====================
*/
static void R_SetTelealpha_f(cvar_t *var) {
  map_telealpha = Cvar_GetValue(var);
}

/*
====================
R_SetSlimealpha_f -- ericw
====================
*/
static void R_SetSlimealpha_f(cvar_t *var) {
  map_slimealpha = Cvar_GetValue(var);
}

/*
====================
GL_WaterAlphaForSurfface -- ericw
====================
*/
float GL_WaterAlphaForSurface(msurface_t *fa) {
  if (fa->flags & SURF_DRAWLAVA)
    return map_lavaalpha > 0 ? map_lavaalpha : map_wateralpha;
  else if (fa->flags & SURF_DRAWTELE)
    return map_telealpha > 0 ? map_telealpha : map_wateralpha;
  else if (fa->flags & SURF_DRAWSLIME)
    return map_slimealpha > 0 ? map_slimealpha : map_wateralpha;
  else
    return map_wateralpha;
}

/*
===============
R_Init
===============
*/
void R_Init(void) {
  extern cvar_t gl_finish;

  Cvar_FakeRegister(&r_norefresh, "r_norefresh");
  Cvar_FakeRegister(&r_lightmap, "r_lightmap");
  Cvar_FakeRegister(&r_fullbright, "r_fullbright");
  Cvar_FakeRegister(&r_drawentities, "r_drawentities");
  Cvar_FakeRegister(&r_drawviewmodel, "r_drawviewmodel");
  Cvar_FakeRegister(&r_shadows, "r_shadows");
  Cvar_FakeRegister(&r_wateralpha, "r_wateralpha");
  Cvar_SetCallback(&r_wateralpha, R_SetWateralpha_f);
  Cvar_FakeRegister(&r_dynamic, "r_dynamic");
  Cvar_FakeRegister(&r_novis, "r_novis");
  Cvar_SetCallback(&r_novis, R_VisChanged);

  Cvar_FakeRegister(&gl_finish, "gl_finish");
  Cvar_FakeRegister(&gl_clear, "gl_clear");
  Cvar_FakeRegister(&gl_cull, "gl_cull");
  Cvar_FakeRegister(&gl_smoothmodels, "gl_smoothmodels");
  Cvar_FakeRegister(&gl_affinemodels, "gl_affinemodels");
  Cvar_FakeRegister(&gl_polyblend, "gl_polyblend");
  Cvar_FakeRegister(&gl_flashblend, "gl_flashblend");
  Cvar_FakeRegister(&gl_playermip, "gl_playermip");
  Cvar_FakeRegister(&gl_nocolors, "gl_nocolors");

  Cvar_FakeRegister(&r_clearcolor, "r_clearcolor");
  Cvar_SetCallback(&r_clearcolor, R_SetClearColor_f);
  Cvar_FakeRegister(&r_waterquality, "r_waterquality");
  Cvar_FakeRegister(&r_oldwater, "r_oldwater");
  Cvar_FakeRegister(&r_waterwarp, "r_waterwarp");
  Cvar_FakeRegister(&r_drawflat, "r_drawflat");
  Cvar_FakeRegister(&r_flatlightstyles, "r_flatlightstyles");
  Cvar_FakeRegister(&r_oldskyleaf, "r_oldskyleaf");
  Cvar_SetCallback(&r_oldskyleaf, R_VisChanged);
  Cvar_FakeRegister(&r_drawworld, "r_drawworld");
  Cvar_FakeRegister(&r_showtris, "r_showtris");
  Cvar_FakeRegister(&gl_farclip, "gl_farclip");
  Cvar_FakeRegister(&gl_fullbrights, "gl_fullbrights");
  Cvar_FakeRegister(&gl_overbright, "gl_overbright");
  Cvar_SetCallback(&gl_overbright, GL_Overbright_f);
  Cvar_FakeRegister(&gl_overbright_models, "gl_overbright_models");
  Cvar_FakeRegister(&r_lerpmodels, "r_lerpmodels");
  Cvar_FakeRegister(&r_lerpmove, "r_lerpmove");
  Cvar_FakeRegister(&r_nolerp_list, "r_nolerp_list");
  Cvar_SetCallback(&r_nolerp_list, R_Model_ExtraFlags_List_f);
  Cvar_FakeRegister(&r_noshadow_list, "r_noshadow_list");

  Cvar_SetCallback(&r_noshadow_list, R_Model_ExtraFlags_List_f);

  Cvar_FakeRegister(&gl_zfix, "gl_zfix");
  Cvar_FakeRegister(&r_lavaalpha, "r_lavaalpha");
  Cvar_SetCallback(&r_lavaalpha, R_SetLavaalpha_f);
  Cvar_FakeRegister(&r_telealpha, "r_telealpha");
  Cvar_SetCallback(&r_telealpha, R_SetTelealpha_f);
  Cvar_FakeRegister(&r_slimealpha, "r_slimealpha");
  Cvar_SetCallback(&r_slimealpha, R_SetSlimealpha_f);

  ParticlesInit();
  R_SetClearColor_f(&r_clearcolor);

  SkyInit();
  Fog_Init();
}

/*
===============
R_TranslateNewPlayerSkin -- johnfitz -- split off of TranslatePlayerSkin -- this
is called when
the skin or model actually changes, instead of just new colors
added bug fix from bengt jardup
===============
*/
// THERJAK: this and the aliashdr_t
void R_TranslateNewPlayerSkin(int playernum) {
  char name[64];
  byte *pixels;
  aliashdr_t *paliashdr;
  int skinnum;
  entity_t* ce;

  // get correct texture pixels
  ce = ClientEntity(playernum);

  if (!ce->model || ce->model->Type != mod_alias) return;

  paliashdr = (aliashdr_t *)Mod_Extradata(ce->model);

  skinnum = ce->skinnum;

  // TODO: move these tests to the place where skinnum gets received from the
  // server
  if (skinnum < 0 || skinnum >= paliashdr->numskins) {
    Con_DPrintf("(%d): Invalid player skin #%d\n", playernum, skinnum);
    skinnum = 0;
  }

  pixels = (byte *)paliashdr +
           paliashdr->texels[skinnum];  // This is not a persistent place!

  // upload new image
  LoadPlayerTexture(playernum, paliashdr->skinwidth, paliashdr->skinheight, pixels);

  // now recolor it
  R_TranslatePlayerSkin(playernum);
}

/*
=============
R_ParseWorldspawn

called at map load
=============
*/
static void R_ParseWorldspawn(void) {
  char key[128], value[4096];
  const char *data;

  map_wateralpha = Cvar_GetValue(&r_wateralpha);
  map_lavaalpha = Cvar_GetValue(&r_lavaalpha);
  map_telealpha = Cvar_GetValue(&r_telealpha);
  map_slimealpha = Cvar_GetValue(&r_slimealpha);

  data = COM_Parse(cl.worldmodel->entities);
  if (!data) return;                // error
  if (com_token[0] != '{') return;  // error
  while (1) {
    data = COM_Parse(data);
    if (!data) return;               // error
    if (com_token[0] == '}') break;  // end of worldspawn
    if (com_token[0] == '_')
      strcpy(key, com_token + 1);
    else
      strcpy(key, com_token);
    while (key[strlen(key) - 1] == ' ')  // remove trailing spaces
      key[strlen(key) - 1] = 0;
    data = COM_Parse(data);
    if (!data) return;  // error
    strcpy(value, com_token);

    if (!strcmp("wateralpha", key)) map_wateralpha = atof(value);

    if (!strcmp("lavaalpha", key)) map_lavaalpha = atof(value);

    if (!strcmp("telealpha", key)) map_telealpha = atof(value);

    if (!strcmp("slimealpha", key)) map_slimealpha = atof(value);
  }
}

/*
===============
R_NewMap
===============
*/
void R_NewMap(void) {
  int i;

  for (i = 0; i < 256; i++) d_lightstylevalue[i] = 264;  // normal light value

  // clear out efrags in case the level hasn't been reloaded
  ClearMapEntityFragments();

  r_viewleaf = NULL;
  ParticlesClear();

  GL_BuildLightmaps();
  GL_BuildBModelVertexBuffer();
  // ericw -- no longer load alias models into a VBO here, it's done in
  // Mod_LoadAliasModel

  R_framecount_reset();     // johnfitz -- paranoid?
  R_visframecount_reset();  // johnfitz -- paranoid?

  SkyNewMap();          // johnfitz -- skybox in worldspawn
  Fog_NewMap();         // johnfitz -- global fog in worldspawn
  R_ParseWorldspawn();  // ericw -- wateralpha, lavaalpha, telealpha, slimealpha
                        // in worldspawn

  // johnfitz -- is this the right place to set this?
  load_subdivide_size = Cvar_GetValue(&gl_subdivide_size);
}

static GLuint gl_programs[16];
static int gl_num_programs;

static qboolean GL_CheckShader(GLuint shader) {
  GLint status;
  glGetShaderiv(shader, GL_COMPILE_STATUS, &status);

  if (status != GL_TRUE) {
    char infolog[1024];

    memset(infolog, 0, sizeof(infolog));
    glGetShaderInfoLog(shader, sizeof(infolog), NULL, infolog);

    Con_Warning("GLSL program failed to compile: %s", infolog);

    return false;
  }
  return true;
}

static qboolean GL_CheckProgram(GLuint program) {
  GLint status;
  glGetProgramiv(program, GL_LINK_STATUS, &status);

  if (status != GL_TRUE) {
    char infolog[1024];

    memset(infolog, 0, sizeof(infolog));
    glGetProgramInfoLog(program, sizeof(infolog), NULL, infolog);

    Con_Warning("GLSL program failed to link: %s", infolog);

    return false;
  }
  return true;
}

/*
=============
GL_GetUniformLocation
=============
*/
GLint GL_GetUniformLocation(GLuint *programPtr, const char *name) {
  GLint location;

  if (!programPtr) return -1;

  location = glGetUniformLocation(*programPtr, name);
  if (location == -1) {
    Con_Warning("GL_GetUniformLocationFunc %s failed\n", name);
    *programPtr = 0;
  }
  return location;
}

/*
====================
GL_CreateProgram

Compiles and returns GLSL program.
====================
*/
GLuint GL_CreateProgram(const GLchar *vertSource, const GLchar *fragSource,
                        int numbindings,
                        const glsl_attrib_binding_t *bindings) {
  int i;
  GLuint program, vertShader, fragShader;

  vertShader = glCreateShader(GL_VERTEX_SHADER);
  glShaderSource(vertShader, 1, &vertSource, NULL);
  glCompileShader(vertShader);
  if (!GL_CheckShader(vertShader)) {
    glDeleteShader(vertShader);
    return 0;
  }

  fragShader = glCreateShader(GL_FRAGMENT_SHADER);
  glShaderSource(fragShader, 1, &fragSource, NULL);
  glCompileShader(fragShader);
  if (!GL_CheckShader(fragShader)) {
    glDeleteShader(vertShader);
    glDeleteShader(fragShader);
    return 0;
  }

  program = glCreateProgram();
  glAttachShader(program, vertShader);
  glDeleteShader(vertShader);
  glAttachShader(program, fragShader);
  glDeleteShader(fragShader);

  for (i = 0; i < numbindings; i++) {
    glBindAttribLocation(program, bindings[i].attrib, bindings[i].name);
  }

  glLinkProgram(program);

  if (!GL_CheckProgram(program)) {
    glDeleteProgram(program);
    return 0;
  } else {
    if (gl_num_programs == (sizeof(gl_programs) / sizeof(GLuint)))
      Host_Error("gl_programs overflow");

    gl_programs[gl_num_programs] = program;
    gl_num_programs++;

    return program;
  }
}

/*
====================
R_DeleteShaders

Deletes any GLSL programs that have been created.
====================
*/
void R_DeleteShaders(void) {
  int i;

  for (i = 0; i < gl_num_programs; i++) {
    glDeleteProgram(gl_programs[i]);
    gl_programs[i] = 0;
  }
  gl_num_programs = 0;
}
GLuint current_array_buffer, current_element_array_buffer;

/*
====================
GL_BindBuffer

glBindBuffer wrapper
====================
*/
void GL_BindBuffer(GLenum target, GLuint buffer) {
  GLuint *cache;

  switch (target) {
    case GL_ARRAY_BUFFER:
      cache = &current_array_buffer;
      break;
    case GL_ELEMENT_ARRAY_BUFFER:
      cache = &current_element_array_buffer;
      break;
    default:
      Host_Error("GL_BindBuffer: unsupported target %d", (int)target);
      return;
  }

  if (*cache != buffer) {
    *cache = buffer;
    glBindBuffer(target, *cache);
  }
}

/*
====================
GL_ClearBufferBindings

This must be called if you do anything that could make the cached bindings
invalid (e.g. manually binding, destroying the context).
====================
*/
void GL_ClearBufferBindings() {
  current_array_buffer = 0;
  current_element_array_buffer = 0;
  glBindBuffer(GL_ARRAY_BUFFER, 0);
  glBindBuffer(GL_ELEMENT_ARRAY_BUFFER, 0);
}
