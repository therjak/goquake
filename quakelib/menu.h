#ifndef _QUAKE_MENU_H
#define _QUAKE_MENU_H

#include "_cgo_export.h"

//
// menus
//

void M_Print(int cx, int cy, const char *str);

void M_Draw(void);

void M_DrawPic(int x, int y, qpic_t *pic);

#endif /* _QUAKE_MENU_H */
