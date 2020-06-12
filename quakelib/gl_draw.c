// draw.c -- 2d drawing

#include "quakedef.h"
#include "draw.h"

canvastype currentcanvas = CANVAS_NONE;  // johnfitz -- for GL_SetCanvas

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
      glViewport(0, GL_Height() - GL_warpimagesize(),
                 GL_warpimagesize(), GL_warpimagesize());
      break;
    default:
      Go_Error("GL_SetCanvas: bad canvas type");
  }
}

void GL_CanvasEnd(void) {
  glMatrixMode(GL_MODELVIEW);
  glLoadIdentity();
}
