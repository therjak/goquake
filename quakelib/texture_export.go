// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

//#include "gl_model.h"
//#include "gl_texmgr.h"
// extern texture_t *r_notexture_mip, *r_notexture_mip2;
import "C"

import (
	"fmt"
	"log"
	"runtime/debug"
	"unsafe"

	"github.com/therjak/goquake/glh"
	"github.com/therjak/goquake/texture"

	"github.com/go-gl/gl/v4.6-core/gl"
)

var (
	texmap map[glh.TexID]*texture.Texture
)

func init() {
	texmap = make(map[glh.TexID]*texture.Texture)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
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
	return uint32(unusedTexture)
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

//export LoadPlayerTexture
func LoadPlayerTexture(playerNum int, width, height int, data *C.byte) {
	// 1 to cl.maxClients
	if playerNum < 0 || playerNum >= cl.maxClients {
		log.Printf("Bad LoadPlayerTextures: %v", playerNum)
		return
	}
	name := fmt.Sprintf("player_%d", playerNum)
	flags := texture.TexPrefPad | texture.TexPrefOverwrite
	d := C.GoBytes(unsafe.Pointer(data), C.int(width*height))

	t := texture.NewTexture(int32(width), int32(height),
		flags, name, texture.ColorTypeIndexed, d)
	textureManager.addActiveTexture(t)
	textureManager.loadIndexed(t, d)

	e := cl.Entities(playerNum + 1)
	playerTextures[e.ptr] = t
}

//export TexMgrLoadImage2
func TexMgrLoadImage2(name *C.char, width C.int,
	height C.int, format C.enum_srcformat, data *C.byte, source_file *C.char,
	flags C.unsigned) uint32 {

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

func textureManagerInit() {
	gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &textureManager.maxAnisotropy)
	gl.GetIntegerv(gl.MAX_TEXTURE_SIZE, &textureManager.maxTextureSize)
	palette.Init()
	textureManager.RecalcWarpImageSize(screen.Width, screen.Height)
	nullTexture = textureManager.LoadNoTex("nulltexture", 2, 2, []byte{
		127, 191, 255, 255, 0, 0, 0, 255,
		0, 0, 0, 255, 127, 191, 255, 255,
	})

	// Mod_Init is called before, so we need to do this here
	C.r_notexture_mip.gltexture = C.uint(unusedTexture)
	C.r_notexture_mip2.gltexture = C.uint(unusedTexture)
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
	if id == 0 {
		debug.PrintStack()
	}
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
