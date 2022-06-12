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

#ifndef _QUAKE_RENDER_H
#define _QUAKE_RENDER_H

//=============================================================================

// johnfitz -- for lerping
#define LERP_MOVESTEP \
  (1 << 0)  // this is a MOVETYPE_STEP entity, enable movement lerp
#define LERP_RESETANIM (1 << 1)  // disable anim lerping until next anim frame
#define LERP_RESETANIM2 \
  (1 << 2)  // set this and previous flag to disable anim lerping for two anim
            // frames
#define LERP_RESETMOVE \
  (1 << 3)  // disable movement lerping until next origin/angles change
#define LERP_FINISH \
  (1 << 4)  // use lerpfinish time from server update instead of assuming
            // interval of 0.1
// johnfitz

extern vec3_t r_origin, vpn, vright, vup;

void R_Init(void);
void R_NewMap(void);
void R_PushDlights(void);

#endif /* _QUAKE_RENDER_H */
