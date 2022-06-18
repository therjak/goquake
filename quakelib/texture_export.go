// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

//#include "gl_model.h"
//#include "gl_texmgr.h"
import "C"

import (
	"log"
	"runtime/debug"

	"goquake/glh"
	"goquake/palette"
	"goquake/texture"

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

//export GetTextureWidth
func GetTextureWidth(id uint32) uint32 {
	return uint32(texmap[glh.TexID(id)].Width)
}

//export GetTextureHeight
func GetTextureHeight(id uint32) int32 {
	return int32(texmap[glh.TexID(id)].Height)
}

//export TexMgrFreeTexturesForOwner
func TexMgrFreeTexturesForOwner(owner *C.qmodel_t) {
	// TODO(therjak): free all activeTextures with this owner
}

//export D8To24Table
func D8To24Table(i, p int) byte {
	return palette.Table[i*4+p]
}

func textureManagerInit() {
	gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &textureManager.maxAnisotropy)
	gl.GetIntegerv(gl.MAX_TEXTURE_SIZE, &textureManager.maxTextureSize)
	textureManager.RecalcWarpImageSize(screen.Width, screen.Height)
	nullTexture = textureManager.LoadNoTex("nulltexture", 2, 2, []byte{
		127, 191, 255, 255, 0, 0, 0, 255,
		0, 0, 0, 255, 127, 191, 255, 255,
	})
}

//export TexMgrReloadImages
func TexMgrReloadImages() {
	// This is the reverse of TexMgrFreeTexturesObjects
	// It is only called on VID_Restart (resolution change, vid_restart)
	textureManager.ReloadImages()
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
