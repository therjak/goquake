// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

//extern unsigned int gl_bmodel_vbo;
//void GL_BuildBModelVertexBufferOld(void);
import "C"

import (

	// "github.com/chewxy/math32"
	"github.com/go-gl/gl/v4.6-core/gl"
	"goquake/bsp"
	"goquake/glh"
	// "goquake/math/vec"
)

type qBrushDrawer struct {
	vao        *glh.VertexArray
	vbo        *glh.Buffer
	ebo        *glh.Buffer
	prog       *glh.Program
	projection int32
	modelview  int32
}

func NewBrushDrawer() *qBrushDrawer {
	d := &qBrushDrawer{}
	// d.vao
	// d.ebo
	d.vbo = glh.NewBuffer(glh.ArrayBuffer)
	// d.prog
	// d.projection
	// d.modelview
	return d
}

var (
	// brushDrawer *qBrushDrawer
	worldDrawer *qBrushDrawer
)

//export GL_BuildBModelVertexBuffer
func GL_BuildBModelVertexBuffer() {
	if worldDrawer == nil {
		worldDrawer = NewBrushDrawer()
		C.gl_bmodel_vbo = C.uint(worldDrawer.vbo.ID())
	}
	worldDrawer.buildVertexBuffer()
	// Provide vboFirstVert on the C side
	C.GL_BuildBModelVertexBufferOld()
}

func (d *qBrushDrawer) buildVertexBuffer() {
	// Gets called once per map
	idx := 0
	var buf []float32
	for _, m := range cl.modelPrecache {
		switch w := m.(type) {
		case *bsp.Model:
			for _, s := range w.Surfaces {
				// Why? We are changing the model again...
				s.VboFirstVert = idx
				idx += len(s.Polys.Verts)
				for _, v := range s.Polys.Verts {
					buf = append(buf,
						v.Pos[0], v.Pos[1], v.Pos[2],
						v.S, v.T, // not in [0,1]
						v.LightMapS, v.LightMapT)
				}
			}
		}
	}
	d.vbo.Bind()
	d.vbo.SetData(4*len(buf), gl.Ptr(buf))
}
