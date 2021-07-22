// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

// This needs test and should probably be a separate package
// It needs a progs.LoadedProg as input
import (
	"fmt"
	"unsafe"

	"goquake/conlog"
	"goquake/progs"
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
	vp := &(entvars[idx][d.Offset])
	switch d.Type {
	case progs.EV_Void:
		return "void"
	case progs.EV_String:
		v := *vp
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
		v := *vp
		return fmt.Sprintf("entity %d", v)
	case progs.EV_Field:
		// TODO:
		return "field"
	case progs.EV_Function:
		v := *vp
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
	return (entvars[idx][off])
}

func SetRawEntVarsI(idx, off int32, value int32) {
	entvars[idx][off] = value
}

func getUnsafe(off int32) unsafe.Pointer {
	return unsafe.Pointer(uintptr(g_entvars) + uintptr(off))
}

func Set0RawEntVarsI(off int32, value int32) {
	*(*int32)(getUnsafe(off)) = value
}

func Set0RawEntVarsF(off int32, value float32) {
	*(*float32)(getUnsafe(off)) = value
}

func RawEntVarsF(idx, off int32) float32 {
	return *(*float32)(unsafe.Pointer(&(entvars[idx][off])))
}

func SetRawEntVarsF(idx, off int32, value float32) {
	*(*float32)(unsafe.Pointer(&(entvars[idx][off]))) = value
}

func EntVarsFieldValue(idx int, name string) (float32, error) {
	// Orig returns the union 'eval_t' but afterwards it is always a float32
	d, err := progsdat.FindFieldDef(name)
	if err != nil {
		return 0, err
	}
	return *(*float32)(unsafe.Pointer(&(entvars[idx][d.Offset]))), nil
}

func EntVarsParsePair(idx int, key progs.Def, val string) {
	// edict number, key, value
	// Def{Type, Offset, uint16, SName int32}
	vp := &(entvars[idx][key.Offset])
	switch key.Type &^ (1 << 15) {
	case progs.EV_String:
		*vp = progsdat.NewString(val)
	case progs.EV_Float:
		var v float32
		_, err := fmt.Sscanf(val, "%f", &v)
		if err != nil {
			conlog.Printf("Can't convert to float32 %s\n", val)
		}
		*(*float32)(unsafe.Pointer(vp)) = v
	case progs.EV_Vector:
		var v [3]float32
		_, err := fmt.Sscanf(val, "%f %f %f", &v[0], &v[1], &v[2])
		if err != nil {
			conlog.Printf("Can't convert to [3]float32 %s\n", val)
		}
		*(*[3]float32)(unsafe.Pointer(vp)) = v
	case progs.EV_Entity:
		var v int32
		_, err := fmt.Sscanf(val, "%d", &v)
		if err != nil {
			conlog.Printf("Can't convert to entity %s\n", val)
			return
		}
		*vp = v
	case progs.EV_Field:
		d, err := progsdat.FindFieldDef(val)
		if err != nil {
			conlog.Printf("Can't find field %s\n", val)
			return
		}
		*vp = progsdat.RawGlobalsI[d.Offset]
	case progs.EV_Function:
		f, err := progsdat.FindFunction(val)
		if err != nil {
			conlog.Printf("Can't find function %s\n", val)
			return
		}
		*vp = int32(f)
	default:
	}
}
