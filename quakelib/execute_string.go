package quakelib

import (
	"github.com/therjak/goquake/alias"
	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/cvar"
	"github.com/therjak/goquake/execute"
)

func init() {
	execute.SetExecutors([](func([]cmd.QArg, int) bool){
		cmd.Execute,
		alias.Execute,
		cvar.Execute,
	})
}
