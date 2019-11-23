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

#ifndef _QUAKE_WAD_H
#define _QUAKE_WAD_H

// johnfitz -- filename is now hard-coded for honesty
#define WADFILENAME "gfx.wad"

typedef struct {
  int width, height;
  unsigned char data[4];  // variably sized
} qpic_t;

extern unsigned char *wad_base;

void W_LoadWadFile(void);  // johnfitz -- filename is now hard-coded for honesty

qpic_t *W_GetQPic(const char *name);

void SwapPic(qpic_t *pic);

#endif /* _QUAKE_WAD_H */
