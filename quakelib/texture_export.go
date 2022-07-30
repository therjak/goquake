// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"github.com/go-gl/gl/v4.6-core/gl"
)

func textureManagerInit() {
	gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &textureManager.maxAnisotropy)
	gl.GetIntegerv(gl.MAX_TEXTURE_SIZE, &textureManager.maxTextureSize)
	nullTexture = textureManager.LoadNoTex("nulltexture", 2, 2, []byte{
		127, 191, 255, 255, 0, 0, 0, 255,
		0, 0, 0, 255, 127, 191, 255, 255,
	})
}
