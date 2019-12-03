#ifndef _QUAKE_DRAW_H
#define _QUAKE_DRAW_H
#include <stdint.h> 
#include "canvas.h"
// draw.h -- these are the only functions outside the refresh allowed
// to touch the vid buffer

typedef struct {
  int width, height;
  unsigned char data[4];  // variably sized
} qpic_t;

typedef struct {
  uint32_t gltexture;
  float sl, tl, sh, th;
} glpic_t;

typedef struct {
  int width;
  int height;
  uint32_t texture;
  float sl, tl, sh, th;
} QPIC;

void SwapPic(qpic_t *pic);
void GL_SetCanvas(canvastype newcanvas);  // johnfitz

#endif /* _QUAKE_DRAW_H */
