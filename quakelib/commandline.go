// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"C"

	cmdl "goquake/commandline"
)

//export CMLDedicated
func CMLDedicated() bool {
	return cmdl.Dedicated()
}

//export CMLMinMemory
func CMLMinMemory() bool {
	return cmdl.MinMemory()
}

func CMLHipnotic() bool {
	// TODO: why isQuoth?
	return cmdl.Hipnotic() || cmdl.Quoth()
}

func CMLRogue() bool {
	return cmdl.Rogue()
}

//export CMLStandardQuake
func CMLStandardQuake() bool {
	return !(cmdl.Quoth() || cmdl.Rogue() || cmdl.Hipnotic())
}

//export CMLZone
func CMLZone() int {
	return cmdl.Zone()
}
