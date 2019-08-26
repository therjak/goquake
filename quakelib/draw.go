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

func DrawCharacter(x, y int, num int) {
	C.Draw_Character(C.int(x), C.int(y), C.int(num))
}

func DrawPicture(x, y int, p Picture) {
	C.Draw_Pic(C.int(x), C.int(y), p)
}

func DrawPictureAlpha(x, y int, p Picture, alpha float32) {
	C.Draw_PicAlpha(C.int(x), C.int(y), p, C.float(alpha))
}

func DrawTransparentPictureTranslate(x, y int, p Picture, top, bottom int) {
	C.Draw_TransPicTranslate(C.int(x), C.int(y), p, C.int(top), C.int(bottom))
}

func GetCachedPicture(name string) Picture {
	n := C.CString(name)
	p := C.Draw_CachePic(n)
	C.free(unsafe.Pointer(n))
	return p
}

func GetPictureFromWad(name string) Picture {
	n := C.CString(name)
	p := C.Draw_PicFromWad(n)
	C.free(unsafe.Pointer(n))
	return p
}
