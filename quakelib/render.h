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

#include "entity_state.h"

// refresh.h -- public interface to refresh functions

#define MAXCLIPPLANES 11

#define TOP_RANGE 16  // soldier uniform colors
#define BOTTOM_RANGE 96

//=============================================================================

typedef struct efrag_s {
  struct mleaf_s *leaf;
  struct efrag_s *leafnext;
  struct entity_s *entity;
  struct efrag_s *entnext;
} efrag_t;

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

typedef struct entity_s {
  qboolean forcelink;  // model changed

  int update_type;

  entity_state_t baseline;  // to fill in defaults in updates

  double msgtime;         // time of last update
  vec3_t msg_origins[2];  // last two updates (0 is newest)
  vec3_t origin;
  vec3_t msg_angles[2];  // last two updates (0 is newest)
  vec3_t angles;
  struct qmodel_s *model;  // NULL = no model
  struct efrag_s *efrag;   // linked list of efrags
  int frame;
  float syncbase;  // for client-side animations
  int effects;     // light, particles, etc
  int skinnum;     // for Alias models
  int visframe;    // last frame this entity was
                   //  found in an active leaf

  int dlightframe;  // dynamic lighting
  int dlightbits;

  // FIXME: could turn these into a union
  int trivial_accept;
  struct mnode_s *topnode;  // for bmodels, first world node
                            //  that splits bmodel, or NULL if
                            //  not split

  byte alpha;          // johnfitz -- alpha
  byte lerpflags;      // johnfitz -- lerping
  float lerpstart;     // johnfitz -- animation lerping
  float lerptime;      // johnfitz -- animation lerping
  float lerpfinish;    // johnfitz -- lerping -- server sent us a more accurate
                       // interval, use it instead of 0.1
  short previouspose;  // johnfitz -- animation lerping
  short currentpose;   // johnfitz -- animation lerping
  //	short					futurepose;
  ////johnfitz
  //-- animation
  // lerping
  float movelerpstart;    // johnfitz -- transform lerping
  vec3_t previousorigin;  // johnfitz -- transform lerping
  vec3_t currentorigin;   // johnfitz -- transform lerping
  vec3_t previousangles;  // johnfitz -- transform lerping
  vec3_t currentangles;   // johnfitz -- transform lerping
} entity_t;

//
// refresh
//
extern int reinit_surfcache;

extern vec3_t r_origin, vpn, vright, vup;

void R_Init(void);
void R_InitTextures(void);
void R_InitEfrags(void);
void R_CheckEfrags(void);
void R_AddEfrags(entity_t *ent);
void R_RemoveEfrags(entity_t *ent);

void R_NewMap(void);

void R_PushDlights(void);

//
// surface cache related
//
extern int reinit_surfcache;  // if 1, surface cache is currently empty and

int D_SurfaceCacheForRes(int width, int height);
void D_FlushCaches(void);
void D_DeleteSurfaceCache(void);
void D_InitCaches(void *buffer, int size);

#endif /* _QUAKE_RENDER_H */
