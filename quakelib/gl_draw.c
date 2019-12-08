// draw.c -- 2d drawing

#include "quakedef.h"
#include "draw.h"

canvastype currentcanvas = CANVAS_NONE;  // johnfitz -- for GL_SetCanvas

void SwapPic(qpic_t *pic) {
  pic->width = LittleLong(pic->width);
  pic->height = LittleLong(pic->height);
}

//==============================================================================
//
//  PIC CACHING
//
//==============================================================================

typedef struct cachepic_s {
  char name[MAX_QPATH];
  qpic_t pic;
  byte padding[32];  // for appended glpic
} cachepic_t;

#define MAX_CACHED_PICS 128
cachepic_t menu_cachepics[MAX_CACHED_PICS];
int menu_numcachepics;

/*
================
GL_SetCanvas -- johnfitz -- support various canvas types
================
*/
void GL_SetCanvas(canvastype newcanvas) {
  float s;
  int lines;

  if (newcanvas == currentcanvas) return;

  currentcanvas = newcanvas;

  glMatrixMode(GL_PROJECTION);
  glLoadIdentity();

  switch (newcanvas) {
    case CANVAS_DEFAULT:
      glOrtho(0, GL_Width(), GL_Height(), 0, -99999, 99999);
      glViewport(0, 0, GL_Width(), GL_Height());
      break;
    case CANVAS_WARPIMAGE:
      glOrtho(0, 128, 0, 128, -99999, 99999);
      glViewport(0, GL_Height() - gl_warpimagesize,
                 gl_warpimagesize, gl_warpimagesize);
      break;
    default:
      Go_Error("GL_SetCanvas: bad canvas type");
  }
}

void GL_CanvasEnd(void) {
  glMatrixMode(GL_MODELVIEW);
  glLoadIdentity();
}

/*
================
GL_Set2D -- johnfitz -- rewritten
================
*/
// THERJAK
void GL_Set2D(void) {
  currentcanvas = CANVAS_NONE;
  GLSetCanvas(CANVAS_DEFAULT);

  glDisable(GL_DEPTH_TEST);
  glDisable(GL_CULL_FACE);
  glDisable(GL_BLEND);
  glEnable(GL_ALPHA_TEST);
  glColor4f(1, 1, 1, 1);
}
