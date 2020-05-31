package quakelib

//void Sky_Init(void);
//void Sky_DrawSky(void);
//void Sky_NewMap(void);
//void Sky_LoadSkyBox(const char *name);
//void Sky_LoadTextureInt(const unsigned char* src, const char* skyName, const char* modelName);
import "C"

import (
	"fmt"
	"unsafe"
)

type qSky struct {
	boxName      string
	boxTextures  [6]*Texture
	solidTexture *Texture
	alphaTexture *Texture
	flat         Color
}

var sky qSky

//export SkyInit
func SkyInit() {
	C.Sky_Init()
}

//export SkyDrawSky
func SkyDrawSky() {
	C.Sky_DrawSky()
}

//export SkyNewMap
func SkyNewMap() {
	C.Sky_NewMap()
}

//export SkyLoadSkyBox
func SkyLoadSkyBox(c *C.char) {
	C.Sky_LoadSkyBox(c)
	return

	name := C.GoString(c)
	sky.LoadBox(name)
}

var (
	skySuf = [6]string{"rt", "bk", "lf", "ft", "up", "dn"}
)

func (s *qSky) LoadBox(name string) {
	if name == s.boxName {
		return
	}
	s.boxName = name
	for i, t := range s.boxTextures {
		textureManager.FreeTexture(t)
		s.boxTextures[i] = noTexture
	}
	if s.boxName == "" {
		return
	}
	noneFound := true
	for i, suf := range skySuf {
		n := fmt.Sprintf("gfx/env/%s%s", s.boxName, suf)
		s.boxTextures[i] = textureManager.LoadSkyBox(n)
		if s.boxTextures[i] != nil {
			noneFound = false
		} else {
			s.boxTextures[i] = noTexture
		}
	}
	if noneFound {
		// boxName == "" => No DrawSkyBox but only DrawSkyLayers
		s.boxName = ""
		return
	}
}

//export SkyLoadTexture
func SkyLoadTexture(src *C.uchar, skyName *C.char, modelName *C.char) {
	C.Sky_LoadTextureInt(src, skyName, modelName)
	return
	s := C.GoString(skyName)
	m := C.GoString(modelName)
	b := C.GoBytes(unsafe.Pointer(src), 256*128)
	sky.LoadTexture(b, s, m)
}

func (s *qSky) LoadTexture(d []byte, skyName, modelName string) {
	// d is a 256*128 texture with the left side being a masked overlay
	// What a mess. It would be better to have the overlay at the bottom.
	front := [128 * 128]byte{}
	back := [128 * 128]byte{}
	var r, g, b, count int
	for i := 0; i < 128; i++ {
		for j := 0; j < 128; j++ {
			sidx := i*256 + j
			didx := i*128 + j
			p := d[sidx]
			if p == 0 {
				front[didx] = 255
			} else {
				front[didx] = p
				pixel := palette.table[p*4 : p*4+4]
				r += int(pixel[0])
				g += int(pixel[1])
				b += int(pixel[2])
				count++ // only count opaque colors
			}
			back[didx] = d[sidx+128]
		}
	}
	fn := fmt.Sprintf("%s:%s_front", modelName, skyName)
	bn := fmt.Sprintf("%s:%s_back", modelName, skyName)
	s.solidTexture = textureManager.LoadSkyTexture(bn, back[:], TexPrefNone)
	s.alphaTexture = textureManager.LoadSkyTexture(fn, front[:], TexPrefAlpha)
	s.flat = Color{
		R: float32(r) / (float32(count) * 255),
		G: float32(g) / (float32(count) * 255),
		B: float32(b) / (float32(count) * 255),
	}
}

// uses
// cl_numvisedicts
// cl_visedicts
// R_CullModelForEntity
// cl.worldmodel->numtextures
// cl.worldmodel->textures
// cl.worldmodel->entities
// DrawGLPoly
// Fog_GetDensity
// Fog_GetColor
// Fog_DisableGFog()
// Fog_EnableGFog()
// r_origin
// gl_mtexable
// rs_skypolys
// rs_skypasses
