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

//==============================================================================
//
//  2D DRAWING
//
//==============================================================================

/*
=============
Draw_Fill

Fills a box of pixels with a single color
=============
*/
void Draw_Fill(int x, int y, int w, int h, int c,
               float alpha)  // johnfitz -- added alpha
{
  byte *pal = (byte *)
      d_8to24table;  // johnfitz -- use d_8to24table instead of host_basepal

  glDisable(GL_TEXTURE_2D);
  glEnable(GL_BLEND);        // johnfitz -- for alpha
  glDisable(GL_ALPHA_TEST);  // johnfitz -- for alpha
  glColor4f(pal[c * 4] / 255.0, pal[c * 4 + 1] / 255.0, pal[c * 4 + 2] / 255.0,
            alpha);  // johnfitz -- added alpha

  glBegin(GL_QUADS);
  glVertex2f(x, y);
  glVertex2f(x + w, y);
  glVertex2f(x + w, y + h);
  glVertex2f(x, y + h);
  glEnd();

  glColor3f(1, 1, 1);
  glDisable(GL_BLEND);      // johnfitz -- for alpha
  glEnable(GL_ALPHA_TEST);  // johnfitz -- for alpha
  glEnable(GL_TEXTURE_2D);
}

/*
================
Draw_FadeScreen -- johnfitz -- revised
================
*/
void Draw_FadeScreen(void) {
  GLSetCanvas(CANVAS_DEFAULT);

  glEnable(GL_BLEND);
  glDisable(GL_ALPHA_TEST);
  glDisable(GL_TEXTURE_2D);
  glColor4f(0, 0, 0, 0.5);
  glBegin(GL_QUADS);

  glVertex2f(0, 0);
  glVertex2f(GL_Width(), 0);
  glVertex2f(GL_Width(), GL_Height());
  glVertex2f(0, GL_Height());

  glEnd();
  glColor4f(1, 1, 1, 1);
  glEnable(GL_TEXTURE_2D);
  glEnable(GL_ALPHA_TEST);
  glDisable(GL_BLEND);

  Sbar_Changed();
}

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
    case CANVAS_CONSOLE:
      lines = ConHeight() -
              (GetScreenConsoleCurrentHeight() * ConHeight() / GL_Height());
      glOrtho(0, ConWidth(), ConHeight() + lines, lines, -99999, 99999);
      glViewport(0, 0, GL_Width(), GL_Height());
      break;
    case CANVAS_MENU:
      s = q_min((float)GL_Width() / 320.0, (float)GL_Height() / 200.0);
      s = CLAMP(1.0, Cvar_GetValue(&scr_menuscale), s);
      // ericw -- doubled width to 640 to accommodate long keybindings
      glOrtho(0, 640, 200, 0, -99999, 99999);
      glViewport((GL_Width() - 320 * s) / 2,
                 (GL_Height() - 200 * s) / 2, 640 * s, 200 * s);
      break;
    case CANVAS_SBAR:
      s = CLAMP(1.0, Cvar_GetValue(&scr_sbarscale), (float)GL_Width() / 320.0);
      if (CL_GameTypeDeathMatch()) {
        glOrtho(0, GL_Width() / s, 48, 0, -99999, 99999);
        glViewport(0, 0, GL_Width(), 48 * s);
      } else {
        glOrtho(0, 320, 48, 0, -99999, 99999);
        glViewport((GL_Width() - 320 * s) / 2, 0, 320 * s,
                   48 * s);
      }
      break;
    case CANVAS_WARPIMAGE:
      glOrtho(0, 128, 0, 128, -99999, 99999);
      glViewport(0, GL_Height() - gl_warpimagesize,
                 gl_warpimagesize, gl_warpimagesize);
      break;
    case CANVAS_BOTTOMRIGHT:               // used by fps/clock
      s = (float)GL_Width() / ConWidth();  // use console scale
      glOrtho(0, 320, 200, 0, -99999, 99999);
      glViewport(GL_Width() - 320 * s, 0, 320 * s, 200 * s);
      break;
    default:
      Go_Error("GL_SetCanvas: bad canvas type");
  }

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
