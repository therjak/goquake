// SPDX-License-Identifier: GPL-2.0-or-later

package progs

import (
	"fmt"
	"strings"
	"unsafe"

	"goquake/conlog"
)

// TODO: rename to Vars?
type EntityVars struct {
	virtmem      []int32
	entityFields int
	maxEdicts    int
	g_entvars    unsafe.Pointer
	entvars      [][]int32
	progsdat     *LoadedProg
}

// TODO: change to (*LoadedProg) AllocVars(numEdicts int) *Vars
func AllocEntvars(numEdicts int, entityfields int, pg *LoadedProg) *EntityVars {
	ev := &EntityVars{}
	ev.entityFields = entityfields
	ev.maxEdicts = numEdicts
	ev.virtmem = make([]int32, ev.maxEdicts*ev.entityFields)
	ev.g_entvars = unsafe.Pointer(&ev.virtmem[0])
	ev.entvars = make([][]int32, ev.maxEdicts)
	for i := 0; i < ev.maxEdicts; i++ {
		ev.entvars[i] = ev.virtmem[i*ev.entityFields : (i+1)*ev.entityFields]
	}
	ev.progsdat = pg
	return ev
}

func (e *EntityVars) Address(idx, off int32) int32 {
	return idx*int32(e.entityFields)*4 + off*4
}

func (e *EntityVars) Free() {
	if e == nil {
		return
	}
	e.g_entvars = nil
	e.entvars = nil
	e.virtmem = nil
}

func (e *EntityVars) Clear(idx int) {
	v := e.entvars[idx]
	for i := 0; i < len(v); i++ {
		v[i] = 0
	}
}

func (e *EntityVars) Get(idx int) *EntVars {
	return (*EntVars)(unsafe.Pointer(&(e.entvars[idx][0])))
}

func (e *EntityVars) Sprint(idx int, d Def) string {
	vp := &(e.entvars[idx][d.Offset])
	switch d.Type {
	case EV_Void:
		return "void"
	case EV_String:
		v := *vp
		s, err := e.progsdat.String(v)
		if err != nil {
			return fmt.Sprintf("bad string %d", v)
		}
		return s
	case EV_Float:
		v := *(*float32)(unsafe.Pointer(vp))
		return fmt.Sprintf("%5.1f", v)
	case EV_Vector:
		v := *(*[3]float32)(unsafe.Pointer(vp))
		return fmt.Sprintf("%5.1f %5.1f %5.1f", v[0], v[1], v[2])
	case EV_Entity:
		v := *vp
		return fmt.Sprintf("entity %d", v)
	case EV_Field:
		// TODO:
		return "field"
	case EV_Function:
		v := *vp
		f := e.progsdat.Functions[int(v)].SName
		s, err := e.progsdat.String(f)
		if err != nil {
			return fmt.Sprintf("bad function %d", v)
		}
		return fmt.Sprintf("%s()", s)
	case EV_Pointer:
		return "pointer"
	default: // also EV_Bad
		return fmt.Sprintf("bad type %d", d.Type)
	}
}

func (e *EntityVars) RawI(idx, off int32) int32 {
	return (e.entvars[idx][off])
}

func (e *EntityVars) SetRawI(idx, off int32, value int32) {
	e.entvars[idx][off] = value
}

func (e *EntityVars) getUnsafe(off int32) unsafe.Pointer {
	// go 1.17:
	// return unsafe.Add(g_entvars, off)
	return unsafe.Pointer(uintptr(e.g_entvars) + uintptr(off))
}

func (e *EntityVars) Set0RawI(off int32, value int32) {
	*(*int32)(e.getUnsafe(off)) = value
}

func (e *EntityVars) Set0RawF(off int32, value float32) {
	*(*float32)(e.getUnsafe(off)) = value
}

func (e *EntityVars) RawF(idx, off int32) float32 {
	return *(*float32)(unsafe.Pointer(&(e.entvars[idx][off])))
}

func (e *EntityVars) SetRawF(idx, off int32, value float32) {
	*(*float32)(unsafe.Pointer(&(e.entvars[idx][off]))) = value
}

func (e *EntityVars) FieldValue(idx int, name string) (float32, error) {
	// Orig returns the union 'eval_t' but afterwards it is always a float32
	d, err := e.progsdat.FindFieldDef(name)
	if err != nil {
		return 0, err
	}
	return *(*float32)(unsafe.Pointer(&(e.entvars[idx][d.Offset]))), nil
}

func (e *EntityVars) ParsePair(idx int, key Def, val string) {
	// edict number, key, value
	// Def{Type, Offset, uint16, SName int32}
	vp := &(e.entvars[idx][key.Offset])
	switch key.Type &^ (1 << 15) {
	case EV_String:
		*vp = e.progsdat.NewString(val)
	case EV_Float:
		var v float32
		_, err := fmt.Sscanf(val, "%f", &v)
		if err != nil {
			conlog.Printf("Can't convert to float32 %s\n", val)
		}
		*(*float32)(unsafe.Pointer(vp)) = v
	case EV_Vector:
		var v [3]float32
		n, err := fmt.Sscanf(val, "%f %f %f", &v[0], &v[1], &v[2])
		if err != nil {
			conlog.Printf("Can't convert to [3]float32 %s\n", val)
		}
		for ; n < 3; n++ {
			v[n] = 0
		}
		*(*[3]float32)(unsafe.Pointer(vp)) = v
	case EV_Entity:
		var v int32
		val = strings.TrimPrefix(val, "entity ") // fix for eto
		_, err := fmt.Sscanf(val, "%d", &v)
		if err != nil {
			conlog.Printf("Can't convert to entity %s\n", val)
			return
		}
		*vp = v
	case EV_Field:
		d, err := e.progsdat.FindFieldDef(val)
		if err != nil {
			conlog.Printf("Can't find field %s\n", val)
			return
		}
		*vp = e.progsdat.RawGlobalsI[d.Offset]
	case EV_Function:
		f, err := e.progsdat.FindFunction(val)
		if err != nil {
			conlog.Printf("Can't find function %s\n", val)
			return
		}
		*vp = int32(f)
	default:
	}
}
