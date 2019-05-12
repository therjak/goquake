package quakelib

// TODO: switch to "github.com/go-gl/glfw/v3.2/glfw"
//       or        "github.com/vulkan-go/glfw/v3.3/glfw"

// void Char_Event(int key);
// void Key_Event(int key, int down);
// void CL_Disconnect(void);
// int Key_TextEntry(void);
import "C"

import (
	cmdl "quake/commandline"
	"quake/conlog"
	"quake/cvars"
	"quake/input"
	kc "quake/keycode"
	"quake/qtime"
	"quake/snd"

	"github.com/veandco/go-sdl2/sdl"
)

//export CL_KeyMLookDown
func CL_KeyMLookDown() C.int {
	if input.MLook.Down() {
		return 1
	}
	return 0
}

func printKeyEvent(e *sdl.KeyboardEvent) {
	var etype string
	if e.State == sdl.PRESSED {
		etype = "SDL_KEYDOWN"
	} else {
		etype = "SDL_KEYUP"
	}
	conlog.Printf("%v scancode: '%v', keycode: '%v', time: %v", etype,
		sdl.GetScancodeName(e.Keysym.Scancode),
		sdl.GetKeyName(e.Keysym.Sym), qtime.QTime().Seconds())
}

func printTextInputEvent(e *sdl.TextInputEvent) {
	// e.Timestamp, e.Type, e.WindowID, e.Text
	conlog.Printf("SDL_TEXTINPUT '%s' time: %d\n", e.Text, e.Timestamp)
}

//export IN_SendKeyEvents
func IN_SendKeyEvents() {
	// TODO? Handle some joystick stuff
	sendKeyEvents()
}

type mouseFilter struct{}

func (mouseFilter) FilterEvent(e sdl.Event, userdata interface{}) bool {
	switch e.GetType() {
	case sdl.MOUSEMOTION:
		return false
	default:
		return true
	}
}

var filterMouseEvents mouseFilter

//export IN_Activate
func IN_Activate() {
	if !cmdl.Mouse() {
		return
	}
	if sdl.SetRelativeMouseMode(true) != 0 {
		conlog.Printf("WARNING: SDL_SetRelativeMouseMode(SDL_TRUE) failed.\n")
	}
	sdl.SetEventFilter(nil, nil)
	resetMouseMotion()
}

var (
	textmode = false // to make entering and leaving text mode lasy
)

//export IN_Init
func IN_Init() {
	textmode = (C.Key_TextEntry() != 0)
	selectTextMode(textmode)
	if !cmdl.Mouse() {
		ef := sdl.GetEventFilter()
		if ef != filterMouseEvents {
			sdl.SetEventFilter(filterMouseEvents, nil)
		}
	}
	IN_Activate()
}

//export IN_UpdateInputMode
func IN_UpdateInputMode() {
	want := (C.Key_TextEntry() != 0)
	if textmode != want {
		textmode = want
		selectTextMode(textmode)
	}
}

func selectTextMode(tm bool) {
	if tm {
		sdl.StartTextInput()
		if cvars.InputDebugKeys.Value() != 0 {
			conlog.Printf("SDL_StartTextInput time: %v\n", qtime.QTime().Seconds())
		}
	} else {
		sdl.StopTextInput()
		if cvars.InputDebugKeys.Value() != 0 {
			conlog.Printf("SDL_StopTextInput time: %v\n", qtime.QTime().Seconds())
		}
	}
}

//export IN_Deactivate
func IN_Deactivate() {
	inputDeactivate(modestate == MS_WINDOWED)
}

func inputDeactivate(freeCursor bool) {
	// free_cursor is qboolean
	if !cmdl.Mouse() {
		return
	}
	if freeCursor {
		sdl.SetRelativeMouseMode(false)
	}
	ef := sdl.GetEventFilter()
	if ef != filterMouseEvents {
		sdl.SetEventFilter(filterMouseEvents, nil)
	}
}

func sdlScancodeToQuake(e *sdl.KeyboardEvent) int {
	// We want the key and not what it is mapped to. So use Scancode
	switch e.Keysym.Scancode {
	case sdl.SCANCODE_TAB:
		return kc.TAB
	case sdl.SCANCODE_RETURN:
		return kc.ENTER
	case sdl.SCANCODE_RETURN2:
		return kc.ENTER
	case sdl.SCANCODE_ESCAPE:
		return kc.ESCAPE
	case sdl.SCANCODE_SPACE:
		return kc.SPACE
	case sdl.SCANCODE_A:
		return int('a')
	case sdl.SCANCODE_B:
		return int('b')
	case sdl.SCANCODE_C:
		return int('c')
	case sdl.SCANCODE_D:
		return int('d')
	case sdl.SCANCODE_E:
		return int('e')
	case sdl.SCANCODE_F:
		return int('f')
	case sdl.SCANCODE_G:
		return int('g')
	case sdl.SCANCODE_H:
		return int('h')
	case sdl.SCANCODE_I:
		return int('i')
	case sdl.SCANCODE_J:
		return int('j')
	case sdl.SCANCODE_K:
		return int('k')
	case sdl.SCANCODE_L:
		return int('l')
	case sdl.SCANCODE_M:
		return int('m')
	case sdl.SCANCODE_N:
		return int('n')
	case sdl.SCANCODE_O:
		return int('o')
	case sdl.SCANCODE_P:
		return int('p')
	case sdl.SCANCODE_Q:
		return int('q')
	case sdl.SCANCODE_R:
		return int('r')
	case sdl.SCANCODE_S:
		return int('s')
	case sdl.SCANCODE_T:
		return int('t')
	case sdl.SCANCODE_U:
		return int('u')
	case sdl.SCANCODE_V:
		return int('v')
	case sdl.SCANCODE_W:
		return int('w')
	case sdl.SCANCODE_X:
		return int('x')
	case sdl.SCANCODE_Y:
		return int('y')
	case sdl.SCANCODE_Z:
		return int('z')

	case sdl.SCANCODE_1:
		return int('1')
	case sdl.SCANCODE_2:
		return int('2')
	case sdl.SCANCODE_3:
		return int('3')
	case sdl.SCANCODE_4:
		return int('4')
	case sdl.SCANCODE_5:
		return int('5')
	case sdl.SCANCODE_6:
		return int('6')
	case sdl.SCANCODE_7:
		return int('7')
	case sdl.SCANCODE_8:
		return int('8')
	case sdl.SCANCODE_9:
		return int('9')
	case sdl.SCANCODE_0:
		return int('0')

	case sdl.SCANCODE_MINUS:
		return int('-')
	case sdl.SCANCODE_EQUALS:
		return int('=')
	case sdl.SCANCODE_LEFTBRACKET:
		return int('[')
	case sdl.SCANCODE_RIGHTBRACKET:
		return int(']')
	case sdl.SCANCODE_BACKSLASH:
		return int('\\')
	case sdl.SCANCODE_NONUSHASH:
		return int('#')
	case sdl.SCANCODE_SEMICOLON:
		return int(';')
	case sdl.SCANCODE_APOSTROPHE:
		return int('\'')
	case sdl.SCANCODE_GRAVE:
		return int('`')
	case sdl.SCANCODE_COMMA:
		return int(',')
	case sdl.SCANCODE_PERIOD:
		return int('.')
	case sdl.SCANCODE_SLASH:
		return int('/')
	case sdl.SCANCODE_NONUSBACKSLASH:
		return int('\\')

	case sdl.SCANCODE_BACKSPACE:
		return kc.BACKSPACE
	case sdl.SCANCODE_UP:
		return kc.UPARROW
	case sdl.SCANCODE_DOWN:
		return kc.DOWNARROW
	case sdl.SCANCODE_LEFT:
		return kc.LEFTARROW
	case sdl.SCANCODE_RIGHT:
		return kc.RIGHTARROW

	case sdl.SCANCODE_LALT:
		return kc.ALT
	case sdl.SCANCODE_RALT:
		return kc.ALT
	case sdl.SCANCODE_LCTRL:
		return kc.CTRL
	case sdl.SCANCODE_RCTRL:
		return kc.CTRL
	case sdl.SCANCODE_LSHIFT:
		return kc.SHIFT
	case sdl.SCANCODE_RSHIFT:
		return kc.SHIFT

	case sdl.SCANCODE_F1:
		return kc.F1
	case sdl.SCANCODE_F2:
		return kc.F2
	case sdl.SCANCODE_F3:
		return kc.F3
	case sdl.SCANCODE_F4:
		return kc.F4
	case sdl.SCANCODE_F5:
		return kc.F5
	case sdl.SCANCODE_F6:
		return kc.F6
	case sdl.SCANCODE_F7:
		return kc.F7
	case sdl.SCANCODE_F8:
		return kc.F8
	case sdl.SCANCODE_F9:
		return kc.F9
	case sdl.SCANCODE_F10:
		return kc.F10
	case sdl.SCANCODE_F11:
		return kc.F11
	case sdl.SCANCODE_F12:
		return kc.F12
	case sdl.SCANCODE_INSERT:
		return kc.INS
	case sdl.SCANCODE_DELETE:
		return kc.DEL
	case sdl.SCANCODE_PAGEDOWN:
		return kc.PGDN
	case sdl.SCANCODE_PAGEUP:
		return kc.PGUP
	case sdl.SCANCODE_HOME:
		return kc.HOME
	case sdl.SCANCODE_END:
		return kc.END

	case sdl.SCANCODE_NUMLOCKCLEAR:
		return kc.KP_NUMLOCK
	case sdl.SCANCODE_KP_DIVIDE:
		return kc.KP_SLASH
	case sdl.SCANCODE_KP_MULTIPLY:
		return kc.KP_STAR
	case sdl.SCANCODE_KP_MINUS:
		return kc.KP_MINUS
	case sdl.SCANCODE_KP_7:
		return kc.KP_HOME
	case sdl.SCANCODE_KP_8:
		return kc.KP_UPARROW
	case sdl.SCANCODE_KP_9:
		return kc.KP_PGUP
	case sdl.SCANCODE_KP_PLUS:
		return kc.KP_PLUS
	case sdl.SCANCODE_KP_4:
		return kc.KP_LEFTARROW
	case sdl.SCANCODE_KP_5:
		return kc.KP_5
	case sdl.SCANCODE_KP_6:
		return kc.KP_RIGHTARROW
	case sdl.SCANCODE_KP_1:
		return kc.KP_END
	case sdl.SCANCODE_KP_2:
		return kc.KP_DOWNARROW
	case sdl.SCANCODE_KP_3:
		return kc.KP_PGDN
	case sdl.SCANCODE_KP_ENTER:
		return kc.KP_ENTER
	case sdl.SCANCODE_KP_0:
		return kc.KP_INS
	case sdl.SCANCODE_KP_PERIOD:
		return kc.KP_DEL

	case sdl.SCANCODE_LGUI:
		return kc.COMMAND
	case sdl.SCANCODE_RGUI:
		return kc.COMMAND

	case sdl.SCANCODE_PAUSE:
		return kc.PAUSE
	}
	return 0
}

func handleMouseButtonEvent(e *sdl.MouseButtonEvent) {
	down := func() C.int {
		if e.State == sdl.PRESSED {
			return 1
		}
		return 0
	}()
	switch e.Button {
	case sdl.BUTTON_LEFT:
		C.Key_Event(kc.MOUSE1, down)
	case sdl.BUTTON_MIDDLE:
		C.Key_Event(kc.MOUSE2, down)
	case sdl.BUTTON_RIGHT:
		C.Key_Event(kc.MOUSE3, down)
	case sdl.BUTTON_X1:
		C.Key_Event(kc.MOUSE4, down)
	case sdl.BUTTON_X2:
		C.Key_Event(kc.MOUSE5, down)
	default:
		conlog.Printf("Ignored event for mouse button %v\n", e.Button)
	}
}

func handleMouseWheelEvent(e *sdl.MouseWheelEvent) {
	// t.Timestamp, t.Type, t.Which, t.X, t.Y)
	if e.Y > 0 {
		C.Key_Event(kc.MWHEELUP, 1)
		C.Key_Event(kc.MWHEELUP, 0)
	} else if e.Y < 0 {
		C.Key_Event(kc.MWHEELDOWN, 1)
		C.Key_Event(kc.MWHEELDOWN, 0)
	}
}

func handleMouseMotionEvent(e *sdl.MouseMotionEvent) {
	// t.Timestamp, t.Type, t.Which, t.X, t.Y, t.XRel, t.YRel
	mouseMotion(int(e.XRel), int(e.YRel))
}

func handleKeyboardEvent(e *sdl.KeyboardEvent) {
	// t.Timestamp, t.Type, t.Keysym.Sym, t.Keysym.Mod, t.State, t.Repeat
	if cvars.InputDebugKeys.Value() != 0 {
		printKeyEvent(e)
	}
	key := C.int(sdlScancodeToQuake(e))
	down := func() C.int {
		if e.State == sdl.PRESSED {
			return 1
		}
		return 0
	}()
	C.Key_Event(key, down)
}

func handleTextInputEvent(e *sdl.TextInputEvent) {
	// e.Timestamp, e.Type, e.WindowID, string(e.Text[:]))
	// SDL2: We use SDL_TEXTINPUT for typing in the console / chat.
	// SDL2 uses the local keyboard layout and handles modifiers
	// (shift for uppercase, etc.) for us.
	if cvars.InputDebugKeys.Value() != 0 {
		printTextInputEvent(e)
	}
	for _, c := range e.Text {
		if c == 0 {
			break
		}
		if c&^0x7F == 0 {
			C.Char_Event(C.int(c))
		}
	}
}

func handleWindowEvent(e *sdl.WindowEvent) {
	// t.Timestamp, t.Type, t.WindowID, t.Event, t.Data1, t.Data2
	switch e.Event {
	case sdl.WINDOWEVENT_FOCUS_GAINED:
		snd.Unblock()
	case sdl.WINDOWEVENT_FOCUS_LOST:
		snd.Block()
	}
}

func sendKeyEvents() {
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch t := event.(type) {
		case *sdl.WindowEvent:
			handleWindowEvent(t)
		case *sdl.TextInputEvent:
			handleTextInputEvent(t)
		case *sdl.KeyboardEvent:
			handleKeyboardEvent(t)
		case *sdl.MouseButtonEvent:
			handleMouseButtonEvent(t)
		case *sdl.MouseWheelEvent:
			handleMouseWheelEvent(t)
		case *sdl.MouseMotionEvent:
			handleMouseMotionEvent(t)
		case *sdl.QuitEvent:
			C.CL_Disconnect()
			Sys_Quit()
		default:
			break
		}
	}
}
