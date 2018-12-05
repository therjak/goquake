package quakelib

import "C"

import (
	"github.com/veandco/go-sdl2/sdl"
)

//export PL_GetClipboardData
func PL_GetClipboardData() *C.char {
	txt, err := sdl.GetClipboardText()
	if err != nil {
		return nil
	}
	return C.CString(txt)
}
