// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"goquake/alias"
	"goquake/cmd"
	"goquake/cvar"
	"goquake/execute"
)

func init() {
	// this should be only:
	// ban, begin, color, fly, give, god
	// kick, kill, name, noclip, notarget, pause,
	// ping, prespawn, say, say_team, setpos, spawn
	// status, tell
	// see sv_client.go
	// all those are defined in hostcmd and added by addClientCommand
	execute.SetClientExecutors([]execute.Efunc{
		cmd.Execute,
		//	alias.Execute,
		//	cvar.Execute,
	})
	execute.SetCommandExecutors([]execute.Efunc{
		cmd.Execute,
		alias.Execute,
		cvar.Execute,
	})
}
