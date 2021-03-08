// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	kc "github.com/therjak/goquake/keycode"
	"github.com/therjak/goquake/keys"
	"github.com/therjak/goquake/menu"
)

func enterMultiPlayerMenu() {
	IN_Deactivate()
	keyDestination = keys.Menu
	qmenu.state = menu.MultiPlayer
	qmenu.playEnterSound = true
}

var (
	multiPlayerMenu = qMultiPlayerMenu{
		items: makeMultiPlayerMenuItems(),
	}
)

type qMultiPlayerMenu struct {
	selectedIndex int
	items         []MenuItem
}

func (m *qMultiPlayerMenu) Draw() {
	DrawPicture(16, 4, GetCachedPicture("gfx/qplaque.lmp"))

	p := GetCachedPicture("gfx/p_multi.lmp")
	DrawPicture((320-p.Width)/2, 4, p)

	DrawPicture(72, 32, GetCachedPicture("gfx/mp_menu.lmp"))

	m.items[m.selectedIndex].DrawCursor()

	if !tcpipAvailable {
		DrawStringWhite((320/2)-((27*8)/2), 148, "No Communications Available")
	}
}

func (m *qMultiPlayerMenu) HandleKey(key kc.KeyCode) {
	switch key {
	case kc.ESCAPE, kc.BBUTTON:
		enterMenuMain()

	case kc.DOWNARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + 1) % len(m.items)

	case kc.UPARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + len(m.items) - 1) % len(m.items)

	case kc.ENTER, kc.KP_ENTER, kc.ABUTTON:
		qmenu.playEnterSound = true
		m.items[m.selectedIndex].Enter()
	}
}

func makeMultiPlayerMenuItems() []MenuItem {
	return []MenuItem{
		&MenuItemNetJoin{qDotMenuItem{qMenuItem{54, 32}}},
		&MenuItemNetNew{qDotMenuItem{qMenuItem{54, 32 + 20}}},
		&MenuItemNetSetup{qDotMenuItem{qMenuItem{54, 32 + 20*2}}},
	}
}

type MenuItemNetJoin struct{ qDotMenuItem }

func (m *MenuItemNetJoin) Enter() {
	if !tcpipAvailable {
		return
	}
	enterNetJoinGameMenu()
}

type MenuItemNetNew struct{ qDotMenuItem }

func (m *MenuItemNetNew) Enter() {
	if !tcpipAvailable {
		return
	}
	enterNetNewGameMenu()
}

type MenuItemNetSetup struct{ qDotMenuItem }

func (m *MenuItemNetSetup) Enter() {
	enterNetSetupMenu()
}
