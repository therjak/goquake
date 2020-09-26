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

#ifndef _QUAKE_PROTOCOL_H
#define _QUAKE_PROTOCOL_H

// protocol.h -- communications protocols

#define PROTOCOL_NETQUAKE 15  // johnfitz -- standard quake protocol
#define PROTOCOL_FITZQUAKE \
  666  // johnfitz -- added new protocol for fitzquake 0.85
#define PROTOCOL_RMQ 999

// PROTOCOL_RMQ protocol flags
#define PRFL_SHORTANGLE (1 << 1)
#define PRFL_FLOATANGLE (1 << 2)
#define PRFL_24BITCOORD (1 << 3)
#define PRFL_FLOATCOORD (1 << 4)
#define PRFL_EDICTSCALE (1 << 5)
#define PRFL_ALPHASANITY (1 << 6)  // cleanup insanity with alpha
#define PRFL_INT32COORD (1 << 7)
#define PRFL_MOREFLAGS (1 << 31)  // not supported

// if the high bit of the servercmd is set, the low bits are fast update flags:
#define U_MOREBITS (1 << 0)
#define U_ORIGIN1 (1 << 1)
#define U_ORIGIN2 (1 << 2)
#define U_ORIGIN3 (1 << 3)
#define U_ANGLE2 (1 << 4)
#define U_STEP \
  (1 << 5)  // johnfitz -- was U_NOLERP, renamed since it's only used for
            // MOVETYPE_STEP
#define U_FRAME (1 << 6)
#define U_SIGNAL (1 << 7)  // just differentiates from other updates

// svc_update can pass all of the fast update bits, plus more
#define U_ANGLE1 (1 << 8)
#define U_ANGLE3 (1 << 9)
#define U_MODEL (1 << 10)
#define U_COLORMAP (1 << 11)
#define U_SKIN (1 << 12)
#define U_EFFECTS (1 << 13)
#define U_LONGENTITY (1 << 14)
// johnfitz -- PROTOCOL_FITZQUAKE -- new bits
#define U_EXTEND1 (1 << 15)
#define U_ALPHA \
  (1 << 16)  // 1 byte, uses ENTALPHA_ENCODE, not sent if equal to baseline
#define U_FRAME2 (1 << 17)  // 1 byte, this is .frame & 0xFF00 (second byte)
#define U_MODEL2 \
  (1 << 18)  // 1 byte, this is .modelindex & 0xFF00 (second byte)
#define U_LERPFINISH \
  (1 << 19)  // 1 byte, 0.0-1.0 maps to 0-255, not sent if exactly 0.1, this is
             // ent->v.nextthink - sv.time, used for lerping
#define U_SCALE \
  (1 << 20)  // 1 byte, for PROTOCOL_RMQ PRFL_EDICTSCALE, currently read but
             // ignored
#define U_UNUSED21 (1 << 21)
#define U_UNUSED22 (1 << 22)
#define U_EXTEND2 (1 << 23)  // another byte to follow, future expansion
// johnfitz

// johnfitz -- PROTOCOL_NEHAHRA transparency
#define U_TRANS (1 << 15)
// johnfitz

// johnfitz -- PROTOCOL_FITZQUAKE -- alpha encoding
#define ENTALPHA_DEFAULT \
  0  // entity's alpha is "default" (i.e. water obeys r_wateralpha) -- must be
     // zero so zeroed out memory works
#define ENTALPHA_ZERO 1   // entity is invisible (lowest possible alpha)
#define ENTALPHA_ENCODE(a)               \
  (((a) == 0)                            \
       ? ENTALPHA_DEFAULT                \
       : Q_rint(CLAMP(1, (a)*254.0f + 1, \
                      255)))  // server convert to byte to send to client
#define ENTALPHA_DECODE(a)   \
  (((a) == ENTALPHA_DEFAULT) \
       ? 1.0f                \
       : ((float)(a)-1) / (254))  // client convert to float for rendering
// johnfitz

#endif /* _QUAKE_PROTOCOL_H */
