package quakelib

//#include <stdlib.h>
//#include "q_stdinc.h"
//#include "progdefs.h"
import "C"
import (
	"quake/progs"
	"unsafe"
)

var (
	virtmem      []int32
	entityFields int
	maxEdicts    int
	g_entvars    *C.entvars_t
)

//export AllocEntvars
func AllocEntvars(edicts C.int, entityfields C.int) {
	entityFields = int(entityfields)
	maxEdicts = int(edicts)
	// virtmem = make([]int32, maxEdicts*entityFields)
	v := C.malloc(C.ulong(edicts * entityfields * 4))
	g_entvars = (*C.entvars_t)(v)
}

//export FreeEntvars
func FreeEntvars() {
	C.free(unsafe.Pointer(g_entvars))
}

//export EVars
func EVars(idx C.int) *C.entvars_t {
	v := uintptr(unsafe.Pointer(g_entvars))
	vp := v + uintptr(idx*C.int(entityFields)*4)
	return (*C.entvars_t)(unsafe.Pointer(vp))
	//return (*C.entvars_t)(unsafe.Pointer(&virtmem[int(idx)*entityFields]))
}

func EntVars(idx int) *progs.EntVars {
	v := uintptr(unsafe.Pointer(g_entvars))
	vp := v + uintptr(idx*entityFields*4)
	return (*progs.EntVars)(unsafe.Pointer(vp))
	//return (*progs.EntVars)(unsafe.Pointer(&virtmem[int(idx)*entityFields]))
}

//export TT_ClearEntVars
func TT_ClearEntVars(e *C.entvars_t) {
	C.memset(unsafe.Pointer(e), 0, C.ulong(entityFields*4))
}

// progs.EntVars
/*
b := []byte{239, 190, 173, 222}
v := *(*uint32)(unsafe.Pointer(&b[0]))
fmt.Printf("0x%X\n", v)
fmt.Printf("%v", *(*[4]byte)(unsafe.Pointer(&v)))
*/