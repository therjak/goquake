package quakelib

import "C"

//export Sbar_Changed
func Sbar_Changed() {
	statusbar.MarkChanged()
}

//export Sbar_Init
func Sbar_Init() {
	statusbar.LoadPictures()
}

//export Sbar_Lines
func Sbar_Lines() int {
	return statusbar.Lines()
}
