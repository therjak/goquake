package quakelib

//#ifndef KEYDEST_T
//#define KEYDEST_T
//typedef enum { key_game, key_console, key_message, key_menu } keydest_t;
//#endif
import "C"

import (
	"quake/keys"
)

var (
	keyDestination = keys.Game
)

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
