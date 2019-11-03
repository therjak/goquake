package quakelib

// int GetRFrameCount();
// void SCR_UpdateScreen2();
// void SetRefdefRect(int x, int y, int w, int h);
// void SetRefdefFov(float x, float y);
import "C"

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"quake/cmd"
	"quake/conlog"
	"quake/cvar"
	"quake/cvars"
	"quake/image"
	"quake/keys"
	"quake/snd"
	"quake/window"
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
	disabled       bool
	recalcViewRect bool

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

	tileClearUpdates int

	fovX float64
	fovY float64

	Width  int
	Height int

	initialized bool
}

func init() {
	f := func(_ *cvar.Cvar) {
		screen.RecalcViewRect()
	}
	cvars.ScreenStatusbarScale.SetCallback(f)
	cvars.ScreenStatusbarAlpha.SetCallback(f)
	cvars.Fov.SetCallback(f)
	cvars.FovAdapt.SetCallback(f)
	cvars.ViewSize.SetCallback(f)
}

func (scr *qScreen) RecalcViewRect() {
	scr.recalcViewRect = true
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
	scr.ResetTileClearUpdates()
}

//export SCR_InitGo
func SCR_InitGo() {
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

//export SCR_SetDrawLoading
func SCR_SetDrawLoading(b int) {
	screen.loading = (b != 0)
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

//export SCR_ModalMessage
func SCR_ModalMessage(c *C.char, timeout C.float) bool {
	return screen.ModalMessage(C.GoString(c), time.Second*time.Duration(timeout))
}

//export SCR_ResetTileClearUpdates
func SCR_ResetTileClearUpdates() {
	screen.ResetTileClearUpdates()
}

func (scr *qScreen) ResetTileClearUpdates() {
	scr.tileClearUpdates = 0
}

func (scr *qScreen) tileClear() {
	if scr.tileClearUpdates >= scr.numPages &&
		!cvars.GlClear.Bool() &&
		cvars.Gamma.Value() == 1 {
		return
	}
	scr.tileClearUpdates++

	h := int(viewport.height) - statusbar.Lines()
	if scr.vrect.x > 0 {
		sw := scr.vrect.x + scr.vrect.width
		DrawTileClear(0, 0, scr.vrect.x, h)
		DrawTileClear(sw, 0, int(viewport.width)-sw, h)
	}
	if scr.vrect.y > 0 {
		sh := scr.vrect.y + scr.vrect.height
		DrawTileClear(scr.vrect.x, 0, scr.vrect.width, scr.vrect.y)
		DrawTileClear(scr.vrect.x, sh, scr.vrect.width, h-sh)
	}
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
		scr.ResetTileClearUpdates()
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
	scr.Update()
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

	scr.ResetTileClearUpdates()
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
	scr.ResetTileClearUpdates()
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

	s.ResetTileClearUpdates()
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
	scr.Update()
	scr.loading = false

	scr.disabled = true
	scr.disabledTime = host.time
}

func (scr *qScreen) EndLoadingPlaque() {
	scr.disabled = false
	console.ClearNotify()
}

func (scr *qScreen) drawDevStats() {
	/*
	   void SCR_DrawDevStats(void) {
	     char str[40];
	     int y = 25 - 9;  // 9=number of lines to print
	     int x = 0;       // margin

	     if (!Cvar_GetValue(&devstats)) return;

	     GL_SetCanvas(CANVAS_BOTTOMLEFT);

	     DrawFillC(x, y * 8, 19 * 8, 9 * 8, 0, 0.5);  // dark rectangle

	     sprintf(str, "devstats |Curr Peak");
	     Draw_String(x, (y++) * 8 - x, str);

	     sprintf(str, "---------+---------");
	     Draw_String(x, (y++) * 8 - x, str);

	     sprintf(str, "Edicts   |%4i %4i", dev_stats.edicts, dev_peakstats.edicts);
	     Draw_String(x, (y++) * 8 - x, str);

	     sprintf(str, "Packet   |%4i %4i", dev_stats.packetsize,
	             dev_peakstats.packetsize);
	     Draw_String(x, (y++) * 8 - x, str);

	     sprintf(str, "Visedicts|%4i %4i", dev_stats.visedicts,
	             dev_peakstats.visedicts);
	     Draw_String(x, (y++) * 8 - x, str);

	     sprintf(str, "Efrags   |%4i %4i", dev_stats.efrags, dev_peakstats.efrags);
	     Draw_String(x, (y++) * 8 - x, str);

	     sprintf(str, "Dlights  |%4i %4i", dev_stats.dlights, dev_peakstats.dlights);
	     Draw_String(x, (y++) * 8 - x, str);

	     sprintf(str, "Beams    |%4i %4i", dev_stats.beams, dev_peakstats.beams);
	     Draw_String(x, (y++) * 8 - x, str);

	     sprintf(str, "Tempents |%4i %4i", dev_stats.tempents, dev_peakstats.tempents);
	     Draw_String(x, (y++) * 8 - x, str);
	   }*/
}

//export SCR_UpdateScreen
func SCR_UpdateScreen() {
	screen.Update()
}

func (scr *qScreen) calcViewRect() {
	// TODO: figure out what Refdef is and rename this stuff

	statusbar.MarkChanged()
	scr.ResetTileClearUpdates()

	// SetByString is the faster one
	if cvars.ViewSize.Value() < 30 {
		cvars.ViewSize.SetByString("30")
	} else if cvars.ViewSize.Value() > 120 {
		cvars.ViewSize.SetByString("120")
	}
	fovx := float64(cvars.Fov.Value())
	if fovx < 10 {
		cvars.Fov.SetByString("10")
		fovx = 10
	} else if fovx > 170 {
		cvars.Fov.SetByString("170")
		fovx = 170
	}

	scr.recalcViewRect = false

	size := cvars.ViewSize.Value()
	if size > 100 {
		size = 1
	} else {
		size /= 100
	}
	w := float32(viewport.width) * size
	if w < 96 {
		w = 96 // lower limit for icons
	}
	h := float32(viewport.height) * size
	hbound := float32(viewport.height) - float32(statusbar.Lines())
	if h > hbound {
		h = hbound // keep space for the statusbar
	}
	x := (float32(viewport.width) - w) / 2
	y := (float32(viewport.height) - float32(statusbar.Lines()) - h) / 2

	scr.vrect = Rect{
		x:      int(x),
		y:      int(y),
		width:  int(w),
		height: int(h),
	}
	sh := float64(scr.Height)
	sw := float64(scr.Width)

	if cvars.FovAdapt.Bool() {
		x := sh / sw
		if x != 0.75 {
			fovx = math.Atan(0.75/x*math.Tan(fovx/360*math.Pi)) * 360 / math.Pi
			if fovx < 1 {
				fovx = 1
			} else if fovx > 179 {
				fovx = 179
			}
		}
	}
	fovy := math.Atan(sh/(sw/math.Tan(fovx/360*math.Pi))) * 360 / math.Pi
	scr.fovX = fovx
	scr.fovY = fovy

	// notify the C side
	C.SetRefdefRect(C.int(scr.vrect.x), C.int(scr.vrect.y),
		C.int(scr.vrect.width), C.int(scr.vrect.height))
	C.SetRefdefFov(C.float(scr.fovX), C.float(scr.fovY))
}

func (scr *qScreen) Update() {
	scr.numPages = 2
	if cvars.GlTripleBuffer.Bool() {
		scr.numPages = 3
	}

	if scr.disabled {
		if host.time-scr.disabledTime > 60 {
			scr.disabled = false
			conlog.Printf("load failed.\n")
		} else {
			return
		}
	}

	if !scr.initialized || !console.initialized {
		return
	}

	UpdateViewport()

	if scr.recalcViewRect {
		scr.calcViewRect()
	}

	scr.setupToDrawConsole()

	view.Render()
	GL_Set2D()

	scr.tileClear()

	if scr.dialog {
		// new game confirm
		if console.forceDuplication {
			DrawConsoleBackground()
		} else {
			statusbar.Draw()
		}
		DrawFadeScreen()
		scr.drawNotifyString()
	} else if scr.loading {
		scr.drawLoading()
		statusbar.Draw()
	} else if cl.intermission == 1 && keyDestination == keys.Game {
		// end of level
		statusbar.IntermissionOverlay()
	} else if cl.intermission == 2 && keyDestination == keys.Game {
		// end of episode
		statusbar.FinaleOverlay()
		scr.CheckDrawCenterPrint()
	} else {
		scr.drawCrosshair()
		scr.drawNet()
		scr.drawTurtle()
		scr.drawPause()
		scr.CheckDrawCenterPrint()
		statusbar.Draw()
		scr.drawDevStats()
		scr.drawFPS()
		scr.drawClock()
		scr.drawConsole()
		qmenu.Draw()
	}

	view.UpdateBlend()
	GLSLGamma_GammaCorrect()
	window.EndRendering()
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
