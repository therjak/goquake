package quakelib

//#ifndef HASCANVAS
//#define HASCANVAS
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
//#endif
// void GL_SetCanvas(canvastype newCanvas);
//#include "stdlib.h"
//#include "wad.h"
//void Draw_Character(int x, int y, int num);
//void Draw_Pic(int x, int y, qpic_t *pic);
//void Draw_TransPicTranslate(int x, int y, qpic_t *pic, int top, int bottom);
//void Draw_PicAlpha(int x, int y, qpic_t *pic, float alpha);
//void Draw_ConsoleBackground(void);
//void Draw_TileClear(int x, int y, int w, int h);
//void Draw_Fill(int x, int y, int w, int h, int c, float alpha);
//void Draw_FadeScreen(void);
//void Draw_String(int x, int y, const char *str);
//qpic_t *Draw_PicFromWad(const char *name);
//qpic_t *Draw_CachePic(const char *path);
//void Draw_NewGame(void);
import "C"

import (
	"unsafe"
)

type canvas int

const (
	CANVAS_NONE        canvas = C.CANVAS_NONE
	CANVAS_DEFAULT     canvas = C.CANVAS_DEFAULT
	CANVAS_CONSOLE     canvas = C.CANVAS_CONSOLE
	CANVAS_MENU        canvas = C.CANVAS_MENU
	CANVAS_STATUSBAR   canvas = C.CANVAS_SBAR
	CANVAS_WARPIMAGE   canvas = C.CANVAS_WARPIMAGE
	CANVAS_CROSSHAIR   canvas = C.CANVAS_CROSSHAIR
	CANVAS_BOTTOMLEFT  canvas = C.CANVAS_BOTTOMLEFT
	CANVAS_BOTTOMRIGHT canvas = C.CANVAS_BOTTOMRIGHT
	CANVAS_TOPRIGHT    canvas = C.CANVAS_TOPRIGHT
	CANVAS_INVALID     canvas = C.CANVAS_INVALID
)

func SetCanvas(c canvas) {
	C.GL_SetCanvas(C.canvastype(c))
}

type Picture *C.qpic_t

type QPic struct {
	pic    Picture
	width  int
	height int
}

func DrawCharacterWhite(x, y int, num int) {
	C.Draw_Character(C.int(x), C.int(y), C.int(num))
}

func DrawCharacterCopper(x, y int, num int) {
	C.Draw_Character(C.int(x), C.int(y), C.int(num+128))
}

func DrawCharacter(x, y int, num int) {
	C.Draw_Character(C.int(x), C.int(y), C.int(num))
}

func DrawPicture(x, y int, p *QPic) {
	C.Draw_Pic(C.int(x), C.int(y), p.pic)
}

func DrawPictureAlpha(x, y int, p *QPic, alpha float32) {
	C.Draw_PicAlpha(C.int(x), C.int(y), p.pic, C.float(alpha))
}

func DrawTransparentPictureTranslate(x, y int, p *QPic, top, bottom int) {
	C.Draw_TransPicTranslate(C.int(x), C.int(y), p.pic, C.int(top), C.int(bottom))
}

func DrawConsoleBackground() {
	C.Draw_ConsoleBackground()
}

func DrawFadeScreen() {
	C.Draw_FadeScreen()
}

func DrawFill(x, y, w, h int, c int, alpha float32) {
	C.Draw_Fill(C.int(x), C.int(y), C.int(w), C.int(h), C.int(c), C.float(alpha))
}

func GetCachedPicture(name string) *QPic {
	n := C.CString(name)
	p := C.Draw_CachePic(n)
	C.free(unsafe.Pointer(n))
	return &QPic{
		pic:    p,
		width:  int(p.width),
		height: int(p.height),
	}
}

func GetPictureFromWad(name string) *QPic {
	n := C.CString(name)
	p := C.Draw_PicFromWad(n)
	C.free(unsafe.Pointer(n))
	return &QPic{
		pic:    p,
		width:  int(p.width),
		height: int(p.height),
	}
}

// 0-127 are white
// 128+ are normal
// We draw on a 320x200 screen
func DrawStringCopper(x, y int, t string) {
	// TODO: unify into one draw call
	nx := x
	for i := 0; i < len(t); i++ {
		DrawCharacterCopper(nx, y, int(t[i]))
		nx += 8
	}
}

func DrawStringWhite(x, y int, t string) {
	// TODO: unify into one draw call
	nx := x
	for i := 0; i < len(t); i++ {
		DrawCharacterWhite(nx, y, int(t[i]))
		nx += 8
	}
}
