// SPDX-License-Identifier: GPL-2.0-or-later
package texture

import (
	"github.com/therjak/goquake/glh"
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
)

type Texture struct {
	glID   glh.Texture
	Width  int32 // mipmap can make it differ from source width
	Height int32
	flags  TexPref
	name   string
	Typ    ColorType
	Data   []byte
}

func NewTexture(w, h int32, flags TexPref, name string, typ ColorType, data []byte) *Texture {
	t := &Texture{
		glID:   glh.NewTexture2D(),
		Width:  w,
		Height: h,
		flags:  flags,
		name:   name,
		Typ:    typ,
		Data:   data,
	}
	return t
}

func (t *Texture) Bind() {
	t.glID.Bind()
}

func (t *Texture) ID() glh.TexID {
	return t.glID.ID()
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
