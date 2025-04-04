// SPDX-License-Identifier: GPL-2.0-or-later

package glh

import (
	"runtime"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/gopxl/mainthread/v2"
)

type TexID uint32

type Texture interface {
	Bind()
}

type texture struct {
	id uint32
}

type texture2D struct {
	texture
}
type textureCube struct {
	texture
}

func (t *texture) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteTextures(1, &t.id)
	})
}

func (t *texture) new() {
	gl.GenTextures(1, &t.id)
	runtime.SetFinalizer(t, (*texture).delete)
}

func NewTexture2D() *texture2D {
	t := &texture2D{}
	t.new()
	return t
}

func (t *texture2D) Bind() {
	gl.BindTexture(gl.TEXTURE_2D, t.id)
}

func NewTextureCube() *textureCube {
	t := &textureCube{}
	t.new()
	return t
}

func (t *textureCube) Bind() {
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, t.id)
}
