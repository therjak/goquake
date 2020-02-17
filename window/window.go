package window

import (
	"log"
	"quake/cvars"
	"unsafe"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	window      *sdl.Window
	context     sdl.GLContext
	skipUpdates bool
)

func Get() *sdl.Window {
	return window
}

func Size() (int, int) {
	w, h := window.GetSize()
	return int(w), int(h)
}

func Shutdown() {
	sdl.GLDeleteContext(context)
	context = nil
	window.Destroy()
	window = nil
}

func InitIcon() {
	rwop, err := sdl.RWFromMem(icon)
	if err != nil {
		log.Printf("Could not create icon: %v", err)
		return
	}
	bmp, err := sdl.LoadBMPRW(rwop, true)
	if err != nil {
		log.Printf("Could not load icon bmp: %v", err)
		return
	}
	ck := sdl.MapRGB(bmp.Format, 255, 0, 255)
	bmp.SetColorKey(true, ck)
	window.SetIcon(bmp)
	bmp.Free()
}

func Fullscreen() bool {
	return window.GetFlags()&sdl.WINDOW_FULLSCREEN != 0
}

func DesktopFullscreen() bool {
	return window.GetFlags()&sdl.WINDOW_FULLSCREEN_DESKTOP != 0
}

func VSync() bool {
	i, _ := sdl.GLGetSwapInterval()
	return i == 1
}

func InputFocus() bool {
	return window.GetFlags()&(sdl.WINDOW_MOUSE_FOCUS|sdl.WINDOW_INPUT_FOCUS) != 0
}

func Minimized() bool {
	return window.GetFlags()&sdl.WINDOW_SHOWN == 0
}

func BPP() int {
	pf, err := window.GetPixelFormat()
	if err != nil {
		return 32
	}
	return int((pf >> 8) & 0xff)
}

func findDisplayMode(width, height, bpp int32) *sdl.DisplayMode {
	num, _ := sdl.GetNumDisplayModes(0)
	for i := 0; i < num; i++ {
		m, err := sdl.GetDisplayMode(0, i)
		if err != nil {
			continue
		}
		mbpp, _, _, _, _, _ := sdl.PixelFormatEnumToMasks(uint(m.Format))
		if m.W == width && m.H == height && mbpp == int(bpp) {
			return &m
		}
	}
	return nil
}

func SetMode(width, height, bpp int32, fullscreen bool) {
	depthbits, stencilbits := func() (int, int) {
		if bpp == 16 {
			return 16, 0
		}
		return 24, 8
	}()
	sdl.GLSetAttribute(sdl.GL_DEPTH_SIZE, depthbits)
	sdl.GLSetAttribute(sdl.GL_STENCIL_SIZE, stencilbits)

	fsaa := int(cvars.VideoFsaa.Value())

	sdl.GLSetAttribute(sdl.GL_MULTISAMPLEBUFFERS, func() int {
		if fsaa > 0 {
			return 1
		}
		return 0
	}())
	sdl.GLSetAttribute(sdl.GL_MULTISAMPLESAMPLES, fsaa)

	if window == nil {
		flags := uint32(sdl.WINDOW_OPENGL | sdl.WINDOW_HIDDEN)
		if cvars.VideoBorderLess.Value() != 0 {
			flags |= sdl.WINDOW_BORDERLESS
		}
		window = func() *sdl.Window {
			w, err := sdl.CreateWindow("GoQuake", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, width, height, flags)
			if err == nil {
				return w
			}
			sdl.GLSetAttribute(sdl.GL_MULTISAMPLEBUFFERS, 0)
			sdl.GLSetAttribute(sdl.GL_MULTISAMPLESAMPLES, 0)
			w, err = sdl.CreateWindow("GoQuake", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, width, height, flags)
			if err == nil {
				return w
			}
			sdl.GLSetAttribute(sdl.GL_DEPTH_SIZE, 16)
			w, err = sdl.CreateWindow("GoQuake", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, width, height, flags)
			if err == nil {
				return w
			}
			sdl.GLSetAttribute(sdl.GL_STENCIL_SIZE, 0)
			w, err = sdl.CreateWindow("GoQuake", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, width, height, flags)
			if err == nil {
				return w
			}
			log.Fatalf("Couldn't create window")
			return nil
		}()
	}
	if Fullscreen() {
		if err := window.SetFullscreen(0); err != nil {
			log.Fatalf("Couln't set fullscreen state mode")
		}
	}
	window.SetSize(width, height)
	window.SetPosition(sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED)
	window.SetDisplayMode(findDisplayMode(width, height, bpp))
	window.SetBordered(cvars.VideoBorderLess.Value() == 0)
	if fullscreen {
		flags := func() uint32 {
			if cvars.VideoDesktopFullscreen.Value() != 0 {
				return sdl.WINDOW_FULLSCREEN_DESKTOP
			}
			return sdl.WINDOW_FULLSCREEN
		}()
		if err := window.SetFullscreen(flags); err != nil {
			log.Fatalf("Couln't set fullscreen state mode: %v", err)
		}
	}

	window.Show()

	if context == nil {
		var err error
		context, err = window.GLCreateContext()
		if err != nil {
			log.Fatalf("Couln't create GL context: %v", err)
		}
		// Initialize Glow
		if err := gl.Init(); err != nil {
			log.Fatalf("Couln't init gl: %v", err)
		}
		gl.DebugMessageCallback(debugCb, unsafe.Pointer(nil))
	}
}

func debugCb(
	source uint32,
	gltype uint32,
	id uint32,
	severity uint32,
	length int32,
	message string,
	userParam unsafe.Pointer) {
	if severity == gl.DEBUG_SEVERITY_HIGH {
		log.Panicf("[GL_DEBUG] source %d gltype %d id %d severity %d length %d: %s", source, gltype, id, severity, length, message)
	} else {
		log.Printf("[GL_DEBUG] source %d gltype %d id %d severity %d length %d: %s", source, gltype, id, severity, length, message)
	}
}

func SetSkipUpdates(skip bool) {
	skipUpdates = skip
}

func EndRendering() {
	if skipUpdates {
		return
	}
	window.GLSwap()
}
