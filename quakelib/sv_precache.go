package quakelib

// const char* PR_GetString(int num);
import "C"

import (
	"log"
)

// sv.modelPrecache
// sv.soundPrecache
// sv.lightStyles

func PR_GetStringWrap(num int) string {
	c := C.PR_GetString(C.int(num))
	return C.GoString(c)
}

//export SetSVModelPrecache
func SetSVModelPrecache(i C.int, c *C.char) {
	s := C.GoString(c)
	if int(i) == len(sv.modelPrecache) {
		sv.modelPrecache = append(sv.modelPrecache, s)
	} else if int(i) > len(sv.modelPrecache) {
		log.Printf("WTF: SetSVModelPrecache")
	} else {
		sv.modelPrecache[int(i)] = s
	}
}

//export SetSVSoundPrecache
func SetSVSoundPrecache(i C.int, c *C.char) {
	s := C.GoString(c)
	if int(i) == len(sv.soundPrecache) {
		sv.soundPrecache = append(sv.soundPrecache, s)
	} else if int(i) > len(sv.soundPrecache) {
		log.Printf("WTF: SetSVSoundPrecache")
	} else {
		sv.soundPrecache[int(i)] = s
	}
}

//export SetSVLightStyles
func SetSVLightStyles(i C.int, c *C.char) {
	s := C.GoString(c)
	if int(i) == len(sv.lightStyles) {
		sv.lightStyles = append(sv.lightStyles, s)
	} else if int(i) > len(sv.lightStyles) {
		log.Printf("WTF: SetSVLightStyles")
	} else {
		sv.lightStyles[int(i)] = s
	}
}
