package quakelib

import (
	"quake/cbuf"
	kc "quake/keycode"
	"quake/keys"
	"quake/menu"
)

func enterSinglePlayerMenu() {
	IN_Deactivate()
	keyDestination = keys.Menu
	qmenu.state = menu.SinglePlayer
	qmenu.playEnterSound = true
}

var (
	singlePlayerMenu = qSinglePlayerMenu{
		items: makeSinglePlayerMenuItems(),
	}
)

type qSinglePlayerMenu struct {
	selectedIndex int
	items         []MenuItem
}

func (m *qSinglePlayerMenu) Draw() {
	DrawPicture(16, 4, GetCachedPicture("gfx/qplaque.lmp"))

	p := GetCachedPicture("gfx/ttl_sgl.lmp")
	DrawPicture((320-p.width)/2, 4, p)

	DrawPicture(72, 32, GetCachedPicture("gfx/sp_menu.lmp"))

	m.items[m.selectedIndex].DrawCursor()
}

func (m *qSinglePlayerMenu) HandleKey(key kc.KeyCode) {
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

func makeSinglePlayerMenuItems() []MenuItem {
	return []MenuItem{
		&MenuItemPlay{qDotMenuItem{qMenuItem{54, 32}}},
		&MenuItemLoad{qDotMenuItem{qMenuItem{54, 32 + 20}}},
		&MenuItemSave{qDotMenuItem{qMenuItem{54, 32 + 20*2}}},
	}
}

type MenuItemPlay struct{ qDotMenuItem }

func (m *MenuItemPlay) Enter() {
	if sv.active &&
		!ModalMessage("Are you sure you want to\nstart a new game?\n", 0) {
		return
	}

	IN_Activate()
	keyDestination = keys.Game
	if sv.active {
		cbuf.AddText("disconnect\n")
	}
	cbuf.AddText("maxplayers 1\n")
	cbuf.AddText("deathmatch 0\n")
	cbuf.AddText("coop 0\n")
	cbuf.AddText("map start\n")
}

type MenuItemLoad struct{ qDotMenuItem }

func (m *MenuItemLoad) Enter() {
	enterLoadMenu()
}

type MenuItemSave struct{ qDotMenuItem }

func (m *MenuItemSave) Enter() {
	enterSaveMenu()
}
