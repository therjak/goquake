package quakelib

import "C"

import (
	"github.com/veandco/go-sdl2/sdl"
	"unsafe"
)

//export GO_GL_GetProcAddress
func GO_GL_GetProcAddress(name *C.char) unsafe.Pointer {
	n := C.GoString(name)
	return sdl.GLGetProcAddress(n)
}
