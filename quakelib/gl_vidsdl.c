#include "quakedef.h"

#define MAX_MODE_LIST 600
#define MAX_BPPS_LIST 5
#define WARP_WIDTH 320
#define WARP_HEIGHT 200
#define MAXWIDTH 10000
#define MAXHEIGHT 10000

//====================================

static cvar_t vid_vsync;
int gl_stencilbits; // TODO: fill with (SDL_GL_GetAttribute(SDL_GL_STENCIL_SIZE, &gl_stencilbits)

cvar_t vid_gamma;
cvar_t vid_contrast;

void GL_CheckExtensions() {
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
void GL_SetupState(void) {
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

  GL_CheckExtensions();

  // johnfitz -- intel video workarounds from Baker
  if (!strcmp(gl_vendor, "Intel")) {
    Con_Printf("Intel Display Adapter detected, enabling gl_clear\n");
    Cbuf_AddText("gl_clear 1");
  }

  GLAlias_CreateShaders();
  GL_ClearBufferBindings();
}

void VID_Init(void) {
  Cvar_FakeRegister(&vid_vsync, "vid_vsync");
  Cvar_FakeRegister(&vid_gamma, "gamma");
  Cvar_FakeRegister(&vid_contrast, "contrast");

  VID_Init_Go();

  GL_Init();
  GL_SetupState();
}

