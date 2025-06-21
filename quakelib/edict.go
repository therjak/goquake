// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"fmt"
	"log/slog"

	"goquake/bsp"
	"goquake/conlog"
	"goquake/cvars"
	"goquake/math"
	"goquake/math/vec"
	"goquake/progs"
)

const (
	defSaveGlobal = 1 << 15
	MIN_EDICTS    = 265
	MAX_EDICTS    = 32000 // edicts past 8192 can't play sounds in standard protocol

)

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

var entvars *progs.EntityVars

func AllocEdicts() {
	entvars = progs.AllocEntvars(sv.maxEdicts, progsdat.EdictSize, progsdat)
	sv.edicts = make([]Edict, sv.maxEdicts)
}

func freeEdicts() {
	entvars.Free()
	sv.edicts = sv.edicts[:0]
}

// Marks the edict as free
// FIXME: walk all entities and NULL out references to this entity
func (v *virtualMachine) edictFree(i int) {
	// unlink from world bsp
	v.UnlinkEdict(i)

	e := edictNum(i)
	e.Free = true
	e.Alpha = 0
	e.FreeTime = sv.time

	ev := entvars.Get(i)
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

func ClearEdict(e int) {
	ent := edictNum(e)
	*ent = Edict{}
	entvars.Clear(e)
}

// Either finds a free edict, or allocates a new one.
// Try to avoid reusing an entity that was recently freed, because it
// can cause the client to think the entity morphed into something else
// instead of being removed and recreated, which can cause interpolated
// angles and bad trails.
func edictAlloc() (int, error) {
	i := svs.maxClients + 1
	for ; i < sv.numEdicts; i++ {
		e := edictNum(i)
		// the first couple seconds of server time can involve a lot of
		// freeing and allocating, so relax the replacement policy
		if e.Free && (e.FreeTime < 2 || sv.time-e.FreeTime > 0.5) {
			entvars.Clear(i)
			edictNum(i).Free = false
			return i, nil
		}
	}

	if i == sv.maxEdicts {
		return 0, fmt.Errorf("ED_Alloc: no free edicts (max_edicts is %d)", sv.maxEdicts)
	}

	sv.numEdicts++
	ClearEdict(i)

	return i, nil
}

func entAlphaEncode(a float32) byte {
	if a == 0 {
		return 0 //ENTALPHA_DEFAULT
	}
	return byte(math.Clamp(1, math.Round(a*254.0+1), 255))
}

func UpdateEdictAlpha(ent int) {
	v, err := entvars.FieldValue(ent, "alpha")
	if err != nil {
		return
	}
	edictNum(ent).Alpha = entAlphaEncode(v)
}

func parse(ed int, data *bsp.Entity) {
	if ed != 0 {
		entvars.Clear(ed)
	}
	for _, on := range data.PropertyNames() {
		// some hacks...
		angleHack := false
		n := on
		if n == "angle" {
			n = "angles"
			angleHack = true
		}
		if n == "light" {
			n = "light_lev"
		}
		def, err := progsdat.FindFieldDef(n)
		if err != nil {
			if n != "sky" && n != "fog" && n != "alpha" {
				conlog.DPrint("Can't find field", slog.String("field", n))
			}
			continue
		}
		p, _ := data.Property(on)
		if angleHack {
			p = fmt.Sprintf("0 %s 0", p)
		}
		entvars.ParsePair(ed, def, p)
	}
}

const (
	SPAWNFLAG_NOT_EASY       = 256
	SPAWNFLAG_NOT_MEDIUM     = 512
	SPAWNFLAG_NOT_HARD       = 1024
	SPAWNFLAG_NOT_DEATHMATCH = 2048
)

// The entities are directly placed in the array, rather than allocated with
// ED_Alloc, because otherwise an error loading the map would have entity
// number references out of order.
//
// Creates a server's entity / program execution context by
// parsing textual entity definitions out of an ent file.
//
// Used for both fresh maps and savegame loads.  A fresh map would also need
// to call ED_CallSpawnFunctions () to let the objects initialize themselves.
func loadEntities(data []*bsp.Entity) error {
	progsdat.Globals.Time = sv.time
	inhibit := 0
	eNr := -1

	currentSkill := int(cvars.Skill.Value())
	// parse ents
	for _, j := range data {
		if eNr < 0 {
			eNr = 0
		} else {
			n, err := edictAlloc()
			if err != nil {
				return err
			}
			eNr = n
		}
		parse(eNr, j)

		ev := entvars.Get(eNr)

		// remove things from different skill levels or deathmatch
		if cvars.DeathMatch.Bool() {
			if (int(ev.SpawnFlags) & SPAWNFLAG_NOT_DEATHMATCH) != 0 {
				vm.edictFree(eNr)
				inhibit++
				continue
			}
		} else if (currentSkill == 0 && int(ev.SpawnFlags)&SPAWNFLAG_NOT_EASY != 0) ||
			(currentSkill == 1 && int(ev.SpawnFlags)&SPAWNFLAG_NOT_MEDIUM != 0) ||
			(currentSkill >= 2 && int(ev.SpawnFlags)&SPAWNFLAG_NOT_HARD != 0) {
			vm.edictFree(eNr)
			inhibit++
			continue
		}

		if ev.ClassName == 0 {
			conlog.SafePrintf("No classname for:\n")
			edictPrint(eNr)
			vm.edictFree(eNr)
			continue
		}

		fname, _ := progsdat.String(ev.ClassName)
		fidx, err := progsdat.FindFunction(fname)

		if err != nil {
			conlog.SafePrintf("No spawn function for:\n")
			edictPrint(eNr)
			vm.edictFree(eNr)
			continue
		}

		progsdat.Globals.Self = int32(eNr)
		if err := vm.ExecuteProgram(int32(fidx)); err != nil {
			return err
		}
	}

	conlog.DPrint("entities inhibited", slog.Int("count", inhibit))
	return nil
}
