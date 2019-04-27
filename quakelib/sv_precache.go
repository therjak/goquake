package quakelib

// unsigned char EAlpha(int num);
import "C"

import (
	"log"
)

// sv.modelPrecache
// sv.soundPrecache
// sv.lightStyles

func EntityAlpha(num int) byte {
	return byte(C.EAlpha(C.int(num)))
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

//export ElementOfSVModelPrecache
func ElementOfSVModelPrecache(c *C.char) C.int {
	s := C.GoString(c)
	for i, m := range sv.modelPrecache {
		if m == s {
			return C.int(i)
		}
	}
	return -1
}

//export ExistSVModelPrecache
func ExistSVModelPrecache(i C.int) C.int {
	if int(i) >= len(sv.modelPrecache) {
		return 0
	}
	return 1
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

//export ElementOfSVSoundPrecache
func ElementOfSVSoundPrecache(c *C.char) C.int {
	s := C.GoString(c)
	for i, m := range sv.soundPrecache {
		if m == s {
			return C.int(i)
		}
	}
	return -1
}

//export ExistSVSoundPrecache
func ExistSVSoundPrecache(i C.int) C.int {
	if int(i) >= len(sv.soundPrecache) {
		return 0
	}
	return 1
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
