// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

//extern unsigned int gl_bmodel_vbo;
//void GL_BuildBModelVertexBufferOld(void);
import "C"

import (

	// "github.com/chewxy/math32"
	"goquake/bsp"
	"goquake/cvars"
	"goquake/glh"
	"goquake/math/vec"

	"github.com/go-gl/gl/v4.6-core/gl"
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

func (r *qRenderer) DrawBrushModel(e *Entity, model *bsp.Model) {
	if r.cullBrush(e, model) {
		return
	}
	modelOrg := vec.Sub(qRefreshRect.viewOrg, e.Origin)
	if e.Angles[0] != 0 || e.Angles[1] != 0 || e.Angles[2] != 0 {
		tmp := modelOrg
		f, r, u := vec.AngleVectors(e.Angles)
		modelOrg[0] = vec.Dot(tmp, f)
		modelOrg[1] = -vec.Dot(tmp, r)
		modelOrg[2] = vec.Dot(tmp, u)
	}

	// calculate dynamic lighting for bmodel if it's not an instanced model
	if !cvars.GlFlashBlend.Bool() /*&& model.firstmodelsurface != 0*/ {
		// R_MarkLights(model.Nodes + model.Hulls[0].firstClipNode)
	}

	if cvars.GlZFix.Bool() {
		e.Origin.Sub(vec.Vec3{DIST_EPSILON, DIST_EPSILON, DIST_EPSILON})
	}

	modelview := view.modelView.Copy()
	modelview.Translate(e.Origin[0], e.Origin[1], e.Origin[2])
	modelview.RotateZ(e.Angles[1])
	// stupid quake bug, it should be -angles[0]
	modelview.RotateY(e.Angles[0])
	modelview.RotateX(e.Angles[2])

	if cvars.GlZFix.Bool() {
		e.Origin.Add(vec.Vec3{DIST_EPSILON, DIST_EPSILON, DIST_EPSILON})
	}

	for _, t := range cl.worldModel.Textures {
		if t != nil {
			t.TextureChains[chainModel] = nil
		}
	}
	// ...
	/*
		for _, n := range cl.worldModel.Nodes {
			for _, s := range n.Surfaces {
				if s.VisFrame == renderer.visFrameCount {
					s.TextureChain = s.TexInfo.Texture.TextureChains[chainWorld]
					s.TexInfo.Texture.TextureChains[chainWorld] = s
				}
			}
		}*/
	// R_DrawTextureChains(model,e,chain_model)
	// R_DrawTextureChains_Water(model,e,chain_model)

	r.DrawBrushModelC(e)
}
