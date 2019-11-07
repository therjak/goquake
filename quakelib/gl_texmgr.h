#ifndef _GL_TEXMAN_H
#define _GL_TEXMAN_H

#include <GL/gl.h>
// gl_texmgr.h -- fitzquake's texture manager. manages opengl texture images

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

enum srcformat { SRC_INDEXED, SRC_LIGHTMAP, SRC_RGBA };

typedef uintptr_t src_offset_t;

typedef struct gltexture_s {
  // managed by texture manager
  GLuint texnum;
  struct gltexture_s *next;
  qmodel_t *owner;
  // managed by image loading
  char name[64];
  unsigned int width;   // size of image as it exists in opengl
  unsigned int height;  // size of image as it exists in opengl
  unsigned int flags;
  char source_file[MAX_QPATH];  // relative filepath to data source, or "" if
                                // source is in memory
  src_offset_t source_offset;   // byte offset into file, or memory address
  enum srcformat
      source_format;  // format of pixel data (indexed, lightmap, or rgba)
  unsigned int source_width;   // size of image in source data
  unsigned int source_height;  // size of image in source data
  unsigned short source_crc;   // generated by source data before modifications
  char shirt;                  // 0-13 shirt color, or -1 if never colormapped
  char pants;                  // 0-13 pants color, or -1 if never colormapped
  // used for rendering
  int visframe;  // matches r_framecount if texture was bound this frame
} gltexture_t;

typedef gltexture_t *gltexture_tp;

extern gltexture_t *notexture;

extern unsigned int d_8to24table[256];

// TEXTURE MANAGER

float TexMgr_FrameUsage(void);
void TexMgr_FreeTexture(gltexture_t *kill);
void TexMgr_FreeTexturesForOwner(qmodel_t *owner);
void TexMgr_Init(void);
void TexMgr_DeleteTextureObjects(void);

// IMAGE LOADING
gltexture_t *TexMgr_LoadImage(qmodel_t *owner, const char *name, int width,
                              int height, enum srcformat format, byte *data,
                              const char *source_file,
                              src_offset_t source_offset, unsigned flags);
void TexMgr_ReloadImage(gltexture_t *glt, int shirt, int pants);
void TexMgr_ReloadImages(void);
void TexMgr_ReloadNobrightImages(void);  // only cvar callback stuff

int TexMgr_PadConditional(int s);

// TEXTURE BINDING & TEXTURE UNIT SWITCHING

void GL_SelectTexture(GLenum target);
void GL_DisableMultitexture(void);  // selects texture unit 0
void GL_EnableMultitexture(void);   // selects texture unit 1
void GL_Bind(gltexture_t *texture);
void GL_ClearBindings(void);

#endif /* _GL_TEXMAN_H */
