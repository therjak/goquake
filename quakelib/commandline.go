package quakelib

import (
	"C"
	cmdl "quake/commandline"
)

//export CMLCurrent
func CMLCurrent() C.int {
	return b2i(cmdl.Current())
}

//export CMLDedicated
func CMLDedicated() C.int {
	return b2i(cmdl.Dedicated())
}

//export CMLDedicatedNum
func CMLDedicatedNum() C.int {
	return C.int(cmdl.DedicatedNum())
}

//export CMLFullscreen
func CMLFullscreen() C.int {
	return b2i(cmdl.Fullscreen())
}

//export CMLMinMemory
func CMLMinMemory() C.int {
	return b2i(cmdl.MinMemory())
}

//export CMLAdd
func CMLAdd() C.int {
	return b2i(cmdl.Add())
}

//export CMLCombine
func CMLCombine() C.int {
	return b2i(cmdl.Combine())
}

//export CMLMtext
func CMLMtext() C.int {
	return b2i(cmdl.Mtext())
}

//export CMLHipnotic
func CMLHipnotic() C.int {
	// TODO: why isQuoth?
	return b2i(cmdl.Hipnotic() || cmdl.Quoth())
}

//export CMLRogue
func CMLRogue() C.int {
	return b2i(cmdl.Rogue())
}

//export CMLStandardQuake
func CMLStandardQuake() C.int {
	return b2i(!(cmdl.Quoth() || cmdl.Rogue() || cmdl.Hipnotic()))
}

//export CMLWindow
func CMLWindow() C.int {
	return b2i(cmdl.Window())
}

//export CMLHeight
func CMLHeight() C.int {
	return C.int(cmdl.Height())
}

//export CMLWidth
func CMLWidth() C.int {
	return C.int(cmdl.Width())
}

//export CMLBpp
func CMLBpp() C.int {
	return C.int(cmdl.Bpp())
}

//export CMLParticles
func CMLParticles() C.int {
	return C.int(cmdl.Particles())
}

//export CMLZone
func CMLZone() C.int {
	return C.int(cmdl.Zone())
}

//export CMLConsoleSize
func CMLConsoleSize() C.int {
	return C.int(cmdl.ConsoleSize())
}
