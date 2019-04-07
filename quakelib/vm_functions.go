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

func setMinMaxSize(e int, min, max math.Vec3) {
	if min.X > max.X || min.Y > max.Y || min.Z > max.Z {
		runError("backwards mins/maxs")
	}
	ev := EntVars(e)
	ev.Mins = [3]float32{min.X, min.Y, min.Z}
	ev.Maxs = [3]float32{max.X, max.Y, max.Z}
	s := math.Sub(max, min)
	ev.Size = [3]float32{s.X, s.Y, s.Z}
	LinkEdict(e, false)
}

//export PF_setsize
func PF_setsize() {
	e := progsdat.RawGlobalsI[progs.OffsetParm0]
	min := math.Vec3{
		progsdat.RawGlobalsF[progs.OffsetParm1],
		progsdat.RawGlobalsF[progs.OffsetParm1+1],
		progsdat.RawGlobalsF[progs.OffsetParm1+2],
	}
	max := math.Vec3{
		progsdat.RawGlobalsF[progs.OffsetParm2],
		progsdat.RawGlobalsF[progs.OffsetParm2+1],
		progsdat.RawGlobalsF[progs.OffsetParm2+2],
	}
	setMinMaxSize(int(e), min, max)
}

//export PF_setmodel
func PF_setmodel() {

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
			setMinMaxSize(e, mod.ClipMins, mod.ClipMaxs)
		} else {
			setMinMaxSize(e, mod.Mins, mod.Maxs)
		}
	} else {
		log.Printf("No Mod")
		setMinMaxSize(e, math.Vec3{}, math.Vec3{})
	}
}
