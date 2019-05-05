package quakelib

//#include <stdlib.h>
//#include "q_stdinc.h"
//#include "progdefs.h"
import "C"
import (
	"fmt"
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
	g_entvars = nil
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

func EntVarsSprint(idx int, d progs.Def) string {
	v := uintptr(unsafe.Pointer(g_entvars))
	vp := v + uintptr(idx*entityFields*4) + uintptr(int(d.Offset)*4)
	// return *(*int32)(unsafe.Pointer(vp))
	switch d.Type {
	case progs.EV_Void:
		return "void"
	case progs.EV_String:
		v := *(*int32)(unsafe.Pointer(vp))
		s := PRGetString(int(v))
		if s == nil {
			return fmt.Sprintf("bad string %d", v)
		}
		return *s
	case progs.EV_Float:
		v := *(*float32)(unsafe.Pointer(vp))
		return fmt.Sprintf("%5.1f", v)
	case progs.EV_Vector:
		v := *(*[3]float32)(unsafe.Pointer(vp))
		return fmt.Sprintf("%5.1f %5.1f %5.1f", v[0], v[1], v[2])
	case progs.EV_Entity:
		v := *(*int32)(unsafe.Pointer(vp))
		return fmt.Sprintf("entity %d", v)
	case progs.EV_Field:
		// TODO:
		return "field"
	case progs.EV_Function:
		v := *(*int32)(unsafe.Pointer(vp))
		f := progsdat.Functions[int(v)].SName
		s := PRGetString(int(f))
		if s == nil {
			return fmt.Sprintf("bad function %d", v)
		}
		return fmt.Sprintf("%s()", *s)
	case progs.EV_Pointer:
		return "pointer"
	default: // also EV_Bad
		return fmt.Sprintf("bad type %d", d.Type)
	}
}

func RawEntVarsI(idx, off int) int32 {
	v := uintptr(unsafe.Pointer(g_entvars))
	vp := v + uintptr(idx*entityFields*4) + uintptr(off*4)
	return *(*int32)(unsafe.Pointer(vp))
}

//export TT_ClearEntVars
func TT_ClearEntVars(e *C.entvars_t) {
	C.memset(unsafe.Pointer(e), 0, C.ulong(entityFields*4))
}
func TTClearEntVars(idx int) {
	ev := EVars(C.int(idx))
	TT_ClearEntVars(ev)
}

// progs.EntVars
/*
b := []byte{239, 190, 173, 222}
v := *(*uint32)(unsafe.Pointer(&b[0]))
fmt.Printf("0x%X\n", v)
fmt.Printf("%v", *(*[4]byte)(unsafe.Pointer(&v)))
*/
