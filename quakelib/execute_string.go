// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"goquake/alias"
	"goquake/cbuf"
	"goquake/cmd"
	"goquake/cvar"
)

func init() {
	cbuf.SetCommandExecutors([]cbuf.Efunc{
		cmd.Execute,
		alias.Execute,
		cvar.Execute,
	})
}
