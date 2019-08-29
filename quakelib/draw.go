package quakelib

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

type Picture *C.qpic_t

type QPic struct {
	pic    Picture
	width  int
	height int
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
