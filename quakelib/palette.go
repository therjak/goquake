package quakelib

//void TexMgrLoadPalette(void);
import "C"

import (
	"quake/filesystem"
)

type qPalette struct {
	table [256 * 4]float32
	/*
	   unsigned int d_8to24table[256];
	   unsigned int d_8to24table_fbright[256];
	   unsigned int d_8to24table_fbright_fence[256];
	   unsigned int d_8to24table_nobright[256];
	   unsigned int d_8to24table_nobright_fence[256];
	   unsigned int d_8to24table_conchars[256];
	*/
}

var (
	palette qPalette
)

//export TexMgr_LoadPalette
func TexMgr_LoadPalette() {
	C.TexMgrLoadPalette()
	palette.Init()
}

func (p *qPalette) Init() {
	b, err := filesystem.GetFileContents("gfx/palette.lmp")
	if err != nil {
		Error("Couln't load gfx/palette.lmp")
	}
	// b is rgb 8bit, we want rgba float32
	if 4*len(b) != 3*len(p.table) {
		Error("Palette has wrong size: %v", len(b))
	}
	bi := 0
	pi := 0
	for i := 0; i < 256; i++ {
		p.table[pi] = float32(b[bi]) / 255
		p.table[pi+1] = float32(b[bi+1]) / 255
		p.table[pi+2] = float32(b[bi+2]) / 255
		p.table[pi+3] = 1
		pi += 4
		bi += 3
	}
	// orig changed the last value to alpha 0?
}

/*
void TexMgr_LoadPalette(void) {
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
}*/
