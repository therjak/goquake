package quakelib

//#ifndef HASMODESTATE
//#define HASMODESTATE
//typedef enum {MS_UNINIT, MS_WINDOWED, MS_FULLSCREEN} modestate_t;
//#endif
import "C"

import (
	"fmt"
	"github.com/therjak/goquake/window"
)

//export VID_Locked
func VID_Locked() bool {
	return videoLocked
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

//export VID_SetMode
func VID_SetMode(width, height, fullscreen C.int) {
	videoSetMode(int32(width), int32(height), fullscreen == 1)
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

//export VID_ValidMode
func VID_ValidMode(width, height, fullscreen C.int) bool {
	return validDisplayMode(int32(width), int32(height), fullscreen != 0)
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

//export VIDGetSwapInterval
func VIDGetSwapInterval() int {
	return getSwapInterval()
}

//export VID_Init_Go
func VID_Init_Go() {
	err := videoInit()
	if err != nil {
		fmt.Printf("%v", err)
		Error(fmt.Sprintf("%v", err))
	}
}
