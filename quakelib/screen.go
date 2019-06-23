package quakelib

//#include <stdlib.h>
// int SCR_ModalMessage(const char *text, float timeout);
// void SCR_BeginLoadingPlaque(void);
// void SCR_EndLoadingPlaque(void);
import "C"

import (
	"unsafe"
)

func ModalMessage(message string, timeout float32) bool {
	m := C.CString(message)
	defer C.free(unsafe.Pointer(m))
	return C.SCR_ModalMessage(m, C.float(timeout)) != 0
}

func SCR_BeginLoadingPlaque() {
	C.SCR_BeginLoadingPlaque()
}

func SCR_EndLoadingPlaque() {
	C.SCR_EndLoadingPlaque()
}
