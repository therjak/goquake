package quakelib

//#include <stdlib.h>
// void SCR_UpdateScreen(void);
// void ResetTileClearUpdates(void);
// int GetRFrameCount();
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

	"github.com/go-gl/gl/v4.6-core/gl"
)

var (
	screen qScreen
)

type fpsAccumulator struct {
	oldTime       time.Time
	lastFPS       float64
	oldFrameCount int
}

func (f *fpsAccumulator) Compute() float64 {
	t := time.Now()
	elapsed := t.Sub(f.oldTime).Seconds()

	fc := int(C.GetRFrameCount())
	fd := fc - f.oldFrameCount

	if elapsed < 0 || fd < 0 {
		// overflow or start
		f.oldTime = t
		f.oldFrameCount = fc
	} else if elapsed > 0.75 {
		f.lastFPS = float64(fd) / elapsed
		f.oldTime = t
		f.oldFrameCount = fc
	}
	return f.lastFPS
}

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

	modalMsg []string

	// needs to match host.time type but should probably be changed to a real time value
	disabledTime float64

	consoleLines int // console lines to display scr_con_current

	clearConsole int

	numPages int // double or tripple buffering

	fps fpsAccumulator
}

func (scr *qScreen) drawFPS() {
	fps := scr.fps.Compute()
	if !cvars.ScreenShowFps.Bool() {
		return
	}

	t := fmt.Sprintf("%4.0f fps", fps)
	x := 320 - len(t)*8
	y := 200 - 8
	if cvars.ScreenClock.Bool() {
		y -= 8
	}
	SetCanvas(CANVAS_BOTTOMRIGHT)
	DrawStringWhite(x, y, t)
	C.ResetTileClearUpdates()
}

//export SCR_DrawFPS
func SCR_DrawFPS() {
	screen.drawFPS()
}

//export SCR_SetUpToDrawConsole
func SCR_SetUpToDrawConsole() {
	screen.setupToDrawConsole()
}

//export GetScreenConsoleCurrentHeight
func GetScreenConsoleCurrentHeight() int {
	return screen.consoleLines
}

//export SCR_DrawConsole
func SCR_DrawConsole() {
	screen.drawConsole()
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

//export SCR_ModalMessage
func SCR_ModalMessage(c *C.char, timeout C.float) bool {
	return screen.ModalMessage(C.GoString(c), time.Second*time.Duration(timeout))
}

//export SCR_DrawNotifyString
func SCR_DrawNotifyString() {
	screen.drawNotifyString()
}

func (scr *qScreen) setupToDrawConsole() {
	console.CheckResize()
	if scr.loading {
		return
	}
	console.forceDuplication = (cls.signon != 4) || cl.worldModel == nil

	lines := 0
	if console.forceDuplication {
		lines = int(viewport.height)
		scr.consoleLines = lines
	} else if keyDestination == keys.Console {
		lines = int(viewport.height / 2)
	}

	if lines != scr.consoleLines {
		timeScale := cvars.HostTimeScale.Value()
		if timeScale <= 0 {
			timeScale = 1
		}
		t := float32(host.frameTime) / timeScale
		s := float32(viewport.height) / 600 // normalize for 800x600 screen
		d := int(cvars.ScreenConsoleSpeed.Value() * s * t)
		if scr.consoleLines < lines {
			scr.consoleLines += d
			if scr.consoleLines > lines {
				scr.consoleLines = lines
			}
		} else {
			scr.consoleLines -= d
			if scr.consoleLines < lines {
				scr.consoleLines = lines
			}
		}
	}

	scr.clearConsole++
	if scr.clearConsole < scr.numPages {
		statusbar.MarkChanged()
	}

	if !console.forceDuplication && scr.consoleLines != 0 {
		C.ResetTileClearUpdates()
	}

}

func (scr *qScreen) drawConsole() {
	if scr.consoleLines > 0 {
		console.Draw(scr.consoleLines)
		scr.clearConsole = 0
	} else {
		if keyDestination == keys.Game || keyDestination == keys.Message {
			console.DrawNotify()
		}
	}
}

func (scr *qScreen) drawNotifyString() {
	y := int(200 * 0.35)
	SetCanvas(CANVAS_MENU)
	for _, s := range scr.modalMsg {
		x := (320 - len(s)*8) / 2
		DrawStringWhite(x, y, s)
		y += 8
	}
}

func (scr *qScreen) ModalMessage(msg string, timeout time.Duration) bool {
	if cls.state == ca_dedicated {
		return true
	}
	scr.modalMsg = strings.Split(msg, "\n")

	scr.dialog = true
	C.SCR_UpdateScreen()
	scr.dialog = false

	// S_ClearBuffer // stop sounds

	return modalResult(timeout)
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

//export SCR_BeginLoadingPlaque
func SCR_BeginLoadingPlaque() {
	screen.BeginLoadingPlaque()
}

func (scr *qScreen) BeginLoadingPlaque() {
	snd.StopAll()
	if cls.state != ca_connected {
		return
	}
	if cls.signon != 4 {
		return
	}
	console.ClearNotify()
	scr.centerTime = time.Now()
	scr.consoleLines = 0

	scr.loading = true
	statusbar.MarkChanged()
	C.SCR_UpdateScreen()
	scr.loading = false

	scr.disabled = true
	scr.disabledTime = host.time
}

func (scr *qScreen) EndLoadingPlaque() {
	scr.disabled = false
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
