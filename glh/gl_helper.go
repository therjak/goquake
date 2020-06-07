package glh

import (
	"fmt"
	"github.com/faiface/mainthread"
	"github.com/go-gl/gl/v4.6-core/gl"
	"runtime"
	"strings"
)

type Program struct {
	prog uint32
}

func NewProgram(vertex, fragment string) (*Program, error) {
	p := &Program{
		prog: gl.CreateProgram(),
	}
	vert, err := GetShader(vertex, gl.VERTEX_SHADER)
	if err != nil {
		return nil, err
	}
	frag, err := GetShader(fragment, gl.FRAGMENT_SHADER)
	if err != nil {
		return nil, err
	}
	gl.AttachShader(p.prog, vert)
	gl.AttachShader(p.prog, frag)
	gl.LinkProgram(p.prog)
	gl.DeleteShader(vert)
	gl.DeleteShader(frag)
	runtime.SetFinalizer(p, (*Program).delete)
	return p, nil
}

func (p *Program) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteProgram(p.prog)
	})
}

func (p *Program) Use() {
	gl.UseProgram(p.prog)
}

func (p *Program) GetAttribLocation(n string) uint32 {
	return uint32(gl.GetAttribLocation(p.prog, gl.Str(n+"\x00")))
}

func (p *Program) GetUniformLocation(n string) int32 {
	return gl.GetUniformLocation(p.prog, gl.Str(n+"\x00"))
}

type Buffer struct {
	buf uint32
}

func NewBuffer() *Buffer {
	b := &Buffer{}
	gl.GenBuffers(1, &b.buf)
	runtime.SetFinalizer(b, (*Buffer).delete)
	return b
}

func (b *Buffer) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteBuffers(1, &b.buf)
	})
}

func (b *Buffer) Bind(target uint32) {
	gl.BindBuffer(target, b.buf)
}

type VertexArray struct {
	a uint32
}

func NewVertexArray() *VertexArray {
	va := &VertexArray{}
	gl.GenVertexArrays(1, &va.a)
	runtime.SetFinalizer(va, (*VertexArray).delete)
	return va
}

func (va *VertexArray) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteVertexArrays(1, &va.a)
	})
}

func (va *VertexArray) Bind() {
	gl.BindVertexArray(va.a)
}

type Texture struct {
	id uint32
}

type TexID uint32

func (t *Texture) ID() TexID {
	return TexID(t.id)
}

func NewTexture() *Texture {
	t := &Texture{}
	gl.GenTextures(1, &t.id)
	runtime.SetFinalizer(t, (*Texture).delete)
	return t
}

func (t *Texture) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteTextures(1, &t.id)
	})
}

func (t *Texture) Bind() {
	gl.BindTexture(gl.TEXTURE_2D, t.id)
}

func GetShader(src string, shaderType uint32) (uint32, error) {
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
		return 0, fmt.Errorf("Failed to compile shader: %v", log)
	}
	return shader, nil
}
