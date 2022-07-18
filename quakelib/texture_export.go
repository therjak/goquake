// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"

import (
	"github.com/go-gl/gl/v4.6-core/gl"
)

//export GetTextureWidth
func GetTextureWidth(id uint32) uint32 {
	return 0
}

//export GetTextureHeight
func GetTextureHeight(id uint32) int32 {
	return 0
}

func textureManagerInit() {
	gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &textureManager.maxAnisotropy)
	gl.GetIntegerv(gl.MAX_TEXTURE_SIZE, &textureManager.maxTextureSize)
	nullTexture = textureManager.LoadNoTex("nulltexture", 2, 2, []byte{
		127, 191, 255, 255, 0, 0, 0, 255,
		0, 0, 0, 255, 127, 191, 255, 255,
	})
}

//export GLBind
func GLBind(id uint32) {
}
