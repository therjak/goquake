package quakelib

import "C"

//export SetSVLightStyles
func SetSVLightStyles(i C.int, c *C.char) {
	s := C.GoString(c)
	sv.lightStyles[int(i)] = s
}
