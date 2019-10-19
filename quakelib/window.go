package quakelib

//#ifndef HASMODESTATE
//#define HASMODESTATE
//typedef enum {MS_UNINIT, MS_WINDOWED, MS_FULLSCREEN} modestate_t;
//#endif
// void S_ClearBuffer();
import "C"

import (
	"quake/cbuf"
	"quake/cmd"
	"quake/conlog"
	"quake/cvar"
	"quake/cvars"
	"quake/keys"
	"quake/math"
	"quake/menu"
	"quake/window"
	"strconv"

	"github.com/veandco/go-sdl2/sdl"
)

var (
	videoChanged     = false
	glSwapControl    = false
	videoInitialized = false
	videoLocked      = false
)

func windowSetMode(width, height, bpp int32, fullscreen bool) {
	window.SetMode(width, height, bpp, fullscreen)
	screenWidth, screenHeight = window.Size()
	UpdateConsoleSize()
}

func videoSetMode(width, height, bpp int32, fullscreen bool) {
	temp := screen.disabled
	screen.disabled = true

	window.SetMode(width, height, bpp, fullscreen)
	screenWidth, screenHeight = window.Size()
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
	numPages = 2

	modestate = func() int {
		if window.Fullscreen() {
			return MS_FULLSCREEN
		}
		return MS_WINDOWED
	}()

	screen.disabled = temp

	Key_ClearStates()

	recalc_refdef = true
	videoChanged = true
}

type DisplayMode struct {
	Width        int32
	Height       int32
	BitsPerPixel []uint32 // sorted, never empty
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
		bpp := (mode.Format >> 8) & 0xff
		addMode(mode.W, mode.H, bpp)
	}
}

func appendMode(w, h int32, bpp uint32) {
	availableDisplayModes = append(availableDisplayModes,
		DisplayMode{
			Width:        w,
			Height:       h,
			BitsPerPixel: []uint32{bpp},
		})
}

func appendBpp(cur []uint32, n uint32) []uint32 {
	if len(cur) == 0 {
		return []uint32{n}
	}
	if cur[len(cur)-1] == n {
		return cur
	}
	return append(cur, n)
}

func addMode(w, h int32, bpp uint32) {
	if len(availableDisplayModes) == 0 {
		appendMode(w, h, bpp)
		return
	}
	last := availableDisplayModes[len(availableDisplayModes)-1]
	if last.Width == w && last.Height == h {
		availableDisplayModes[len(availableDisplayModes)-1].BitsPerPixel =
			appendBpp(last.BitsPerPixel, bpp)
	} else {
		appendMode(w, h, bpp)
	}

}

func hasDisplayMode(width, height int32, bpp uint32) bool {
	for _, m := range availableDisplayModes {
		if m.Height == height && m.Width == width {
			for _, b := range m.BitsPerPixel {
				if b == bpp {
					return true
				}
			}
		}
	}
	return false
}

func validDisplayMode(width, height int32, bpp uint32, fullscreen bool) bool {
	if fullscreen {
		if cvars.VideoDesktopFullscreen.Value() != 0 {
			return true
		}
	}

	if width < 320 || height < 200 {
		return false
	}

	if fullscreen && !hasDisplayMode(width, height, bpp) {
		return false
	}

	switch bpp {
	case 16, 24, 32:
		return true
	}

	return false
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
	C.S_ClearBuffer()
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
	screenWidth, screenHeight = window.Size()
	recalc_refdef = true
	UpdateConsoleSize()
}

var (
	screenWidth   int
	screenHeight  int
	recalc_refdef bool
	numPages      int
)

func init() {
	f := func(_ *cvar.Cvar) {
		recalc_refdef = true
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
			return int(float32(screenWidth) / cvars.ScreenConsoleScale.Value())
		}
		return screenWidth
	}()
	w = math.ClampI(320, w, screenWidth)
	w &= 0xFFFFFFF8

	console.width = int(w)
	console.height = console.width * screenHeight / screenWidth
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
	cvars.VideoBitsPerPixel.SetCallback(f)
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
		cvars.VideoBitsPerPixel.SetByString(strconv.FormatInt(int64(window.BPP()), 10))
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
		for _, d := range m.BitsPerPixel {
			conlog.Printf("  %4d x %4d x %d\n", m.Width, m.Height, d)
			count++
		}
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

func getIndexCurrentBpp(m int) (int, bool) {
	if m < 0 || m > len(availableDisplayModes) {
		return 0, false
	}
	cb := uint32(cvars.VideoBitsPerPixel.Value())
	for i, b := range availableDisplayModes[m].BitsPerPixel {
		if b == cb {
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

func chooseBpp(b uint32) {
	bs := strconv.FormatInt(int64(b), 10)
	cvars.VideoBitsPerPixel.SetByString(bs)
}

func chooseMode(f func(int) int) {
	mi, ok := getIndexCurrentDisplayMode()
	if !ok {
		if len(availableDisplayModes) != 0 {
			m := availableDisplayModes[0]
			chooseDisplayMode(m.Width, m.Height)
			chooseBpp(m.BitsPerPixel[0])
		}
		return
	}
	ni := f(mi)
	mode := availableDisplayModes[ni]
	chooseDisplayMode(mode.Width, mode.Height)
	_, ok = getIndexCurrentBpp(ni)
	if !ok {
		chooseBpp(mode.BitsPerPixel[0])
	}
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

func chooseNextBpp() {
	m, ok := getIndexCurrentDisplayMode()
	if !ok {
		return
	}
	i, ok := getIndexCurrentBpp(m)
	if !ok {
		chooseBpp(availableDisplayModes[m].BitsPerPixel[0])
		return
	}
	i = (i + 1) % len(availableDisplayModes[m].BitsPerPixel)
	chooseBpp(availableDisplayModes[m].BitsPerPixel[i])
}

func choosePrevBpp() {
	m, ok := getIndexCurrentDisplayMode()
	if !ok {
		return
	}
	i, ok := getIndexCurrentBpp(m)
	if !ok {
		chooseBpp(availableDisplayModes[m].BitsPerPixel[0])
		return
	}
	i = (i - 1 + len(availableDisplayModes[m].BitsPerPixel)) % len(availableDisplayModes[m].BitsPerPixel)
	chooseBpp(availableDisplayModes[m].BitsPerPixel[i])
}

//TODO
func VID_Init() {}

/*
SDL_GL_GetSwapInterval
int SDL_GetNumDisplayModes(int)
int SDL_GetDisplayMode(int,int,&mode)
*/
