// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"log"

	"goquake/glh"
	"goquake/math/vec"
	"goquake/spr"

	"github.com/chewxy/math32"
	"github.com/go-gl/gl/v4.6-core/gl"
)

func newSpriteDrawProgram() (*glh.Program, error) {
	return glh.NewProgram(vertexTextureSource2, fragmentSourceDrawer)
}

type qSpriteDrawer struct {
	vao        *glh.VertexArray
	vbo        *glh.Buffer
	ebo        *glh.Buffer
	prog       *glh.Program
	projection int32
	modelview  int32
}

func NewSpriteDrawer() (*qSpriteDrawer, error) {
	d := &qSpriteDrawer{}
	elements := []uint32{
		0, 1, 2,
		2, 3, 0,
	}
	d.vao = glh.NewVertexArray()
	d.vbo = glh.NewBuffer(glh.ArrayBuffer)
	d.ebo = glh.NewBuffer(glh.ElementArrayBuffer)
	d.ebo.Bind()
	d.ebo.SetData(4*len(elements), gl.Ptr(elements))
	var err error
	d.prog, err = newSpriteDrawProgram()
	if err != nil {
		return nil, err
	}

	d.projection = d.prog.GetUniformLocation("projection")
	d.modelview = d.prog.GetUniformLocation("modelview")
	return d, nil
}

var (
	spriteDrawer *qSpriteDrawer
)

func CreateSpriteDrawer() error {
	var err error
	spriteDrawer, err = NewSpriteDrawer()
	return err
}

func (r *qRenderer) DrawSpriteModel(e *Entity, m *spr.Model) {
	const piDiv180 = math32.Pi / 180

	t := float32(cl.time) + e.SyncBase
	cf := e.Frame
	if cf >= len(m.Data.Frames) || cf < 0 {
		log.Printf("R_DrawSprite: no such frame %v for '%s'", cf, m.Name())
		cf = 0
	}

	sprite := m.Data
	frame := sprite.Frames[cf].Frame(t)

	var sUp, sRight vec.Vec3
	switch sprite.Type {
	case spr.SPR_VP_PARALLEL_UPRIGHT:
		sUp = vec.Vec3{0, 0, 1}
		sRight = qRefreshRect.viewRight
	case spr.SPR_FACING_UPRIGHT:
		fwd := vec.Sub(e.Origin, qRefreshRect.viewOrg)
		fwd[2] = 0
		fwd = fwd.Normalize()
		sRight = vec.Vec3{
			fwd[1],
			-fwd[0],
			0,
		}
		sUp = vec.Vec3{0, 0, 1}
	case spr.SPR_VP_PARALLEL:
		sUp = qRefreshRect.viewUp
		sRight = qRefreshRect.viewRight
	case spr.SPR_ORIENTED:
		_, sRight, sUp = vec.AngleVectors(e.Angles)
	case spr.SPR_VP_PARALLEL_ORIENTED:
		angle := piDiv180 * e.Angles[ROLL]
		s, c := math32.Sincos(angle)
		sUp = vec.Add(vec.Scale(c, qRefreshRect.viewRight), vec.Scale(s, qRefreshRect.viewUp))
		sRight = vec.Add(vec.Scale(-s, qRefreshRect.viewRight), vec.Scale(c, qRefreshRect.viewUp))
	}

	if sprite.Type == spr.SPR_ORIENTED {
		gl.Enable(gl.POLYGON_OFFSET_FILL)
		defer gl.Disable(gl.POLYGON_OFFSET_FILL)
		gl.Enable(gl.POLYGON_OFFSET_LINE)
		defer gl.Disable(gl.POLYGON_OFFSET_LINE)
	}

	textureManager.BindUnit(frame.Texture, gl.TEXTURE0)

	p1 := vec.FMA(e.Origin, frame.Down, sUp)
	p1 = vec.FMA(p1, frame.Left, sRight)
	p2 := vec.FMA(e.Origin, frame.Up, sUp)
	p2 = vec.FMA(p2, frame.Left, sRight)
	p3 := vec.FMA(e.Origin, frame.Up, sUp)
	p3 = vec.FMA(p3, frame.Right, sRight)
	p4 := vec.FMA(e.Origin, frame.Down, sUp)
	p4 = vec.FMA(p4, frame.Right, sRight)

	vertices := []float32{
		p2[0], p2[1], p2[2], 0, 0,
		p3[0], p3[1], p3[2], 1, 0,
		p4[0], p4[1], p4[2], 1, 1,
		p1[0], p1[1], p1[2], 0, 1,
	}

	spriteDrawer.prog.Use()
	spriteDrawer.vao.Bind()
	spriteDrawer.ebo.Bind()
	spriteDrawer.vbo.Bind()
	spriteDrawer.vbo.SetData(4*len(vertices), gl.Ptr(vertices))

	gl.EnableVertexAttribArray(0)
	defer gl.DisableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1)
	defer gl.DisableVertexAttribArray(1)

	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 4*5, 0)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 4*5, 3*4)

	view.projection.SetAsUniform(spriteDrawer.projection)
	view.modelView.SetAsUniform(spriteDrawer.modelview)

	gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(0))
}
