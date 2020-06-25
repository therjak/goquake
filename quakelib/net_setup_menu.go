package quakelib

import (
	"fmt"

	"github.com/therjak/goquake/cbuf"
	"github.com/therjak/goquake/cvars"
	kc "github.com/therjak/goquake/keycode"
	"github.com/therjak/goquake/keys"
	"github.com/therjak/goquake/menu"
)

func enterNetSetupMenu() {
	IN_Deactivate()
	keyDestination = keys.Menu
	qmenu.state = menu.Setup
	qmenu.playEnterSound = true
	netSetupMenu.update()
}

var (
	netSetupMenu = qNetSetupMenu{
		items: makeNetSetupMenuItems(),
	}
)

type qNetSetupMenu struct {
	selectedIndex int
	items         []MenuItem
	hostname      string
	playername    string
	topColor      int
	bottomColor   int
}

func (m *qNetSetupMenu) update() {
	m.playername = cvars.ClientName.String()
	m.hostname = cvars.HostName.String()
	c := int(cvars.ClientColor.Value())
	m.topColor = c >> 4
	m.bottomColor = c & 15
}

func (m *qNetSetupMenu) TextEntry() bool {
	switch m.selectedIndex {
	case 0, 1:
		return true
	}
	return false
}

func (m *qNetSetupMenu) Draw() {
	DrawPicture(16, 4, GetCachedPicture("gfx/qplaque.lmp"))
	p := GetCachedPicture("gfx/p_multi.lmp")
	DrawPicture((320-p.Width)/2, 4, p)

	DrawPicture(160, 64, GetCachedPicture("gfx/bigbox.lmp"))
	DrawPictureTranslate(172, 72, GetCachedPicture("gfx/menuplyr.lmp"), m.topColor, m.bottomColor)

	for _, item := range m.items {
		item.Draw()
	}
	m.items[m.selectedIndex].DrawCursor()
}

func (m *qNetSetupMenu) HandleKey(key kc.KeyCode) {
	switch key {
	case kc.ESCAPE, kc.BBUTTON:
		enterMultiPlayerMenu()
	case kc.UPARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + len(m.items) - 1) % len(m.items)
	case kc.DOWNARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + 1) % len(m.items)
	case kc.LEFTARROW:
		m.items[m.selectedIndex].Left()
	case kc.RIGHTARROW:
		m.items[m.selectedIndex].Right()
	case kc.ENTER, kc.KP_ENTER, kc.ABUTTON:
		m.items[m.selectedIndex].Enter()
	case kc.BACKSPACE:
		m.items[m.selectedIndex].Backspace()
	}
}

func (m *qNetSetupMenu) HandleRune(key rune) {
	switch m.selectedIndex {
	case 0:
		if len(m.hostname) < 15 {
			m.hostname += string(key)
		}
	case 1:
		if len(m.playername) < 15 {
			m.playername += string(key)
		}
	}
}

func makeNetSetupMenuItems() []MenuItem {
	return []MenuItem{
		&hostnameMenuItem{qMenuItem{56, 40}},
		&playernameMenuItem{qMenuItem{56, 56}},
		&topColorMenuItem{qMenuItem{56, 80}},
		&bottomColorMenuItem{qMenuItem{56, 104}},
		&netSetupAcceptMenuItem{qMenuItem{56, 140}},
	}
}

type hostnameMenuItem struct {
	qMenuItem
}

func (m *hostnameMenuItem) Draw() {
	drawString(64, m.Y, "Hostname")
	drawTextbox(160, m.Y-8, 16, 1)
	drawString(168, m.Y, netSetupMenu.hostname)
}

func (m *hostnameMenuItem) DrawCursor() {
	m.qMenuItem.DrawCursor()
	DrawCharacterWhite(168+8*len(netSetupMenu.hostname), m.Y,
		10+(int(Time()*4)&1))
}

func removeLast(s string) string {
	if len(s) == 0 {
		return s
	}
	return s[:len(s)-1]
}

func (m *hostnameMenuItem) Backspace() {
	netSetupMenu.hostname = removeLast(netSetupMenu.hostname)
}

type playernameMenuItem struct {
	qMenuItem
}

func (m *playernameMenuItem) Draw() {
	drawString(64, m.Y, "Your name")
	drawTextbox(160, m.Y-8, 16, 1)
	drawString(168, m.Y, netSetupMenu.playername)
}

func (m *playernameMenuItem) DrawCursor() {
	m.qMenuItem.DrawCursor()
	DrawCharacterWhite(168+8*len(netSetupMenu.playername), m.Y,
		10+(int(Time()*4)&1))
}

func (m *playernameMenuItem) Backspace() {
	netSetupMenu.playername = removeLast(netSetupMenu.playername)
}

type topColorMenuItem struct {
	qMenuItem
}

func (m *topColorMenuItem) Draw() {
	drawString(64, m.Y, "Shirt color")
}
func (m *topColorMenuItem) Left() {
	localSound("misc/menu3.wav")
	netSetupMenu.topColor = (netSetupMenu.topColor + 14 - 1) % 14
}
func (m *topColorMenuItem) Right() {
	localSound("misc/menu3.wav")
	netSetupMenu.topColor = (netSetupMenu.topColor + 1) % 14
}

type bottomColorMenuItem struct {
	qMenuItem
}

func (m *bottomColorMenuItem) Draw() {
	drawString(64, m.Y, "Pants color")
}
func (m *bottomColorMenuItem) Left() {
	localSound("misc/menu3.wav")
	netSetupMenu.bottomColor = (netSetupMenu.bottomColor + 14 - 1) % 14
}
func (m *bottomColorMenuItem) Right() {
	localSound("misc/menu3.wav")
	netSetupMenu.bottomColor = (netSetupMenu.bottomColor + 1) % 14
}

type netSetupAcceptMenuItem struct {
	qMenuItem
}

func (m *netSetupAcceptMenuItem) Draw() {
	drawTextbox(64, m.Y-8, 14, 1)
	drawString(72, m.Y, "Accept Changes")
}

func (mi *netSetupAcceptMenuItem) Enter() {
	m := &netSetupMenu
	if m.playername != cvars.ClientName.String() {
		cbuf.AddText(fmt.Sprintf("name \"%s\"\n", m.playername))
	}
	if m.hostname != cvars.HostName.String() {
		cvars.HostName.SetByString(m.hostname)
	}
	c := int(cvars.ClientColor.Value())
	if m.topColor != (c>>4) || m.bottomColor != (c&15) {
		cbuf.AddText(fmt.Sprintf("color %d %d\n", m.topColor, m.bottomColor))
	}

	enterMultiPlayerMenu()
}
