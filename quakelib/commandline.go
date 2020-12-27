package quakelib

import (
	"C"

	cmdl "github.com/therjak/goquake/commandline"
)

//export CMLDedicated
func CMLDedicated() bool {
	return cmdl.Dedicated()
}

//export CMLMinMemory
func CMLMinMemory() bool {
	return cmdl.MinMemory()
}

//export CMLHipnotic
func CMLHipnotic() bool {
	// TODO: why isQuoth?
	return cmdl.Hipnotic() || cmdl.Quoth()
}

//export CMLRogue
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
