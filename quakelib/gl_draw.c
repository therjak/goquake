// draw.c -- 2d drawing

#include "quakedef.h"
#include "wad.h"

// extern unsigned char d_15to8table[65536]; //johnfitz -- never used

cvar_t scr_conalpha;

qpic_t *draw_backtile;

uint32_t char_texture2;

qpic_t *pic_ovr, *pic_ins;  // johnfitz -- new cursor handling
qpic_t *pic_nul;            // johnfitz -- for missing gfx, don't crash

// johnfitz -- new pics
byte pic_ovr_data[8][8] = {
    {255, 255, 255, 255, 255, 255, 255, 255},
    {255, 15, 15, 15, 15, 15, 15, 255},
    {255, 15, 15, 15, 15, 15, 15, 2},
    {255, 15, 15, 15, 15, 15, 15, 2},
    {255, 15, 15, 15, 15, 15, 15, 2},
    {255, 15, 15, 15, 15, 15, 15, 2},
    {255, 15, 15, 15, 15, 15, 15, 2},
    {255, 255, 2, 2, 2, 2, 2, 2},
};

byte pic_ins_data[9][8] = {
    {15, 15, 255, 255, 255, 255, 255, 255},
    {15, 15, 2, 255, 255, 255, 255, 255},
    {15, 15, 2, 255, 255, 255, 255, 255},
    {15, 15, 2, 255, 255, 255, 255, 255},
    {15, 15, 2, 255, 255, 255, 255, 255},
    {15, 15, 2, 255, 255, 255, 255, 255},
    {15, 15, 2, 255, 255, 255, 255, 255},
    {15, 15, 2, 255, 255, 255, 255, 255},
    {255, 2, 2, 255, 255, 255, 255, 255},
};

byte pic_nul_data[8][8] = {
    {252, 252, 252, 252, 0, 0, 0, 0}, {252, 252, 252, 252, 0, 0, 0, 0},
    {252, 252, 252, 252, 0, 0, 0, 0}, {252, 252, 252, 252, 0, 0, 0, 0},
    {0, 0, 0, 0, 252, 252, 252, 252}, {0, 0, 0, 0, 252, 252, 252, 252},
    {0, 0, 0, 0, 252, 252, 252, 252}, {0, 0, 0, 0, 252, 252, 252, 252},
};

byte pic_stipple_data[8][8] = {
    {255, 0, 0, 0, 255, 0, 0, 0}, {0, 0, 255, 0, 0, 0, 255, 0},
    {255, 0, 0, 0, 255, 0, 0, 0}, {0, 0, 255, 0, 0, 0, 255, 0},
    {255, 0, 0, 0, 255, 0, 0, 0}, {0, 0, 255, 0, 0, 0, 255, 0},
    {255, 0, 0, 0, 255, 0, 0, 0}, {0, 0, 255, 0, 0, 0, 255, 0},
};

byte pic_crosshair_data[8][8] = {
    {255, 255, 255, 255, 255, 255, 255, 255},
    {255, 255, 255, 8, 9, 255, 255, 255},
    {255, 255, 255, 6, 8, 2, 255, 255},
    {255, 6, 8, 8, 6, 8, 8, 255},
    {255, 255, 2, 8, 8, 2, 2, 2},
    {255, 255, 255, 7, 8, 2, 255, 255},
    {255, 255, 255, 255, 2, 2, 255, 255},
    {255, 255, 255, 255, 255, 255, 255, 255},
};
// johnfitz

canvastype currentcanvas = CANVAS_NONE;  // johnfitz -- for GL_SetCanvas

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

byte menuplyr_pixels[4096];

//  scrap allocation
//  Allocate all the little status bar obejcts into a single texture
//  to crutch up stupid hardware / drivers

#define MAX_SCRAPS 2
#define BLOCK_WIDTH 256
#define BLOCK_HEIGHT 256

int scrap_allocated[MAX_SCRAPS][BLOCK_WIDTH];
byte scrap_texels[MAX_SCRAPS][BLOCK_WIDTH * BLOCK_HEIGHT];  // johnfitz --
                                                            // removed *4 after
                                                            // BLOCK_HEIGHT
uint32_t scrap_textures2[MAX_SCRAPS];  // johnfitz

/*
================
Draw_PicFromWad
================
*/
qpic_t *Draw_PicFromWad(const char *name) {
  qpic_t *p;
  glpic_t gl;
  src_offset_t offset;  // johnfitz

  p = W_GetQPic(name);
  if (!p) return pic_nul;  // johnfitz

    char texturename[64];  // johnfitz
    q_snprintf(texturename, sizeof(texturename), "%s:%s", WADFILENAME,
               name);  // johnfitz

    offset =
        (src_offset_t)p - (src_offset_t)wad_base + sizeof(int) * 2;  // johnfitz

    gl.gltexture = TexMgrLoadImage(
        NULL, texturename, p->width, p->height, SRC_INDEXED, p->data,
        WADFILENAME, offset,
        TEXPREF_ALPHA | TEXPREF_PAD | TEXPREF_NOPICMIP);  // johnfitz -- TexMgr
    gl.sl = 0;
    gl.sh = 1;
    gl.tl = 0;
    gl.th = 1;

  memcpy(p->data, &gl, sizeof(glpic_t));

  return p;
}

/*
================
Draw_CachePic
================
*/
qpic_t *Draw_CachePic(const char *path) {
  cachepic_t *pic;
  int i;
  qpic_t *dat;
  glpic_t gl;

  for (pic = menu_cachepics, i = 0; i < menu_numcachepics; pic++, i++) {
    if (!strcmp(path, pic->name)) return &pic->pic;
  }
  if (menu_numcachepics == MAX_CACHED_PICS)
    Go_Error("menu_numcachepics == MAX_CACHED_PICS");
  menu_numcachepics++;
  strcpy(pic->name, path);

  //
  // load the pic from disk
  //
  int length = 0;
  dat = (qpic_t *)COM_LoadFileGo(path, &length);
  if (!dat) Go_Error_S("Draw_CachePic: failed to load %v", path);
  SwapPic(dat);

  // HACK HACK HACK --- we need to keep the bytes for
  // the translatable player picture just for the menu
  // configuration dialog
  if (!strcmp(path, "gfx/menuplyr.lmp"))
    memcpy(menuplyr_pixels, dat->data, dat->width * dat->height);

  pic->pic.width = dat->width;
  pic->pic.height = dat->height;

  gl.gltexture = TexMgrLoadImage(
      NULL, path, dat->width, dat->height, SRC_INDEXED, dat->data, path,
      sizeof(int) * 2,
      TEXPREF_ALPHA | TEXPREF_PAD | TEXPREF_NOPICMIP);  // johnfitz -- TexMgr
  gl.sl = 0;
  gl.sh = 1;
  gl.tl = 0;
  gl.th = 1;
  memcpy(pic->pic.data, &gl, sizeof(glpic_t));

  return &pic->pic;
}

/*
================
Draw_MakePic -- johnfitz -- generate pics from internal data
================
*/
qpic_t *Draw_MakePic(const char *name, int width, int height, byte *data) {
  int flags = TEXPREF_NEAREST | TEXPREF_ALPHA | TEXPREF_PERSIST |
              TEXPREF_NOPICMIP | TEXPREF_PAD;
  qpic_t *pic;
  glpic_t gl;

  pic = (qpic_t *)Hunk_Alloc(sizeof(qpic_t) - 4 + sizeof(glpic_t));
  pic->width = width;
  pic->height = height;

  gl.gltexture = TexMgrLoadImage(NULL, name, width, height, SRC_INDEXED, data,
                                 "", (src_offset_t)data, flags);
  gl.sl = 0;
  gl.sh = 1;
  gl.tl = 0;
  gl.th = 1;
  memcpy(pic->data, &gl, sizeof(glpic_t));

  return pic;
}

//==============================================================================
//
//  INIT
//
//==============================================================================

/*
===============
Draw_LoadPics -- johnfitz
===============
*/
void Draw_LoadPics(void) {
  char_texture2 = TexMgrLoadConsoleChars();
  if (!char_texture2) Go_Error("Draw_LoadPics: couldn't load conchars");
  draw_backtile = Draw_PicFromWad("backtile");
}

/*
===============
Draw_Init -- johnfitz -- rewritten
===============
*/
void Draw_Init(void) {
  Cvar_FakeRegister(&scr_conalpha, "scr_conalpha");

  // create internal pics
  pic_ins = Draw_MakePic("ins", 8, 9, &pic_ins_data[0][0]);
  pic_ovr = Draw_MakePic("ovr", 8, 8, &pic_ovr_data[0][0]);
  pic_nul = Draw_MakePic("nul", 8, 8, &pic_nul_data[0][0]);

  // load game pics
  Draw_LoadPics();
}

//==============================================================================
//
//  2D DRAWING
//
//==============================================================================

/*
================
Draw_CharacterQuad -- johnfitz -- seperate function to spit out verts
================
*/
void Draw_CharacterQuad(int x, int y, char num) {
  int row, col;
  float frow, fcol, size;

  row = num >> 4;
  col = num & 15;

  frow = row * 0.0625;
  fcol = col * 0.0625;
  size = 0.0625;

  GLBind(char_texture2);
  glBegin(GL_QUADS);

  glTexCoord2f(fcol, frow);
  glVertex2f(x, y);
  glTexCoord2f(fcol + size, frow);
  glVertex2f(x + 8, y);
  glTexCoord2f(fcol + size, frow + size);
  glVertex2f(x + 8, y + 8);
  glTexCoord2f(fcol, frow + size);
  glVertex2f(x, y + 8);

  glEnd();
}

/*
================
Draw_Character -- johnfitz -- modified to call Draw_CharacterQuad
================
*/
void Draw_Character(int x, int y, int num) {
  if (y <= -8) return;  // totally off screen

  num &= 255;

  if (num == 32) return;  // don't waste verts on spaces

  Draw_CharacterQuad(x, y, (char)num);
}

/*
=============
Draw_Pic -- johnfitz -- modified
=============
*/
void Draw_Pic(int x, int y, qpic_t *pic) {
  glpic_t *gl;

  gl = (glpic_t *)pic->data;
  GLBind(gl->gltexture);
  glBegin(GL_QUADS);
  glTexCoord2f(gl->sl, gl->tl);
  glVertex2f(x, y);
  glTexCoord2f(gl->sh, gl->tl);
  glVertex2f(x + pic->width, y);
  glTexCoord2f(gl->sh, gl->th);
  glVertex2f(x + pic->width, y + pic->height);
  glTexCoord2f(gl->sl, gl->th);
  glVertex2f(x, y + pic->height);
  glEnd();
}

void Draw_Pic2(int x, int y, QPIC pic) {
  GLBind(pic.texture);
  glBegin(GL_QUADS);
  glTexCoord2f(pic.sl, pic.tl);
  glVertex2f(x, y);
  glTexCoord2f(pic.sh, pic.tl);
  glVertex2f(x + pic.width, y);
  glTexCoord2f(pic.sh, pic.th);
  glVertex2f(x + pic.width, y + pic.height);
  glTexCoord2f(pic.sl, pic.th);
  glVertex2f(x, y + pic.height);
  glEnd();
}

void Draw_TransPicTranslate2(int x, int y, QPIC pic, int top, int bottom) {
  static int oldtop = -2;
  static int oldbottom = -2;

  if (top != oldtop || bottom != oldbottom) {
    uint32_t glt = pic.texture;
    oldtop = top;
    oldbottom = bottom;
    TexMgrReloadImage(glt, top, bottom);
  }
  Draw_Pic2(x, y, pic);
}

/*
=============
Draw_TileClear

This repeats a 64*64 tile graphic to fill the screen around a sized down
refresh window.
=============
*/
void Draw_TileClear(int x, int y, int w, int h) {
  glpic_t *gl;

  gl = (glpic_t *)draw_backtile->data;

  glColor3f(1, 1, 1);
  GLBind(gl->gltexture);
  glBegin(GL_QUADS);
  glTexCoord2f(x / 64.0, y / 64.0);
  glVertex2f(x, y);
  glTexCoord2f((x + w) / 64.0, y / 64.0);
  glVertex2f(x + w, y);
  glTexCoord2f((x + w) / 64.0, (y + h) / 64.0);
  glVertex2f(x + w, y + h);
  glTexCoord2f(x / 64.0, (y + h) / 64.0);
  glVertex2f(x, y + h);
  glEnd();
}

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
  GL_SetCanvas(CANVAS_DEFAULT);

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
      glViewport(GL_X(), GL_Y(), GL_Width(), GL_Height());
      break;
    case CANVAS_CONSOLE:
      lines = ConHeight() -
              (GetScreenConsoleCurrentHeight() * ConHeight() / GL_Height());
      glOrtho(0, ConWidth(), ConHeight() + lines, lines, -99999, 99999);
      glViewport(GL_X(), GL_Y(), GL_Width(), GL_Height());
      break;
    case CANVAS_MENU:
      s = q_min((float)GL_Width() / 320.0, (float)GL_Height() / 200.0);
      s = CLAMP(1.0, Cvar_GetValue(&scr_menuscale), s);
      // ericw -- doubled width to 640 to accommodate long keybindings
      glOrtho(0, 640, 200, 0, -99999, 99999);
      glViewport(GL_X() + (GL_Width() - 320 * s) / 2,
                 GL_Y() + (GL_Height() - 200 * s) / 2, 640 * s, 200 * s);
      break;
    case CANVAS_SBAR:
      s = CLAMP(1.0, Cvar_GetValue(&scr_sbarscale), (float)GL_Width() / 320.0);
      if (CL_GameTypeDeathMatch()) {
        glOrtho(0, GL_Width() / s, 48, 0, -99999, 99999);
        glViewport(GL_X(), GL_Y(), GL_Width(), 48 * s);
      } else {
        glOrtho(0, 320, 48, 0, -99999, 99999);
        glViewport(GL_X() + (GL_Width() - 320 * s) / 2, GL_Y(), 320 * s,
                   48 * s);
      }
      break;
    case CANVAS_WARPIMAGE:
      glOrtho(0, 128, 0, 128, -99999, 99999);
      glViewport(GL_X(), GL_Y() + GL_Height() - gl_warpimagesize,
                 gl_warpimagesize, gl_warpimagesize);
      break;
    case CANVAS_CROSSHAIR:  // 0,0 is center of viewport
      s = CLAMP(1.0, Cvar_GetValue(&scr_crosshairscale), 10.0);
      glOrtho(SCR_GetVRectWidth() / -2 / s, SCR_GetVRectWidth() / 2 / s,
              SCR_GetVRectHeight() / 2 / s, SCR_GetVRectHeight() / -2 / s,
              -99999, 99999);
      glViewport(SCR_GetVRectX(),
                 GL_Height() - SCR_GetVRectY() - SCR_GetVRectHeight(),
                 SCR_GetVRectWidth() & ~1, SCR_GetVRectHeight() & ~1);
      break;
    case CANVAS_BOTTOMLEFT:                // used by devstats
      s = (float)GL_Width() / ConWidth();  // use console scale
      glOrtho(0, 320, 200, 0, -99999, 99999);
      glViewport(GL_X(), GL_Y(), 320 * s, 200 * s);
      break;
    case CANVAS_BOTTOMRIGHT:               // used by fps/clock
      s = (float)GL_Width() / ConWidth();  // use console scale
      glOrtho(0, 320, 200, 0, -99999, 99999);
      glViewport(GL_X() + GL_Width() - 320 * s, GL_Y(), 320 * s, 200 * s);
      break;
    case CANVAS_TOPRIGHT:  // used by disc
      s = 1;
      glOrtho(0, 320, 200, 0, -99999, 99999);
      glViewport(GL_X() + GL_Width() - 320 * s, GL_Y() + GL_Height() - 200 * s,
                 320 * s, 200 * s);
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
  currentcanvas = CANVAS_INVALID;
  GL_SetCanvas(CANVAS_DEFAULT);

  glDisable(GL_DEPTH_TEST);
  glDisable(GL_CULL_FACE);
  glDisable(GL_BLEND);
  glEnable(GL_ALPHA_TEST);
  glColor4f(1, 1, 1, 1);
}
