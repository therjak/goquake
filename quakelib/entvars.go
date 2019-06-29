package quakelib

//#include <stdlib.h>
//#include "q_stdinc.h"
//#include "progdefs.h"
import "C"
import (
	"fmt"
	"quake/conlog"
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
func AllocEntvars(edicts int, entityfields int) {
	entityFields = entityfields
	maxEdicts = edicts
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
		s, err := progsdat.String(v)
		if err != nil {
			return fmt.Sprintf("bad string %d", v)
		}
		return s
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
		s, err := progsdat.String(f)
		if err != nil {
			return fmt.Sprintf("bad function %d", v)
		}
		return fmt.Sprintf("%s()", s)
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

func EntVarsFieldValue(idx int, name string) (float32, error) {
	// Orig returns the union 'eval_t' but afterwards it is always a float32
	d, err := progsdat.FindFieldDef(name)
	if err != nil {
		return 0, err
	}
	v := uintptr(unsafe.Pointer(g_entvars))
	vp := v + uintptr(idx*entityFields*4) + uintptr(d.Offset*4)
	return *(*float32)(unsafe.Pointer(vp)), nil
}

func TT_ClearEntVars(e *C.entvars_t) {
	C.memset(unsafe.Pointer(e), 0, C.ulong(entityFields*4))
}

//export TTClearEntVars
func TTClearEntVars(idx int) {
	ev := EVars(C.int(idx))
	TT_ClearEntVars(ev)
}

//export TT_ClearEdict
func TT_ClearEdict(e int) {
	ClearEdict(e)
}

func ClearEdict(e int) {
	ent := edictNum(e)
	*ent = Edict{}
	TTClearEntVars(e)
}

func EntVarsParsePair(e int, key progs.Def, val string) {
	// edict number, key, value
	// Def{Type, Offset, uint16, SName int32}
	v := uintptr(unsafe.Pointer(g_entvars))
	vp := v + uintptr(e*entityFields*4) + uintptr(key.Offset*4)
	p := unsafe.Pointer(vp)
	switch key.Type &^ (1 << 15) {
	case progs.EV_String:
		*(*int32)(p) = progsdat.NewString(val)
	case progs.EV_Float:
		var v float32
		_, err := fmt.Sscanf(val, "%f", &v)
		if err != nil {
			conlog.Printf("Can't convert to float32 %s\n", val)
		}
		*(*float32)(p) = v
	case progs.EV_Vector:
		var v [3]float32
		_, err := fmt.Sscanf(val, "%f %f %f", &v[0], &v[1], &v[2])
		if err != nil {
			conlog.Printf("Can't convert to [3]float32 %s\n", val)
		}
		*(*[3]float32)(p) = v
	case progs.EV_Entity:
		var v int32
		_, err := fmt.Sscanf(val, "%d", &v)
		if err != nil {
			conlog.Printf("Can't convert to entity %s\n", val)
			return
		}
		*(*int32)(p) = int32(v)
	case progs.EV_Field:
		d, err := progsdat.FindFieldDef(val)
		if err != nil {
			conlog.Printf("Can't find field %s\n", val)
			return
		}
		*(*int32)(p) = progsdat.RawGlobalsI[d.Offset]
	case progs.EV_Function:
		idx, err := progsdat.FindFunction(val)
		if err != nil {
			conlog.Printf("Can't find function %s\n", val)
			return
		}
		*(*int32)(p) = int32(idx)
	default:
	}
}

// progs.EntVars
/*
b := []byte{239, 190, 173, 222}
v := *(*uint32)(unsafe.Pointer(&b[0]))
fmt.Printf("0x%X\n", v)
fmt.Printf("%v", *(*[4]byte)(unsafe.Pointer(&v)))
*/
