// gl_texmgr.c -- fitzquake's texture manager. manages opengl texture images

#include "quakedef.h"

unsigned int d_8to24table[256];

/*
=================
TexMgr_LoadPalette -- johnfitz -- was VID_SetPalette, moved here, renamed,
rewritten
=================
*/
void TexMgrLoadPalette(void) {
  byte *pal, *src, *dst;
  int i;

  int length = 0;
  pal = COM_LoadFileGo("gfx/palette.lmp", &length);
  if (!pal) Go_Error("Couldn't load gfx/palette.lmp");

  // standard palette, 255 is transparent
  dst = (byte *)d_8to24table;
  src = pal;
  for (i = 0; i < 256; i++) {
    *dst++ = *src++;
    *dst++ = *src++;
    *dst++ = *src++;
    *dst++ = 255;
  }
  ((byte *)&d_8to24table[255])[3] = 0;
}

/*
================
TexMgr_Init

must be called before any texture loading
================
*/
void TexMgr_Init(void) {
  TexMgr_LoadPalette();

  // set safe size for warpimages
  gl_warpimagesize = 0;
  TexMgrRecalcWarpImageSize();
}
