// SPDX-License-Identifier: GPL-2.0-or-later

package texture

import (
	"goquake/glh"
)

type TexPref uint32

const (
	TexPrefMipMap TexPref = 1 << iota
	TexPrefLinear
	TexPrefNearest
	TexPrefAlpha
	TexPrefPad
	TexPrefPersist
	TexPrefOverwrite
	TexPrefNoPicMip
	TexPrefFullBright
	TexPrefNoBright
	TexPrefConChars
	TexPrefWarpImage
	TexPrefNone TexPref = 0
)

type ColorType int

const (
	ColorTypeIndexed ColorType = iota
	ColorTypeRGBA
	ColorTypeLightmap
	// ColorTypeRaw: one byte per texel stored as-is (palette indices).
	// The texture is uploaded as GL_R8 / GL_RED without palette conversion.
	ColorTypeRaw
)

type texType int

const (
	texType2D texType = iota
	texTypeCube
	texType1D
)

type Texture struct {
	glID   glh.Texture
	name   string
	Data   []byte
	Typ    ColorType
	tt     texType
	Width  int32 // mipmap can make it differ from source width
	Height int32
	flags  TexPref
}

func NewTexture(w, h int32, flags TexPref, name string, typ ColorType, data []byte) *Texture {
	t := &Texture{
		Width:  w,
		Height: h,
		flags:  flags,
		name:   name,
		Typ:    typ,
		Data:   data,
		tt:     texType2D,
	}
	return t
}

func NewCubeTexture(w, h int32, flags TexPref, name string, typ ColorType, data []byte) *Texture {
	// TODO
	t := &Texture{
		Width:  w,
		Height: h,
		flags:  flags,
		name:   name,
		Typ:    typ,
		Data:   data,
		tt:     texTypeCube,
	}
	return t
}

func (t *Texture) Bind() {
	if t.glID == nil {
		switch t.tt {
		case texTypeCube:
			t.glID = glh.NewTextureCube()
		case texType1D:
			t.glID = glh.NewTexture1D()
		default: // texType2D
			t.glID = glh.NewTexture2D()
		}
	}
	t.glID.Bind()
}

// NewTexture1D creates a 1-dimensional texture (e.g. palette or LUT).
func NewTexture1D(w int32, flags TexPref, name string, typ ColorType, data []byte) *Texture {
	t := &Texture{
		Width:  w,
		Height: 1,
		flags:  flags,
		name:   name,
		Typ:    typ,
		Data:   data,
		tt:     texType1D,
	}
	return t
}

func (t *Texture) Name() string {
	return t.name
}

func (t *Texture) Texels() int {
	if t.Flags(TexPrefMipMap) {
		return int(t.Width * t.Height * 4 / 3)
	}
	return int(t.Width * t.Height)
}

func (t *Texture) Flags(f TexPref) bool {
	return t.flags&f != 0
}
