// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"fmt"

	"goquake/cbuf"
	"goquake/cvars"
	kc "goquake/keycode"
	"goquake/keys"
	"goquake/menu"
)

type qVideoMenu struct {
	selectedIndex int
	items         []MenuItem
}

func (m *qVideoMenu) HandleKey(key kc.KeyCode) {
	switch key {
	case kc.ESCAPE, kc.BBUTTON:
		// FIXME: there are other ways to leave menu
		syncVideoCvars()
		localSound(lsMenu1)
		enterMenuOptions()
	case kc.UPARROW:
		localSound(lsMenu1)
		m.selectedIndex = (m.selectedIndex + len(m.items) - 1) % len(m.items)
	case kc.DOWNARROW:
		localSound(lsMenu1)
		m.selectedIndex = (m.selectedIndex + 1) % len(m.items)
	case kc.LEFTARROW:
		localSound(lsMenu3)
		m.items[m.selectedIndex].Left()
	case kc.RIGHTARROW:
		localSound(lsMenu3)
		m.items[m.selectedIndex].Right()
	case kc.ENTER, kc.KP_ENTER, kc.ABUTTON:
		qmenu.playEnterSound = true
		m.items[m.selectedIndex].Enter()
	}
}

func (m *qVideoMenu) Draw() {
	// We draw on a 320x200 screen
	DrawPicture(16, 4, GetCachedPicture("gfx/qplaque.lmp"))

	p := GetCachedPicture("gfx/p_option.lmp")
	DrawPicture((320-p.Width)/2, 4, p)

	title := "Video Options"
	DrawStringWhite((320-8*len(title))/2, 32, title)

	for _, item := range m.items {
		item.Draw()
	}
	m.items[m.selectedIndex].DrawCursor()
}

var (
	videoMenu = qVideoMenu{
		items: makeVideoMenuItems(),
	}
)

func makeVideoMenuItems() []MenuItem {
	return []MenuItem{
		&MenuItemResolution{qMenuItem{168, 8 * 6}},
		&MenuItemFullscreen{qMenuItem{168, 8 * 8}},
		&MenuItemVerticalSync{qMenuItem{168, 8 * 9}},
		&MenuItemVideoTest{qMenuItem{168, 8 * 11}},
		&MenuItemVideoAccept{qMenuItem{168, 8 * 12}},
	}
}

type MenuItemResolution struct{ qMenuItem }

func (m *MenuItemResolution) Left()  { chooseNextMode() }
func (m *MenuItemResolution) Right() { choosePrevMode() }
func (m *MenuItemResolution) Enter() { chooseNextMode() }
func (m *MenuItemResolution) Draw() {
	drawString(16, m.Y, "        Video mode")
	drawString(184, m.Y, fmt.Sprintf("%.fx%.f", cvars.VideoWidth.Value(), cvars.VideoHeight.Value()))
}

type MenuItemFullscreen struct{ qMenuItem }

func (m *MenuItemFullscreen) Left()  { cbuf.AddText("toggle vid_fullscreen\n") }
func (m *MenuItemFullscreen) Right() { cbuf.AddText("toggle vid_fullscreen\n") }
func (m *MenuItemFullscreen) Enter() { cbuf.AddText("toggle vid_fullscreen\n") }
func (m *MenuItemFullscreen) Draw() {
	drawString(16, m.Y, "        Fullscreen")
	drawCheckbox(184, m.Y, cvars.VideoFullscreen.Value() != 0)
}

type MenuItemVerticalSync struct{ qMenuItem }

func (m *MenuItemVerticalSync) Left()  { cbuf.AddText("toggle vid_vsync\n") }
func (m *MenuItemVerticalSync) Right() { cbuf.AddText("toggle vid_vsync\n") }
func (m *MenuItemVerticalSync) Enter() { cbuf.AddText("toggle vid_vsync\n") }
func (m *MenuItemVerticalSync) Draw() {
	drawString(16, m.Y, "     Vertical sync")
	if glSwapControl {
		drawCheckbox(184, m.Y, cvars.VideoVerticalSync.Value() != 0)
	} else {
		drawString(184, m.Y, "N/A")
	}
}

type MenuItemVideoTest struct{ qMenuItem }

func (m *MenuItemVideoTest) Enter() { cbuf.AddText("vid_test\n") }
func (m *MenuItemVideoTest) Draw() {
	drawString(16, m.Y, "      Test changes")
}

type MenuItemVideoAccept struct{ qMenuItem }

func (m *MenuItemVideoAccept) Enter() {
	cbuf.AddText("vid_restart\n")
	keyDestination = keys.Game
	qmenu.state = menu.None
	inputActivate()
}
func (m *MenuItemVideoAccept) Draw() {
	drawString(16, m.Y, "     Apply changes")
}
