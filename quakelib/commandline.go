// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	cmdl "goquake/commandline"
)

func CMLHipnotic() bool {
	// TODO: why isQuoth?
	return cmdl.Hipnotic() || cmdl.Quoth()
}

func CMLRogue() bool {
	return cmdl.Rogue()
}
