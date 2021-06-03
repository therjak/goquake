// SPDX-License-Identifier: GPL-2.0-or-later

package glh

import (
	"fmt"
	"runtime"
	"strings"
	"unsafe"

	"github.com/faiface/mainthread"
	"github.com/go-gl/gl/v4.6-core/gl"
)

const (
	ArrayBuffer        = gl.ARRAY_BUFFER
	ElementArrayBuffer = gl.ELEMENT_ARRAY_BUFFER
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

func NewProgramWithGeometry(vertex, geometry, fragment string) (*Program, error) {
	p := &Program{
		prog: gl.CreateProgram(),
	}
	vert, err := GetShader(vertex, gl.VERTEX_SHADER)
	if err != nil {
		return nil, err
	}
	geo, err := GetShader(geometry, gl.GEOMETRY_SHADER)
	if err != nil {
		return nil, err
	}
	frag, err := GetShader(fragment, gl.FRAGMENT_SHADER)
	if err != nil {
		return nil, err
	}
	gl.AttachShader(p.prog, vert)
	gl.AttachShader(p.prog, geo)
	gl.AttachShader(p.prog, frag)
	gl.LinkProgram(p.prog)
	gl.DeleteShader(vert)
	gl.DeleteShader(geo)
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
	buf    uint32
	target uint32
}

func NewBuffer(target uint32) *Buffer {
	b := &Buffer{
		target: target,
	}
	gl.GenBuffers(1, &b.buf)
	runtime.SetFinalizer(b, (*Buffer).delete)
	return b
}

func (b *Buffer) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteBuffers(1, &b.buf)
	})
}

func (b *Buffer) Bind() {
	gl.BindBuffer(b.target, b.buf)
}

// SetData sets the data for this buffer. It needs to be bound first.
func (b *Buffer) SetData(size int, data unsafe.Pointer) {
	// It would be nice to just call b.Bind() first.
	// But even in the effective noop case this is not free.
	gl.BufferData(b.target, size, data, gl.STATIC_DRAW)
}

func Ptr(data interface{}) unsafe.Pointer {
	return gl.Ptr(data)
}

func (b *Buffer) ID() uint32 {
	return b.buf
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
