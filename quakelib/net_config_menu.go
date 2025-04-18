// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"fmt"
	"strconv"

	"goquake/cbuf"
	kc "goquake/keycode"
	"goquake/keys"
	"goquake/menu"
	"goquake/net"
)

func enterNetNewGameMenu() {
	IN_Deactivate()
	keyDestination = keys.Menu
	qmenu.state = menu.NetNewGame
	qmenu.playEnterSound = true
	netNewGameMenu.Update()
}

func enterNetJoinGameMenu() {
	IN_Deactivate()
	keyDestination = keys.Menu
	qmenu.state = menu.NetJoinGame
	qmenu.playEnterSound = true
	netJoinGameMenu.Update()
}

var (
	netNewGameMenu  = makeNetNewMenu()
	netJoinGameMenu = makeNetJoinMenu()
)

func makeNetNewMenu() *qNetNewMenu {
	return &qNetNewMenu{
		qNetConfigMenu{
			text: "New Game",
			items: []MenuItem{
				&portMenuItem{qMenuItem{52, 72}, 0, ""},
				&newGameOkMenuItem{qMenuItem{52, 92}, nil},
			},
		},
	}
}

func makeNetJoinMenu() *qNetJoinMenu {
	return &qNetJoinMenu{
		qNetConfigMenu{
			text: "Join Game",
			items: []MenuItem{
				&portMenuItem{qMenuItem{52, 72}, 0, ""},
				// Therjak: Removed server search. Net broadcast is not implemented.
				// &joinGameSearchMenuItem{qMenuItem{52, 92}, nil},
				&serverNameMenuItem{qMenuItem{52, 124}, "", nil},
			},
		},
	}
}

/*
	type joinGameSearchMenuItem struct {
		qMenuItem
		accepter qAccept
	}

	func (m *joinGameSearchMenuItem) Draw() {
		drawString(60, m.Y, "Search for local games...")
		// the following is a little bit hacky as is does not belong to this item
		drawString(60, 108, "Join game at:")
	}

	func (m *joinGameSearchMenuItem) Enter() {
		qmenu.playEnterSound = true
		m.accepter.Accept() // to update the port
		//  M_Menu_Search_f();
	}

	func (m *joinGameSearchMenuItem) Update(a qAccept) {
		m.accepter = a
	}
*/
type newGameOkMenuItem struct {
	qMenuItem
	accepter qAccept
}

func (m *newGameOkMenuItem) Draw() {
	drawTextbox(60, m.Y-8, 2, 1)
	drawString(60+8, m.Y, "OK")
}

func (m *newGameOkMenuItem) Update(a qAccept) {
	m.accepter = a
}

func (m *newGameOkMenuItem) Enter() {
	qmenu.playEnterSound = true
	m.accepter.Accept() // to update the port
	enterGameOptionsMenu()
}

type serverNameMenuItem struct {
	qMenuItem
	serverName string
	accepter   qAccept
}

func (m *serverNameMenuItem) Draw() {
	drawTextbox(60+8, m.Y-8, 22, 1)
	drawString(60+16, m.Y, m.serverName)
}

func (m *serverNameMenuItem) Update(a qAccept) {
	m.serverName = net.ServerName()
	m.accepter = a
}

func (m *serverNameMenuItem) Backspace() {
	m.serverName = removeLast(m.serverName)
}

func (m *serverNameMenuItem) HandleRune(key rune) {
	if len(m.serverName) < 21 {
		m.serverName += string(key)
	}
}

/*
* This is probably better done in Enter
func (m *serverNameMenuItem) Accept() {
	cbuf.AddText(fmt.Sprintf("connect \"%s\"\n", m.serverName))
}
*/

func (m *serverNameMenuItem) DrawCursor() {
	m.qMenuItem.DrawCursor()
	DrawCharacterWhite(60+16+8*len(m.serverName), m.Y, 10+blink())
}

func (m *serverNameMenuItem) Enter() {
	qmenu.playEnterSound = true
	m.accepter.Accept()
	enterMenuNone()
	cbuf.AddText(fmt.Sprintf("connect \"%s\"\n", m.serverName))
}

type portMenuItem struct {
	qMenuItem
	port     int
	portName string
}

func (m *portMenuItem) Draw() {
	drawString(60, m.Y, "Port")
	drawTextbox(60+8*8, m.Y-8, 6, 1)
	drawString(60+9*8, m.Y, m.portName)
}

func (m *portMenuItem) Update(a qAccept) {
	m.port = net.Port()
	m.portName = fmt.Sprintf("%d", m.port)
}

func (m *portMenuItem) Backspace() {
	m.portName = removeLast(m.portName)
}

func (m *portMenuItem) HandleRune(key rune) {
	if key < '0' || key > '9' {
		return
	}
	if len(m.portName) < 5 {
		m.portName += string(key)
	}
}

func (m *portMenuItem) Accept() {
	p, err := strconv.Atoi(m.portName)
	if err != nil || p > 65535 {
		p = m.port
	}
	cbuf.AddText("stopdemo\n")
	net.SetPort(p)
}

func (m *portMenuItem) DrawCursor() {
	m.qMenuItem.DrawCursor()
	DrawCharacterWhite(60+9*8+8*len(m.portName), m.Y, 10+blink())
}

type qNetConfigMenu struct {
	selectedIndex int
	items         []MenuItem
	text          string
}

type qNetNewMenu struct {
	qNetConfigMenu
}

type qNetJoinMenu struct {
	qNetConfigMenu
}

func (m *qNetNewMenu) Update() {
	m.qNetConfigMenu.Update()
	m.selectedIndex = 1
}

func (m *qNetJoinMenu) Update() {
	m.qNetConfigMenu.Update()
	m.selectedIndex = len(m.items) - 1
}

func (m *qNetConfigMenu) HandleKey(key kc.KeyCode) {
	switch key {
	case kc.ESCAPE, kc.BBUTTON:
		enterMultiPlayerMenu()

	case kc.DOWNARROW:
		localSound(lsMenu1)
		m.selectedIndex = (m.selectedIndex + 1) % len(m.items)

	case kc.UPARROW:
		localSound(lsMenu1)
		m.selectedIndex = (m.selectedIndex + len(m.items) - 1) % len(m.items)

	case kc.ENTER, kc.KP_ENTER, kc.ABUTTON:
		m.items[m.selectedIndex].Enter()

	case kc.BACKSPACE:
		m.items[m.selectedIndex].Backspace()
	}
}

func (m *qNetConfigMenu) HandleRune(key rune) {
	m.items[m.selectedIndex].HandleRune(key)
}

func (m *qNetConfigMenu) Accept() {
	for _, item := range m.items {
		item.Accept()
	}
}

func (m *qNetConfigMenu) Update() {
	for _, item := range m.items {
		item.Update(m)
	}
}

func (m *qNetConfigMenu) TextEntry() bool {
	// Note: if the joinGameSearch gets reintroduced this needs an update!
	return m.selectedIndex == 0 || m.selectedIndex == 1
}

func (m *qNetConfigMenu) Draw() {
	DrawPicture(16, 4, GetCachedPicture("gfx/qplaque.lmp"))
	p := GetCachedPicture("gfx/p_multi.lmp")
	DrawPicture((320-p.Width)/2, 4, p)
	drawString(52, 32, fmt.Sprintf("%s - TCP/IP", m.text))
	drawString(60, 52, "Address:")
	drawString(60+9*8, 52, net.Address())

	for _, item := range m.items {
		item.Draw()
	}
	m.items[m.selectedIndex].DrawCursor()
	// if errorString != "" {
	// drawWhiteString(60,148,errorString)
	// }
}
