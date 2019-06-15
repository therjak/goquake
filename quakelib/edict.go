package quakelib

//#include "edict.h"
import "C"

import (
	"quake/cmd"
	"quake/conlog"
	"quake/math"
	"quake/progs"
)

const (
	defSaveGlobal = 1 << 15
)

func init() {
	cmd.AddCommand("edict", edictPrintEdictFunc)
	cmd.AddCommand("edicts", edictPrintEdicts)
	cmd.AddCommand("edictcount", edictCount)
}

func edictNum(i int) *C.edict_t {
	return C.EDICT_NUM(C.int(i))
}

// Marks the edict as free
// FIXME: walk all entities and NULL out references to this entity
func edictFree(i int) {
	// unlink from world bsp
	UnlinkEdict(i)

	e := edictNum(i)
	e.free = b2i(true)
	e.alpha = 0
	e.freetime = C.float(sv.time)

	ev := EntVars(i)
	ev.Model = 0
	ev.TakeDamage = 0
	ev.ModelIndex = 0
	ev.ColorMap = 0
	ev.Skin = 0
	ev.Frame = 0
	ev.Origin = [3]float32{}
	ev.Angles = [3]float32{}
	ev.NextThink = -1
	ev.Solid = 0
}

//export ED_Free
func ED_Free(ed int) {
	edictFree(ed)
}

// Sets everything to NULL
func edClearEdict(e int) {
	TTClearEntVars(e)
	edictNum(e).free = b2i(false)
}

// Either finds a free edict, or allocates a new one.
// Try to avoid reusing an entity that was recently freed, because it
// can cause the client to think the entity morphed into something else
// instead of being removed and recreated, which can cause interpolated
// angles and bad trails.
func edictAlloc() int {
	i := svs.maxClients + 1
	for ; i < sv.numEdicts; i++ {
		e := edictNum(i)
		// the first couple seconds of server time can involve a lot of
		// freeing and allocating, so relax the replacement policy
		if e.free != 0 && (e.freetime < 2 || sv.time-float32(e.freetime) > 0.5) {
			edClearEdict(i)
			return i
		}
	}

	if i == sv.maxEdicts {
		HostError("ED_Alloc: no free edicts (max_edicts is %d)", sv.maxEdicts)
	}

	sv.numEdicts++
	TTClearEdict(i)

	return i
}

//export ED_Alloc
func ED_Alloc() int {
	return edictAlloc()
}

// For debugging
func edictPrint(ed int) {
	if edictNum(ed).free != 0 {
		conlog.Printf("FREE\n")
		return
	}
	conlog.SafePrintf("\nEDICT %d:\n", ed)
	for i := 1; i < len(progsdat.FieldDefs); i++ {
		d := progsdat.FieldDefs[i]
		name, err := progsdat.String(d.SName)
		if err != nil {
			continue
		}
		l := len(name)
		if l > 1 && (name)[l-2] == '_' {
			// skip _x, _y, _z vars
			continue
		}
		// TODO: skip 0 values
		conlog.SafePrintf(name)
		for ; l < 15; l++ {
			conlog.SafePrintf(" ")
		}
		conlog.SafePrintf("%s\n", EntVarsSprint(ed, d))
	}
}

//export ED_PrintNum
func ED_PrintNum(ent C.int) {
	edictPrint(int(ent))
}

// For debugging, prints all the entities in the current server
func edictPrintEdicts(_ []cmd.QArg) {
	if !sv.active {
		return
	}

	conlog.Printf("%d entities\n", sv.numEdicts)
	for i := 0; i < sv.numEdicts; i++ {
		edictPrint(i)
	}
}

// For debugging, prints a single edict
func edictPrintEdictFunc(args []cmd.QArg) {
	if !sv.active || len(args) == 0 {
		return
	}

	i := args[0].Int()
	if i < 0 || i >= sv.numEdicts {
		conlog.Printf("Bad edict number\n")
		return
	}
	edictPrint(i)
}

// For debugging
func edictCount(_ []cmd.QArg) {
	if !sv.active {
		return
	}

	active := 0
	models := 0
	solid := 0
	step := 0
	for i := 0; i < sv.numEdicts; i++ {
		if edictNum(i).free != 0 {
			continue
		}
		active++
		if EntVars(i).Solid != 0 {
			solid++
		}
		if EntVars(i).Model != 0 {
			models++
		}
		if EntVars(i).MoveType == progs.MoveTypeStep {
			step++
		}
	}

	conlog.Printf("num_edicts:%3d\n", sv.numEdicts)
	conlog.Printf("active    :%3d\n", active)
	conlog.Printf("view      :%3d\n", models)
	conlog.Printf("touch     :%3d\n", solid)
	conlog.Printf("step      :%3d\n", step)
}

func entAlphaEncode(a float32) byte {
	if a == 0 {
		return 0 //ENTALPHA_DEFAULT
	}
	return byte(math.Clamp32(1, math.Round(a*254.0+1), 255))
}

//export UpdateEdictAlpha
func UpdateEdictAlpha(ent int) {
	v, err := EntVarsFieldValue(ent, "alpha")
	if err != nil {
		return
	}
	edictNum(ent).alpha = C.uchar(entAlphaEncode(v))
}
