package quakelib

import "C"

import (
	"quake/cmd"
	"quake/conlog"
	"quake/math"
	"quake/math/vec"
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

type EntityState struct {
	Origin     vec.Vec3
	Angles     vec.Vec3
	ModelIndex uint16
	Frame      uint16
	ColorMap   byte
	Skin       byte
	Alpha      byte
	Effects    int
}

type Edict struct {
	Free bool

	num_leafs int
	leafnums  [MAX_ENT_LEAFS]int

	Baseline     EntityState
	Alpha        byte // hack to support alpha since it's not part of entvars_t
	SendInterval bool // send time until nextthink to client for better lerp timing

	FreeTime float32 // sv.time when the object was freed
}

func edictNum(i int) *Edict {
	return &sv.edicts[i]
}

//export EDICT_FREE
func EDICT_FREE(n int) C.int {
	return b2i(edictNum(n).Free)
}

//export EDICT_SETFREE
func EDICT_SETFREE(n, free int) {
	edictNum(n).Free = (free != 0)
}

//export EDICT_ALPHA
func EDICT_ALPHA(n int) byte {
	return edictNum(n).Alpha
}

//export EDICT_SETALPHA
func EDICT_SETALPHA(n int, alpha byte) {
	edictNum(n).Alpha = alpha
}

//export AllocEdicts
func AllocEdicts() {
	AllocEntvars(sv.maxEdicts, progsdat.EdictSize)
	sv.edicts = make([]Edict, sv.maxEdicts)
}

//export FreeEdicts
func FreeEdicts() {
	FreeEntvars()
	sv.edicts = sv.edicts[:0]
}

// Marks the edict as free
// FIXME: walk all entities and NULL out references to this entity
func edictFree(i int) {
	// unlink from world bsp
	UnlinkEdict(i)

	e := edictNum(i)
	e.Free = true
	e.Alpha = 0
	e.FreeTime = sv.time

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
	edictNum(e).Free = false
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
		if e.Free && (e.FreeTime < 2 || sv.time-e.FreeTime > 0.5) {
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
	if edictNum(ed).Free {
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
		if edictNum(i).Free {
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
	edictNum(ent).Alpha = entAlphaEncode(v)
}
