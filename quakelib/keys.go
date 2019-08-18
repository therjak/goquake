package quakelib

//#ifndef KEYDEST_T
//#define KEYDEST_T
//typedef enum { key_game, key_console, key_message, key_menu } keydest_t;
//#endif
//void Key_EndChat(void);
import "C"

import (
	"fmt"
	"io"
	"quake/cmd"
	"quake/conlog"
	"quake/cvars"
	kc "quake/keycode"
	"quake/keys"
	"sort"
)

var (
	keyDestination = keys.Game
	// if true, can't be rebound while in console
	consoleKeys map[kc.KeyCode]bool
	// if true, can't be rebound while in menu
	menuBound   map[kc.KeyCode]bool
	keyDown     map[kc.KeyCode]bool
	keyBindings map[kc.KeyCode]string
)

func init() {
	consoleKeys = make(map[kc.KeyCode]bool)
	menuBound = make(map[kc.KeyCode]bool)
	keyDown = make(map[kc.KeyCode]bool)
	keyBindings = make(map[kc.KeyCode]string)

	for i := 32; i < 127; i++ {
		// ascii characters
		consoleKeys[kc.KeyCode(i)] = true
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
		consoleKeys[k] = true
	}
	// only on MAC?
	// consolekeys[K_COMMAND] = true;

	for _, k := range []kc.KeyCode{kc.ESCAPE,
		kc.F1, kc.F2, kc.F3, kc.F4, kc.F5,
		kc.F6, kc.F7, kc.F8, kc.F9, kc.F10,
		kc.F11, kc.F12} {
		menuBound[k] = true
	}

	cmd.AddCommand("bindlist", keyBindlist)
	cmd.AddCommand("bind", keyBind)
	cmd.AddCommand("unbind", keyUnbind)
	cmd.AddCommand("unbindall", keyUnbindAll)
}

func keyBindlist(args []cmd.QArg, _ int) {
	count := 0
	for k, v := range keyBindings {
		if v != "" {
			count++
			conlog.SafePrintf("  %s \"%s\"\n", kc.KeyToString(k), v)
		}
	}
	conlog.SafePrintf("%d bindings\n", count)
}

func keyUnbind(args []cmd.QArg, _ int) {
	if len(args) != 1 {
		conlog.Printf("unbind <key> : remove commands from a key\n")
		return
	}

	b := kc.StringToKey(args[0].String())
	if b == -1 {
		conlog.Printf("\"%s\" isn't a valid key\n", args[0].String())
		return
	}

	delete(keyBindings, b)
}

func keyUnbindAll(_ []cmd.QArg, _ int) {
	keyBindings = make(map[kc.KeyCode]string)
}

func keyBind(args []cmd.QArg, _ int) {
	c := len(args)
	if c != 1 && c != 2 {
		conlog.Printf("bind <key> [command] : attach a command to a key\n")
	}
	k := kc.StringToKey(args[0].String())
	if k == -1 {
		conlog.Printf("\"%s\" isn't a valid key\n", args[0].String())
		return
	}
	if c == 1 {
		if b, ok := keyBindings[k]; ok && b != "" {
			conlog.Printf("\"%s\" = \"%s\"\n", args[0].String(), b)
		} else {
			conlog.Printf("\"%s\" is not bound\n", args[0].String())
		}
		return
	}
	keyBindings[k] = args[1].String()
}

func writeKeyBindings(w io.Writer) {
	if cvars.CfgUnbindAll.Bool() {
		w.Write([]byte("unbindall\n"))
	}
	b := []string{}
	for k, c := range keyBindings {
		if c == "" {
			continue
		}
		b = append(b, fmt.Sprintf("bind \"%s\" \"%s\"\n", kc.KeyToString(k), c))
	}
	sort.Strings(b)
	for _, s := range b {
		w.Write([]byte(s))
	}
}

func getKeysForCommand(command string) (kc.KeyCode, kc.KeyCode, kc.KeyCode) {
	ks := kc.KeyCodeSlice{}
	for k, c := range keyBindings {
		if c == command {
			ks = append(ks, k)
		}
	}
	sort.Sort(ks)
	for len(ks) < 3 {
		ks = append(ks, kc.KeyCode(-1))
	}

	return ks[0], ks[1], ks[2]
}

func unbindCommand(command string) {
	for k, c := range keyBindings {
		if c == command {
			delete(keyBindings, k)
		}
	}
}

//export ConsoleKeys
func ConsoleKeys(k C.int) C.int {
	return b2i(consoleKeys[kc.KeyCode(k)])
}

//export MenuBound
func MenuBound(k C.int) C.int {
	return b2i(menuBound[kc.KeyCode(k)])
}

//export Key_SetBinding
func Key_SetBinding(keynum C.int, binding *C.char) {
	keyBindings[kc.KeyCode(keynum)] = C.GoString(binding)
}

//export Key_HasBinding
func Key_HasBinding(keynum C.int) C.int {
	return b2i("" != keyBindings[kc.KeyCode(keynum)])
}

//export Key_Bindings
func Key_Bindings(keynum C.int) *C.char {
	b := keyBindings[kc.KeyCode(keynum)]
	if b != "" {
		return C.CString(b)
	}
	return nil
}

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
