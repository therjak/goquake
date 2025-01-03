// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"goquake/glh"
	"goquake/math"

	"github.com/go-gl/gl/v4.6-core/gl"
)

type postProcess struct {
	vao      *glh.VertexArray
	vbo      *glh.Buffer
	ebo      *glh.Buffer
	prog     *glh.Program
	gamma    int32
	contrast int32

	// just used to check for changes as a change requires a new texture
	width  int32
	height int32

	texture glh.Texture
}

var pprocess *postProcess

func CreatePostProcess() {
	pprocess = newPostProcessor()
}

func newPostProcessor() *postProcess {
	p := &postProcess{}
	elements := []uint32{
		0, 1, 2,
		2, 3, 0,
	}
	vertices := []float32{
		// vertex, tex
		-1, -1, 0, 0,
		1, -1, 1, 0,
		1, 1, 1, 1,
		-1, 1, 0, 1,
	}
	p.vao = glh.NewVertexArray()
	p.vbo = glh.NewBuffer(glh.ArrayBuffer)
	p.vbo.Bind()
	p.vbo.SetData(4*len(vertices), gl.Ptr(vertices))
	p.ebo = glh.NewBuffer(glh.ElementArrayBuffer)
	p.ebo.Bind()
	p.ebo.SetData(4*len(elements), gl.Ptr(elements))
	var err error
	p.prog, err = glh.NewProgram(vertexTextureSource, postProcessFragment)
	if err != nil {
		Error(err.Error())
	}
	p.gamma = p.prog.GetUniformLocation("gamma")
	p.contrast = p.prog.GetUniformLocation("contrast")
	return p
}

func (p *postProcess) Draw(gamma, contrast float32) {
	var viewport [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])

	if p.texture == nil || p.width != viewport[2] || p.height != viewport[3] {
		p.texture = glh.NewTexture2D()
		p.texture.Bind()
		p.width = viewport[2]
		p.height = viewport[3]
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, p.width, p.height,
			0, gl.BGRA, gl.UNSIGNED_INT_8_8_8_8_REV, nil)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	}

	textureManager.DisableMultiTexture()
	p.texture.Bind()
	gl.CopyTexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, 0, 0, p.width, p.height)

	p.prog.Use()
	p.vao.Bind()
	p.ebo.Bind()
	p.vbo.Bind()
	gl.EnableVertexAttribArray(0)
	defer gl.DisableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1)
	defer gl.DisableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, 4*4, 0)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 4*4, 2*4)
	gl.Uniform1f(p.gamma, gamma)
	gl.Uniform1f(p.contrast, contrast)
	gl.Disable(gl.DEPTH_TEST)

	gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(0))

	// We bound a texture without the texture manager.
	// Tell the texture manager that its cache is invalid.
	textureManager.ClearBindings()
}

func postProcessGammaContrast(gamma, contrast float32) {
	contrast = math.Clamp(1, contrast, 2)
	pprocess.Draw(gamma, contrast)
}
