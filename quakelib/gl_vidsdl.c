#include "quakedef.h"

#define MAX_MODE_LIST 600
#define MAX_BPPS_LIST 5
#define WARP_WIDTH 320
#define WARP_HEIGHT 200
#define MAXWIDTH 10000
#define MAXHEIGHT 10000

typedef struct {
  int width;
  int height;
} vmode_t;

static char *gl_extensions_nice;

static void GL_Init(void);
static void GL_SetupState(void);

qboolean gl_glsl_able = false;
int gl_stencilbits;

//====================================

static cvar_t vid_fullscreen;
static cvar_t vid_width;
static cvar_t vid_height;
static cvar_t vid_vsync;

cvar_t vid_gamma;
cvar_t vid_contrast;

static void VID_Restart(void) {
  int width, height;
  qboolean fullscreen;

  if (VID_Locked() || !VIDChanged()) return;

  width = (int)Cvar_GetValue(&vid_width);
  height = (int)Cvar_GetValue(&vid_height);
  fullscreen = Cvar_GetValue(&vid_fullscreen) ? true : false;

  //
  // validate new mode
  //
  if (!VID_ValidMode(width, height, fullscreen)) {
    Sys_Print("VID_ValidMode == false");
    Con_Printf("%dx%d %s is not a valid mode\n", width, height,
               fullscreen ? "fullscreen" : "windowed");
    return;
  }

  // ericw -- OS X, SDL1: textures, VBO's invalid after mode change
  //          OS X, SDL2: still valid after mode change
  // To handle both cases, delete all GL objects (textures, VBO, GLSL) now.
  // We must not interleave deleting the old objects with creating new ones,
  // because
  // one of the new objects could be given the same ID as an invalid handle
  // which is later deleted.

  TexMgrDeleteTextureObjects();
  GLSLGamma_DeleteTexture();
  R_DeleteShaders();
  GL_DeleteBModelVertexBuffer();
  GLMesh_DeleteVertexBuffers();

  //
  // set new mode
  //
  VID_SetMode(width, height, fullscreen);

  GL_Init();
  TexMgrReloadImages();
  GL_BuildBModelVertexBuffer();
  GLMesh_LoadVertexBuffers();
  GL_SetupState();
  Fog_SetupState();

  // warpimages needs to be recalculated
  TexMgrRecalcWarpImageSize();

  UpdateConsoleSize();
  //
  // keep cvars in line with actual mode
  //
  VID_SyncCvars();
  //
  // update mouse grab
  //
  if (GetKeyDest() == key_console || GetKeyDest() == key_menu) {
    if (VID_GetModeState() == MS_WINDOWED)
      IN_Deactivate();
    else if (VID_GetModeState() == MS_FULLSCREEN)
      IN_Activate();
  }
}

static void VID_Test(void) {
  int old_width, old_height, old_fullscreen;

  if (VID_Locked() || !VIDChanged()) return;
  old_width = VID_GetCurrentWidth();
  old_height = VID_GetCurrentHeight();
  old_fullscreen = VID_GetFullscreen();

  VID_Restart();

  if (!SCR_ModalMessage("Would you like to keep this\nvideo mode? (y/n)\n",
                        5.0f)) {
    Cvar_SetValueQuick(&vid_width, old_width);
    Cvar_SetValueQuick(&vid_height, old_height);
    Cvar_SetQuick(&vid_fullscreen, old_fullscreen ? "1" : "0");
    VID_Restart();
  }
}

static qboolean GL_ParseExtensionList(const char *list, const char *name) {
  const char *start;
  const char *where, *terminator;

  if (!list || !name || !*name) return false;
  if (strchr(name, ' ') != NULL)
    return false;  // extension names must not have spaces

  start = list;
  while (1) {
    where = strstr(start, name);
    if (!where) break;
    terminator = where + strlen(name);
    if (where == start || where[-1] == ' ')
      if (*terminator == ' ' || *terminator == '\0') return true;
    start = terminator;
  }
  return false;
}

static void GL_CheckExtensions(const char *gl_extensions) {
  int swap_control;
  // swap control
  //
  if (!VIDGLSwapControl()) {
    Con_Warning(
        "vertical sync not supported (SDL_GL_SetSwapInterval failed)\n");
  } else if ((swap_control = VIDGetSwapInterval()) == -1) {
    SetVIDGLSwapControl(false);
    Con_Warning(
        "vertical sync not supported (SDL_GL_GetSwapInterval failed)\n");
  } else if ((Cvar_GetValue(&vid_vsync) && swap_control != 1) ||
             (!Cvar_GetValue(&vid_vsync) && swap_control != 0)) {
    SetVIDGLSwapControl(false);
    Con_Warning(
        "vertical sync not supported (swap_control doesn't match vid_vsync)\n");
  } else {
    Con_Printf("FOUND: SDL_GL_SetSwapInterval\n");
  }
}

/*
===============
GL_SetupState -- johnfitz

does all the stuff from GL_Init that needs to be done every time a new GL render
context is created
===============
*/
static void GL_SetupState(void) {
  glClearColor(0.15, 0.15, 0.15, 0);
  glCullFace(GL_BACK);
  glFrontFace(GL_CW);
  glEnable(GL_TEXTURE_2D);
  glEnable(GL_ALPHA_TEST);
  glAlphaFunc(GL_GREATER, 0.666);
  glPolygonMode(GL_FRONT_AND_BACK, GL_FILL);
  glShadeModel(GL_FLAT);
  glHint(GL_PERSPECTIVE_CORRECTION_HINT, GL_NICEST);
  glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);
  glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_REPLACE);
  glTexParameterf(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_NEAREST);
  glTexParameterf(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_NEAREST);
  glTexParameterf(GL_TEXTURE_2D, GL_TEXTURE_WRAP_S, GL_REPEAT);
  glTexParameterf(GL_TEXTURE_2D, GL_TEXTURE_WRAP_T, GL_REPEAT);
  glDepthRange(0, 1);
  glDepthFunc(GL_LEQUAL);
}

/*
===============
GL_Init
===============
*/
void GL_Init(void) {
  int gl_version_major;
  int gl_version_minor;
  const char *gl_vendor = (const char *)glGetString(GL_VENDOR);
  const char *gl_renderer = (const char *)glGetString(GL_RENDERER);
  const char *gl_version = (const char *)glGetString(GL_VERSION);
  const char *gl_extensions = (const char *)glGetString(GL_EXTENSIONS);

  Con_SafePrintf("GL_VENDOR: %s\n", gl_vendor);
  Con_SafePrintf("GL_RENDERER: %s\n", gl_renderer);
  Con_SafePrintf("GL_VERSION: %s\n", gl_version);

  if (gl_version == NULL ||
      sscanf(gl_version, "%d.%d", &gl_version_major, &gl_version_minor) < 2) {
    gl_version_major = 0;
  }
  if (gl_version_major < 2) {
    Go_Error("Need OpenGL version 2 or later");
  }

  GL_CheckExtensions(gl_extensions);

  // johnfitz -- intel video workarounds from Baker
  if (!strcmp(gl_vendor, "Intel")) {
    Con_Printf("Intel Display Adapter detected, enabling gl_clear\n");
    Cbuf_AddText("gl_clear 1");
  }

  GLAlias_CreateShaders();
  GL_ClearBufferBindings();
}

void VID_Init(void) {
  Cvar_FakeRegister(&vid_fullscreen, "vid_fullscreen");
  Cvar_FakeRegister(&vid_width, "vid_width");
  Cvar_FakeRegister(&vid_height, "vid_height");
  Cvar_FakeRegister(&vid_vsync, "vid_vsync");
  Cvar_FakeRegister(&vid_gamma, "gamma");
  Cvar_FakeRegister(&vid_contrast, "contrast");

  Cmd_AddCommand("vid_restart", VID_Restart);
  Cmd_AddCommand("vid_test", VID_Test);

  VID_Init_Go();

  GL_Init();
  GL_SetupState();
}

