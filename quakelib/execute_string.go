package quakelib

import "C"

import (
	"quake/alias"
	"quake/cmd"
	"quake/cvar"
	"quake/execute"
)

//export Cmd_ExecuteString
func Cmd_ExecuteString(s *C.char, source C.int) {
	if source == execute.Client {
		execute.Execute(C.GoString(s), execute.Client, sv_player)
	} else {
		execute.Execute(C.GoString(s), execute.Command, sv_player)
	}
}

func init() {
	execute.SetExecutors([](func([]cmd.QArg, int) bool){
		cmd.Execute,
		alias.Execute,
		cvar.Execute,
	})
}
