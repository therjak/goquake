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
// void Draw_Character(int cx, int line, int num);
// qpic_t *Draw_CachePic(const char *path);
// void Draw_Pic(int x, int y, qpic_t *pic);
// void Draw_TransPicTranslate(int x, int y, qpic_t *pic, int top, int bottom);
// void	Con_ToggleConsole_f(void);
// void CL_NextDemo(void);
// void M_Setup_Key(int);
// void M_GameOptions_Key(int);
// void M_Search_Key(int);
// void M_ServerList_Key(int);
// void M_Setup_Char(int);
// int M_Setup_TextEntry();
import "C"

import (
	"quake/menu"
)

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

//export M_Charinput
func M_Charinput(key C.int) {
	switch qmenu.state {
	case menu.Setup:
		netSetupMenu.HandleChar(int(key))
	case menu.NetNewGame:
		netNewGameMenu.HandleChar(int(key))
	case menu.NetJoinGame:
		netJoinGameMenu.HandleChar(int(key))
	}
}

//export M_TextEntry
func M_TextEntry() C.int {
	switch qmenu.state {
	case menu.Setup:
		return b2i(netSetupMenu.TextEntry())
	case menu.NetNewGame:
		return b2i(netNewGameMenu.TextEntry())
	case menu.NetJoinGame:
		return b2i(netJoinGameMenu.TextEntry())
	}
	return 0
}

//export M_Draw
func M_Draw() {
	qmenu.Draw()
}

//export M_Keydown
func M_Keydown(k C.int) {
	qmenu.HandleKey(int(k))
}
