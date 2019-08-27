package quakelib

//#ifndef HASMSTATE
//#define HASMSTATE
//typedef enum {
//  m_none,
//  m_main,
//  m_singleplayer,
//  m_load,
//  m_save,
//  m_multiplayer,
//  m_setup,
//  m_options,
//  m_video,
//  m_keys,
//  m_help,
//  m_lanconfig,
//  m_gameoptions,
//  m_search,
//  m_slist
//} m_state_e;
//
//typedef enum {
//  CANVAS_NONE,
//  CANVAS_DEFAULT,
//  CANVAS_CONSOLE,
//  CANVAS_MENU,
//  CANVAS_SBAR,
//  CANVAS_WARPIMAGE,
//  CANVAS_CROSSHAIR,
//  CANVAS_BOTTOMLEFT,
//  CANVAS_BOTTOMRIGHT,
//  CANVAS_TOPRIGHT,
//  CANVAS_INVALID = -1
//} canvastype;
//
//#endif
// #include "stdlib.h"
// #include "wad.h"
// void	Con_ToggleConsole_f(void);
// void M_Search_Key(int);
// void M_ServerList_Key(int);
// void GL_SetCanvas(canvastype newCanvas);
// float GetScreenConsoleCurrentHeight(void);
// void M_Search_Draw(void);
// void M_ServerList_Draw(void);
// void S_ExtraUpdate(void);
import "C"

import (
	"fmt"
	"quake/cmd"
	kc "quake/keycode"
	"quake/keys"
	"quake/math"
	"quake/menu"
)

const ()

var ()

func nextDemo() {
	CL_NextDemo()
}

func init() {
	cmd.AddCommand("togglemenu", func(_ []cmd.QArg, _ int) { toggleMenu() })

	cmd.AddCommand("menu_main", func(_ []cmd.QArg, _ int) { enterMenuMain() })
	cmd.AddCommand("menu_singleplayer", func(_ []cmd.QArg, _ int) { enterSinglePlayerMenu() })
	cmd.AddCommand("menu_load", func(_ []cmd.QArg, _ int) { enterLoadMenu() })
	cmd.AddCommand("menu_save", func(_ []cmd.QArg, _ int) { enterSaveMenu() })
	cmd.AddCommand("menu_multiplayer", func(_ []cmd.QArg, _ int) { enterMultiPlayerMenu() })
	cmd.AddCommand("menu_setup", func(_ []cmd.QArg, _ int) { enterNetSetupMenu() })
	cmd.AddCommand("menu_options", func(_ []cmd.QArg, _ int) { enterMenuOptions() })
	cmd.AddCommand("menu_keys", func(_ []cmd.QArg, _ int) { enterMenuKeys() })
	cmd.AddCommand("menu_video", func(_ []cmd.QArg, _ int) { enterMenuVideo() })
	cmd.AddCommand("help", func(_ []cmd.QArg, _ int) { enterMenuHelp() })
	cmd.AddCommand("menu_quit", func(_ []cmd.QArg, _ int) { enterQuitMenu() })
}

// 0-127 are white
// 128+ are normal
// We draw on a 320x200 screen
func drawString(x, y int, t string) {
	// TODO: unify into one draw call
	nx := x
	for i := 0; i < len(t); i++ {
		DrawCharacter(nx, y, int(t[i])+128)
		nx += 8
	}
}

func drawStringWhite(x, y int, t string) {
	// TODO: unify into one draw call
	nx := x
	for i := 0; i < len(t); i++ {
		DrawCharacter(nx, y, int(t[i]))
		nx += 8
	}
}

func drawCheckbox(x, y int, checked bool) {
	if checked {
		drawString(x, y, "on")
	} else {
		drawString(x, y, "off")
	}
}

func drawSlider(x, y int, r float32) {
	// TODO: unify into one draw call
	r = math.Clamp32(0, r, 1)
	DrawCharacter(x-8, y, 128)
	for i := 0; i < 10; i++ {
		DrawCharacter(x+i*8, y, 129)
	}
	DrawCharacter(x+10*8, y, 130)
	DrawCharacter(x+int(9*8*r), y, 131)
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
		toggleConsole()
	} else {
		enterMenuMain()
	}
}

func enterMenuNone() {
	IN_Activate()
	keyDestination = keys.Game
	qmenu.state = menu.None
}

func toggleConsole() {
	C.Con_ToggleConsole_f()
}

type MenuItem interface {
	Draw()
	DrawCursor()
	Enter()
	Backspace()
	Left()
	Right()
	HandleChar(key kc.KeyCode)
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
	DrawCharacter(m.Xcursor, m.Y, 12+(int(Time()*4))&1)
}
func (m *qMenuItem) Enter()                    {}
func (m *qMenuItem) Backspace()                {}
func (m *qMenuItem) Left()                     {}
func (m *qMenuItem) Right()                    {}
func (m *qMenuItem) HandleChar(key kc.KeyCode) {}
func (m *qMenuItem) Update(a qAccept)          {}
func (m *qMenuItem) Accept()                   {}

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

func (m *qMenu) Draw() {
	if m.state == menu.None || keyDestination != keys.Menu {
		return
	}

	if C.GetScreenConsoleCurrentHeight() != 0 {
		DrawConsoleBackground()
		C.S_ExtraUpdate()
	}

	DrawFadeScreen()

	C.GL_SetCanvas(C.CANVAS_MENU)

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

	case menu.Search:
		C.M_Search_Draw()

	case menu.ServerList:
		C.M_ServerList_Draw()
	}

	if m.playEnterSound {
		localSound("misc/menu2.wav")
		m.playEnterSound = false
	}

	C.S_ExtraUpdate()
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
	case menu.Search:
		C.M_Search_Key(C.int(k))
	case menu.ServerList:
		C.M_ServerList_Key(C.int(k))
	}
}
