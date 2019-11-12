// gl_texmgr.c -- fitzquake's texture manager. manages opengl texture images

// THERJAK:
// change gltexture_t to simple int (index)

#include "quakedef.h"

const int gl_solid_format = 3;
const int gl_alpha_format = 4;

static cvar_t gl_max_size;
static cvar_t gl_picmip;
static GLint gl_hardware_maxsize;

#define MAX_GLTEXTURES 2048
static int numgltextures;
static gltexture_t *active_gltextures, *free_gltextures;
gltexture_t *notexture, *nulltexture;

unsigned int d_8to24table[256];
unsigned int d_8to24table_fbright[256];
unsigned int d_8to24table_fbright_fence[256];
unsigned int d_8to24table_nobright[256];
unsigned int d_8to24table_nobright_fence[256];
unsigned int d_8to24table_conchars[256];

int TexMgr_SafeTextureSize(int s);
/*
================================================================================

        COMMANDS

================================================================================
*/

typedef struct {
  int magfilter;
  int minfilter;
  const char *name;
} glmode_t;
static glmode_t glmodes[] = {
    {GL_NEAREST, GL_NEAREST, "GL_NEAREST"},
    {GL_NEAREST, GL_NEAREST_MIPMAP_NEAREST, "GL_NEAREST_MIPMAP_NEAREST"},
    {GL_NEAREST, GL_NEAREST_MIPMAP_LINEAR, "GL_NEAREST_MIPMAP_LINEAR"},
    {GL_LINEAR, GL_LINEAR, "GL_LINEAR"},
    {GL_LINEAR, GL_LINEAR_MIPMAP_NEAREST, "GL_LINEAR_MIPMAP_NEAREST"},
    {GL_LINEAR, GL_LINEAR_MIPMAP_LINEAR, "GL_LINEAR_MIPMAP_LINEAR"},
};
#define NUM_GLMODES (int)(sizeof(glmodes) / sizeof(glmodes[0]))
static int glmode_idx = NUM_GLMODES - 1; /* trilinear */

/*
================
TexMgr_FindTexture
================
*/
gltexture_t *TexMgr_FindTexture(qmodel_t *owner, const char *name) {
  gltexture_t *glt;

  if (name) {
    for (glt = active_gltextures; glt; glt = glt->next) {
      if (glt->owner == owner && !strcmp(glt->name, name)) return glt;
    }
  }

  return NULL;
}

/*
================
TexMgr_NewTexture
================
*/
gltexture_t *TexMgr_NewTexture(void) {
  gltexture_t *glt;

  if (numgltextures == MAX_GLTEXTURES)
    Go_Error("numgltextures == MAX_GLTEXTURES\n");

  glt = free_gltextures;
  free_gltextures = glt->next;
  glt->next = active_gltextures;
  active_gltextures = glt;

  // THERJAK: glGenTextures is opengl
  GL_GenTextures2(glt);
  numgltextures++;
  return glt;
}

// void GL_DeleteTexture(gltexture_t *texture);

// ericw -- workaround for preventing TexMgr_FreeTexture during
// TexMgr_ReloadImages
static qboolean in_reload_images;

/*
================
TexMgr_FreeTexture
================
*/
void TexMgr_FreeTexture(gltexture_t *kill) {
  gltexture_t *glt;

  if (in_reload_images) return;

  if (kill == NULL) {
    Con_Printf("TexMgr_FreeTexture: NULL texture\n");
    return;
  }

  if (active_gltextures == kill) {
    active_gltextures = kill->next;
    kill->next = free_gltextures;
    free_gltextures = kill;

    GL_DeleteTexture2(kill);
    numgltextures--;
    return;
  }

  for (glt = active_gltextures; glt; glt = glt->next) {
    if (glt->next == kill) {
      glt->next = kill->next;
      kill->next = free_gltextures;
      free_gltextures = kill;

      GL_DeleteTexture2(kill);
      numgltextures--;
      return;
    }
  }

  Con_Printf("TexMgr_FreeTexture: not found\n");
}

/*
================
TexMgr_FreeTexturesForOwner
================
*/
void TexMgr_FreeTexturesForOwner(qmodel_t *owner) {
  gltexture_t *glt, *next;

  for (glt = active_gltextures; glt; glt = next) {
    next = glt->next;
    if (glt && glt->owner == owner) TexMgr_FreeTexture(glt);
  }
}

/*
=================
TexMgr_LoadPalette -- johnfitz -- was VID_SetPalette, moved here, renamed,
rewritten
=================
*/
void TexMgrLoadPalette(void) {
  byte *pal, *src, *dst;
  int i, mark;
  FILE *f;

  int length = 0;
  pal = COM_LoadFileGo("gfx/palette.lmp", &length);
  if (!pal) Go_Error("Couldn't load gfx/palette.lmp");

  //  mark = Hunk_LowMark();
  //  pal = (byte *)Hunk_Alloc(768);
  //  fread(pal, 1, 768, f);
  //  fclose(f);

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

  // fullbright palette, 0-223 are black (for additive blending)
  src = pal + 224 * 3;
  dst = (byte *)&d_8to24table_fbright[224];
  for (i = 224; i < 256; i++) {
    *dst++ = *src++;
    *dst++ = *src++;
    *dst++ = *src++;
    *dst++ = 255;
  }
  for (i = 0; i < 224; i++) {
    dst = (byte *)&d_8to24table_fbright[i];
    dst[3] = 255;
    dst[2] = dst[1] = dst[0] = 0;
  }

  // nobright palette, 224-255 are black (for additive blending)
  dst = (byte *)d_8to24table_nobright;
  src = pal;
  for (i = 0; i < 256; i++) {
    *dst++ = *src++;
    *dst++ = *src++;
    *dst++ = *src++;
    *dst++ = 255;
  }
  for (i = 224; i < 256; i++) {
    dst = (byte *)&d_8to24table_nobright[i];
    dst[3] = 255;
    dst[2] = dst[1] = dst[0] = 0;
  }

  // fullbright palette, for fence textures
  memcpy(d_8to24table_fbright_fence, d_8to24table_fbright, 256 * 4);
  d_8to24table_fbright_fence[255] = 0;  // Alpha of zero.

  // nobright palette, for fence textures
  memcpy(d_8to24table_nobright_fence, d_8to24table_nobright, 256 * 4);
  d_8to24table_nobright_fence[255] = 0;  // Alpha of zero.

  // conchars palette, 0 and 255 are transparent
  memcpy(d_8to24table_conchars, d_8to24table, 256 * 4);
  ((byte *)&d_8to24table_conchars[0])[3] = 0;

  //  Hunk_FreeToLowMark(mark);
}

/*
=============
TexMgr_RecalcWarpImageSize -- called during init, and after a vid_restart

choose safe warpimage size and resize existing warpimage textures
=============
*/
void TexMgr_RecalcWarpImageSize(void) {
  int mark;
  // int oldsize;
  gltexture_t *glt;
  byte *dummy;

  //
  // find the new correct size
  //
  // oldsize = gl_warpimagesize;

  gl_warpimagesize = TexMgr_SafeTextureSize(512);

  while (gl_warpimagesize > ScreenWidth()) gl_warpimagesize >>= 1;
  while (gl_warpimagesize > ScreenHeight()) gl_warpimagesize >>= 1;

  // ericw -- removed early exit if (gl_warpimagesize == oldsize).
  // after vid_restart TexMgr_ReloadImage reloads textures
  // to tx->source_width/source_height, which might not match oldsize.
  // fixes: https://sourceforge.net/p/quakespasm/bugs/13/

  //
  // resize the textures in opengl
  //
  mark = Hunk_LowMark();
  dummy = (byte *)Hunk_Alloc(gl_warpimagesize * gl_warpimagesize * 4);

  for (glt = active_gltextures; glt; glt = glt->next) {
    if (glt->flags & TEXPREF_WARPIMAGE) {
      GL_Bind(glt);
      // THERJAK: glTexImage2D is opengl
      GL_TexImage2D(GL_TEXTURE_2D, 0, gl_solid_format, gl_warpimagesize,
                    gl_warpimagesize, 0, GL_RGBA, GL_UNSIGNED_BYTE, dummy);
      glt->width = glt->height = gl_warpimagesize;
    }
  }

  Hunk_FreeToLowMark(mark);
}

/*
================
TexMgr_Init

must be called before any texture loading
================
*/
void TexMgr_Init(void) {
  int i;
  static byte notexture_data[16] = {
      159, 91, 83, 255, 0,   0,  0,  255,
      0,   0,  0,  255, 159, 91, 83, 255};  // black and pink checker
  static byte nulltexture_data[16] = {
      127, 191, 255, 255, 0,   0,   0,   255,
      0,   0,   0,   255, 127, 191, 255, 255};  // black and blue checker
  extern texture_t *r_notexture_mip, *r_notexture_mip2;

  // init texture list
  free_gltextures = (gltexture_t *)Hunk_AllocName(
      MAX_GLTEXTURES * sizeof(gltexture_t), "gltextures");
  active_gltextures = NULL;
  for (i = 0; i < MAX_GLTEXTURES - 1; i++)
    free_gltextures[i].next = &free_gltextures[i + 1];
  free_gltextures[i].next = NULL;
  numgltextures = 0;

  // palette
  TexMgr_LoadPalette();

  Cvar_FakeRegister(&gl_max_size, "gl_max_size");
  Cvar_FakeRegister(&gl_picmip, "gl_picmip");

  // poll max size from hardware
  // THERJAK: glGetIntegerv is opengl
  GL_GetIntegerv(GL_MAX_TEXTURE_SIZE, &gl_hardware_maxsize);

  // load notexture images
  notexture =
      TexMgrLoadImage2(NULL, "notexture", 2, 2, SRC_RGBA, notexture_data, "",
                       (src_offset_t)notexture_data,
                       TEXPREF_NEAREST | TEXPREF_PERSIST | TEXPREF_NOPICMIP);
  nulltexture =
      TexMgrLoadImage2(NULL, "nulltexture", 2, 2, SRC_RGBA, nulltexture_data,
                       "", (src_offset_t)nulltexture_data,
                       TEXPREF_NEAREST | TEXPREF_PERSIST | TEXPREF_NOPICMIP);

  // have to assign these here becuase Mod_Init is called before TexMgr_Init
  r_notexture_mip->gltexture = r_notexture_mip2->gltexture = notexture;

  // set safe size for warpimages
  gl_warpimagesize = 0;
  TexMgr_RecalcWarpImageSize();
}

/*
================================================================================

        IMAGE LOADING

================================================================================
*/

/*
================
TexMgr_Pad -- return smallest power of two greater than or equal to s
================
*/
int TexMgr_Pad(int s) {
  int i;
  for (i = 1; i < s; i <<= 1)
    ;
  return i;
}

/*
===============
TexMgr_SafeTextureSize -- return a size with hardware and user prefs in mind
===============
*/
int TexMgr_SafeTextureSize(int s) {
  if ((int)Cvar_GetValue(&gl_max_size) > 0)
    s = q_min(TexMgr_Pad((int)Cvar_GetValue(&gl_max_size)), s);
  s = q_min(gl_hardware_maxsize, s);
  return s;
}

/*
================
TexMgr_PadConditional -- only pad if a texture of that size would be padded.
(used for tex coords)
================
*/
int TexMgr_PadConditional(int s) {
  if (s < TexMgr_SafeTextureSize(s))
    return TexMgr_Pad(s);
  else
    return s;
}

/*
================
TexMgr_MipMapW
================
*/
static unsigned *TexMgr_MipMapW(unsigned *data, int width, int height) {
  int i, size;
  byte *out, *in;

  out = in = (byte *)data;
  size = (width * height) >> 1;

  for (i = 0; i < size; i++, out += 4, in += 8) {
    out[0] = (in[0] + in[4]) >> 1;
    out[1] = (in[1] + in[5]) >> 1;
    out[2] = (in[2] + in[6]) >> 1;
    out[3] = (in[3] + in[7]) >> 1;
  }

  return data;
}

/*
================
TexMgr_MipMapH
================
*/
static unsigned *TexMgr_MipMapH(unsigned *data, int width, int height) {
  int i, j;
  byte *out, *in;

  out = in = (byte *)data;
  height >>= 1;
  width <<= 2;

  for (i = 0; i < height; i++, in += width) {
    for (j = 0; j < width; j += 4, out += 4, in += 4) {
      out[0] = (in[0] + in[width + 0]) >> 1;
      out[1] = (in[1] + in[width + 1]) >> 1;
      out[2] = (in[2] + in[width + 2]) >> 1;
      out[3] = (in[3] + in[width + 3]) >> 1;
    }
  }

  return data;
}

/*
================
TexMgr_ResampleTexture -- bilinear resample
================
*/
static unsigned *TexMgr_ResampleTexture(unsigned *in, int inwidth, int inheight,
                                        qboolean alpha) {
  byte *nwpx, *nepx, *swpx, *sepx, *dest;
  unsigned xfrac, yfrac, x, y, modx, mody, imodx, imody, injump, outjump;
  unsigned *out;
  int i, j, outwidth, outheight;

  if (inwidth == TexMgr_Pad(inwidth) && inheight == TexMgr_Pad(inheight))
    return in;

  outwidth = TexMgr_Pad(inwidth);
  outheight = TexMgr_Pad(inheight);
  out = (unsigned *)Hunk_Alloc(outwidth * outheight * 4);

  xfrac = ((inwidth - 1) << 16) / (outwidth - 1);
  yfrac = ((inheight - 1) << 16) / (outheight - 1);
  y = outjump = 0;

  for (i = 0; i < outheight; i++) {
    mody = (y >> 8) & 0xFF;
    imody = 256 - mody;
    injump = (y >> 16) * inwidth;
    x = 0;

    for (j = 0; j < outwidth; j++) {
      modx = (x >> 8) & 0xFF;
      imodx = 256 - modx;

      nwpx = (byte *)(in + (x >> 16) + injump);
      nepx = nwpx + 4;
      swpx = nwpx + inwidth * 4;
      sepx = swpx + 4;

      dest = (byte *)(out + outjump + j);

      dest[0] = (nwpx[0] * imodx * imody + nepx[0] * modx * imody +
                 swpx[0] * imodx * mody + sepx[0] * modx * mody) >>
                16;
      dest[1] = (nwpx[1] * imodx * imody + nepx[1] * modx * imody +
                 swpx[1] * imodx * mody + sepx[1] * modx * mody) >>
                16;
      dest[2] = (nwpx[2] * imodx * imody + nepx[2] * modx * imody +
                 swpx[2] * imodx * mody + sepx[2] * modx * mody) >>
                16;
      if (alpha)
        dest[3] = (nwpx[3] * imodx * imody + nepx[3] * modx * imody +
                   swpx[3] * imodx * mody + sepx[3] * modx * mody) >>
                  16;
      else
        dest[3] = 255;

      x += xfrac;
    }
    outjump += outwidth;
    y += yfrac;
  }

  return out;
}

/*
===============
TexMgr_AlphaEdgeFix

eliminate pink edges on sprites, etc.
operates in place on 32bit data
===============
*/
static void TexMgr_AlphaEdgeFix(byte *data, int width, int height) {
  int i, j, n = 0, b, c[3] = {0, 0, 0}, lastrow, thisrow, nextrow, lastpix,
            thispix, nextpix;
  byte *dest = data;

  for (i = 0; i < height; i++) {
    lastrow = width * 4 * ((i == 0) ? height - 1 : i - 1);
    thisrow = width * 4 * i;
    nextrow = width * 4 * ((i == height - 1) ? 0 : i + 1);

    for (j = 0; j < width; j++, dest += 4) {
      if (dest[3])  // not transparent
        continue;

      lastpix = 4 * ((j == 0) ? width - 1 : j - 1);
      thispix = 4 * j;
      nextpix = 4 * ((j == width - 1) ? 0 : j + 1);

      b = lastrow + lastpix;
      if (data[b + 3]) {
        c[0] += data[b];
        c[1] += data[b + 1];
        c[2] += data[b + 2];
        n++;
      }
      b = thisrow + lastpix;
      if (data[b + 3]) {
        c[0] += data[b];
        c[1] += data[b + 1];
        c[2] += data[b + 2];
        n++;
      }
      b = nextrow + lastpix;
      if (data[b + 3]) {
        c[0] += data[b];
        c[1] += data[b + 1];
        c[2] += data[b + 2];
        n++;
      }
      b = lastrow + thispix;
      if (data[b + 3]) {
        c[0] += data[b];
        c[1] += data[b + 1];
        c[2] += data[b + 2];
        n++;
      }
      b = nextrow + thispix;
      if (data[b + 3]) {
        c[0] += data[b];
        c[1] += data[b + 1];
        c[2] += data[b + 2];
        n++;
      }
      b = lastrow + nextpix;
      if (data[b + 3]) {
        c[0] += data[b];
        c[1] += data[b + 1];
        c[2] += data[b + 2];
        n++;
      }
      b = thisrow + nextpix;
      if (data[b + 3]) {
        c[0] += data[b];
        c[1] += data[b + 1];
        c[2] += data[b + 2];
        n++;
      }
      b = nextrow + nextpix;
      if (data[b + 3]) {
        c[0] += data[b];
        c[1] += data[b + 1];
        c[2] += data[b + 2];
        n++;
      }

      // average all non-transparent neighbors
      if (n) {
        dest[0] = (byte)(c[0] / n);
        dest[1] = (byte)(c[1] / n);
        dest[2] = (byte)(c[2] / n);

        n = c[0] = c[1] = c[2] = 0;
      }
    }
  }
}

/*
===============
TexMgr_PadEdgeFixW -- special case of AlphaEdgeFix for textures that only need
it because they were padded

operates in place on 32bit data, and expects unpadded height and width values
===============
*/
static void TexMgr_PadEdgeFixW(byte *data, int width, int height) {
  byte *src, *dst;
  int i, padw, padh;

  padw = TexMgr_PadConditional(width);
  padh = TexMgr_PadConditional(height);

  // copy last full column to first empty column, leaving alpha byte at zero
  src = data + (width - 1) * 4;
  for (i = 0; i < padh; i++) {
    src[4] = src[0];
    src[5] = src[1];
    src[6] = src[2];
    src += padw * 4;
  }

  // copy first full column to last empty column, leaving alpha byte at zero
  src = data;
  dst = data + (padw - 1) * 4;
  for (i = 0; i < padh; i++) {
    dst[0] = src[0];
    dst[1] = src[1];
    dst[2] = src[2];
    src += padw * 4;
    dst += padw * 4;
  }
}

/*
===============
TexMgr_PadEdgeFixH -- special case of AlphaEdgeFix for textures that only need
it because they were padded

operates in place on 32bit data, and expects unpadded height and width values
===============
*/
static void TexMgr_PadEdgeFixH(byte *data, int width, int height) {
  byte *src, *dst;
  int i, padw, padh;

  padw = TexMgr_PadConditional(width);
  padh = TexMgr_PadConditional(height);

  // copy last full row to first empty row, leaving alpha byte at zero
  dst = data + height * padw * 4;
  src = dst - padw * 4;
  for (i = 0; i < padw; i++) {
    dst[0] = src[0];
    dst[1] = src[1];
    dst[2] = src[2];
    src += 4;
    dst += 4;
  }

  // copy first full row to last empty row, leaving alpha byte at zero
  dst = data + (padh - 1) * padw * 4;
  src = data;
  for (i = 0; i < padw; i++) {
    dst[0] = src[0];
    dst[1] = src[1];
    dst[2] = src[2];
    src += 4;
    dst += 4;
  }
}

/*
================
TexMgr_8to32
================
*/
static unsigned *TexMgr_8to32(byte *in, int pixels, unsigned int *usepal) {
  int i;
  unsigned *out, *data;

  out = data = (unsigned *)Hunk_Alloc(pixels * 4);

  for (i = 0; i < pixels; i++) *out++ = usepal[*in++];

  return data;
}

/*
================
TexMgr_PadImageW -- return image with width padded up to power-of-two dimentions
================
*/
static byte *TexMgr_PadImageW(byte *in, int width, int height, byte padbyte) {
  int i, j, outwidth;
  byte *out, *data;

  if (width == TexMgr_Pad(width)) return in;

  outwidth = TexMgr_Pad(width);

  out = data = (byte *)Hunk_Alloc(outwidth * height);

  for (i = 0; i < height; i++) {
    for (j = 0; j < width; j++) *out++ = *in++;
    for (; j < outwidth; j++) *out++ = padbyte;
  }

  return data;
}

/*
================
TexMgr_PadImageH -- return image with height padded up to power-of-two
dimentions
================
*/
static byte *TexMgr_PadImageH(byte *in, int width, int height, byte padbyte) {
  int i, srcpix, dstpix;
  byte *data, *out;

  if (height == TexMgr_Pad(height)) return in;

  srcpix = width * height;
  dstpix = width * TexMgr_Pad(height);

  out = data = (byte *)Hunk_Alloc(dstpix);

  for (i = 0; i < srcpix; i++) *out++ = *in++;
  for (; i < dstpix; i++) *out++ = padbyte;

  return data;
}

/*
================
TexMgr_LoadImage32 -- handles 32bit source data
================
*/
static void TexMgr_LoadImage32(gltexture_t *glt, unsigned *data) {
  int internalformat, miplevel, mipwidth, mipheight, picmip;

  // mipmap down
  picmip = (glt->flags & TEXPREF_NOPICMIP)
               ? 0
               : q_max((int)Cvar_GetValue(&gl_picmip), 0);
  mipwidth = TexMgr_SafeTextureSize(glt->width >> picmip);
  mipheight = TexMgr_SafeTextureSize(glt->height >> picmip);
  while ((int)glt->width > mipwidth) {
    TexMgr_MipMapW(data, glt->width, glt->height);
    glt->width >>= 1;
    if (glt->flags & TEXPREF_ALPHA)
      TexMgr_AlphaEdgeFix((byte *)data, glt->width, glt->height);
  }
  while ((int)glt->height > mipheight) {
    TexMgr_MipMapH(data, glt->width, glt->height);
    glt->height >>= 1;
    if (glt->flags & TEXPREF_ALPHA)
      TexMgr_AlphaEdgeFix((byte *)data, glt->width, glt->height);
  }

  // upload
  GL_Bind(glt);
  internalformat =
      (glt->flags & TEXPREF_ALPHA) ? gl_alpha_format : gl_solid_format;
  // THERJAK: glTexImage2D is opengl
  GL_TexImage2D(GL_TEXTURE_2D, 0, internalformat, glt->width, glt->height, 0,
                GL_RGBA, GL_UNSIGNED_BYTE, data);

  // upload mipmaps
  if (glt->flags & TEXPREF_MIPMAP) {
    mipwidth = glt->width;
    mipheight = glt->height;

    for (miplevel = 1; mipwidth > 1 || mipheight > 1; miplevel++) {
      if (mipwidth > 1) {
        TexMgr_MipMapW(data, mipwidth, mipheight);
        mipwidth >>= 1;
      }
      if (mipheight > 1) {
        TexMgr_MipMapH(data, mipwidth, mipheight);
        mipheight >>= 1;
      }
      // THERJAK: glTexImage2D is opengl
      GL_TexImage2D(GL_TEXTURE_2D, miplevel, internalformat, mipwidth,
                    mipheight, 0, GL_RGBA, GL_UNSIGNED_BYTE, data);
    }
  }

  // set filter modes
  TexMgr_SetFilterModes(glt);
}

/*
================
TexMgr_LoadImage8 -- handles 8bit source data, then passes it to LoadImage32
================
*/
static void TexMgr_LoadImage8(gltexture_t *glt, byte *data) {
  extern cvar_t gl_fullbrights;
  qboolean padw = false, padh = false;
  byte padbyte;
  unsigned int *usepal;
  int i;

  // HACK HACK HACK -- taken from tomazquake
  if (strstr(glt->name, "shot1sid") && glt->width == 32 && glt->height == 32 &&
      CRC_Block(data, 1024) == 65393) {
    // This texture in b_shell1.bsp has some of the first 32 pixels painted
    // white.
    // They are invisible in software, but look really ugly in GL. So we just
    // copy
    // 32 pixels from the bottom to make it look nice.
    memcpy(data, data + 32 * 31, 32);
  }

  // detect false alpha cases
  if (glt->flags & TEXPREF_ALPHA && !(glt->flags & TEXPREF_CONCHARS)) {
    for (i = 0; i < (int)(glt->width * glt->height); i++)
      if (data[i] == 255)  // transparent index
        break;
    if (i == (int)(glt->width * glt->height)) glt->flags -= TEXPREF_ALPHA;
  }

  // choose palette and padbyte
  if (glt->flags & TEXPREF_FULLBRIGHT) {
    if (glt->flags & TEXPREF_ALPHA)
      usepal = d_8to24table_fbright_fence;
    else
      usepal = d_8to24table_fbright;
    padbyte = 0;
  } else if (glt->flags & TEXPREF_NOBRIGHT && Cvar_GetValue(&gl_fullbrights)) {
    if (glt->flags & TEXPREF_ALPHA)
      usepal = d_8to24table_nobright_fence;
    else
      usepal = d_8to24table_nobright;
    padbyte = 0;
  } else if (glt->flags & TEXPREF_CONCHARS) {
    usepal = d_8to24table_conchars;
    padbyte = 0;
  } else {
    usepal = d_8to24table;
    padbyte = 255;
  }

  // pad each dimention, but only if it's not going to be downsampled later
  if (glt->flags & TEXPREF_PAD) {
    if ((int)glt->width < TexMgr_SafeTextureSize(glt->width)) {
      data = TexMgr_PadImageW(data, glt->width, glt->height, padbyte);
      glt->width = TexMgr_Pad(glt->width);
      padw = true;
    }
    if ((int)glt->height < TexMgr_SafeTextureSize(glt->height)) {
      data = TexMgr_PadImageH(data, glt->width, glt->height, padbyte);
      glt->height = TexMgr_Pad(glt->height);
      padh = true;
    }
  }

  // convert to 32bit
  data = (byte *)TexMgr_8to32(data, glt->width * glt->height, usepal);

  // fix edges
  if (glt->flags & TEXPREF_ALPHA)
    TexMgr_AlphaEdgeFix(data, glt->width, glt->height);
  else {
    if (padw) TexMgr_PadEdgeFixW(data, glt->source_width, glt->source_height);
    if (padh) TexMgr_PadEdgeFixH(data, glt->source_width, glt->source_height);
  }

  // upload it
  TexMgr_LoadImage32(glt, (unsigned *)data);
}

/*
================
TexMgr_LoadLightmap -- handles lightmap data
================
*/
static void TexMgr_LoadLightmap(gltexture_t *glt, byte *data) {
  // upload it
  GL_Bind(glt);
  // THERJAK
  GL_TexImage2D(GL_TEXTURE_2D, 0, lightmap_bytes, glt->width, glt->height, 0,
                gl_lightmap_format, GL_UNSIGNED_BYTE, data);

  // set filter modes
  TexMgr_SetFilterModes(glt);
}

/*
================
TexMgr_LoadImage -- the one entry point for loading all textures
================
*/
gltexture_t *TexMgr_LoadImage(qmodel_t *owner, const char *name, int width,
                              int height, enum srcformat format, byte *data,
                              const char *source_file,
                              src_offset_t source_offset, unsigned flags) {
  unsigned short crc;
  gltexture_t *glt;
  int mark;

  if (CMLDedicated()) return NULL;

  // cache check
  switch (format) {
    case SRC_INDEXED:
      crc = CRC_Block(data, width * height);
      break;
    case SRC_LIGHTMAP:
      crc = CRC_Block(data, width * height * lightmap_bytes);
      break;
    case SRC_RGBA:
      crc = CRC_Block(data, width * height * 4);
      break;
    default: /* not reachable but avoids compiler warnings */
      crc = 0;
  }
  if ((flags & TEXPREF_OVERWRITE) && (glt = TexMgr_FindTexture(owner, name))) {
    if (glt->source_crc == crc) return glt;
  } else
    glt = TexMgr_NewTexture();

  // copy data
  glt->owner = owner;
  q_strlcpy(glt->name, name, sizeof(glt->name));
  glt->width = width;
  glt->height = height;
  glt->flags = flags;
  glt->shirt = -1;
  glt->pants = -1;
  q_strlcpy(glt->source_file, source_file, sizeof(glt->source_file));
  glt->source_offset = source_offset;
  glt->source_format = format;
  glt->source_width = width;
  glt->source_height = height;
  glt->source_crc = crc;

  // upload it
  mark = Hunk_LowMark();

  switch (glt->source_format) {
    case SRC_INDEXED:
      TexMgr_LoadImage8(glt, data);
      break;
    case SRC_LIGHTMAP:
      TexMgr_LoadLightmap(glt, data);
      break;
    case SRC_RGBA:
      TexMgr_LoadImage32(glt, (unsigned *)data);
      break;
  }

  Hunk_FreeToLowMark(mark);

  return glt;
}

/*
================================================================================

        COLORMAPPING AND TEXTURE RELOADING

================================================================================
*/

/*
================
TexMgr_ReloadImage -- reloads a texture, and colormaps it if needed
================
*/
void TexMgr_ReloadImage(gltexture_t *glt, int shirt, int pants) {
  byte translation[256];
  byte *src, *dst, *data = NULL, *translated;
  int mark, size, i;
  //
  // get source data
  //
  mark = Hunk_LowMark();

  if (glt->source_file[0] && glt->source_offset) {
    // lump inside file
    long size;
    int length = 0;
    byte *file = COM_LoadFileGo(glt->source_file, &length);
    if (!file) goto invalid;
    size = (long)(glt->source_width * glt->source_height);
    /* should be SRC_INDEXED, but no harm being paranoid:  */
    if (glt->source_format == SRC_RGBA)
      size *= 4;
    else if (glt->source_format == SRC_LIGHTMAP)
      size *= lightmap_bytes;

    if (glt->source_offset + size > (long)(length)) {
      free(file);
      goto invalid;
    }

    data = (byte *)Hunk_Alloc(size);
    byte *fs = file + glt->source_offset;
    memcpy(data, fs, size);

    free(file);
  } else if (glt->source_file[0] && !glt->source_offset)
    data = Image_LoadImage(glt->source_file, (int *)&glt->source_width,
                           (int *)&glt->source_height);  // simple file
  else if (!glt->source_file[0] && glt->source_offset)
    data = (byte *)glt->source_offset;  // image in memory

  if (!data) {
  invalid:
    Con_Printf("TexMgr_ReloadImage: invalid source for %s\n", glt->name);
    Hunk_FreeToLowMark(mark);
    return;
  }

  glt->width = glt->source_width;
  glt->height = glt->source_height;
  //
  // apply shirt and pants colors
  //
  // if shirt and pants are -1,-1, use existing shirt and pants colors
  // if existing shirt and pants colors are -1,-1, don't bother colormapping
  if (shirt > -1 && pants > -1) {
    if (glt->source_format == SRC_INDEXED) {
      glt->shirt = shirt;
      glt->pants = pants;
    } else
      Con_Printf(
          "TexMgr_ReloadImage: can't colormap a non SRC_INDEXED texture: %s\n",
          glt->name);
  }
  if (glt->shirt > -1 && glt->pants > -1) {
    // create new translation table
    for (i = 0; i < 256; i++) translation[i] = i;

    shirt = glt->shirt * 16;
    if (shirt < 128) {
      for (i = 0; i < 16; i++) translation[TOP_RANGE + i] = shirt + i;
    } else {
      for (i = 0; i < 16; i++) translation[TOP_RANGE + i] = shirt + 15 - i;
    }

    pants = glt->pants * 16;
    if (pants < 128) {
      for (i = 0; i < 16; i++) translation[BOTTOM_RANGE + i] = pants + i;
    } else {
      for (i = 0; i < 16; i++) translation[BOTTOM_RANGE + i] = pants + 15 - i;
    }

    // translate texture
    size = glt->width * glt->height;
    dst = translated = (byte *)Hunk_Alloc(size);
    src = data;

    for (i = 0; i < size; i++) *dst++ = translation[*src++];

    data = translated;
  }
  //
  // upload it
  //
  switch (glt->source_format) {
    case SRC_INDEXED:
      TexMgr_LoadImage8(glt, data);
      break;
    case SRC_LIGHTMAP:
      TexMgr_LoadLightmap(glt, data);
      break;
    case SRC_RGBA:
      TexMgr_LoadImage32(glt, (unsigned *)data);
      break;
  }

  Hunk_FreeToLowMark(mark);
}
