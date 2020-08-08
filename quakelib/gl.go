package quakelib

import "C"

//export GL_Height
func GL_Height() int {
	return screen.Height
}

//export GL_Width
func GL_Width() int {
	return screen.Width
}
