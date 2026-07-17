// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

// translateDrawer draws a 2D quad using a three-texture lookup:
//   - TEXTURE0: raw index texture (one palette index per texel, GL_R8)
//   - TEXTURE1: 256-entry RGBA palette (1D texture)
//   - TEXTURE2: 256-entry remap LUT   (1D texture, GL_R8)
//
// This lets DrawPictureTranslate perform the shirt/pants colour remap
// entirely on the GPU without re-uploading the source texture.

import (
	"log"

	"goquake/glh"
	"goquake/palette"
	"goquake/texture"

	"github.com/go-gl/gl/v4.6-core/gl"
)

// Quake palette colour ranges used for player customisation.
const (
	topColorStart    = 144
	topColorStop     = 159
	bottomColorStart = 160
	bottomColorStop  = 175
)

// translateDrawer holds the GPU resources for palette-translate drawing.
type translateDrawer struct {
	vao  *glh.VertexArray
	vbo  *glh.Buffer
	ebo  *glh.Buffer
	prog *glh.Program

	// uniform locations
	uIndexTex   int32
	uPalette    int32
	uTranslation int32

	// shared 1D textures (created once, owned here)
	paletteTex    *texture.Texture // 256-entry RGBA palette
	translationTex *texture.Texture // 256-entry remap LUT
}


// newTranslateDrawProgram compiles the shader pair for palette translation.
func newTranslateDrawProgram() (*glh.Program, error) {
	return glh.NewProgram(vertexTextureSource, fragmentSourceTranslate)
}

// NewTranslateDrawer allocates all GPU objects and uploads the palette once.
func NewTranslateDrawer() (*translateDrawer, error) {
	d := &translateDrawer{}

	elements := []uint32{0, 1, 2, 2, 3, 0}
	d.vao = glh.NewVertexArray()
	d.vbo = glh.NewBuffer(glh.ArrayBuffer)
	d.ebo = glh.NewBuffer(glh.ElementArrayBuffer)
	d.ebo.Bind()
	d.ebo.SetData(4*len(elements), gl.Ptr(elements))

	var err error
	d.prog, err = newTranslateDrawProgram()
	if err != nil {
		return nil, err
	}

	d.uIndexTex    = d.prog.GetUniformLocation("indexTex")
	d.uPalette     = d.prog.GetUniformLocation("palette")
	d.uTranslation = d.prog.GetUniformLocation("translation")

	// ---- palette 1D texture (uploaded once) --------------------------------
	// palette.Table is [256*4]uint8 (RGBA). We upload it as a GL_TEXTURE_1D
	// with internal format GL_RGBA8.
	d.paletteTex = texture.NewTexture1D(256, texture.TexPrefNearest|texture.TexPrefNoPicMip,
		"_translate_palette", texture.ColorTypeRGBA, nil)
	d.paletteTex.Bind()
	gl.TexImage1D(gl.TEXTURE_1D, 0, gl.RGBA, 256, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&palette.Table[0]))
	gl.TexParameteri(gl.TEXTURE_1D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_1D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_1D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)

	// ---- translation LUT texture (updated per draw when colours change) -----
	// Identity mapping to start with.
	var lut [256]uint8
	for i := range lut {
		lut[i] = uint8(i)
	}
	d.translationTex = texture.NewTexture1D(256, texture.TexPrefNearest|texture.TexPrefNoPicMip,
		"_translate_lut", texture.ColorTypeRaw, nil)
	d.translationTex.Bind()
	// GL_R8 stores one red channel (0..255 normalised to 0..1 in the shader).
	gl.TexImage1D(gl.TEXTURE_1D, 0, gl.R8, 256, 0, gl.RED, gl.UNSIGNED_BYTE, gl.Ptr(&lut[0]))
	gl.TexParameteri(gl.TEXTURE_1D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_1D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_1D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)

	return d, nil
}

// buildTranslation produces the 256-byte remap table for the given top/bottom
// colour indices (0..13).  The logic mirrors the original C Quake code.
func buildTranslation(top, bottom int) [256]uint8 {
	var t [256]uint8
	for i := range t {
		t[i] = uint8(i)
	}

	shirt := top * 16
	if shirt < 128 {
		for i := 0; i < 16; i++ {
			t[topColorStart+i] = uint8(shirt + i)
		}
	} else {
		for i := 0; i < 16; i++ {
			t[topColorStart+i] = uint8(shirt + 15 - i)
		}
	}

	pants := bottom * 16
	if pants < 128 {
		for i := 0; i < 16; i++ {
			t[bottomColorStart+i] = uint8(pants + i)
		}
	} else {
		for i := 0; i < 16; i++ {
			t[bottomColorStart+i] = uint8(pants + 15 - i)
		}
	}

	return t
}

// UpdateTranslation recomputes and re-uploads the translation LUT.
func (d *translateDrawer) UpdateTranslation(top, bottom int) {
	lut := buildTranslation(top, bottom)
	d.translationTex.Bind()
	gl.TexSubImage1D(gl.TEXTURE_1D, 0, 0, 256, gl.RED, gl.UNSIGNED_BYTE, gl.Ptr(&lut[0]))
}

// Draw renders the textured quad at (x, y) with the given index texture.
// The palette and translation LUT are bound automatically.
func (d *translateDrawer) Draw(x, y, w, h float32, t *texture.Texture) {
	sx, sy := qCanvas.Apply()
	x1, x2 := x, x+w
	y1, y2 := y+h, y
	x1 = x1*sx - 1
	x2 = x2*sx - 1
	ys := qCanvas.YShift()
	y1 = -y1*sy + ys
	y2 = -y2*sy + ys

	vertices := []float32{
		x1, y2, 0, 0, 0,
		x2, y2, 0, 1, 0,
		x2, y1, 0, 1, 1,
		x1, y1, 0, 0, 1,
	}

	d.prog.Use()
	d.vao.Bind()
	d.ebo.Bind()
	d.vbo.Bind()
	d.vbo.SetData(4*len(vertices), gl.Ptr(vertices))

	gl.EnableVertexAttribArray(0)
	defer gl.DisableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1)
	defer gl.DisableVertexAttribArray(1)

	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 4*5, 0)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 4*5, 3*4)

	// TEXTURE0 = raw index texture
	gl.ActiveTexture(gl.TEXTURE0)
	t.Bind()
	gl.Uniform1i(d.uIndexTex, 0)

	// TEXTURE1 = palette
	gl.ActiveTexture(gl.TEXTURE1)
	d.paletteTex.Bind()
	gl.Uniform1i(d.uPalette, 1)

	// TEXTURE2 = translation LUT
	gl.ActiveTexture(gl.TEXTURE2)
	d.translationTex.Bind()
	gl.Uniform1i(d.uTranslation, 2)

	gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(0))

	// Restore active texture unit to the conventional unit 0.
	gl.ActiveTexture(gl.TEXTURE0)
}

// initTranslateDrawer is called during renderer startup.
func initTranslateDrawer() {
	var err error
	qTranslateDrawer, err = NewTranslateDrawer()
	if err != nil {
		log.Fatalf("failed to create translate drawer: %v", err)
	}
}
