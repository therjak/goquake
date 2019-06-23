package quakelib

import "C"

func EntityAlpha(num int) byte {
	return EDICT_ALPHA(num)
}

//export SetSVLightStyles
func SetSVLightStyles(i C.int, c *C.char) {
	s := C.GoString(c)
	sv.lightStyles[int(i)] = s
}
