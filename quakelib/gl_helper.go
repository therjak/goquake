package quakelib

import (
	"github.com/faiface/mainthread"
	"github.com/go-gl/gl/v4.6-core/gl"
	"runtime"
	"strings"
)

type GlProgram struct {
	prog uint32
}

func newGlProgram(vertex, fragment string) *GlProgram {
	p := &GlProgram{
		prog: gl.CreateProgram(),
	}
	vert := getShader(vertex, gl.VERTEX_SHADER)
	frag := getShader(fragment, gl.FRAGMENT_SHADER)
	gl.AttachShader(p.prog, vert)
	gl.AttachShader(p.prog, frag)
	gl.LinkProgram(p.prog)
	gl.DeleteShader(vert)
	gl.DeleteShader(frag)
	runtime.SetFinalizer(p, (*GlProgram).delete)
	return p
}

func (p *GlProgram) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteProgram(p.prog)
	})
}

func (p *GlProgram) Use() {
	gl.UseProgram(p.prog)
}

func (p *GlProgram) GetAttribLocation(n string) uint32 {
	return uint32(gl.GetAttribLocation(p.prog, gl.Str(n+"\x00")))
}

func (p *GlProgram) GetUniformLocation(n string) int32 {
	return gl.GetUniformLocation(p.prog, gl.Str(n+"\x00"))
}

type GlBuffer struct {
	buf uint32
}

func newGlBuffer() *GlBuffer {
	b := &GlBuffer{}
	gl.GenBuffers(1, &b.buf)
	runtime.SetFinalizer(b, (*GlBuffer).delete)
	return b
}

func (b *GlBuffer) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteBuffers(1, &b.buf)
	})
}

func (b *GlBuffer) Bind(target uint32) {
	gl.BindBuffer(target, b.buf)
}

func getShader(src string, shaderType uint32) uint32 {
	shader := gl.CreateShader(shaderType)
	csource, free := gl.Strs(src)
	gl.ShaderSource(shader, 1, csource, nil)
	free()
	gl.CompileShader(shader)
	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		Error("Failed to compile shader: %v", log)
	}
	return shader
}
