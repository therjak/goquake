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
	for _, v := range ks {
		_, ok := engineStringMap[v]
		if !ok {
			engineStringMap[v] = len(engineStrings)
			engineStrings = append(engineStrings, v)
		}
	}
}

// export ED_NewString
func ED_NewString(str *C.char) C.int {
	// TODO:
	// replace \n with '\n' and all other \x with just '\'
	log.Printf("ED_NewString %x", C.GoString(str))
	return PR_SetEngineString(str)
}

// export PR_SetEngineString
func PR_SetEngineString(str *C.char) C.int {
	s := C.GoString(str)
	i, ok := engineStringMap[s]
	if !ok {
		i = len(engineStrings)
		engineStringMap[s] = i
		engineStrings = append(engineStrings, s)
		log.Printf("PR_SetEngineStringNEW")
	}
	log.Printf("PR_SetEngineString %v, %d", s, i)
	return C.int(-(i + 1))
}

//export PR_GetStringInt
func PR_GetStringInt(num C.int) *C.char {
	n := int(num)
	if num >= 0 {
		s, ok := progsdat.Strings[n]
		if !ok {
			log.Printf("PR_GetStringInt: request of %v, is unknown", n)
			return C.CString("")
		}
		// log.Printf("PR_GetEngineString1 %v", s)
		return C.CString(s)
	}
	// n is negative, so -(n + 1) is our index
	index := -(n + 1)
	if len(engineStrings) <= index {
		log.Printf("PR_GetStringInt: request of %v, is unknown", n)
		return C.CString("")
	}
	log.Printf("PR_GetEngineString2 %v, %v", index, engineStrings[index])
	return C.CString(engineStrings[index])
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

//export Pr_global_struct_other
func Pr_global_struct_other() int32 {
	return progsdat.Globals.Other
}

//export Pr_global_struct_world
func Pr_global_struct_world() int32 {
	return progsdat.Globals.World
}

//export Pr_global_struct_time
func Pr_global_struct_time() float32 {
	return progsdat.Globals.Time
}

//export Pr_global_struct_force_retouch
func Pr_global_struct_force_retouch() float32 {
	return progsdat.Globals.ForceRetouch
}

//export Pr_global_struct_parm
func Pr_global_struct_parm(i int) float32 {
	return progsdat.Globals.Parm[i]
}

//export Dec_pr_global_struct_force_retouch
func Dec_pr_global_struct_force_retouch() {
	progsdat.Globals.ForceRetouch--
}

//export Pr_global_struct_deathmatch
func Pr_global_struct_deathmatch() float32 {
	return progsdat.Globals.DeathMatch
}

//export Pr_global_struct_coop
func Pr_global_struct_coop() float32 {
	return progsdat.Globals.Coop
}

//export Pr_global_struct_teamplay
func Pr_global_struct_teamplay() float32 {
	return progsdat.Globals.TeamPlay
}

//export Pr_global_struct_serverflags
func Pr_global_struct_serverflags() float32 {
	return progsdat.Globals.ServerFlags
}

//export Pr_global_struct_total_secrets
func Pr_global_struct_total_secrets() float32 {
	return progsdat.Globals.TotalSecrets
}

//export Pr_global_struct_total_monsters
func Pr_global_struct_total_monsters() float32 {
	return progsdat.Globals.TotalMonsters
}

//export Pr_global_struct_found_secrets
func Pr_global_struct_found_secrets() float32 {
	return progsdat.Globals.FoundSecrets
}

//export Pr_global_struct_killed_monsters
func Pr_global_struct_killed_monsters() float32 {
	return progsdat.Globals.KilledMonsters
}

//export Pr_global_struct_PlayerPreThink
func Pr_global_struct_PlayerPreThink() int32 {
	return progsdat.Globals.PlayerPreThink
}

//export Pr_global_struct_PlayerPostThink
func Pr_global_struct_PlayerPostThink() int32 {
	return progsdat.Globals.PlayerPostThink
}

//export Pr_global_struct_StartFrame
func Pr_global_struct_StartFrame() int32 {
	return progsdat.Globals.StartFrame
}

//export Pr_global_struct_SetNewParms
func Pr_global_struct_SetNewParms() int32 {
	return progsdat.Globals.SetNewParms
}

//export Pr_global_struct_SetChangeParms
func Pr_global_struct_SetChangeParms() int32 {
	return progsdat.Globals.SetChangeParms
}

//export Pr_global_struct_msg_entity
func Pr_global_struct_msg_entity() int32 {
	return progsdat.Globals.MsgEntity
}

//export Pr_global_struct_ClientKill
func Pr_global_struct_ClientKill() int32 {
	return progsdat.Globals.ClientKill
}

//export Pr_global_struct_ClientConnect
func Pr_global_struct_ClientConnect() int32 {
	return progsdat.Globals.ClientConnect
}

//export Pr_global_struct_ClientDisconnect
func Pr_global_struct_ClientDisconnect() int32 {
	return progsdat.Globals.ClientDisconnect
}

//export Pr_global_struct_PutClientInServer
func Pr_global_struct_PutClientInServer() int32 {
	return progsdat.Globals.PutClientInServer
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

//export Set_pr_global_struct_trace_allsolid
func Set_pr_global_struct_trace_allsolid(t float32) {
	progsdat.Globals.TraceAllSolid = t
}

//export Set_pr_global_struct_trace_startsolid
func Set_pr_global_struct_trace_startsolid(t float32) {
	progsdat.Globals.TraceStartSolid = t
}

//export Set_pr_global_struct_trace_fraction
func Set_pr_global_struct_trace_fraction(t float32) {
	progsdat.Globals.TraceFraction = t
}

//export Set_pr_global_struct_trace_inwater
func Set_pr_global_struct_trace_inwater(t float32) {
	progsdat.Globals.TraceInWater = t
}

//export Set_pr_global_struct_trace_inopen
func Set_pr_global_struct_trace_inopen(t float32) {
	progsdat.Globals.TraceInOpen = t
}

//export Set_pr_global_struct_trace_plane_dist
func Set_pr_global_struct_trace_plane_dist(t float32) {
	progsdat.Globals.TracePlaneDist = t
}

//export Set_pr_global_struct_trace_ent
func Set_pr_global_struct_trace_ent(o int32) {
	progsdat.Globals.TraceEnt = o
}

//export Set_pr_global_struct_parm
func Set_pr_global_struct_parm(i int, t float32) {
	progsdat.Globals.Parm[i] = t
}

//export Pr_global_struct_v_forward
func Pr_global_struct_v_forward(x, y, z *float32) {
	*x = progsdat.Globals.VForward[0]
	*y = progsdat.Globals.VForward[1]
	*z = progsdat.Globals.VForward[2]
}

//export Set_pr_global_struct_v_forward
func Set_pr_global_struct_v_forward(x, y, z float32) {
	progsdat.Globals.VForward[0] = x
	progsdat.Globals.VForward[1] = y
	progsdat.Globals.VForward[2] = z
}

//export Set_pr_global_struct_v_up
func Set_pr_global_struct_v_up(x, y, z float32) {
	progsdat.Globals.VUp[0] = x
	progsdat.Globals.VUp[1] = y
	progsdat.Globals.VUp[2] = z
}

//export Set_pr_global_struct_v_right
func Set_pr_global_struct_v_right(x, y, z float32) {
	progsdat.Globals.VRight[0] = x
	progsdat.Globals.VRight[1] = y
	progsdat.Globals.VRight[2] = z
}

//export Set_pr_global_struct_trace_endpos
func Set_pr_global_struct_trace_endpos(x, y, z float32) {
	progsdat.Globals.TraceEndPos[0] = x
	progsdat.Globals.TraceEndPos[1] = y
	progsdat.Globals.TraceEndPos[2] = z
}

//export Set_pr_global_struct_trace_plane_normal
func Set_pr_global_struct_trace_plane_normal(x, y, z float32) {
	progsdat.Globals.TracePlaneNormal[0] = x
	progsdat.Globals.TracePlaneNormal[1] = y
	progsdat.Globals.TracePlaneNormal[2] = z
}
