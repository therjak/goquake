// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	cmdl "goquake/commandline"
	kc "goquake/keycode"
	"goquake/keys"
	"goquake/menu"
)

var (
	menuSaveDemoNumber = -1
)

func enterMenuMain() {
	if keyDestination != keys.Menu {
		// TODO(therjak): isn't what is wanted to pause the demo?
		// I guess this restarts the same
		menuSaveDemoNumber = cls.demoNum
		cls.demoNum = -1
	}
	IN_Deactivate()
	keyDestination = keys.Menu
	qmenu.state = menu.Main
	qmenu.playEnterSound = true
}

var (
	mainMenu = qMainMenu{
		items: makeMainMenuItems(),
	}
)

type qMainMenu struct {
	selectedIndex int
	items         []MenuItem
}

func (m *qMainMenu) Draw() {
	// We draw on a 320x200 screen
	DrawPicture(16, 4, GetCachedPicture("gfx/qplaque.lmp"))

	p := GetCachedPicture("gfx/ttl_main.lmp")
	DrawPicture((320-p.Width)/2, 4, p)

	DrawPicture(72, 32, GetCachedPicture("gfx/mainmenu.lmp"))

	m.items[m.selectedIndex].DrawCursor()
}

func (m *qMainMenu) HandleKey(key kc.KeyCode) {
	switch key {
	case kc.ESCAPE, kc.BBUTTON:
		inputActivate()
		keyDestination = keys.Game
		qmenu.state = menu.None
		cls.demoNum = menuSaveDemoNumber
		if !cmdl.Fitz() {
			return
		}
		if cls.demoNum != -1 && !cls.demoPlayback && cls.state != ca_connected {
			nextDemo()
		}
	case kc.UPARROW:
		localSound(lsMenu1)
		m.selectedIndex = (m.selectedIndex + len(m.items) - 1) % len(m.items)
	case kc.DOWNARROW:
		localSound(lsMenu1)
		m.selectedIndex = (m.selectedIndex + 1) % len(m.items)
	case kc.ENTER, kc.KP_ENTER, kc.ABUTTON:
		qmenu.playEnterSound = true
		m.items[m.selectedIndex].Enter()
	}
}

func makeMainMenuItems() []MenuItem {
	return []MenuItem{
		&MenuItemSinglePlayer{qDotMenuItem{qMenuItem{54, 32}}},
		&MenuItemMultiPlayer{qDotMenuItem{qMenuItem{54, 32 + 20}}},
		&MenuItemOptions{qDotMenuItem{qMenuItem{54, 32 + 20*2}}},
		&MenuItemHelp{qDotMenuItem{qMenuItem{54, 32 + 20*3}}},
		&MenuItemQuit{qDotMenuItem{qMenuItem{54, 32 + 20*4}}},
	}
}

type MenuItemSinglePlayer struct{ qDotMenuItem }

func (m *MenuItemSinglePlayer) Enter() {
	enterSinglePlayerMenu()
}

type MenuItemMultiPlayer struct{ qDotMenuItem }

func (m *MenuItemMultiPlayer) Enter() {
	enterMultiPlayerMenu()
}

type MenuItemOptions struct{ qDotMenuItem }

func (m *MenuItemOptions) Enter() {
	enterMenuOptions()
}

type MenuItemHelp struct{ qDotMenuItem }

func (m *MenuItemHelp) Enter() {
	enterMenuHelp()
}

type MenuItemQuit struct{ qDotMenuItem }

func (m *MenuItemQuit) Enter() {
	enterQuitMenu()
}
