package quakelib

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
	g_entvars    unsafe.Pointer
	entvars      [][]int32
)

func AllocEntvars(numEdicts int, entityfields int) {
	entityFields = entityfields
	maxEdicts = numEdicts
	virtmem = make([]int32, maxEdicts*entityFields)
	g_entvars = unsafe.Pointer(&virtmem[0])
	entvars = make([][]int32, maxEdicts)
	for i := 0; i < maxEdicts; i++ {
		entvars[i] = virtmem[i*entityFields : (i+1)*entityFields]
	}
}

func FreeEntvars() {
	g_entvars = nil
	entvars = nil
	virtmem = nil
}

func ClearEntVars(idx int) {
	v := entvars[idx]
	for i := 0; i < len(v); i++ {
		v[i] = 0
	}
}

func EntVars(idx int) *progs.EntVars {
	return (*progs.EntVars)(unsafe.Pointer(&(entvars[idx][0])))
}

func EntVarsSprint(idx int, d progs.Def) string {
	v := uintptr(g_entvars)
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

func RawEntVarsI(idx, off int32) int32 {
	v := uintptr(g_entvars)
	vp := v + uintptr(int(idx)*entityFields*4) + uintptr(off*4)
	return *(*int32)(unsafe.Pointer(vp))
}

func SetRawEntVarsI(idx, off int32, value int32) {
	v := uintptr(g_entvars)
	vp := v + uintptr(int(idx)*entityFields*4) + uintptr(off*4)
	*(*int32)(unsafe.Pointer(vp)) = value
}

func Raw0EntVarsI(off int32) int32 {
	v := uintptr(g_entvars)
	vp := v + uintptr(off)
	return *(*int32)(unsafe.Pointer(vp))
}

func Set0RawEntVarsI(off int32, value int32) {
	v := uintptr(g_entvars)
	vp := v + uintptr(off)
	*(*int32)(unsafe.Pointer(vp)) = value
}

func RawEntVarsF(idx, off int32) float32 {
	v := uintptr(g_entvars)
	vp := v + uintptr(int(idx)*entityFields*4) + uintptr(off*4)
	return *(*float32)(unsafe.Pointer(vp))
}

func SetRawEntVarsF(idx, off int32, value float32) {
	v := uintptr(g_entvars)
	vp := v + uintptr(int(idx)*entityFields*4) + uintptr(off*4)
	*(*float32)(unsafe.Pointer(vp)) = value
}

func Raw0EntVarsF(off int32) float32 {
	v := uintptr(g_entvars)
	vp := v + uintptr(off)
	return *(*float32)(unsafe.Pointer(vp))
}

func Set0RawEntVarsF(off int32, value float32) {
	v := uintptr(g_entvars)
	vp := v + uintptr(off)
	*(*float32)(unsafe.Pointer(vp)) = value
}

func EntVarsFieldValue(idx int, name string) (float32, error) {
	// Orig returns the union 'eval_t' but afterwards it is always a float32
	d, err := progsdat.FindFieldDef(name)
	if err != nil {
		return 0, err
	}
	v := uintptr(g_entvars)
	vp := v + uintptr(idx*entityFields*4) + uintptr(d.Offset*4)
	return *(*float32)(unsafe.Pointer(vp)), nil
}

func ClearEdict(e int) {
	ent := edictNum(e)
	*ent = Edict{}
	ClearEntVars(e)
}

func EntVarsParsePair(e int, key progs.Def, val string) {
	// edict number, key, value
	// Def{Type, Offset, uint16, SName int32}
	v := uintptr(g_entvars)
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
