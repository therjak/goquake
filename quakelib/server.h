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

#ifndef _QUAKE_SERVER_H
#define _QUAKE_SERVER_H

// server.h

//=============================================================================

typedef enum { ss_loading, ss_active } server_state_t;

typedef struct {
  struct qmodel_s *worldmodel;
  struct qmodel_s *models[MAX_MODELS];
  const char *lightstyles[MAX_LIGHTSTYLES];
  edict_t *edicts;  // can NOT be array indexed, because
                    // edict_t is variable sized, but can
                    // be used to reference the world ent
} server_t;

const char *SV_Name();
const char *SV_ModelName();

#define NUM_PING_TIMES 16
#define NUM_SPAWN_PARMS 16

edict_t *SV_GetEdict(int cl);

void SV_SetEdictNum(int cl, int num);

//=============================================================================

// edict->movetype values
#define MOVETYPE_NONE 0  // never moves
#define MOVETYPE_ANGLENOCLIP 1
#define MOVETYPE_ANGLECLIP 2
#define MOVETYPE_WALK 3  // gravity
#define MOVETYPE_STEP 4  // gravity, special edge handling
#define MOVETYPE_FLY 5
#define MOVETYPE_TOSS 6  // gravity
#define MOVETYPE_PUSH 7  // no clip to world, push and crush
#define MOVETYPE_NOCLIP 8
#define MOVETYPE_FLYMISSILE 9  // extra size to monsters
#define MOVETYPE_BOUNCE 10

// edict->solid values
#define SOLID_NOT 0       // no interaction with other objects
#define SOLID_TRIGGER 1   // touch on edge, but not blocking
#define SOLID_BBOX 2      // touch on edge, block
#define SOLID_SLIDEBOX 3  // touch on edge, but not an onground
#define SOLID_BSP 4       // bsp clip, touch on edge, block

// edict->deadflag values
#define DEAD_NO 0
#define DEAD_DYING 1
#define DEAD_DEAD 2

#define DAMAGE_NO 0
#define DAMAGE_YES 1
#define DAMAGE_AIM 2

// edict->flags
#define FL_FLY 1
#define FL_SWIM 2
//#define	FL_GLIMPSE				4
#define FL_CONVEYOR 4
#define FL_CLIENT 8
#define FL_INWATER 16
#define FL_MONSTER 32
#define FL_GODMODE 64
#define FL_NOTARGET 128
#define FL_ITEM 256
#define FL_ONGROUND 512
#define FL_PARTIALGROUND 1024  // not all corners are valid
#define FL_WATERJUMP 2048      // player jumping out of water
#define FL_JUMPRELEASED 4096   // for jump debouncing

// entity effects

#define EF_BRIGHTFIELD 1
#define EF_MUZZLEFLASH 2
#define EF_BRIGHTLIGHT 4
#define EF_DIMLIGHT 8

#define SPAWNFLAG_NOT_EASY 256
#define SPAWNFLAG_NOT_MEDIUM 512
#define SPAWNFLAG_NOT_HARD 1024
#define SPAWNFLAG_NOT_DEATHMATCH 2048

//============================================================================

extern cvar_t teamplay;
extern cvar_t skill;
extern cvar_t deathmatch;
extern cvar_t coop;
extern cvar_t fraglimit;
extern cvar_t timelimit;

extern server_t sv;  // local server

extern int host_client;
int HostClient(void);

//===========================================================

void SV_Init(void);

void SV_StartParticle(vec3_t org, vec3_t dir, int color, int count);
void SV_StartSound(edict_t *entity, int channel, const char *sample, int volume,
                   float attenuation);

void SV_DropClient(int client, qboolean crash);

void SV_SendClientMessages(void);
void SV_ClearDatagram(void);

void SV_SetIdealPitch(void);

void SV_AddUpdates(void);

void SV_AddClientToServer(int ret);

void SV_ClientPrintf2(int client, const char *fmt, ...);
//    __attribute__((__format__(__printf__, 1, 2)));
void SV_BroadcastPrintf(const char *fmt, ...)
    __attribute__((__format__(__printf__, 1, 2)));

void SV_Physics(void);

qboolean SV_CheckBottom(edict_t *ent);
qboolean SV_movestep(edict_t *ent, vec3_t move, qboolean relink);

void SV_WriteClientdataToMessage(edict_t *ent);

void SV_MoveToGoal(void);

void SV_CheckForNewClients(void);
void SV_RunClients(void);
void SV_SaveSpawnparms();
void SV_SpawnServer(const char *server);

#endif /* _QUAKE_SERVER_H */
