package quakelib

import "C"

import (
	"log"
	"quake/math"
	"quake/model"
	"quake/progs"
)

func runError(format string, v ...interface{}) {
	// TODO: see PR_RunError
	conPrintf(format, v...)
}

/*
This is the only valid way to move an object without using the physics
of the world (setting velocity and waiting).  Directly changing origin
will not set internal links correctly, so clipping would be messed up.

This should be called when an object is spawned, and then only if it is
teleported.
*/
//export PF_setorigin
func PF_setorigin() {
	e := int(progsdat.RawGlobalsI[progs.OffsetParm0])
	ev := EntVars(e)
	ev.Origin = *progsdat.Globals.Parm1f()

	LinkEdict(e, false)
}

func setMinMaxSize(ev *progs.EntVars, min, max math.Vec3) {
	if min.X > max.X || min.Y > max.Y || min.Z > max.Z {
		runError("backwards mins/maxs")
	}
	ev.Mins = min.Array()
	ev.Maxs = max.Array()
	s := math.Sub(max, min)
	ev.Size = s.Array()
}

//export PF_setsize
func PF_setsize() {
	e := int(progsdat.RawGlobalsI[progs.OffsetParm0])
	min := math.VFromA(*progsdat.Globals.Parm1f())
	max := math.VFromA(*progsdat.Globals.Parm2f())
	setMinMaxSize(EntVars(e), min, max)
	LinkEdict(e, false)
}

//export PF_setmodel2
func PF_setmodel2() {

	e := int(progsdat.RawGlobalsI[progs.OffsetParm0])
	mi := progsdat.RawGlobalsI[progs.OffsetParm1]
	m := PR_GetStringWrap(int(mi))

	idx := -1
	for i, mp := range sv.modelPrecache {
		if mp == m {
			idx = i
			break
		}
	}
	if idx == -1 {
		runError("no precache: %s", m)
	}

	ev := EntVars(e)
	ev.Model = mi
	ev.ModelIndex = float32(idx)

	mod := sv.models[idx]
	if mod != nil {
		if mod.Type == model.ModBrush {
			log.Printf("ModBrush")
			log.Printf("mins: %v, maxs: %v", mod.ClipMins, mod.ClipMaxs)
			setMinMaxSize(ev, mod.ClipMins, mod.ClipMaxs)
		} else {
			log.Printf("!!!ModBrush")
			setMinMaxSize(ev, mod.Mins, mod.Maxs)
		}
	} else {
		log.Printf("No Mod")
		setMinMaxSize(ev, math.Vec3{}, math.Vec3{})
	}
	LinkEdict(e, false)
}

//export PF_normalize
func PF_normalize() {
	v := math.VFromA(*progsdat.Globals.Parm0f())
	vn := v.Normalize()
	*progsdat.Globals.Returnf() = vn.Array()
}
