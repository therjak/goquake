package quakelib

import (
	"fmt"

	kc "github.com/therjak/goquake/keycode"
	"github.com/therjak/goquake/keys"
	"github.com/therjak/goquake/menu"
)

func enterMenuHelp() {
	IN_Deactivate()
	keyDestination = keys.Menu
	qmenu.state = menu.Help
	qmenu.playEnterSound = true
	helpMenu.Reset()
}

var (
	helpMenu = qHelpMenu{
		items: makeHelpMenuItems(),
	}
)

type qHelpMenu struct {
	selectedIndex int
	items         []MenuItem
}

type helpMenuItem struct {
	index int
	qMenuItem
}

func (h *helpMenuItem) DrawCursor() {
	name := fmt.Sprintf("gfx/help%d.lmp", h.index)
	DrawPicture(0, 0, GetCachedPicture(name))
}

func makeHelpMenuItems() []MenuItem {
	return []MenuItem{
		&helpMenuItem{index: 0},
		&helpMenuItem{index: 1},
		&helpMenuItem{index: 2},
		&helpMenuItem{index: 3},
		&helpMenuItem{index: 4},
		&helpMenuItem{index: 5},
	}
}

func (m *qHelpMenu) Draw() {
	m.items[m.selectedIndex].DrawCursor()
}

func (m *qHelpMenu) Reset() {
	m.selectedIndex = 0
}

func (m *qHelpMenu) HandleKey(key kc.KeyCode) {
	switch key {
	case kc.ESCAPE, kc.BBUTTON:
		enterMenuMain()
	case kc.UPARROW, kc.RIGHTARROW:
		qmenu.playEnterSound = true
		m.selectedIndex = (m.selectedIndex + len(m.items) - 1) % len(m.items)
	case kc.DOWNARROW, kc.LEFTARROW:
		qmenu.playEnterSound = true
		m.selectedIndex = (m.selectedIndex + 1) % len(m.items)
	}
}
