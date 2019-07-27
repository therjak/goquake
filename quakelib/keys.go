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

/*
char *keybindings[MAX_KEYS];
qboolean consolekeys[MAX_KEYS];  // if true, can't be rebound while in console
qboolean menubound[MAX_KEYS];    // if true, can't be rebound while in menu
qboolean keydown[MAX_KEYS];
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
