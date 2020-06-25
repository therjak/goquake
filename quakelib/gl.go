package quakelib

// void GLSLGamma_GammaCorrect(void);
import "C"

import (
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

type glrect struct {
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

//export UpdateViewport
func UpdateViewport() {
	viewport.width = int32(screen.Width)
	viewport.height = int32(screen.Height)
}

func GLSLGamma_GammaCorrect() {
	C.GLSLGamma_GammaCorrect()
}
