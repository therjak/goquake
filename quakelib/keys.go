// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unsafe"

	"goquake/cbuf"
	"goquake/conlog"
	"goquake/cvars"
	qhistory "goquake/history"
	kc "goquake/keycode"
	"goquake/keys"

	"github.com/veandco/go-sdl2/sdl"
)

var (
	keyDestination = keys.Game
	// if true, can't be rebound while in console
	consoleKeys map[kc.KeyCode]bool
	// if true, can't be rebound while in menu
	menuBound   map[kc.KeyCode]bool
	keyDown     map[kc.KeyCode]bool
	keyBindings map[kc.KeyCode]string

	keyInput qKeyInput
	history  qhistory.History
)

type qKeyInput struct {
	buf        []byte
	cursorXPos int
	insert     bool
	blinkTime  time.Time
}

func (k *qKeyInput) Cursor() *QPic {
	return getCursorPic(k.insert)
}

func (k *qKeyInput) String() string {
	return *(*string)(unsafe.Pointer(&k.buf))
}

func (k *qKeyInput) consoleKeyEvent(key kc.KeyCode) {
	switch key {
	case kc.KP_ENTER, kc.ENTER:
		t := k.String() + "\n"
		cbuf.AddText(t)
		history.Add(k.String()) // do not duplicate the '\n'
		conlog.Printf(t)
		k.buf = make([]byte, 0, 40)
		k.cursorXPos = 0
	case kc.TAB:
		// TODO(therjak): tap completion
	case kc.BACKSPACE:
		if k.cursorXPos > 0 {
			k.buf = append(k.buf[:k.cursorXPos-1], k.buf[k.cursorXPos:]...)
			k.cursorXPos--
			if k.cursorXPos < 0 {
				k.cursorXPos = 0
			}
		}
	case kc.DEL:
		if k.cursorXPos < len(k.buf) {
			k.buf = append(k.buf[:k.cursorXPos], k.buf[k.cursorXPos+1:]...)
		}
	case kc.HOME:
		if keyDown[kc.CTRL] {
			console.BackScrollHome()
		} else {
			k.cursorXPos = 0
		}
	case kc.END:
		if keyDown[kc.CTRL] {
			console.BackScrollEnd()
		} else {
			k.cursorXPos = len(k.buf)
		}
	case kc.PGUP, kc.MWHEELUP:
		console.BackScrollUp(keyDown[kc.CTRL])
	case kc.PGDN, kc.MWHEELDOWN:
		console.BackScrollDown(keyDown[kc.CTRL])
	case kc.LEFTARROW:
		if k.cursorXPos > 0 {
			k.cursorXPos--
			k.blinkTime = time.Now()
		}
	case kc.RIGHTARROW:
		if k.cursorXPos < len(k.buf) {
			k.cursorXPos++
			k.blinkTime = time.Now()
		}
	case kc.UPARROW:
		history.Up()
		k.buf = []byte(history.String())
		k.cursorXPos = len(k.buf)
	case kc.DOWNARROW:
		history.Down()
		k.buf = []byte(history.String())
		k.cursorXPos = len(k.buf)
	case kc.INS:
		k.insert = !k.insert
	case 'V', 'v':
		if keyDown[kc.CTRL] {
			k.paste()
		}
	case 'C', 'c':
		if keyDown[kc.CTRL] {
			k.buf = k.buf[:0]
			k.cursorXPos = 0
		}
	}
}

var (
	ovrPic *QPic
	insPic *QPic
)

func getCursorPic(insert bool) *QPic {
	if insert {
		return getInsPic()
	}
	return getOvrPic()
}

func getInsPic() *QPic {
	if insPic == nil {
		insPic = GetPictureFromBytes("ins", 8, 9, []byte{
			15, 15, 255, 255, 255, 255, 255, 255,
			15, 15, 2, 255, 255, 255, 255, 255,
			15, 15, 2, 255, 255, 255, 255, 255,
			15, 15, 2, 255, 255, 255, 255, 255,
			15, 15, 2, 255, 255, 255, 255, 255,
			15, 15, 2, 255, 255, 255, 255, 255,
			15, 15, 2, 255, 255, 255, 255, 255,
			15, 15, 2, 255, 255, 255, 255, 255,
			255, 2, 2, 255, 255, 255, 255, 255},
		)
	}
	return insPic
}

func getOvrPic() *QPic {
	if ovrPic == nil {
		ovrPic = GetPictureFromBytes("ovr", 8, 8, []byte{
			255, 255, 255, 255, 255, 255, 255, 255,
			255, 15, 15, 15, 15, 15, 15, 255,
			255, 15, 15, 15, 15, 15, 15, 2,
			255, 15, 15, 15, 15, 15, 15, 2,
			255, 15, 15, 15, 15, 15, 15, 2,
			255, 15, 15, 15, 15, 15, 15, 2,
			255, 15, 15, 15, 15, 15, 15, 2,
			255, 255, 2, 2, 2, 2, 2, 2},
		)
	}
	return ovrPic
}

func (k *qKeyInput) paste() {
	t, err := sdl.GetClipboardText()
	if err != nil {
		return
	}
	if k.cursorXPos == len(k.buf) {
		k.buf = append(k.buf[:k.cursorXPos], t...)
	} else {
		k.buf = append(k.buf[:k.cursorXPos],
			append([]byte(t), k.buf[k.cursorXPos+1:]...)...)
	}
	k.cursorXPos += len(t)
}

func (k *qKeyInput) consoleTextEvent(key rune) {
	// TODO(therjak): fix rune handling
	if k.cursorXPos == len(k.buf) {
		k.buf = append(k.buf[:k.cursorXPos], byte(key))
	} else if k.insert {
		k.buf = append(k.buf[:k.cursorXPos],
			append([]byte{byte(key)}, k.buf[k.cursorXPos:]...)...)
	} else {
		k.buf = append(k.buf[:k.cursorXPos],
			append([]byte{byte(key)}, k.buf[k.cursorXPos+1:]...)...)
	}
	k.cursorXPos++
}

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

	addCommand("bindlist", keyBindlist)
	addCommand("bind", keyBind)
	addCommand("unbind", keyUnbind)
	addCommand("unbindall", keyUnbindAll)
}

func keyBindlist(a cbuf.Arguments) error {
	count := 0
	for k, v := range keyBindings {
		if v != "" {
			count++
			conlog.SafePrintf("  %s \"%s\"\n", kc.KeyToString(k), v)
		}
	}
	conlog.SafePrintf("%d bindings\n", count)
	return nil
}

func keyUnbind(a cbuf.Arguments) error {
	args := a.Args()[1:]
	if len(args) != 1 {
		conlog.Printf("unbind <key> : remove commands from a key\n")
		return nil
	}

	b := kc.StringToKey(args[0].String())
	if b == -1 {
		conlog.Printf("\"%s\" isn't a valid key\n", args[0].String())
		return nil
	}

	delete(keyBindings, b)
	return nil
}

func keyUnbindAll(_ cbuf.Arguments) error {
	keyBindings = make(map[kc.KeyCode]string)
	return nil
}

func keyBind(a cbuf.Arguments) error {
	args := a.Args()[1:]
	c := len(args)
	if c != 1 && c != 2 {
		conlog.Printf("bind <key> [command] : attach a command to a key\n")
	}
	k := kc.StringToKey(args[0].String())
	if k == -1 {
		conlog.Printf("\"%s\" isn't a valid key\n", args[0].String())
		return nil
	}
	if c == 1 {
		if b, ok := keyBindings[k]; ok && b != "" {
			conlog.Printf("\"%s\" = \"%s\"\n", args[0].String(), b)
		} else {
			conlog.Printf("\"%s\" is not bound\n", args[0].String())
		}
		return nil
	}
	keyBindings[k] = args[1].String()
	return nil
}

func writeKeyBindings(w io.Writer) error {
	if cvars.CfgUnbindAll.Bool() {
		if _, err := w.Write([]byte("unbindall\n")); err != nil {
			return err
		}
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
		if _, err := w.Write([]byte(s)); err != nil {
			return err
		}
	}
	return nil
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

var (
	updateKeyDestForced = false
)

func updateKeyDest() {
	switch keyDestination {
	case keys.Console:
		if updateKeyDestForced && cls.state == ca_connected {
			updateKeyDestForced = false
			inputActivate()
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

type qInputGrab struct {
	active   bool
	lastKey  kc.KeyCode
	lastChar rune
}

var (
	inputGrab qInputGrab
)

func modalResult(timeout time.Duration) bool {
	Key_ClearStates()
	inputGrab = qInputGrab{
		active: true,
	}
	updateInputMode()

	endTime := time.Now().Add(timeout)
	result := false

	for {
		sendKeyEvents()
		// TODO(therjak): this Sleep should go
		time.Sleep(time.Millisecond * 16)
		if inputGrab.lastKey == kc.ABUTTON ||
			inputGrab.lastChar == 'Y' ||
			inputGrab.lastChar == 'y' {
			result = true
			break
		}
		if inputGrab.lastKey == kc.ESCAPE ||
			inputGrab.lastKey == kc.BBUTTON ||
			inputGrab.lastChar == 'N' ||
			inputGrab.lastChar == 'n' {
			result = false
			break
		}
		if timeout != 0 && endTime.Before(time.Now()) {
			result = false
			break
		}
	}

	Key_ClearStates()
	inputGrab.active = false
	updateInputMode()

	return result
}

func keyEvent(key kc.KeyCode, down bool) {
	if down && (key == kc.ENTER || key == kc.KP_ENTER) && keyDown[kc.ALT] {
		toggleFullScreen()
		return
	}

	if down {
		if keyDown[key] {
			if keyDestination == keys.Game && !console.forceDuplication {
				return
			}
			if key >= 200 && ("" != keyBindings[key]) {
				// >= 200 is mouse + joystick
				// TODO(therjak): is this the right condidition, do we want this at all?
				conlog.Printf("%s is unbound, hit F4 to set.\n", kc.KeyToString(key))
			}
		}
	} else {
		// ignore stray key up events
		if !keyDown[key] {
			return
		}
	}

	keyDown[key] = down

	if inputGrab.active {
		if down {
			inputGrab.lastKey = key
		}
		return
	}

	if key == kc.ESCAPE {
		// handled specially to disallow unbind
		if !down {
			return
		}
		if keyDown[kc.SHIFT] {
			console.Toggle()
			return
		}
		switch keyDestination {
		default: //keys.Game & keys.Console
			toggleMenu()
		case keys.Message:
			chatKey(key)
		case keys.Menu:
			qmenu.HandleKey(key)
		}
		return
	}

	if !down {
		// up presses are only relevant for "+"commands.
		// to be able to match multiple ones make them unique by adding the keynum
		b := keyBindings[key]
		if strings.HasPrefix(b, "+") {
			cmd := strings.Replace(b, "+", "-", 1)
			cbuf.AddText(fmt.Sprintf("%s %d\n", cmd, key))
		}
		return
	}

	if cls.demoPlayback &&
		consoleKeys[key] &&
		keyDestination == keys.Game &&
		key != kc.TAB {
		toggleMenu()
		return
	}

	if (keyDestination == keys.Menu && menuBound[key]) ||
		(keyDestination == keys.Console && !consoleKeys[key]) ||
		(keyDestination == keys.Game && (!console.forceDuplication || !consoleKeys[key])) {
		b := keyBindings[key]
		if strings.HasPrefix(b, "+") {
			cbuf.AddText(fmt.Sprintf("%s %d\n", b, key))
		} else {
			cbuf.AddText(b + "\n")
		}
	}
	switch keyDestination {
	default: //keys.Game & keys.Console
		keyInput.consoleKeyEvent(key)
	case keys.Message:
		chatKey(key)
	case keys.Menu:
		qmenu.HandleKey(key)
	}
}

var (
	chatBuilder strings.Builder
)

func chatEnd() {
	keyDestination = keys.Game
	chatBuilder.Reset()
}

func chatKey(key kc.KeyCode) {
	switch key {
	case kc.ENTER, kc.KP_ENTER:
		if console.chatTeam {
			cbuf.AddText("say_team \"")
		} else {
			cbuf.AddText("say \"")
		}
		cbuf.AddText(chatBuilder.String())
		cbuf.AddText("\"\n")
		chatEnd()
	case kc.ESCAPE:
		chatEnd()
	case kc.BACKSPACE:
		cur := chatBuilder.String()
		chatBuilder.Reset()
		last := true
		cur = strings.TrimRightFunc(cur, func(_ rune) bool {
			if last {
				last = false
				return true
			}
			return false
		})
		chatBuilder.WriteString(cur)
	}
}

func charEvent(key rune) {
	if key < 32 || key > 126 {
		// only ascii chars
		conlog.Printf("Got non ascii char in charEvent: %d", key)
		return
	}
	if keyDown[kc.CTRL] {
		return
	}
	if inputGrab.active {
		inputGrab.lastChar = key
		return
	}
	switch keyDestination {
	case keys.Game:
		if console.forceDuplication {
			keyInput.consoleTextEvent(key)
		}
	case keys.Console:
		keyInput.consoleTextEvent(key)
	case keys.Message:
		chatBuilder.WriteRune(key)
	case keys.Menu:
		qmenu.RuneInput(key)
	default:
	}
}

func Key_ClearStates() {
	for k, v := range keyDown {
		if v {
			keyEvent(k, false)
		}
	}
}

func keyTextEntry() bool {
	if inputGrab.active {
		return true
	}

	switch keyDestination {
	case keys.Game:
		return console.forceDuplication
	case keys.Console, keys.Message:
		return true
	case keys.Menu:
		return qmenu.TextEntry()
	default:
		return false
	}
}
