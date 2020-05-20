package quakelib

//#ifndef HASMODESTATE
//#define HASMODESTATE
//typedef enum {MS_UNINIT, MS_WINDOWED, MS_FULLSCREEN} modestate_t;
//#endif
import "C"

import (
	"github.com/therjak/goquake/window"
)

//export VID_Locked
func VID_Locked() bool {
	return videoLocked
}

//export VID_Initialized
func VID_Initialized() bool {
	return videoInitialized
}

//export SetVID_Locked
func SetVID_Locked(v C.int) {
	videoLocked = (v != 0)
}

//export SetVID_Initialized
func SetVID_Initialized(v C.int) {
	videoInitialized = (v != 0)
}

//export VIDGLSwapControl
func VIDGLSwapControl() bool {
	return glSwapControl
}

//export SetVIDGLSwapControl
func SetVIDGLSwapControl(v C.int) {
	glSwapControl = (v != 0)
}

//export VIDChanged
func VIDChanged() bool {
	return videoChanged
}

//export PL_SetWindowIcon
func PL_SetWindowIcon() {
	window.InitIcon()
}

//export VID_SetMode
func VID_SetMode(width, height, bpp, fullscreen C.int) {
	videoSetMode(int32(width), int32(height), int32(bpp), fullscreen == 1)
}

//export VID_GetCurrentWidth
func VID_GetCurrentWidth() C.int {
	w, _ := window.Size()
	return C.int(w)
}

//export VID_GetCurrentHeight
func VID_GetCurrentHeight() C.int {
	_, h := window.Size()
	return C.int(h)
}

//export GL_EndRendering
func GL_EndRendering() {
	window.EndRendering()
}

//export VID_GetCurrentBPP
func VID_GetCurrentBPP() C.int {
	return C.int(window.BPP())
}

//export VID_GetFullscreen
func VID_GetFullscreen() bool {
	return window.Fullscreen()
}

//export VID_GetVSync
func VID_GetVSync() bool {
	return window.VSync()
}

//export VID_InitModelist
func VID_InitModelist() {
	updateAvailableDisplayModes()
}

//export VID_ValidMode
func VID_ValidMode(width, height, bpp, fullscreen C.int) bool {
	return validDisplayMode(int32(width), int32(height), uint32(bpp), fullscreen != 0)
}

//export VID_GetModeState
func VID_GetModeState() C.modestate_t {
	switch modestate {
	case MS_WINDOWED:
		return C.MS_WINDOWED
	case MS_FULLSCREEN:
		return C.MS_FULLSCREEN
	default:
		return MS_UNINIT
	}
}

//export SetRecalcRefdef
func SetRecalcRefdef(v C.int) {
	screen.recalcViewRect = (v != 0)
}

//export UpdateConsoleSize
func UpdateConsoleSize() {
	updateConsoleSize()
}

//export VID_SyncCvars
func VID_SyncCvars() {
	syncVideoCvars()
}
