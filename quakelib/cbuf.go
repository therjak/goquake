package quakelib

import "C"
import (
	"quake/cbuf"
)

//export Cbuf_Execute
func Cbuf_Execute() {
	cbuf.Execute(sv_player)
}

//export Cbuf_AddText
func Cbuf_AddText(text *C.char) {
	cbuf.AddText(C.GoString(text))
}

//export Cbuf_InsertText
func Cbuf_InsertText(text *C.char) {
	cbuf.InsertText(C.GoString(text))
}
