#ifndef _QUAKE_WAD_H
#define _QUAKE_WAD_H

#include <stdint.h>

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

#endif /* _QUAKE_WAD_H */
