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
// #include "draw.h"
import "C"

import (
	kc "quake/keycode"
	"quake/menu"
)

//export DrawConsoleBackgroundC
func DrawConsoleBackgroundC() {
	DrawConsoleBackground()
}

//export DrawFillC
func DrawFillC(x, y, w, h int, c int, alpha float32) {
	DrawFill(x, y, w, h, c, alpha)
}

//export MENU_SetEnterSound
func MENU_SetEnterSound(v C.int) {
	qmenu.playEnterSound = (v != 0)
}

//export MENU_SetState
func MENU_SetState(s C.m_state_e) {
	switch s {
	default: //case C.m_none:
		qmenu.state = menu.None
	case C.m_gameoptions:
		qmenu.state = menu.GameOptions
	case C.m_search:
		qmenu.state = menu.Search
	case C.m_slist:
		qmenu.state = menu.ServerList
	}
}

//export M_ToggleMenu_f
func M_ToggleMenu_f() {
	toggleMenu()
}

//export M_Menu_Main_f
func M_Menu_Main_f() {
	enterMenuMain()
}

//export M_Menu_LanConfig_f
func M_Menu_LanConfig_f() {
	enterNetJoinGameMenu()
}

//export M_Draw
func M_Draw() {
	qmenu.Draw()
}

//export M_Keydown
func M_Keydown(k C.int) {
	qmenu.HandleKey(kc.KeyCode(k))
}
