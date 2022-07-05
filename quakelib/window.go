// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"fmt"
	"strconv"
	"time"

	"goquake/cbuf"
	"goquake/cmd"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/keys"
	"goquake/menu"
	"goquake/window"

	"github.com/go-gl/gl/v4.6-core/gl"

	"github.com/veandco/go-sdl2/sdl"
)

var (
	videoChanged     = false
	glSwapControl    = false
	videoInitialized = false
	videoLocked      = false
)

func videoSetMode(width, height int32, fullscreen bool) {
	temp := screen.disabled
	screen.disabled = true

	window.SetMode(width, height, fullscreen)
	// setupGLState should get called whenever a new gl context gets created
	// and window.SetMode could have created a context
	// TODO: This window stuff needs a cleanup.
	setupGLState()
	w, h := window.Size()
	screen.UpdateSize(w, h)

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
		switch keyDestination {
		case keys.Console, keys.Menu:
			switch modestate {
			case MS_WINDOWED:
				inputDeactivate(true) // free cursor
			case MS_FULLSCREEN:
				inputActivate()
			}
		}
	}
	// this addition fixes at least the 'to fullscreen'
	// not sure what the issue is with 'from fullscreen' as it looks distorted
	width, height := window.Size()
	screen.UpdateSize(width, height)
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

func describeCurrentMode(_ []cmd.QArg, _ int) error {
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
	return nil
}

func describeModes(_ []cmd.QArg, _ int) error {
	count := 0
	for _, m := range availableDisplayModes {
		conlog.Printf("  %4d x %4d\n", m.Width, m.Height)
		count++
	}
	conlog.Printf("%d modes\n", count)
	return nil
}

func init() {
	addCommand("vid_describecurrentmode", describeCurrentMode)
	addCommand("vid_describemodes", describeModes)
	addCommand("vid_unlock", vidUnlock)
	addCommand("vid_restart", vidRestart)
	addCommand("vid_test", vidTest)
}

func vidRestart(_ []cmd.QArg, _ int) error {
	if videoLocked || !videoChanged {
		return nil
	}

	width := int32(cvars.VideoWidth.Value())
	height := int32(cvars.VideoHeight.Value())
	fullscreen := cvars.VideoFullscreen.Bool()

	if !validDisplayMode(width, height, fullscreen) {
		mode := "fullscreen"
		if !fullscreen {
			mode = "windowed"
		}
		conlog.Printf("%dx%d %s is not a valid mode\n", width, height, mode)
		return nil
	}

	videoSetMode(width, height, fullscreen)

	// warpimages needs to be recalculated
	textureManager.RecalcWarpImageSize(screen.Width, screen.Height)

	updateConsoleSize()
	// keep cvars in line with actual mode
	syncVideoCvars()

	qCanvas.UpdateSize()

	// update mouse grab
	switch keyDestination {
	case keys.Console, keys.Menu:
		switch modestate {
		case MS_WINDOWED:
			inputDeactivate(true) // free cursor
		case MS_FULLSCREEN:
			inputActivate()
		}
	}
	return nil
}

func vidTest(_ []cmd.QArg, _ int) error {
	if videoLocked || !videoChanged {
		return nil
	}
	oldWidth, oldHeight := window.Size()
	oldFullscreen := window.Fullscreen()

	if err := vidRestart(nil, 0); err != nil {
		return err
	}

	if !screen.ModalMessage("Would you like to keep this\nvideo mode? (y/n)\n", time.Second*5) {
		cvars.VideoWidth.SetValue(float32(oldWidth))
		cvars.VideoHeight.SetValue(float32(oldHeight))
		if oldFullscreen {
			cvars.VideoFullscreen.SetByString("1")
		} else {
			cvars.VideoFullscreen.SetByString("0")
		}
		if err := vidRestart(nil, 0); err != nil {
			return err
		}
	}
	return nil
}

func vidUnlock(_ []cmd.QArg, _ int) error {
	videoLocked = false
	syncVideoCvars()
	return nil
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

	videoSetMode(width, height, fullscreen)

	window.InitIcon()

	// QuakeSpasm: current vid settings should override config file settings.
	// so we have to lock the vid mode from now until after all config files are
	// read.
	videoLocked = true

	inputInit()
	printGLInfo()
	checkVSync()
	return nil
}

func checkVSync() {
	if !glSwapControl {
		conlog.Warning("vertical sync not supported (SDL_GL_SetSwapInterval failed)\n")
		return
	}
	i, _ := sdl.GLGetSwapInterval()
	wantVSync := cvars.VideoVerticalSync.Bool()
	switch {
	case i == -1:
		glSwapControl = false
		conlog.Warning("vertical sync not supported (SDL_GL_GetSwapInterval failed)\n")
	case i == 0 && wantVSync, i == 1 && !wantVSync:
		glSwapControl = false
		conlog.Warning("vertical sync not supported (swap_control doesn't match vid_vsync)\n")
	default:
		conlog.Printf("FOUND: SDL_GL_SetSwapInterval\n")
	}
}

func printGLInfo() {
	vendor := gl.GoStr(gl.GetString(gl.VENDOR))
	conlog.SafePrintf("GL_VENDOR: %s\n", vendor)

	renderer := gl.GoStr(gl.GetString(gl.RENDERER))
	conlog.SafePrintf("GL_RENDERER: %s\n", renderer)

	version := gl.GoStr(gl.GetString(gl.VERSION))
	conlog.SafePrintf("GL_VERSION: %s\n", version)

	if vendor == "Intel" {
		conlog.Printf("Intel Display Adapter detected, enabling gl_clear\n")
		cbuf.AddText("gl_clear 1\n") // queue to override config file setting
	}
}
