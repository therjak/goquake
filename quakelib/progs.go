package quakelib

import "C"

import (
	"log"
	"quake/progs"
)

var (
	progsdat *progs.LoadedProg
)

//export PR_LoadProgsGo
func PR_LoadProgsGo() {
	log.Printf("LOADING PROGS")
	p, err := progs.LoadProgs()
	if err != nil {
		log.Fatalf("Failed to load progs.dat: %v", err)
	}
	progsdat = p
}

//export ED_NewString
func ED_NewString(str *C.char) C.int {
	s := C.GoString(str)
	i := progsdat.NewString(s)
	return C.int(i)
}

//export PR_SetEngineString
func PR_SetEngineString(str *C.char) C.int {
	s := C.GoString(str)
	if len(s) == 0 && progsdat == nil {
		log.Printf("Trying to add an empty string in PR_SetEngineString")
		return 0
	}
	if progsdat == nil {
		log.Printf("Trying to add: '%s' before PR_LoadProgsGo, len %v", s, len(s))
	}
	i := progsdat.AddString(s)
	return C.int(i)
}

//export PR_GetString
func PR_GetString(num C.int) *C.char {
	n := int(num)
	if progsdat == nil {
		return nil
	}
	s, err := progsdat.String(n)
	if err != nil {
		return nil
	}
	// TODO: FIX memory leak
	return C.CString(s)
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
