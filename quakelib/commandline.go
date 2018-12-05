package quakelib

import (
	"C"
	cmdl "quake/commandline"
)

//export CMLConsoleDebug
func CMLConsoleDebug() C.int {
	return b2i(cmdl.ConsoleDebug())
}

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

//export CMLFitz
func CMLFitz() C.int {
	return b2i(cmdl.Fitz())
}

//export CMLFullscreen
func CMLFullscreen() C.int {
	return b2i(cmdl.Fullscreen())
}

//export CMLListen
func CMLListen() C.int {
	return b2i(cmdl.Listen())
}

//export CMLListenNum
func CMLListenNum() C.int {
	return C.int(cmdl.ListenNum())
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

//export CMLSound
func CMLSound() C.int {
	return b2i(!cmdl.Sound())
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

//export CMLQuoth
func CMLQuoth() C.int {
	return b2i(cmdl.Quoth())
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

//export CMLFsaa
func CMLFsaa() C.int {
	return C.int(cmdl.Fsaa())
}

//export CMLPort
func CMLPort() C.int {
	return C.int(cmdl.Port())
}

//export CMLParticles
func CMLParticles() C.int {
	return C.int(cmdl.Particles())
}

//export CMLProtocol
func CMLProtocol() C.int {
	return C.int(cmdl.Protocol())
}

//export CMLZone
func CMLZone() C.int {
	return C.int(cmdl.Zone())
}

//export CMLConsoleSize
func CMLConsoleSize() C.int {
	return C.int(cmdl.ConsoleSize())
}
