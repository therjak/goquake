package quakelib

//#include <stdlib.h>
// int SCR_ModalMessage(const char *text, float timeout);
import "C"

import (
	"unsafe"
)

func ModalMessage(message string, timeout float32) bool {
	m := C.CString(message)
	defer C.free(unsafe.Pointer(m))
	return C.SCR_ModalMessage(m, C.float(timeout)) != 0
}
