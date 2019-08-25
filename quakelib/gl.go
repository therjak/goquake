package quakelib

import "C"

import (
	"github.com/veandco/go-sdl2/sdl"
	"unsafe"
)

type glrect struct {
	x      int32
	y      int32
	height int32
	width  int32
}

var (
	viewport glrect
)

//export GO_GL_GetProcAddress
func GO_GL_GetProcAddress(name *C.char) unsafe.Pointer {
	n := C.GoString(name)
	return sdl.GLGetProcAddress(n)
}

//export GL_Height
func GL_Height() int32 {
	return viewport.height
}

//export GL_Width
func GL_Width() int32 {
	return viewport.width
}

//export GL_X
func GL_X() int32 {
	return viewport.x
}

//export GL_Y
func GL_Y() int32 {
	return viewport.y
}

//export UpdateViewport
func UpdateViewport() {
	viewport.x = 0
	viewport.y = 0
	viewport.width = int32(screenWidth)
	viewport.height = int32(screenHeight)
}
