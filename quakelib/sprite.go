package quakelib

import (
	"log"

	"github.com/chewxy/math32"
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/therjak/goquake/glh"
	"github.com/therjak/goquake/math/vec"
	"github.com/therjak/goquake/spr"
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

func NewSpriteDrawer() *qSpriteDrawer {
	d := &qSpriteDrawer{}
	elements := []uint32{
		0, 1, 2,
		2, 3, 0,
	}
	d.vao = glh.NewVertexArray()
	d.vbo = glh.NewBuffer()
	d.ebo = glh.NewBuffer()
	d.ebo.Bind(gl.ELEMENT_ARRAY_BUFFER)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 4*len(elements), gl.Ptr(elements), gl.STATIC_DRAW)
	var err error
	d.prog, err = newSpriteDrawProgram()
	if err != nil {
		Error(err.Error())
	}

	d.projection = d.prog.GetUniformLocation("projection")
	d.modelview = d.prog.GetUniformLocation("modelview")
	return d
}

var (
	spriteDrawer *qSpriteDrawer
)

func (r *qRenderer) DrawSpriteModel(e *Entity, m *spr.Model) {
	const piDiv180 = math32.Pi / 180

	if spriteDrawer == nil {
		spriteDrawer = NewSpriteDrawer()
	}

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

	textureManager.DisableMultiTexture()
	textureManager.Bind(frame.Texture)

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

	// TODO: why do the model/view stuff outside the shader?
	projection := [16]float32{}
	gl.GetFloatv(0x0BA7, &projection[0])
	modelview := [16]float32{}
	gl.GetFloatv(0x0BA6, &modelview[0])

	spriteDrawer.prog.Use()
	spriteDrawer.vao.Bind()
	spriteDrawer.ebo.Bind(gl.ELEMENT_ARRAY_BUFFER)
	spriteDrawer.vbo.Bind(gl.ARRAY_BUFFER)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices), gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	defer gl.DisableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1)
	defer gl.DisableVertexAttribArray(1)

	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 4*5, gl.PtrOffset(0))
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 4*5, gl.PtrOffset(3*4))

	gl.UniformMatrix4fv(skyDrawer.projection, 1, false, &projection[0])
	gl.UniformMatrix4fv(skyDrawer.modelview, 1, false, &modelview[0])

	gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(0))
}
