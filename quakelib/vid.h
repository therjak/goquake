/*
Copyright (C) 1996-2001 Id Software, Inc.
Copyright (C) 2002-2009 John Fitzgibbons and others
Copyright (C) 2007-2008 Kristian Duske
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

#ifndef __VID_DEFS_H
#define __VID_DEFS_H

// vid.h -- video driver defs

#define VID_CBITS 6
#define VID_GRADES (1 << VID_CBITS)

typedef struct vrect_s {
  int x, y, width, height;
  struct vrect_s *pnext;
} vrect_t;

void VID_MenuDraw(void);
void VID_MenuKey(int key);
void VID_Menu_f(void);
void VID_Init(void); 
void VID_Shutdown(void);

#endif /* __VID_DEFS_H */
