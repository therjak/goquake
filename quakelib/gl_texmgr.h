#ifndef _GL_TEXMAN_H
#define _GL_TEXMAN_H

#define TEXPREF_NONE 0x0000
#define TEXPREF_MIPMAP 0x0001  // generate mipmaps
// TEXPREF_NEAREST and TEXPREF_LINEAR aren't supposed to be ORed with TEX_MIPMAP
#define TEXPREF_LINEAR 0x0002      // force linear
#define TEXPREF_NEAREST 0x0004     // force nearest
#define TEXPREF_ALPHA 0x0008       // allow alpha
#define TEXPREF_PAD 0x0010         // allow padding
#define TEXPREF_PERSIST 0x0020     // never free
#define TEXPREF_OVERWRITE 0x0040   // overwrite existing same-name texture
#define TEXPREF_NOPICMIP 0x0080    // always load full-sized
#define TEXPREF_FULLBRIGHT 0x0100  // use fullbright mask palette
#define TEXPREF_NOBRIGHT 0x0200    // use nobright mask palette
#define TEXPREF_CONCHARS 0x0400    // use conchars palette
#define TEXPREF_WARPIMAGE \
  0x0800  // resize this texture when warpimagesize changes

enum srcformat { SRC_INDEXED, SRC_RGBA };
typedef uintptr_t src_offset_t;

extern unsigned int d_8to24table[256];

void TexMgr_Init(void);
#endif /* _GL_TEXMAN_H */
