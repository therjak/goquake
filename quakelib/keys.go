package quakelib

//#ifndef KEYDEST_T
//#define KEYDEST_T
//typedef enum { key_game, key_console, key_message, key_menu } keydest_t;
//#endif
//void Key_EndChat(void);
import "C"

import (
	kc "quake/keycode"
	"quake/keys"
)

var (
	keyDestination = keys.Game
	// if true, can't be rebound while in console
	consolekeys map[kc.KeyCode]bool
	// if true, can't be rebound while in menu
	menubound map[kc.KeyCode]bool
	keydown   map[kc.KeyCode]bool
)

func init() {
	consolekeys = make(map[kc.KeyCode]bool)
	menubound = make(map[kc.KeyCode]bool)
	keydown = make(map[kc.KeyCode]bool)

	for i := 32; i < 127; i++ {
		// ascii characters
		consolekeys[kc.KeyCode(i)] = true
	}
	for _, k := range []kc.KeyCode{kc.TAB, kc.ENTER, kc.ESCAPE, kc.BACKSPACE,
		kc.UPARROW, kc.DOWNARROW, kc.LEFTARROW, kc.RIGHTARROW,
		kc.CTRL, kc.SHIFT,
		kc.INS, kc.DEL, kc.PGDN, kc.PGUP, kc.HOME, kc.END,
		kc.KP_NUMLOCK, kc.KP_SLASH, kc.KP_STAR, kc.KP_MINUS, kc.KP_HOME,
		kc.KP_UPARROW, kc.KP_PGUP, kc.KP_PLUS,
		kc.KP_LEFTARROW, kc.KP_5, kc.KP_RIGHTARROW,
		kc.KP_END, kc.KP_DOWNARROW, kc.KP_PGDN,
		kc.KP_ENTER, kc.KP_INS, kc.KP_DEL,
		kc.MWHEELUP, kc.MWHEELDOWN} {
		consolekeys[k] = true
	}
	// only on MAC?
	// consolekeys[K_COMMAND] = true;

	for _, k := range []kc.KeyCode{kc.ESCAPE,
		kc.F1, kc.F2, kc.F3, kc.F4, kc.F5,
		kc.F6, kc.F7, kc.F8, kc.F9, kc.F10,
		kc.F11, kc.F12} {
		menubound[k] = true
	}
}

/*
char *keybindings[MAX_KEYS];
*/

//export GetKeyDest
func GetKeyDest() C.keydest_t {
	switch keyDestination {
	default:
		return C.key_game
	case keys.Console:
		return C.key_console
	case keys.Message:
		return C.key_message
	case keys.Menu:
		return C.key_menu
	}
}

//export SetKeyDest
func SetKeyDest(k C.keydest_t) {
	switch k {
	default:
		keyDestination = keys.Game
	case C.key_console:
		keyDestination = keys.Console
	case C.key_message:
		keyDestination = keys.Message
	case C.key_menu:
		keyDestination = keys.Menu
	}
}

func keyEndChat() {
	C.Key_EndChat()
}

var (
	updateKeyDestForced = false
)

func updateKeyDest() {
	if cls.state == ca_dedicated {
		return
	}

	switch keyDestination {
	case keys.Console:
		if updateKeyDestForced && cls.state == ca_connected {
			updateKeyDestForced = false
			IN_Activate()
			keyDestination = keys.Game
		}
	case keys.Game:
		if cls.state != ca_connected {
			updateKeyDestForced = true
			IN_Deactivate()
			keyDestination = keys.Console
		} else {
			updateKeyDestForced = false
		}
	default:
		updateKeyDestForced = false
	}
}
