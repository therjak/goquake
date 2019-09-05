/*
Copyright (C) 1996-2001 Id Software, Inc.
Copyright (C) 2002-2009 John Fitzgibbons and others
Copyright (C) 2010-2014 QuakeSpasm developers

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 2
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.

See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software
Foundation, Inc., 59 Temple Place - Suite 330, Boston, MA  02111-1307, USA.

*/

#ifndef _QUAKE_SCREEN_H
#define _QUAKE_SCREEN_H
#include "_cgo_export.h"

// screen.h

void SCR_Init(void);
void SCR_LoadPics(void);

void SCR_UpdateScreen(void);

void SCR_SizeUp(void);
void SCR_SizeDown(void);
void SCR_BringDownConsole(void);
void SCR_CenterPrint(const char *str);

void SCR_BeginLoadingPlaque(void);
void SCR_EndLoadingPlaque(void);

int SCR_ModalMessage(const char *text, float timeout);

float GetScreenConsoleCurrentHeight(void);

extern int clearnotify;  // set to 0 whenever notify text is drawn

extern cvar_t scr_viewsize;

extern cvar_t scr_sbaralpha;

void SCR_UpdateWholeScreen(void);

// johnfitz -- stuff for 2d drawing control
/*
typedef enum {
  CANVAS_NONE,
  CANVAS_DEFAULT,
  CANVAS_CONSOLE,
  CANVAS_MENU,
  CANVAS_SBAR,
  CANVAS_WARPIMAGE,
  CANVAS_CROSSHAIR,
  CANVAS_BOTTOMLEFT,
  CANVAS_BOTTOMRIGHT,
  CANVAS_TOPRIGHT,
  CANVAS_INVALID = -1
} canvastype;
*/
extern cvar_t scr_menuscale;
extern cvar_t scr_sbarscale;
extern cvar_t scr_scale;
extern cvar_t scr_crosshairscale;
// johnfitz

extern int scr_tileclear_updates;

#endif /* _QUAKE_SCREEN_H */
