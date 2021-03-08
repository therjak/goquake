// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"fmt"

	"github.com/therjak/goquake/cbuf"
	kc "github.com/therjak/goquake/keycode"
	"github.com/therjak/goquake/keys"
	"github.com/therjak/goquake/menu"
)

func enterMenuKeys() {
	IN_Deactivate()
	keyDestination = keys.Menu
	qmenu.state = menu.Keys
	qmenu.playEnterSound = true
}

var (
	keysMenu = qKeysMenu{
		items: makeKeysMenuItems(),
	}
)

func makeKeysMenuItems() []*keysMenuItem {
	i := 0
	y := func() int {
		defer func() { i++ }()
		return 48 + 8*i
	}
	return []*keysMenuItem{
		makeKeysMenuItem("+attack", "attack", y()),
		makeKeysMenuItem("impulse 10", "next weapon", y()),
		makeKeysMenuItem("impulse 12", "prev weapon", y()),
		makeKeysMenuItem("+jump", "jump / swim up", y()),
		makeKeysMenuItem("+forward", "walk forward", y()),
		makeKeysMenuItem("+back", "backpedal", y()),
		makeKeysMenuItem("+left", "turn left", y()),
		makeKeysMenuItem("+right", "turn right", y()),
		makeKeysMenuItem("+speed", "run", y()),
		makeKeysMenuItem("+moveleft", "step left", y()),
		makeKeysMenuItem("+moveright", "step right", y()),
		makeKeysMenuItem("+strafe", "sidestep", y()),
		makeKeysMenuItem("+lookup", "look up", y()),
		makeKeysMenuItem("+lookdown", "look down", y()),
		makeKeysMenuItem("centerview", "center view", y()),
		makeKeysMenuItem("+mlook", "mouse look", y()),
		makeKeysMenuItem("+klook", "keyboard look", y()),
		makeKeysMenuItem("+moveup", "swim up", y()),
		makeKeysMenuItem("+movedown", "swim down", y()),
	}
}

type qKeysMenu struct {
	selectedIndex int
	items         []*keysMenuItem
	grabbed       bool
}

func (m *qKeysMenu) HandleKey(key kc.KeyCode) {
	if m.grabbed {
		localSound("misc/menu1.wav")
		if (key != kc.ESCAPE) && (key != '`') {
			m.items[m.selectedIndex].Change(key)
		}
		m.grabbed = false
		IN_Deactivate()
		return
	}
	switch key {
	case kc.ESCAPE, kc.BBUTTON:
		enterMenuOptions()
	case kc.LEFTARROW, kc.UPARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + len(m.items) - 1) % len(m.items)
	case kc.DOWNARROW, kc.RIGHTARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + 1) % len(m.items)
	case kc.ENTER, kc.KP_ENTER, kc.ABUTTON:
		m.items[m.selectedIndex].Enter()
		m.grabbed = true
		inputActivate()
	case kc.BACKSPACE, kc.DEL:
		m.items[m.selectedIndex].Backspace()
	}
}

func (m *qKeysMenu) Draw() {
	p := GetCachedPicture("gfx/ttl_cstm.lmp")
	DrawPicture((320-p.Width)/2, 4, p)
	if m.grabbed {
		drawString(12, 32, "Press a key or button for this action")
	} else {
		drawString(18, 32, "Enter to change, backspace to clear")
	}
	for i, item := range m.items {
		item.Draw(i == m.selectedIndex, m.grabbed)
	}
}

type keysMenuItem struct {
	bind        string
	description string
	qMenuItem
}

func makeKeysMenuItem(b, d string, y int) *keysMenuItem {
	return &keysMenuItem{b, d, qMenuItem{0, y}}
}

func (m *keysMenuItem) Draw(s, grab bool) {
	drawString(16, m.Y, m.description)

	k0, k1, k2 := getKeysForCommand(m.bind)

	if k0 == -1 {
		drawString(140, m.Y, "???")
	} else {
		name := kc.KeyToString(k0)
		drawString(140, m.Y, name)
		x := len(name) * 8
		if k1 != -1 {
			name = kc.KeyToString(k1)
			drawString(140+x+8, m.Y, "or")
			drawString(140+x+32, m.Y, name)
			x = x + 32 + len(name)*8
			if k2 != -1 {
				drawString(140+x+8, m.Y, "or")
				drawString(140+x+32, m.Y, kc.KeyToString(k2))
			}
		}
	}

	if s {
		if grab {
			DrawCharacterWhite(130, m.Y, '=')
		} else {
			DrawCharacterWhite(130, m.Y, 12+(int(Time()*4)&1))
		}
	}
}
func (m *keysMenuItem) Enter() {
	localSound("misc/menu2.wav")
	_, _, k2 := getKeysForCommand(m.bind)
	if k2 != -1 {
		unbindCommand(m.bind)
	}
}

func (m *keysMenuItem) Backspace() {
	localSound("misc/menu2.wav")
	unbindCommand(m.bind)
}

func (m *keysMenuItem) Change(key kc.KeyCode) {
	cmd := fmt.Sprintf("bind \"%s\" \"%s\"\n", kc.KeyToString(key), m.bind)
	cbuf.InsertText(cmd)
}
