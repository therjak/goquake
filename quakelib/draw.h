#ifndef _QUAKE_DRAW_H
#define _QUAKE_DRAW_H
#include "_cgo_export.h"
// draw.h -- these are the only functions outside the refresh allowed
// to touch the vid buffer

void Draw_Init(void);
void Draw_Character(int x, int y, int num);
void Draw_Pic(int x, int y, qpic_t *pic);
void Draw_ConsoleBackground(void);  // johnfitz -- removed parameter int lines
void Draw_TileClear(int x, int y, int w, int h);
void Draw_Fill(int x, int y, int w, int h, int c,
               float alpha);  // johnfitz -- added alpha
void Draw_FadeScreen(void);

void GL_SetCanvas(canvastype newcanvas);  // johnfitz

#endif /* _QUAKE_DRAW_H */
