// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"goquake/alias"
	"goquake/cmd"
	"goquake/cvar"
	"goquake/execute"
)

func init() {
	execute.SetCommandExecutors([]execute.Efunc{
		cmd.Execute,
		alias.Execute,
		cvar.Execute,
	})
}
