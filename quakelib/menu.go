// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"fmt"

	"goquake/cmd"
	kc "goquake/keycode"
	"goquake/keys"
	"goquake/math"
	"goquake/menu"
)

func nextDemo() {
	if err := CL_NextDemo(); err != nil {
		HostError(err)
	}
}

func init() {
	addCommand("togglemenu", func(_ cmd.Arguments, p, s int) error {
		toggleMenu()
		return nil
	})
	addCommand("menu_main", func(_ cmd.Arguments, p, s int) error {
		enterMenuMain()
		return nil
	})
	addCommand("menu_singleplayer", func(_ cmd.Arguments, p, s int) error {
		enterSinglePlayerMenu()
		return nil
	})
	addCommand("menu_load", func(_ cmd.Arguments, p, s int) error {
		enterLoadMenu()
		return nil
	})
	addCommand("menu_save", func(_ cmd.Arguments, p, s int) error {
		enterSaveMenu()
		return nil
	})
	addCommand("menu_multiplayer", func(_ cmd.Arguments, p, s int) error {
		enterMultiPlayerMenu()
		return nil
	})
	addCommand("menu_setup", func(_ cmd.Arguments, p, s int) error {
		enterNetSetupMenu()
		return nil
	})
	addCommand("menu_options", func(_ cmd.Arguments, p, s int) error {
		enterMenuOptions()
		return nil
	})
	addCommand("menu_keys", func(_ cmd.Arguments, p, s int) error {
		enterMenuKeys()
		return nil
	})
	addCommand("menu_video", func(_ cmd.Arguments, p, s int) error {
		enterMenuVideo()
		return nil
	})
	addCommand("help", func(_ cmd.Arguments, p, s int) error {
		enterMenuHelp()
		return nil
	})
	addCommand("menu_quit", func(_ cmd.Arguments, p, s int) error {
		enterQuitMenu()
		return nil
	})
	addCommand("menu_credits", func(_ cmd.Arguments, p, s int) error {
		return nil
	})
}

// 0-127 are white
// 128+ are normal
// We draw on a 320x200 screen
func drawString(x, y int, t string) {
	DrawStringCopper(x, y, t)
}

func drawCheckbox(x, y int, checked bool) {
	if checked {
		drawString(x, y, "on")
	} else {
		drawString(x, y, "off")
	}
}

func drawSlider(x, y int, r float32) {
	r = math.Clamp32(0, r, 1)
	DrawCharacterWhite(x-8, y, 128)
	for i := 0; i < 10; i++ {
		DrawCharacterWhite(x+i*8, y, 129)
	}
	DrawCharacterWhite(x+10*8, y, 130)
	DrawCharacterWhite(x+int(9*8*r), y, 131)
}

// w: width, l: lines
func drawTextbox(x, y, w, l int) {
	tm := GetCachedPicture("gfx/box_tm.lmp")
	mm := GetCachedPicture("gfx/box_tm.lmp")
	mm2 := GetCachedPicture("gfx/box_tm.lmp")
	bm := GetCachedPicture("gfx/box_tm.lmp")

	for i := 0; i < w/2; i++ {
		mx := x + 8 + 16*i
		DrawPicture(mx, y, tm)
		for n := 0; n < l; n++ {
			if n == 1 {
				DrawPicture(mx, y+8*n, mm2)
			} else {
				DrawPicture(mx, y+8*n, mm)
			}
		}
		DrawPicture(mx, y+8*l, bm)
	}

	fx := x + 8 + 16*(w/2)
	DrawPicture(x, y, GetCachedPicture("gfx/box_tl.lmp"))
	DrawPicture(fx, y, GetCachedPicture("gfx/box_tr.lmp"))
	p1 := GetCachedPicture("gfx/box_ml.lmp")
	p2 := GetCachedPicture("gfx/box_mr.lmp")
	for i := 0; i < l; i++ {
		my := y + 8 + 8*i
		DrawPicture(x, my, p1)
		DrawPicture(fx, my, p2)
	}
	fy := y + 8 + 8*l
	DrawPicture(x, fy, GetCachedPicture("gfx/box_bl.lmp"))
	DrawPicture(fx, fy, GetCachedPicture("gfx/box_br.lmp"))
}

func enterMenuOptions() {
	IN_Deactivate()
	keyDestination = keys.Menu
	qmenu.state = menu.Options
	qmenu.playEnterSound = true
}

func toggleMenu() {
	qmenu.playEnterSound = true

	if keyDestination == keys.Menu {
		if qmenu.state != menu.Main {
			enterMenuMain()
			return
		}
		enterMenuNone()
		return
	}
	if keyDestination == keys.Console {
		console.Toggle()
	} else {
		enterMenuMain()
	}
}

func enterMenuNone() {
	inputActivate()
	keyDestination = keys.Game
	qmenu.state = menu.None
}

type MenuItem interface {
	Draw()
	DrawCursor()
	Enter()
	Backspace()
	Left()
	Right()
	HandleRune(key rune)
	Update(m qAccept)
	Accept()
}

type qAccept interface {
	Accept()
}

type qMenuItem struct {
	Xcursor int
	Y       int
}

func (m *qMenuItem) Draw() {}
func (m *qMenuItem) DrawCursor() {
	DrawCharacterWhite(m.Xcursor, m.Y, 12+(int(Time()*4))&1)
}
func (m *qMenuItem) Enter()              {}
func (m *qMenuItem) Backspace()          {}
func (m *qMenuItem) Left()               {}
func (m *qMenuItem) Right()              {}
func (m *qMenuItem) HandleRune(key rune) {}
func (m *qMenuItem) Update(a qAccept)    {}
func (m *qMenuItem) Accept()             {}

type qDotMenuItem struct {
	qMenuItem
}

func (m *qDotMenuItem) DrawCursor() {
	i := (int(Time()*10) % 6) + 1
	name := fmt.Sprintf("gfx/menudot%d.lmp", i)
	DrawPicture(m.Xcursor, m.Y, GetCachedPicture(name))
}

type qMenu struct {
	state          int
	playEnterSound bool
}

var (
	qmenu = qMenu{
		state:          menu.None,
		playEnterSound: false,
	}
)

func (m *qMenu) TextEntry() bool {
	switch m.state {
	case menu.Setup:
		return netSetupMenu.TextEntry()
	case menu.NetNewGame:
		return netNewGameMenu.TextEntry()
	case menu.NetJoinGame:
		return netJoinGameMenu.TextEntry()
	}
	return false
}

func (m *qMenu) RuneInput(key rune) {
	switch m.state {
	case menu.Setup:
		netSetupMenu.HandleRune(key)
	case menu.NetNewGame:
		netNewGameMenu.HandleRune(key)
	case menu.NetJoinGame:
		netJoinGameMenu.HandleRune(key)
	}
}

func (m *qMenu) Draw() {
	if m.state == menu.None || keyDestination != keys.Menu {
		return
	}

	if console.currentHeight() != 0 {
		DrawConsoleBackground()
	}

	DrawFadeScreen()

	qCanvas.Set(CANVAS_MENU)

	switch qmenu.state {
	case menu.Main:
		mainMenu.Draw()

	case menu.SinglePlayer:
		singlePlayerMenu.Draw()

	case menu.Load:
		loadMenu.Draw()

	case menu.Save:
		saveMenu.Draw()

	case menu.MultiPlayer:
		multiPlayerMenu.Draw()

	case menu.Setup:
		netSetupMenu.Draw()

	case menu.Options:
		optionsMenu.Draw()

	case menu.Keys:
		keysMenu.Draw()

	case menu.Video:
		videoMenu.Draw()

	case menu.Help:
		helpMenu.Draw()

	case menu.NetJoinGame:
		netJoinGameMenu.Draw()

	case menu.NetNewGame:
		netNewGameMenu.Draw()

	case menu.GameOptions:
		gameOptionsMenu.Draw()
		/*
			case menu.Search:
				C.M_Search_Draw()

			case menu.ServerList:
				C.M_ServerList_Draw()
		*/
	}
	if m.playEnterSound {
		localSound("misc/menu2.wav")
		m.playEnterSound = false
	}
}

func (m *qMenu) HandleKey(k kc.KeyCode) {
	switch m.state {
	case menu.Main:
		mainMenu.HandleKey(k)
	case menu.SinglePlayer:
		singlePlayerMenu.HandleKey(k)
	case menu.Load:
		loadMenu.HandleKey(k)
	case menu.Save:
		saveMenu.HandleKey(k)
	case menu.MultiPlayer:
		multiPlayerMenu.HandleKey(k)
	case menu.Setup:
		netSetupMenu.HandleKey(k)
	case menu.Options:
		optionsMenu.HandleKey(k)
	case menu.Keys:
		keysMenu.HandleKey(k)
	case menu.Video:
		videoMenu.HandleKey(k)
	case menu.Help:
		helpMenu.HandleKey(k)
	case menu.NetNewGame:
		netNewGameMenu.HandleKey(k)
	case menu.NetJoinGame:
		netJoinGameMenu.HandleKey(k)
	case menu.GameOptions:
		gameOptionsMenu.HandleKey(k)
		/*
			case menu.Search:
				C.M_Search_Key(C.int(k))
			case menu.ServerList:
				C.M_ServerList_Key(C.int(k))
		*/
	}
}
