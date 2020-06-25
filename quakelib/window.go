package quakelib

import (
	"fmt"
	"strconv"

	"github.com/therjak/goquake/cbuf"
	"github.com/therjak/goquake/cmd"
	cmdl "github.com/therjak/goquake/commandline"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/cvar"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/keys"
	"github.com/therjak/goquake/math"
	"github.com/therjak/goquake/menu"
	"github.com/therjak/goquake/window"

	"github.com/veandco/go-sdl2/sdl"
)

var (
	videoChanged     = false
	glSwapControl    = false
	videoInitialized = false
	videoLocked      = false
)

/*
func windowSetMode(width, height, bpp int32, fullscreen bool) {
	window.SetMode(width, height, bpp, fullscreen)
	screen.Width, screen.Height = window.Size()
	UpdateConsoleSize()
}*/

func videoSetMode(width, height int32, fullscreen bool) {
	temp := screen.disabled
	screen.disabled = true

	window.SetMode(width, height, fullscreen)
	screen.Width, screen.Height = window.Size()
	UpdateConsoleSize()

	glSwapControl = true
	vsync := func() int {
		if cvars.VideoVerticalSync.Value() == 0 {
			return 0
		}
		return 1
	}()
	if err := sdl.GLSetSwapInterval(vsync); err != nil {
		glSwapControl = false
	}
	screen.numPages = 2

	modestate = func() int {
		if window.Fullscreen() {
			return MS_FULLSCREEN
		}
		return MS_WINDOWED
	}()

	screen.disabled = temp

	Key_ClearStates()

	screen.RecalcViewRect()
	videoChanged = true
}

type DisplayMode struct {
	Width  int32
	Height int32
}

var (
	// sorted alphabetically by width, height
	availableDisplayModes []DisplayMode
)

func updateAvailableDisplayModes() {
	const display = 0
	availableDisplayModes = availableDisplayModes[:0]
	// SDL says the returned modes are ordered by width,height,bpp,...
	num, err := sdl.GetNumDisplayModes(display)
	if err != nil {
		return
	}
	for i := 0; i < num; i++ {
		mode, err := sdl.GetDisplayMode(display, i)
		if err != nil {
			continue
		}
		addMode(mode.W, mode.H)
	}
}

func appendMode(w, h int32) {
	availableDisplayModes = append(availableDisplayModes,
		DisplayMode{
			Width:  w,
			Height: h,
		})
}

func addMode(w, h int32) {
	if len(availableDisplayModes) == 0 {
		appendMode(w, h)
		return
	}
	last := availableDisplayModes[len(availableDisplayModes)-1]
	if last.Width != w && last.Height != h {
		appendMode(w, h)
	}

}

func hasDisplayMode(width, height int32) bool {
	for _, m := range availableDisplayModes {
		if m.Height == height && m.Width == width {
			return true
		}
	}
	return false
}

func validDisplayMode(width, height int32, fullscreen bool) bool {
	if fullscreen {
		if cvars.VideoDesktopFullscreen.Value() != 0 {
			return true
		}
	}

	if width < 320 || height < 200 {
		return false
	}

	if fullscreen && !hasDisplayMode(width, height) {
		return false
	}

	return true
}

const (
	MS_UNINIT = iota
	MS_WINDOWED
	MS_FULLSCREEN
)

var (
	modestate = MS_UNINIT
)

func toggleFullScreen() {
	// This is buggy. It seems to miss changing the global 'vid' object and whatnot.
	flags := func() uint32 {
		if window.Fullscreen() {
			return 0 // windowed
		}
		if cvars.VideoDesktopFullscreen.Value() != 0 {
			return sdl.WINDOW_FULLSCREEN
		}
		return sdl.WINDOW_FULLSCREEN_DESKTOP
	}()
	w := window.Get()
	if err := w.SetFullscreen(flags); err != nil {
		if window.Fullscreen() {
			cvars.VideoFullscreen.SetByString("0")
		} else {
			cvars.VideoFullscreen.SetByString("1")
		}
		cbuf.AddText("vid_restart\n")
	} else {
		StatusbarChanged()
		if window.Fullscreen() {
			modestate = MS_FULLSCREEN
		} else {
			modestate = MS_WINDOWED
		}
		syncVideoCvars()
		if keyDestination == keys.Console || keyDestination == keys.Menu {
			switch modestate {
			case MS_WINDOWED:
				IN_Deactivate()
			case MS_FULLSCREEN:
				IN_Activate()
			}
		}
	}
	// this addition fixes at least the 'to fullscreen'
	// not sure what the issue is with 'from fullscreen' as it looks distorted
	screen.Width, screen.Height = window.Size()
	screen.RecalcViewRect()
	UpdateConsoleSize()
}

func init() {
	f := func(_ *cvar.Cvar) {
		screen.RecalcViewRect()
		UpdateConsoleSize()
	}
	cvars.ScreenConsoleWidth.SetCallback(f)
	cvars.ScreenConsoleScale.SetCallback(f)
}

func updateConsoleSize() {
	w := func() int {
		if cvars.ScreenConsoleWidth.Value() > 0 {
			return int(cvars.ScreenConsoleWidth.Value())
		}
		if cvars.ScreenConsoleScale.Value() > 0 {
			return int(float32(screen.Width) / cvars.ScreenConsoleScale.Value())
		}
		return screen.Width
	}()
	w = math.ClampI(320, w, screen.Width)
	w &= 0xFFFFFFF8

	console.width = int(w)
	console.height = console.width * screen.Height / screen.Width
}

func b2s(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func init() {
	f := func(_ *cvar.Cvar) { videoChanged = true }
	cvars.VideoFullscreen.SetCallback(f)
	cvars.VideoWidth.SetCallback(f)
	cvars.VideoHeight.SetCallback(f)
	cvars.VideoVerticalSync.SetCallback(f)
	cvars.VideoDesktopFullscreen.SetCallback(f)
	cvars.VideoBorderLess.SetCallback(f)
	cvars.VideoFsaa.SetCallback(func(cv *cvar.Cvar) {
		if !videoInitialized {
			return
		}
		conlog.Printf("%s %d requires engine restart to take effect\n",
			cv.Name(), int(cv.Value()))
	})
}

func syncVideoCvars() {
	if window.Get() != nil {
		if !window.DesktopFullscreen() {
			w, h := window.Size()
			cvars.VideoWidth.SetByString(strconv.FormatInt(int64(w), 10))
			cvars.VideoHeight.SetByString(strconv.FormatInt(int64(h), 10))
		}
		cvars.VideoFullscreen.SetByString(b2s(window.Fullscreen()))
		cvars.VideoVerticalSync.SetByString(b2s(window.VSync()))
	}

	videoChanged = false
}

func describeCurrentMode(_ []cmd.QArg, _ int) {
	if window.Get() != nil {
		w, h := window.Size()
		fs := func() string {
			if window.Fullscreen() {
				return "fullscreen"
			}
			return "windowed"
		}()
		conlog.Printf("%dx%dx%d %s\n", w, h, window.BPP(), fs)
	}
}

func describeModes(_ []cmd.QArg, _ int) {
	count := 0
	for _, m := range availableDisplayModes {
		conlog.Printf("  %4d x %4d\n", m.Width, m.Height)
		count++
	}
	conlog.Printf("%d modes\n", count)
}

func init() {
	cmd.AddCommand("vid_describecurrentmode", describeCurrentMode)
	cmd.AddCommand("vid_describemodes", describeModes)
	cmd.AddCommand("vid_unlock", vidUnlock)
}

func vidUnlock(_ []cmd.QArg, _ int) {
	videoLocked = false
	syncVideoCvars()
}

func videoShutdown() {
	inputDeactivate(true) // frow IN_Shutdown
	if videoInitialized {
		sdl.QuitSubSystem(sdl.INIT_VIDEO)
		window.Shutdown()
	}
}

func enterMenuVideo() {
	inputDeactivate(modestate == MS_WINDOWED)
	keyDestination = keys.Menu
	qmenu.state = menu.Video
	qmenu.playEnterSound = true

	syncVideoCvars()
}

func getIndexCurrentDisplayMode() (int, bool) {
	cw := int32(cvars.VideoWidth.Value())
	ch := int32(cvars.VideoHeight.Value())
	for i, m := range availableDisplayModes {
		if m.Width == cw && m.Height == ch {
			return i, true
		}
	}
	return 0, false
}

func chooseDisplayMode(w, h int32) {
	ws := strconv.FormatInt(int64(w), 10)
	hs := strconv.FormatInt(int64(h), 10)
	cvars.VideoWidth.SetByString(ws)
	cvars.VideoHeight.SetByString(hs)
}

func chooseMode(f func(int) int) {
	mi, ok := getIndexCurrentDisplayMode()
	if !ok {
		if len(availableDisplayModes) != 0 {
			m := availableDisplayModes[0]
			chooseDisplayMode(m.Width, m.Height)
		}
		return
	}
	ni := f(mi)
	mode := availableDisplayModes[ni]
	chooseDisplayMode(mode.Width, mode.Height)
}

func chooseNextMode() {
	chooseMode(func(i int) int {
		return (i + 1) % len(availableDisplayModes)
	})
}

func choosePrevMode() {
	chooseMode(func(i int) int {
		return ((i - 1) + len(availableDisplayModes)) % len(availableDisplayModes)
	})
}

func videoInit() error {
	err := sdl.InitSubSystem(sdl.INIT_VIDEO)
	if err != nil {
		return fmt.Errorf("Couldn't init SDL video: %v", err)
	}
	mode, err := sdl.GetDesktopDisplayMode(0) // TODO: fix multi monitor support
	if err != nil {
		return fmt.Errorf("Could not get desktop display mode")
	}

	// TODO(therjak): It would be good to have read the configs already
	// quakespams reads at least config.cfg here for its cvars. But cvars
	// exist in autoexec.cfg and default.cfg as well.

	updateAvailableDisplayModes()
	width := int32(cvars.VideoWidth.Value())
	height := int32(cvars.VideoHeight.Value())
	fullscreen := cvars.VideoFullscreen.Bool()

	if cmdl.Current() {
		width = mode.W
		height = mode.H
		fullscreen = true
	} else {
		clWidth := cmdl.Width()
		clHeight := cmdl.Height()
		if clWidth >= 0 {
			width = int32(clWidth)
			if clHeight < 0 {
				height = width * 3 / 4
			}
		}
		if clHeight >= 0 {
			height = int32(clHeight)
			if clWidth < 0 {
				width = height * 4 / 3
			}
		}
		if cmdl.Window() {
			fullscreen = false
		} else if cmdl.Fullscreen() {
			fullscreen = true
		}
	}
	if !validDisplayMode(width, height, fullscreen) {
		width = int32(cvars.VideoWidth.Value())
		height = int32(cvars.VideoHeight.Value())
		fullscreen = cvars.VideoFullscreen.Bool()
	}
	if !validDisplayMode(width, height, fullscreen) {
		width = 640
		height = 480
		fullscreen = false
	}
	videoInitialized = true

	window.InitIcon()

	videoSetMode(width, height, fullscreen)

	// QuakeSpasm: current vid settings should override config file settings.
	// so we have to lock the vid mode from now until after all config files are
	// read.
	videoLocked = true

	inputInit()
	return nil
}

func getSwapInterval() int {
	i, _ := sdl.GLGetSwapInterval()
	return i
}
