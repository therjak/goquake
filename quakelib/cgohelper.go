package quakelib

import "C"

func b2i(b bool) C.int {
	if b {
		return 1
	}
	return 0
}
