// SPDX-License-Identifier: GPL-2.0-or-later
package bsp

import (
	"fmt"
	"goquake/palette"
	"goquake/texture"
	"strings"
)

func (t *Texture) loadSkyTexture(data []byte, textureName, modelName string) {
	// d is a 256*128 texture with the left side being a masked overlay
	// What a mess. It would be better to have the overlay at the bottom.
	front := [128 * 128]byte{}
	back := [128 * 128]byte{}
	var r, g, b, count int
	for i := 0; i < 128; i++ {
		for j := 0; j < 128; j++ {
			sidx := i*256 + j
			didx := i*128 + j
			p := data[sidx]
			if p == 0 {
				front[didx] = 255
			} else {
				front[didx] = p
				pixel := palette.Table[p*4 : p*4+4]
				r += int(pixel[0])
				g += int(pixel[1])
				b += int(pixel[2])
				count++ // only count opaque colors
			}
			back[didx] = data[sidx+128]
		}
	}

	fn := fmt.Sprintf("%s:%s_front", modelName, textureName)
	bn := fmt.Sprintf("%s:%s_back", modelName, textureName)
	t.SolidSky = texture.NewTexture(128, 128, texture.TexPrefNone, fn, texture.ColorTypeIndexed, front[:])
	t.AlphaSky = texture.NewTexture(128, 128, texture.TexPrefAlpha, bn, texture.ColorTypeIndexed, back[:])

	t.FlatSky = Color{
		R: float32(r) / (float32(count) * 255),
		G: float32(g) / (float32(count) * 255),
		B: float32(b) / (float32(count) * 255),
	}
}

func checkFullbrights(data []byte) bool {
	for _, d := range data {
		if d > 223 {
			return true
		}
	}
	return false
}

func (t *Texture) loadBspTexture(data []byte, textureName, modelName string) {
	var extraFlag texture.TexPref
	if strings.HasPrefix(textureName, "{") {
		extraFlag = texture.TexPrefAlpha
	}

	// TODO: integrate texMgr.loadIndexed and return as RGBA instead of Indexed
	if checkFullbrights(t.Data) {
		tName := fmt.Sprintf("%s:%s", modelName, textureName)
		t.Texture = texture.NewTexture(
			int32(t.Width),
			int32(t.Height),
			texture.TexPrefMipMap|texture.TexPrefNoBright|extraFlag,
			tName,
			texture.ColorTypeIndexed,
			data)
		fbName := fmt.Sprintf("%s:%s_glow", modelName, textureName)
		t.Fullbright = texture.NewTexture(
			int32(t.Width),
			int32(t.Height),
			texture.TexPrefMipMap|texture.TexPrefFullBright|extraFlag,
			fbName,
			texture.ColorTypeIndexed,
			data)
	} else {
		t.Texture = texture.NewTexture(
			int32(t.Width),
			int32(t.Height),
			texture.TexPrefMipMap|extraFlag,
			textureName,
			texture.ColorTypeIndexed,
			data)
	}
}
