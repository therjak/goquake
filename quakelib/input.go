package quakelib

// TODO: switch to "github.com/go-gl/glfw/v3.2/glfw"
//       or        "github.com/vulkan-go/glfw/v3.3/glfw"

import "C"

import (
	cmdl "github.com/therjak/goquake/commandline"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/cvars"
	kc "github.com/therjak/goquake/keycode"
	"github.com/therjak/goquake/qtime"
	"github.com/therjak/goquake/snd"

	"github.com/veandco/go-sdl2/sdl"
)

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

func inputActivate() {
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

func inputInit() {
	textmode = keyTextEntry()
	selectTextMode(textmode)
	if !cmdl.Mouse() {
		ef := sdl.GetEventFilter()
		if ef != filterMouseEvents {
			sdl.SetEventFilter(filterMouseEvents, nil)
		}
	}
	inputActivate()
}

func updateInputMode() {
	want := keyTextEntry()
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

func sdlScancodeToQuake(e *sdl.KeyboardEvent) kc.KeyCode {
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
		return kc.KeyCode('a')
	case sdl.SCANCODE_B:
		return kc.KeyCode('b')
	case sdl.SCANCODE_C:
		return kc.KeyCode('c')
	case sdl.SCANCODE_D:
		return kc.KeyCode('d')
	case sdl.SCANCODE_E:
		return kc.KeyCode('e')
	case sdl.SCANCODE_F:
		return kc.KeyCode('f')
	case sdl.SCANCODE_G:
		return kc.KeyCode('g')
	case sdl.SCANCODE_H:
		return kc.KeyCode('h')
	case sdl.SCANCODE_I:
		return kc.KeyCode('i')
	case sdl.SCANCODE_J:
		return kc.KeyCode('j')
	case sdl.SCANCODE_K:
		return kc.KeyCode('k')
	case sdl.SCANCODE_L:
		return kc.KeyCode('l')
	case sdl.SCANCODE_M:
		return kc.KeyCode('m')
	case sdl.SCANCODE_N:
		return kc.KeyCode('n')
	case sdl.SCANCODE_O:
		return kc.KeyCode('o')
	case sdl.SCANCODE_P:
		return kc.KeyCode('p')
	case sdl.SCANCODE_Q:
		return kc.KeyCode('q')
	case sdl.SCANCODE_R:
		return kc.KeyCode('r')
	case sdl.SCANCODE_S:
		return kc.KeyCode('s')
	case sdl.SCANCODE_T:
		return kc.KeyCode('t')
	case sdl.SCANCODE_U:
		return kc.KeyCode('u')
	case sdl.SCANCODE_V:
		return kc.KeyCode('v')
	case sdl.SCANCODE_W:
		return kc.KeyCode('w')
	case sdl.SCANCODE_X:
		return kc.KeyCode('x')
	case sdl.SCANCODE_Y:
		return kc.KeyCode('y')
	case sdl.SCANCODE_Z:
		return kc.KeyCode('z')

	case sdl.SCANCODE_1:
		return kc.KeyCode('1')
	case sdl.SCANCODE_2:
		return kc.KeyCode('2')
	case sdl.SCANCODE_3:
		return kc.KeyCode('3')
	case sdl.SCANCODE_4:
		return kc.KeyCode('4')
	case sdl.SCANCODE_5:
		return kc.KeyCode('5')
	case sdl.SCANCODE_6:
		return kc.KeyCode('6')
	case sdl.SCANCODE_7:
		return kc.KeyCode('7')
	case sdl.SCANCODE_8:
		return kc.KeyCode('8')
	case sdl.SCANCODE_9:
		return kc.KeyCode('9')
	case sdl.SCANCODE_0:
		return kc.KeyCode('0')

	case sdl.SCANCODE_MINUS:
		return kc.KeyCode('-')
	case sdl.SCANCODE_EQUALS:
		return kc.KeyCode('=')
	case sdl.SCANCODE_LEFTBRACKET:
		return kc.KeyCode('[')
	case sdl.SCANCODE_RIGHTBRACKET:
		return kc.KeyCode(']')
	case sdl.SCANCODE_BACKSLASH:
		return kc.KeyCode('\\')
	case sdl.SCANCODE_NONUSHASH:
		return kc.KeyCode('#')
	case sdl.SCANCODE_SEMICOLON:
		return kc.KeyCode(';')
	case sdl.SCANCODE_APOSTROPHE:
		return kc.KeyCode('\'')
	case sdl.SCANCODE_GRAVE:
		return kc.KeyCode('`')
	case sdl.SCANCODE_COMMA:
		return kc.KeyCode(',')
	case sdl.SCANCODE_PERIOD:
		return kc.KeyCode('.')
	case sdl.SCANCODE_SLASH:
		return kc.KeyCode('/')
	case sdl.SCANCODE_NONUSBACKSLASH:
		return kc.KeyCode('\\')

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
	down := e.State == sdl.PRESSED
	switch e.Button {
	case sdl.BUTTON_LEFT:
		keyEvent(kc.MOUSE1, down)
	case sdl.BUTTON_MIDDLE:
		keyEvent(kc.MOUSE2, down)
	case sdl.BUTTON_RIGHT:
		keyEvent(kc.MOUSE3, down)
	case sdl.BUTTON_X1:
		keyEvent(kc.MOUSE4, down)
	case sdl.BUTTON_X2:
		keyEvent(kc.MOUSE5, down)
	default:
		conlog.Printf("Ignored event for mouse button %v\n", e.Button)
	}
}

func handleMouseWheelEvent(e *sdl.MouseWheelEvent) {
	// t.Timestamp, t.Type, t.Which, t.X, t.Y)
	if e.Y > 0 {
		keyEvent(kc.MWHEELUP, true)
		keyEvent(kc.MWHEELUP, false)
	} else if e.Y < 0 {
		keyEvent(kc.MWHEELDOWN, true)
		keyEvent(kc.MWHEELDOWN, false)
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
	key := sdlScancodeToQuake(e)
	keyEvent(key, e.State == sdl.PRESSED)
}

func handleTextInputEvent(e *sdl.TextInputEvent) {
	// e.Timestamp, e.Type, e.WindowID, string(e.Text[:]))
	// SDL2: We use SDL_TEXTINPUT for typing in the console / chat.
	// SDL2 uses the local keyboard layout and handles modifiers
	// (shift for uppercase, etc.) for us.
	if cvars.InputDebugKeys.Value() != 0 {
		printTextInputEvent(e)
	}
	for _, c := range e.GetText() {
		if c == 0 {
			break
		}
		if c&^0x7F == 0 {
			charEvent(c)
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
			cls.Disconnect()
			Sys_Quit()
		default:
			break
		}
	}
}
