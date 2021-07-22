// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"
import (
	"goquake/cbuf"
)

//export Cbuf_Execute
func Cbuf_Execute() {
	cbuf.Execute(0)
}

//export Cbuf_AddText
func Cbuf_AddText(text *C.char) {
	cbuf.AddText(C.GoString(text))
}

//export Cbuf_InsertText
func Cbuf_InsertText(text *C.char) {
	cbuf.InsertText(C.GoString(text))
}
