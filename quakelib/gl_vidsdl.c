#include "quakedef.h"

//====================================

int gl_stencilbits = 0; // TODO(therjak): fill with (SDL_GL_GetAttribute(SDL_GL_STENCIL_SIZE, &gl_stencilbits)

cvar_t vid_gamma;
cvar_t vid_contrast;

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

void VID_Init(void) {
  Cvar_FakeRegister(&vid_gamma, "gamma");
  Cvar_FakeRegister(&vid_contrast, "contrast");

  VID_Init_Go();

	GLAlias_CreateShaders();
	GL_ClearBufferBindings();
  GL_SetupState();
}

