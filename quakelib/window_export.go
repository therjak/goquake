package quakelib

//#ifndef HASMODESTATE
//#define HASMODESTATE
//typedef enum {MS_UNINIT, MS_WINDOWED, MS_FULLSCREEN} modestate_t;
//#endif
import "C"

import (
	"quake/window"
	"unsafe"
)

//export ConsoleWidth
func ConsoleWidth() C.int {
	return C.int(consoleLineWidth)
}

//export SetConsoleWidth
func SetConsoleWidth(w C.int) {
	consoleLineWidth = int(w)
}

//export VID_Locked
func VID_Locked() C.int {
	return b2i(videoLocked)
}

//export VID_Initialized
func VID_Initialized() C.int {
	return b2i(videoInitialized)
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
func VIDGLSwapControl() C.int {
	return b2i(glSwapControl)
}

//export SetVIDGLSwapControl
func SetVIDGLSwapControl(v C.int) {
	glSwapControl = (v != 0)
}

//export VIDChanged
func VIDChanged() C.int {
	return b2i(videoChanged)
}

//export SetVIDChanged
func SetVIDChanged(v C.int) {
	videoChanged = (v != 0)
}

//export WINDOW_Get
func WINDOW_Get() unsafe.Pointer {
	return unsafe.Pointer(window.Get())
}

//export PL_SetWindowIcon
func PL_SetWindowIcon() {
	window.InitIcon()
}

//export WINDOW_Shutdown
func WINDOW_Shutdown() {
	window.Shutdown()
}

//export WINDOW_SetMode
func WINDOW_SetMode(width, height, bpp, fullscreen C.int) {
	windowSetMode(int32(width), int32(height), int32(bpp), fullscreen == 1)
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
func VID_GetFullscreen() C.int {
	return b2i(window.Fullscreen())
}

//export VID_GetDesktopFullscreen
func VID_GetDesktopFullscreen() C.int {
	return b2i(window.DesktopFullscreen())
}

//export VID_GetVSync
func VID_GetVSync() C.int {
	return b2i(window.VSync())
}

//export VID_InitModelist
func VID_InitModelist() {
	updateAvailableDisplayModes()
}

//export VID_ValidMode
func VID_ValidMode(width, height, bpp, fullscreen C.int) C.int {
	return b2i(validDisplayMode(int32(width), int32(height), uint32(bpp), fullscreen != 0))
}

//export VID_SetModeState
func VID_SetModeState(s C.modestate_t) {
	switch s {
	case C.MS_WINDOWED:
		modestate = MS_WINDOWED
	case C.MS_FULLSCREEN:
		modestate = MS_FULLSCREEN
	default:
		modestate = MS_UNINIT
	}
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

//export VID_Toggle
func VID_Toggle() {
	videoToggle()
}

//export GetNumPages
func GetNumPages() C.int {
	return C.int(numPages)
}

//export SetNumPages
func SetNumPages(v C.int) {
	numPages = int(v)
}

//export GetRecalcRefdef
func GetRecalcRefdef() C.int {
	return b2i(recalc_refdef)
}

//export SetRecalcRefdef
func SetRecalcRefdef(v C.int) {
	recalc_refdef = (v != 0)
}

//export ConWidth
func ConWidth() C.int {
	return C.int(consoleWidth)
}

//export ConHeight
func ConHeight() C.int {
	return C.int(consoleHeight)
}

//export ScreenWidth
func ScreenWidth() C.int {
	return C.int(screenWidth)
}

//export ScreenHeight
func ScreenHeight() C.int {
	return C.int(screenHeight)
}

//export UpdateConsoleSize
func UpdateConsoleSize() {
	updateConsoleSize()
}

//export VID_SyncCvars
func VID_SyncCvars() {
	syncVideoCvars()
}

//export VID_Shutdown
func VID_Shutdown() {
	videoShutdown()
}
