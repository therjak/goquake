package quakelib

//#include "gl_model.h"
//#include "gl_texmgr.h"
// extern texture_t *r_notexture_mip, *r_notexture_mip2;
import "C"

import (
	"github.com/therjak/goquake/glh"
	"github.com/therjak/goquake/texture"
	"github.com/therjak/goquake/wad"
	"log"
	"unsafe"

	"github.com/go-gl/gl/v4.6-core/gl"
)

var (
	texmap map[glh.TexID]*texture.Texture
)

func init() {
	texmap = make(map[glh.TexID]*texture.Texture)
}

//export GL_warpimagesize
func GL_warpimagesize() int32 {
	return glWarpImageSize
}

//export GetMTexEnabled
func GetMTexEnabled() bool {
	return textureManager.multiTextureEnabled
}

//export GetNoTexture
func GetNoTexture() uint32 {
	return uint32(noTexture.ID())
}

//export GetTextureWidth
func GetTextureWidth(id uint32) uint32 {
	return uint32(texmap[glh.TexID(id)].Width)
}

//export GetTextureHeight
func GetTextureHeight(id uint32) int32 {
	return int32(texmap[glh.TexID(id)].Height)
}

//export TexMgrLoadLightMapImage
func TexMgrLoadLightMapImage(owner *C.qmodel_t, name *C.char, width C.int,
	height C.int, data *C.byte, flags C.unsigned) uint32 {

	// TODO(therjak): add cache ala
	// if TexPrefOverWrite && owner&name&crc match
	//  return old one

	d := C.GoBytes(unsafe.Pointer(data), width*height*4)
	t := texture.NewTexture(int32(width), int32(height),
		texture.TexPref(flags), C.GoString(name), texture.ColorTypeLightmap, d)

	textureManager.addActiveTexture(t)
	textureManager.loadLightMap(t, d)
	texmap[t.ID()] = t
	return uint32(t.ID())
}

//export Go_LoadWad
func Go_LoadWad() {
	err := wad.Load()
	if err != nil {
		Error("Could not load wad: %v", err)
	}
}

//export TexMgrLoadParticleImage
func TexMgrLoadParticleImage(name *C.char, width C.int,
	height C.int, data *C.byte) uint32 {
	d := C.GoBytes(unsafe.Pointer(data), width*height*4)
	t := textureManager.loadParticleImage(C.GoString(name), int32(width), int32(height), d)
	texmap[t.ID()] = t
	return uint32(t.ID())
}

//export TexMgrLoadSkyTexture
func TexMgrLoadSkyTexture(name *C.char, data *C.byte, flags C.unsigned) uint32 {
	n := C.GoString(name)
	d := C.GoBytes(unsafe.Pointer(data), 128*128)
	t := textureManager.LoadSkyTexture(n, d, texture.TexPref(flags))
	texmap[t.ID()] = t
	return uint32(t.ID())
}

//export TexMgrLoadSkyBox
func TexMgrLoadSkyBox(name *C.char) uint32 {
	n := C.GoString(name)
	t := textureManager.LoadSkyBox(n)
	if t == nil {
		return 0
	}
	texmap[t.ID()] = t
	return uint32(t.ID())
}

//export TexMgrLoadImage2
func TexMgrLoadImage2(owner *C.qmodel_t, name *C.char, width C.int,
	height C.int, format C.enum_srcformat, data *C.byte, source_file *C.char,
	source_offset C.src_offset_t, flags C.unsigned) uint32 {

	d, ct := func() ([]byte, texture.ColorType) {
		switch format {
		case C.SRC_RGBA:
			return C.GoBytes(unsafe.Pointer(data), width*height*4), texture.ColorTypeRGBA
		default: // C.SRC_INDEXED
			return C.GoBytes(unsafe.Pointer(data), width*height), texture.ColorTypeIndexed
		}
	}()

	t := texture.NewTexture(int32(width), int32(height), texture.TexPref(flags), C.GoString(name), ct, d)
	textureManager.addActiveTexture(t)
	switch format {
	case C.SRC_RGBA:
		textureManager.loadRGBA(t, d)
	default: // C.SRC_INDEXED
		textureManager.loadIndexed(t, d)
	}
	texmap[t.ID()] = t
	return uint32(t.ID())
}

//export TexMgrReloadImage
func TexMgrReloadImage(id uint32, shirt C.int, pants C.int) {
	t := texmap[glh.TexID(id)]
	textureManager.ReloadImage(t)
	// this is actually a 'map texture & reload'
	// it should work quite different to the others.
	// It also implies indexed colors
}

//export TexMgrFreeTexture
func TexMgrFreeTexture(id uint32) {
	textureManager.FreeTexture(texmap[glh.TexID(id)])
	delete(texmap, glh.TexID(id))
}

//export TexMgrFrameUsage
func TexMgrFrameUsage() float32 {
	return textureManager.FrameUsage()
}

//export TexMgrFreeTexturesForOwner
func TexMgrFreeTexturesForOwner(owner *C.qmodel_t) {
	// TODO(therjak): free all activeTextures with this owner
}

//export D8To24Table
func D8To24Table(i, p int) byte {
	return palette.table[i*4+p]
}

//export TexMgrInit
func TexMgrInit() {
	gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &textureManager.maxAnisotropy)
	gl.GetIntegerv(gl.MAX_TEXTURE_SIZE, &textureManager.maxTextureSize)
	palette.Init()
	TexMgrRecalcWarpImageSize()
	noTexture = textureManager.LoadNoTex("notexture", 2, 2, []byte{
		159, 91, 83, 255, 0, 0, 0, 255,
		0, 0, 0, 255, 159, 91, 83, 255,
	})
	nullTexture = textureManager.LoadNoTex("nulltexture", 2, 2, []byte{
		127, 191, 255, 255, 0, 0, 0, 255,
		0, 0, 0, 255, 127, 191, 255, 255,
	})

	// Mod_Init is called before, so we need to do this here
	C.r_notexture_mip.gltexture = C.uint(noTexture.ID())
	C.r_notexture_mip2.gltexture = C.uint(noTexture.ID())
	texmap[noTexture.ID()] = noTexture
	texmap[nullTexture.ID()] = nullTexture
}

//export TexMgrDeleteTextureObjects
func TexMgrDeleteTextureObjects() {
	// This only discards all opengl objects. They get recreated
	// in TexMgrReloadImages
	textureManager.DeleteTextureObjects()
}

//export TexMgrReloadImages
func TexMgrReloadImages() {
	// This is the reverse of TexMgrFreeTexturesObjects
	// It is only called on VID_Restart (resolution change, vid_restart)
	textureManager.ReloadImages()
}

//export TexMgrRecalcWarpImageSize
func TexMgrRecalcWarpImageSize() {
	textureManager.RecalcWarpImageSize(screen.Width, screen.Height)
}

//export GLDisableMultitexture
func GLDisableMultitexture() {
	textureManager.DisableMultiTexture()
}

//export GLEnableMultitexture
func GLEnableMultitexture() {
	textureManager.EnableMultiTexture()
}

//export GLSelectTexture
func GLSelectTexture(target uint32) {
	textureManager.SelectTextureUnit(target)
}

//export GLBind
func GLBind(id uint32) {
	qid := glh.TexID(id)
	textureManager.Bind(texmap[qid])
	if texmap[qid].ID() != qid {
		log.Printf("broken glID: %v, %v", texmap[qid].ID(), id)
	}
}

//export GLClearBindings
func GLClearBindings() {
	textureManager.ClearBindings()
}