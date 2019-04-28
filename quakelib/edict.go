package quakelib

//#include "edict.h"
//void ED_Free(int ed);
//int ED_Alloc(void);
import "C"

import (
	"quake/cmd"
	"quake/progs"
)

const (
	defSaveGlobal = 1 << 15
)

func edictNum(i int) *C.edict_t {
	return C.EDICT_NUM(C.int(i))
}

func edictFree(i int) {
	C.ED_Free(C.int(i))
}

func edictAlloc() int {
	return int(C.ED_Alloc())
}

//export ED_Print
func ED_Print(ed C.int) {
	edictPrint(int(ed))
}

// For debugging
func edictPrint(ed int) {
	if edictNum(ed).free != 0 {
		conPrintf("FREE\n")
		return
	}
	conSafePrintf("\nEDICT %d:\n", ed)
	for i := 1; i < len(progsdat.FieldDefs); i++ {
		d := progsdat.FieldDefs[i]
		name := PRGetString(int(d.SName))
		if name == nil {
			continue
		}
		l := len(*name)
		if l > 1 && (*name)[l-2] == '_' {
			// skip _x, _y, _z vars
			continue
		}
		// TODO: skip 0 values
		conSafePrintf(*name)
		for ; l < 15; l++ {
			conSafePrintf(" ")
		}
		conSafePrintf("%s\n", EntVarsSprint(ed, d))
	}
}

//export ED_PrintNum
func ED_PrintNum(ent C.int) {
	edictPrint(int(ent))
}

// For debugging, prints all the entities in the current server
//export ED_PrintEdicts
func ED_PrintEdicts() {
	edictPrintEdicts()
}

func edictPrintEdicts() {
	if !sv.active {
		return
	}

	conPrintf("%d entities\n", sv.numEdicts)
	for i := 0; i < sv.numEdicts; i++ {
		edictPrint(i)
	}
}

// For debugging, prints a single edicy
//export ED_PrintEdict_f
func ED_PrintEdict_f() {
	edictPrintEdictFunc()
}

func edictPrintEdictFunc() {
	if !sv.active {
		return
	}

	i := cmd.CmdArgv(1).Int()
	if i < 0 || i >= sv.numEdicts {
		conPrintf("Bad edict number\n")
		return
	}
	edictPrint(i)
}

// For debugging
//export ED_Count
func ED_Count() {
	edictCount()
}

func edictCount() {
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

	conPrintf("num_edicts:%3d\n", sv.numEdicts)
	conPrintf("active    :%3d\n", active)
	conPrintf("view      :%3d\n", models)
	conPrintf("touch     :%3d\n", solid)
	conPrintf("step      :%3d\n", step)
}
