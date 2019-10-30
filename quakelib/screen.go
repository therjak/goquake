package quakelib

//#include <stdlib.h>
// int SCR_ModalMessage(const char *text, float timeout);
// void SCR_BeginLoadingPlaque(void);
// void SCR_UpdateScreen(void);
// void ResetTileClearUpdates(void);
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"quake/cmd"
	"quake/conlog"
	"quake/cvars"
	"quake/image"
	"quake/keys"
	"quake/snd"
	"strings"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v4.6-core/gl"
)

var (
	screen qScreen
)

type qScreen struct {
	disabled bool

	centerString []string
	centerTime   time.Time

	loading bool
	dialog  bool

	turtlePic   *QPic
	turtleCount int

	netPic *QPic

	vrect Rect

	disabledTime float64 // needs to match host.time type
}

//export SCR_CenterPrint
func SCR_CenterPrint(s *C.char) {
	screen.CenterPrint(C.GoString(s))
}

//export SCR_DrawCrosshair
func SCR_DrawCrosshair() {
	screen.drawCrosshair()
}

//export SCR_CheckDrawCenterString
func SCR_CheckDrawCenterString() {
	screen.CheckDrawCenterPrint()
}

//export SCR_DrawPause
func SCR_DrawPause() {
	screen.drawPause()
}

//export SCR_DrawClock
func SCR_DrawClock() {
	screen.drawClock()
}

//export SCR_IsDrawLoading
func SCR_IsDrawLoading() C.int {
	return b2i(screen.loading)
}

//export SCR_SetDrawLoading
func SCR_SetDrawLoading(b int) {
	screen.loading = (b != 0)
}

//export SCR_IsDrawDialog
func SCR_IsDrawDialog() C.int {
	return b2i(screen.dialog)
}

//export SCR_SetDrawDialog
func SCR_SetDrawDialog(b int) {
	screen.dialog = (b != 0)
}

//export SCR_DrawLoading
func SCR_DrawLoading() {
	screen.drawLoading()
}

//export SCR_DrawTurtle
func SCR_DrawTurtle() {
	screen.drawTurtle()
}

//export SCR_DrawNet
func SCR_DrawNet() {
	screen.drawNet()
}

//export SCR_SetVRect
func SCR_SetVRect(x, y, w, h int) {
	screen.vrect = Rect{x: x, y: y, width: w, height: h}
}

//export SCR_GetVRectX
func SCR_GetVRectX() int {
	return screen.vrect.x
}

//export SCR_GetVRectY
func SCR_GetVRectY() int {
	return screen.vrect.y
}

//export SCR_GetVRectHeight
func SCR_GetVRectHeight() int {
	return screen.vrect.height
}

//export SCR_GetVRectWidth
func SCR_GetVRectWidth() int {
	return screen.vrect.width
}

//export SCR_UpdateDisabledTime
func SCR_UpdateDisabledTime() {
	screen.disabledTime = host.time
}

//export SCR_GetDisabledTime
func SCR_GetDisabledTime() C.double {
	return C.double(screen.disabledTime)
}

func (scr *qScreen) drawNet() {
	if host.time-cl.lastReceivedMessageTime < 0.3 {
		return
	}
	if cls.demoPlayback {
		return
	}
	if scr.netPic == nil {
		scr.netPic = GetPictureFromWad("net")
	}
	SetCanvas(CANVAS_DEFAULT)
	DrawPicture(scr.vrect.x, scr.vrect.y, scr.netPic)
}

func (scr *qScreen) drawTurtle() {
	if !cvars.ShowTurtle.Bool() {
		return
	}
	if scr.turtlePic == nil {
		scr.turtlePic = GetPictureFromWad("turtle")
	}
	if host.frameTime < 0.1 {
		scr.turtleCount = 0
	}
	scr.turtleCount++
	if scr.turtleCount < 3 {
		return
	}
	SetCanvas(CANVAS_DEFAULT)
	DrawPicture(scr.vrect.x, scr.vrect.y, scr.turtlePic)
}

func (scr *qScreen) drawLoading() {
	if !scr.loading {
		return // probably impossible to reach
	}
	SetCanvas(CANVAS_MENU)

	p := GetCachedPicture("gfx/loading.lmp")
	DrawPicture((320-p.width)/2, (240-48-p.height)/2, p)

	C.ResetTileClearUpdates()
}

func (scr *qScreen) drawClock() {
	if !cvars.ScreenClock.Bool() {
		return
	}
	t := int(cl.time)

	m, s := t/60, t%60
	str := fmt.Sprintf("%d:%02d", m, s)

	SetCanvas(CANVAS_BOTTOMRIGHT)
	DrawStringWhite(320-len(str)*8, 200-8, str)
	C.ResetTileClearUpdates()
}

func (s *qScreen) drawPause() {
	if !cl.paused {
		return
	}
	if !cvars.ShowPause.Bool() {
		return
	}
	SetCanvas(CANVAS_MENU)

	p := GetCachedPicture("gfx/pause.lmp")
	DrawPicture((320-p.width)/2, (240-48-p.height)/2, p)

	C.ResetTileClearUpdates()
}

func (s *qScreen) drawCrosshair() {
	if !cvars.Crosshair.Bool() {
		return
	}
	SetCanvas(CANVAS_CROSSHAIR)
	DrawCharacterWhite(-4, -4, '+')
}

func (s *qScreen) CenterPrint(str string) {
	s.centerTime = time.Now().Add(time.Second * 2) // scr_centertime
	s.centerString = strings.Split(str, "\n")
}

func (s *qScreen) drawCenterPrint() {
	SetCanvas(CANVAS_MENU)

	remaining := 9999
	if cl.intermission != 0 {
		r := time.Duration(cvars.ScreenPrintSpeed.Value()) * time.Now().Sub(s.centerTime)
		remaining = int(r.Seconds())
	}
	if remaining < 1 {
		return
	}

	// 320x200 coordinate system
	y := int(200 * 0.35)
	if len(s.centerString) > 4 {
		y = 48
	}
	if cvars.Crosshair.Bool() {
		y -= 8
	}

	for _, t := range s.centerString {
		runes := []rune(t)
		x := (320 - (len(runes) * 8)) / 2
		if remaining < len(runes) {
			runes = runes[:remaining]
		}
		l := len(runes)
		remaining -= l
		for _, r := range runes {
			DrawCharacterWhite(x, y, int(r))
			x += 8
		}

		if remaining < 1 {
			return
		}
		y += 8
	}

}

func (s *qScreen) CheckDrawCenterPrint() {
	if keyDestination != keys.Game {
		return
	}
	if cl.paused {
		return
	}
	if time.Now().After(s.centerTime) && cl.intermission == 0 {
		return
	}
	s.drawCenterPrint()
}

func ModalMessage(message string, timeout float32) bool {
	m := C.CString(message)
	defer C.free(unsafe.Pointer(m))
	return C.SCR_ModalMessage(m, C.float(timeout)) != 0
}

func SCR_BeginLoadingPlaque() {
	snd.StopAll()
	if cls.state != ca_connected {
		return
	}
	if cls.signon != 4 {
		return
	}
	console.ClearNotify()
	screen.centerTime = time.Now()

	C.SCR_BeginLoadingPlaque()
}

func SCR_EndLoadingPlaque() {
	screen.disabled = false
	console.ClearNotify()
}

func SCR_UpdateScreen() {
	C.SCR_UpdateScreen()
}

func init() {
	cmd.AddCommand("screenshot", screenShot)
	cmd.AddCommand("sizeup", screenSizeup)
	cmd.AddCommand("sizedown", screenSizedown)
}

func screenSizeup(_ []cmd.QArg, _ int) {
	cvars.ViewSize.SetValue(cvars.ViewSize.Value() + 10)
}

func screenSizedown(_ []cmd.QArg, _ int) {
	cvars.ViewSize.SetValue(cvars.ViewSize.Value() - 10)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func screenShot(_ []cmd.QArg, _ int) {
	var pngName string
	var fileName string
	for i := 0; i < 10000; i++ {
		pngName = fmt.Sprintf("spasm%04d.png", i)
		fileName = filepath.Join(gameDirectory, pngName)
		if !fileExists(fileName) {
			break
		}
		if i == 9999 {
			conlog.Printf("Coun't find an unused filename\n")
			return
		}
	}
	buffer := make([]byte, viewport.width*viewport.height*4)
	gl.PixelStorei(gl.PACK_ALIGNMENT, 1)
	gl.ReadPixels(viewport.x, viewport.y, viewport.width, viewport.height,
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(buffer))
	err := image.Write(fileName, buffer, int(viewport.width), int(viewport.height))
	if err != nil {
		conlog.Printf("Coudn't create screenshot file\n")
	} else {
		conlog.Printf("Wrote %s\n", pngName)
	}
}
