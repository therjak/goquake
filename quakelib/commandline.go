package quakelib

import (
	"C"

	cmdl "github.com/therjak/goquake/commandline"
)

//export CMLCurrent
func CMLCurrent() bool {
	return cmdl.Current()
}

//export CMLDedicated
func CMLDedicated() bool {
	return cmdl.Dedicated()
}

//export CMLFullscreen
func CMLFullscreen() bool {
	return cmdl.Fullscreen()
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

//export CMLWindow
func CMLWindow() bool {
	return cmdl.Window()
}

//export CMLHeight
func CMLHeight() int {
	return cmdl.Height()
}

//export CMLWidth
func CMLWidth() int {
	return cmdl.Width()
}

//export CMLBpp
func CMLBpp() int {
	return cmdl.Bpp()
}

//export CMLZone
func CMLZone() int {
	return cmdl.Zone()
}
