// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"

import (
	"log"
	"runtime/debug"

	"goquake/glh"
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

//export GetTextureWidth
func GetTextureWidth(id uint32) uint32 {
	return uint32(texmap[glh.TexID(id)].Width)
}

//export GetTextureHeight
func GetTextureHeight(id uint32) int32 {
	return int32(texmap[glh.TexID(id)].Height)
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
