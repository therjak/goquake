package quakelib

import "C"

import (
	"log"
	"quake/progs"
)

var (
	progsdat        *progs.LoadedProg
	engineStrings   []string
	engineStringMap map[string]int
)

func init() {
	engineStringMap = make(map[string]int)
}

//export PR_LoadProgsGo
func PR_LoadProgsGo() {
	p, err := progs.LoadProgs()
	if err != nil {
		log.Fatalf("Failed to load progs.dat: %v", err)
	}
	progsdat = p
	fillEngineStrings(p.Strings)
}

func fillEngineStrings(ks map[int]string) {
	// just ignore duplicates
	for k, v := range ks {
		_, ok := engineStringMap[v]
		if !ok {
			engineStringMap[v] = k
		}
	}
}

//export ED_NewString
func ED_NewString(str *C.char) C.int {
	s := C.GoString(str)
	i := EDNewString(s)
	return C.int(i)
}

func EDNewString(s string) int {
	// TODO:
	// replace \n with '\n' and all other \x with just '\'
	engineStrings = append(engineStrings, s)
	i := len(engineStrings)
	engineStringMap[s] = -i
	return -i
}

//export PR_SetEngineString
func PR_SetEngineString(str *C.char) C.int {
	s := C.GoString(str)
	i := PRSetEngineString(s)
	return C.int(i)
}

func PRSetEngineString(s string) int {
	/*
		v, ok := engineStringMap[s]
		if ok {
			log.Printf("PR_SetEngineString1 %v, %d", s, v)
			return C.int(v)
		}
	*/
	engineStrings = append(engineStrings, s)
	i := len(engineStrings)
	engineStringMap[s] = -i
	return -i
}

//export PR_GetString
func PR_GetString(num C.int) *C.char {
	n := int(num)
	s := PRGetString(n)
	if s == nil {
		return nil
	}
	// TODO: FIX memory leak
	return C.CString(*s)
}

func PRGetString(n int) *string {
	if n >= 0 {
		s, ok := progsdat.Strings[n]
		if !ok {
			log.Printf("PR_GetStringInt: request of %v, is unknown", n)
			return nil
		}
		return &s
	}
	// n is negative, so -(n + 1) is our index
	index := -(n + 1)
	if len(engineStrings) <= index {
		log.Printf("PR_GetStringInt: request of %v, is unknown", n)
		return nil
	}
	return &engineStrings[index]
}

//export Pr_globalsf
func Pr_globalsf(i int) float32 {
	return progsdat.RawGlobalsF[i]
}

//export Set_Pr_globalsf
func Set_Pr_globalsf(i int, f float32) {
	progsdat.RawGlobalsF[i] = f
}

//export Pr_globalsi
func Pr_globalsi(i int) int32 {
	return progsdat.RawGlobalsI[i]
}

//export Set_Pr_globalsi
func Set_Pr_globalsi(i int, f int32) {
	progsdat.RawGlobalsI[i] = f
}

//export Pr_global_struct_self
func Pr_global_struct_self() int32 {
	return progsdat.Globals.Self
}

//export Pr_global_struct_time
func Pr_global_struct_time() float32 {
	return progsdat.Globals.Time
}

//export Set_pr_global_struct_mapname
func Set_pr_global_struct_mapname(n int32) {
	progsdat.Globals.MapName = n
}

//export Set_pr_global_struct_self
func Set_pr_global_struct_self(s int32) {
	progsdat.Globals.Self = s
}

//export Set_pr_global_struct_other
func Set_pr_global_struct_other(o int32) {
	progsdat.Globals.Other = o
}

//export Set_pr_global_struct_time
func Set_pr_global_struct_time(t float32) {
	progsdat.Globals.Time = t
}

//export Set_pr_global_struct_frametime
func Set_pr_global_struct_frametime(t float32) {
	progsdat.Globals.FrameTime = t
}

//export Set_pr_global_struct_deathmatch
func Set_pr_global_struct_deathmatch(t float32) {
	progsdat.Globals.DeathMatch = t
}

//export Set_pr_global_struct_coop
func Set_pr_global_struct_coop(t float32) {
	progsdat.Globals.Coop = t
}

//export Set_pr_global_struct_serverflags
func Set_pr_global_struct_serverflags(t float32) {
	progsdat.Globals.ServerFlags = t
}
