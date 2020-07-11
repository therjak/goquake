package quakelib

import "C"

import (
	"time"
)

//export SCR_Init
func SCR_Init() {
	screen.initialized = true
}

//export GetScreenConsoleCurrentHeight
func GetScreenConsoleCurrentHeight() int {
	return screen.consoleLines
}

//export SCR_CenterPrint
func SCR_CenterPrint(s *C.char) {
	screen.CenterPrint(C.GoString(s))
}

//export SCR_ModalMessage
func SCR_ModalMessage(c *C.char, timeout C.float) bool {
	return screen.ModalMessage(C.GoString(c), time.Second*time.Duration(timeout))
}

//export SCR_ResetTileClearUpdates
func SCR_ResetTileClearUpdates() {
	screen.ResetTileClearUpdates()
}

//export SCR_BeginLoadingPlaque
func SCR_BeginLoadingPlaque() {
	screen.BeginLoadingPlaque()
}
