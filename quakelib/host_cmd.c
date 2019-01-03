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

#include "quakedef.h"
#ifndef _WIN32
#include <dirent.h>
#endif

extern cvar_t pausable;

int current_skill;

void Mod_Print(void);

/*
==================
Host_Quit_f
==================
*/

void Host_Quit_f(void) {
  if (GetKeyDest() != key_console && CLS_GetState() != ca_dedicated) {
    M_Menu_Quit_f();
    return;
  }
  CL_Disconnect();
  Host_ShutdownServer(false);

  Sys_Quit();
}

//==============================================================================
// johnfitz -- extramaps management
//==============================================================================

// ericw -- was extralevel_t, renamed and now used with mods list as well
// to simplify completion code
typedef struct filelist_item_s {
  char name[32];
  struct filelist_item_s *next;
} filelist_item_t;

/*
==================
FileList_Add
==================
*/
void FileList_Add(const char *name, filelist_item_t **list) {
  filelist_item_t *item, *cursor, *prev;

  // ignore duplicate
  for (item = *list; item; item = item->next) {
    if (!Q_strcmp(name, item->name)) return;
  }

  item = (filelist_item_t *)Z_Malloc(sizeof(filelist_item_t));
  q_strlcpy(item->name, name, sizeof(item->name));

  // insert each entry in alphabetical order
  if (*list == NULL ||
      q_strcasecmp(item->name, (*list)->name) < 0)  // insert at front
  {
    item->next = *list;
    *list = item;
  } else  // insert later
  {
    prev = *list;
    cursor = (*list)->next;
    while (cursor && (q_strcasecmp(item->name, cursor->name) > 0)) {
      prev = cursor;
      cursor = cursor->next;
    }
    item->next = prev->next;
    prev->next = item;
  }
}

static void FileList_Clear(filelist_item_t **list) {
  filelist_item_t *blah;

  while (*list) {
    blah = (*list)->next;
    Z_Free(*list);
    *list = blah;
  }
}

filelist_item_t *extralevels;

void ExtraMaps_Add(const char *name) { FileList_Add(name, &extralevels); }

void ExtraMaps_Init(void) {
  /*
  DIR *dir_p;
  struct dirent *dir_t;
  char filestring[MAX_OSPATH];
  char mapname[32];
  char ignorepakdir[32];
  searchpath_t *search;
  pack_t *pak;
  int i;

  // we don't want to list the maps in id1 pakfiles,
  // because these are not "add-on" levels
  q_snprintf(ignorepakdir, sizeof(ignorepakdir), "/%s/", GAMENAME);

  for (search = com_searchpaths; search; search = search->next) {
    if (*search->filename)  // directory
    {
      q_snprintf(filestring, sizeof(filestring), "%s/maps/", search->filename);
      dir_p = opendir(filestring);
      if (dir_p == NULL) continue;
      while ((dir_t = readdir(dir_p)) != NULL) {
        if (q_strcasecmp(COM_FileGetExtension(dir_t->d_name), "bsp") != 0)
          continue;
        COM_StripExtension(dir_t->d_name, mapname, sizeof(mapname));
        ExtraMaps_Add(mapname);
      }
      closedir(dir_p);
    } else  // pakfile
    {
      if (!strstr(search->pack->filename,
                  ignorepakdir)) {  // don't list standard id maps
        for (i = 0, pak = search->pack; i < pak->numfiles; i++) {
          if (!strcmp(COM_FileGetExtension(pak->files[i].name), "bsp")) {
            if (pak->files[i].filelen >
                32 * 1024) {  // don't list files under 32k (ammo boxes etc)
              COM_StripExtension(pak->files[i].name + 5, mapname,
                                 sizeof(mapname));
              ExtraMaps_Add(mapname);
            }
          }
        }
      }
    }
  }
  */
}

static void ExtraMaps_Clear(void) { FileList_Clear(&extralevels); }

void ExtraMaps_NewGame(void) {
  ExtraMaps_Clear();
  ExtraMaps_Init();
}

/*
==================
Host_Maps_f
==================
*/
void Host_Maps_f(void) {
  int i;
  filelist_item_t *level;

  for (level = extralevels, i = 0; level; level = level->next, i++)
    Con_SafePrintf("   %s\n", level->name);

  if (i)
    Con_SafePrintf("%i map(s)\n", i);
  else
    Con_SafePrintf("no maps found\n");
}

//==============================================================================
// johnfitz -- modlist management
//==============================================================================

filelist_item_t *modlist;

void Modlist_Add(const char *name) { FileList_Add(name, &modlist); }

#ifdef _WIN32
void Modlist_Init(void) {
  WIN32_FIND_DATA fdat, mod_fdat;
  HANDLE fhnd, mod_fhnd;
  char dir_string[MAX_OSPATH], mod_string[MAX_OSPATH];

  q_snprintf(dir_string, sizeof(dir_string), "%s/*", Com_Basedir());
  fhnd = FindFirstFile(dir_string, &fdat);
  if (fhnd == INVALID_HANDLE_VALUE) return;

  do {
    if (!strcmp(fdat.cFileName, ".")) continue;

    q_snprintf(mod_string, sizeof(mod_string), "%s/%s/progs.dat", Com_Basedir(),
               fdat.cFileName);
    mod_fhnd = FindFirstFile(mod_string, &mod_fdat);
    if (mod_fhnd != INVALID_HANDLE_VALUE) {
      FindClose(mod_fhnd);
      Modlist_Add(fdat.cFileName);
    } else {
      q_snprintf(mod_string, sizeof(mod_string), "%s/%s/*.pak", Com_Basedir(),
                 fdat.cFileName);
      mod_fhnd = FindFirstFile(mod_string, &mod_fdat);
      if (mod_fhnd != INVALID_HANDLE_VALUE) {
        FindClose(mod_fhnd);
        Modlist_Add(fdat.cFileName);
      }
    }
  } while (FindNextFile(fhnd, &fdat));

  FindClose(fhnd);
}
#else
void Modlist_Init(void) {
  DIR *dir_p, *mod_dir_p;
  struct dirent *dir_t, *mod_dir_t;
  char dir_string[MAX_OSPATH], mod_string[MAX_OSPATH];

  q_snprintf(dir_string, sizeof(dir_string), "%s/", Com_Basedir());
  dir_p = opendir(dir_string);
  if (dir_p == NULL) return;

  while ((dir_t = readdir(dir_p)) != NULL) {
    if (!strcmp(dir_t->d_name, ".") || !strcmp(dir_t->d_name, "..")) continue;
    q_snprintf(mod_string, sizeof(mod_string), "%s%s/", dir_string,
               dir_t->d_name);
    mod_dir_p = opendir(mod_string);
    if (mod_dir_p == NULL) continue;
    // find progs.dat and pak file(s)
    while ((mod_dir_t = readdir(mod_dir_p)) != NULL) {
      if (!q_strcasecmp(mod_dir_t->d_name, "progs.dat")) {
        Modlist_Add(dir_t->d_name);
        break;
      }
      if (!q_strcasecmp(COM_FileGetExtension(mod_dir_t->d_name), "pak")) {
        Modlist_Add(dir_t->d_name);
        break;
      }
    }
    closedir(mod_dir_p);
  }

  closedir(dir_p);
}
#endif

//==============================================================================
// ericw -- demo list management
//==============================================================================

filelist_item_t *demolist;

static void DemoList_Clear(void) { FileList_Clear(&demolist); }

void DemoList_Rebuild(void) {
  DemoList_Clear();
  DemoList_Init();
}

// TODO: Factor out to a general-purpose file searching function
void DemoList_Init(void) {
  /*
  DIR *dir_p;
  struct dirent *dir_t;
  char filestring[MAX_OSPATH];
  char demname[32];
  char ignorepakdir[32];
  searchpath_t *search;
  pack_t *pak;
  int i;

  // we don't want to list the demos in id1 pakfiles,
  // because these are not "add-on" demos
  q_snprintf(ignorepakdir, sizeof(ignorepakdir), "/%s/", GAMENAME);

  for (search = com_searchpaths; search; search = search->next) {
    if (*search->filename)  // directory
    {
      q_snprintf(filestring, sizeof(filestring), "%s/", search->filename);
      dir_p = opendir(filestring);
      if (dir_p == NULL) continue;
      while ((dir_t = readdir(dir_p)) != NULL) {
        if (q_strcasecmp(COM_FileGetExtension(dir_t->d_name), "dem") != 0)
          continue;
        COM_StripExtension(dir_t->d_name, demname, sizeof(demname));
        FileList_Add(demname, &demolist);
      }
      closedir(dir_p);
    } else  // pakfile
    {
      if (!strstr(search->pack->filename,
                  ignorepakdir)) {  // don't list standard id demos
        for (i = 0, pak = search->pack; i < pak->numfiles; i++) {
          if (!strcmp(COM_FileGetExtension(pak->files[i].name), "dem")) {
            COM_StripExtension(pak->files[i].name, demname, sizeof(demname));
            FileList_Add(demname, &demolist);
          }
        }
      }
    }
  }
  */
}

/*
==================
Host_Mods_f -- johnfitz

list all potential mod directories (contain either a pak file or a progs.dat)
==================
*/
void Host_Mods_f(void) {
  int i;
  filelist_item_t *mod;

  for (mod = modlist, i = 0; mod; mod = mod->next, i++)
    Con_SafePrintf("   %s\n", mod->name);

  if (i)
    Con_SafePrintf("%i mod(s)\n", i);
  else
    Con_SafePrintf("no mods found\n");
}

//==============================================================================

/*
=============
Host_Mapname_f -- johnfitz
=============
*/
void Host_Mapname_f(void) {
  if (SV_Active()) {
    Con_Printf("\"mapname\" is \"%s\"\n", sv.name);
    return;
  }

  if (CLS_GetState() == ca_connected) {
    Con_Printf("\"mapname\" is \"%s\"\n", cl.mapname);
    return;
  }

  Con_Printf("no map loaded\n");
}

/*
==================
Host_Status_f
==================
*/
void host_status_clientPrintf(const char *fmt, ...) {
  va_list argptr;
  char string[1024];

  va_start(argptr, fmt);
  q_vsnprintf(string, sizeof(string), fmt, argptr);
  va_end(argptr);
  SV_ClientPrint2(HostClient(), string);
}

void Host_Status_f(void) {
  int seconds;
  int minutes;
  int hours = 0;
  int j;
  void (*print_fn)(const char *fmt, ...)
      __fp_attribute__((__format__(__printf__, 1, 2)));

  if (IsSrcCommand()) {
    if (!SV_Active()) {
      Cmd_ForwardToServer();
      return;
    }
    print_fn = Con_Printf;
  } else {
    print_fn = host_status_clientPrintf;
  }

  print_fn("host:    %s\n", Cvar_GetString(&hostname));
  print_fn("version: %4.2f\n", VERSION);
  if (NETtcpipAvailable()) print_fn("tcp/ip:  %s\n", my_tcpip_address);
  print_fn("map:     %s\n", sv.name);
  int active = 0;
  for (j = 0; j < SVS_GetMaxClients(); j++) {
    if (GetClientActive(j)) {
      active++;
    }
  }
  print_fn("players: %i active (%i max)\n\n", active,
           SVS_GetMaxClients());
  for (j = 0; j < SVS_GetMaxClients(); j++) {
    if (!GetClientActive(j)) continue;
    seconds = (int)(NET_GetTime() - ClientConnectTime(j));
    minutes = seconds / 60;
    if (minutes) {
      seconds -= (minutes * 60);
      hours = minutes / 60;
      if (hours) minutes -= (hours * 60);
    } else
      hours = 0;
    char *name = GetClientName(j);
    print_fn("#%-2u %-16.16s  %3i  %2i:%02i:%02i\n", j + 1, name,
             (int)EdictV(SV_GetEdict(j))->frags, hours, minutes, seconds);
    free(name);
    print_fn("   %s\n", NET_QSocketGetAddressString(j));
  }
}

/*
==================
Host_God_f

Sets client to godmode
==================
*/
void Host_God_f(void) {
  if (IsSrcCommand()) {
    Cmd_ForwardToServer();
    return;
  }

  if (pr_global_struct->deathmatch) return;

  // johnfitz -- allow user to explicitly set god mode to on or off
  switch (Cmd_Argc()) {
    case 1:
      EdictV(sv_player)->flags = (int)EdictV(sv_player)->flags ^ FL_GODMODE;
      if (!((int)EdictV(sv_player)->flags & FL_GODMODE))
        SV_ClientPrintf2(HostClient(), "godmode OFF\n");
      else
        SV_ClientPrintf2(HostClient(), "godmode ON\n");
      break;
    case 2:
      if (Cmd_ArgvAsInt(1)) {
        EdictV(sv_player)->flags = (int)EdictV(sv_player)->flags | FL_GODMODE;
        SV_ClientPrintf2(HostClient(), "godmode ON\n");
      } else {
        EdictV(sv_player)->flags = (int)EdictV(sv_player)->flags & ~FL_GODMODE;
        SV_ClientPrintf2(HostClient(), "godmode OFF\n");
      }
      break;
    default:
      Con_Printf("god [value] : toggle god mode. values: 0 = off, 1 = on\n");
      break;
  }
  // johnfitz
}

/*
==================
Host_Notarget_f
==================
*/
void Host_Notarget_f(void) {
  if (IsSrcCommand()) {
    Cmd_ForwardToServer();
    return;
  }

  if (pr_global_struct->deathmatch) return;

  // johnfitz -- allow user to explicitly set notarget to on or off
  switch (Cmd_Argc()) {
    case 1:
      EdictV(sv_player)->flags = (int)EdictV(sv_player)->flags ^ FL_NOTARGET;
      if (!((int)EdictV(sv_player)->flags & FL_NOTARGET))
        SV_ClientPrintf2(HostClient(), "notarget OFF\n");
      else
        SV_ClientPrintf2(HostClient(), "notarget ON\n");
      break;
    case 2:
      if (Cmd_ArgvAsInt(1)) {
        EdictV(sv_player)->flags = (int)EdictV(sv_player)->flags | FL_NOTARGET;
        SV_ClientPrintf2(HostClient(), "notarget ON\n");
      } else {
        EdictV(sv_player)->flags = (int)EdictV(sv_player)->flags & ~FL_NOTARGET;
        SV_ClientPrintf2(HostClient(), "notarget OFF\n");
      }
      break;
    default:
      Con_Printf(
          "notarget [value] : toggle notarget mode. values: 0 = off, 1 = on\n");
      break;
  }
  // johnfitz
}

qboolean noclip_anglehack;

/*
==================
Host_Noclip_f
==================
*/
void Host_Noclip_f(void) {
  if (IsSrcCommand()) {
    Cmd_ForwardToServer();
    return;
  }

  if (pr_global_struct->deathmatch) return;

  // johnfitz -- allow user to explicitly set noclip to on or off
  switch (Cmd_Argc()) {
    case 1:
      if (EdictV(sv_player)->movetype != MOVETYPE_NOCLIP) {
        noclip_anglehack = true;
        EdictV(sv_player)->movetype = MOVETYPE_NOCLIP;
        SV_ClientPrintf2(HostClient(), "noclip ON\n");
      } else {
        noclip_anglehack = false;
        EdictV(sv_player)->movetype = MOVETYPE_WALK;
        SV_ClientPrintf2(HostClient(), "noclip OFF\n");
      }
      break;
    case 2:
      if (Cmd_ArgvAsInt(1)) {
        noclip_anglehack = true;
        EdictV(sv_player)->movetype = MOVETYPE_NOCLIP;
        SV_ClientPrintf2(HostClient(), "noclip ON\n");
      } else {
        noclip_anglehack = false;
        EdictV(sv_player)->movetype = MOVETYPE_WALK;
        SV_ClientPrintf2(HostClient(), "noclip OFF\n");
      }
      break;
    default:
      Con_Printf(
          "noclip [value] : toggle noclip mode. values: 0 = off, 1 = on\n");
      break;
  }
  // johnfitz
}

/*
====================
Host_SetPos_f

adapted from fteqw, originally by Alex Shadowalker
====================
*/
void Host_SetPos_f(void) {
  if (IsSrcCommand()) {
    Cmd_ForwardToServer();
    return;
  }

  if (pr_global_struct->deathmatch) return;

  if (Cmd_Argc() != 7 && Cmd_Argc() != 4) {
    SV_ClientPrintf2(HostClient(), "usage:\n");
    SV_ClientPrintf2(HostClient(), "   setpos <x> <y> <z>\n");
    SV_ClientPrintf2(HostClient(),
                     "   setpos <x> <y> <z> <pitch> <yaw> <roll>\n");
    SV_ClientPrintf2(HostClient(), "current values:\n");
    SV_ClientPrintf2(HostClient(), "   %i %i %i %i %i %i\n",
                     (int)EdictV(sv_player)->origin[0],
                     (int)EdictV(sv_player)->origin[1],
                     (int)EdictV(sv_player)->origin[2],
                     (int)EdictV(sv_player)->v_angle[0],
                     (int)EdictV(sv_player)->v_angle[1],
                     (int)EdictV(sv_player)->v_angle[2]);
    return;
  }

  if (EdictV(sv_player)->movetype != MOVETYPE_NOCLIP) {
    noclip_anglehack = true;
    EdictV(sv_player)->movetype = MOVETYPE_NOCLIP;
    SV_ClientPrintf2(HostClient(), "noclip ON\n");
  }

  // make sure they're not going to whizz away from it
  EdictV(sv_player)->velocity[0] = 0;
  EdictV(sv_player)->velocity[1] = 0;
  EdictV(sv_player)->velocity[2] = 0;

  EdictV(sv_player)->origin[0] = Cmd_ArgvAsDouble(1);
  EdictV(sv_player)->origin[1] = Cmd_ArgvAsDouble(2);
  EdictV(sv_player)->origin[2] = Cmd_ArgvAsDouble(3);

  if (Cmd_Argc() == 7) {
    EdictV(sv_player)->angles[0] = Cmd_ArgvAsDouble(4);
    EdictV(sv_player)->angles[1] = Cmd_ArgvAsDouble(5);
    EdictV(sv_player)->angles[2] = Cmd_ArgvAsDouble(6);
    EdictV(sv_player)->fixangle = 1;
  }

  SV_LinkEdict(sv_player, false);
}

/*
==================
Host_Fly_f

Sets client to flymode
==================
*/
void Host_Fly_f(void) {
  if (IsSrcCommand()) {
    Cmd_ForwardToServer();
    return;
  }

  if (pr_global_struct->deathmatch) return;

  // johnfitz -- allow user to explicitly set noclip to on or off
  switch (Cmd_Argc()) {
    case 1:
      if (EdictV(sv_player)->movetype != MOVETYPE_FLY) {
        EdictV(sv_player)->movetype = MOVETYPE_FLY;
        SV_ClientPrintf2(HostClient(), "flymode ON\n");
      } else {
        EdictV(sv_player)->movetype = MOVETYPE_WALK;
        SV_ClientPrintf2(HostClient(), "flymode OFF\n");
      }
      break;
    case 2:
      if (Cmd_ArgvAsInt(1)) {
        EdictV(sv_player)->movetype = MOVETYPE_FLY;
        SV_ClientPrintf2(HostClient(), "flymode ON\n");
      } else {
        EdictV(sv_player)->movetype = MOVETYPE_WALK;
        SV_ClientPrintf2(HostClient(), "flymode OFF\n");
      }
      break;
    default:
      Con_Printf("fly [value] : toggle fly mode. values: 0 = off, 1 = on\n");
      break;
  }
  // johnfitz
}

/*
==================
Host_Ping_f

==================
*/
void Host_Ping_f(void) {
  int i, j;
  float total;

  if (IsSrcCommand()) {
    Cmd_ForwardToServer();
    return;
  }

  SV_ClientPrintf2(HostClient(), "Client ping times:\n");
  for (i = 0; i < SVS_GetMaxClients(); i++) {
    if (!GetClientActive(i)) continue;
    total = 0;
    for (j = 0; j < NUM_PING_TIMES; j++) {
      total += GetClientPingTime(i, j);
    }
    total /= NUM_PING_TIMES;
    char *name = GetClientName(i);
    SV_ClientPrintf2(HostClient(), "%4i %s\n", (int)(total * 1000), name);
    free(name);
  }
}

/*
===============================================================================

SERVER TRANSITIONS

===============================================================================
*/

/*
======================
Host_Map_f

handle a
map <servername>
command from the console.  Active clients are kicked off.
======================
*/
void Host_Map_f(void) {
  int i;
  char name[MAX_QPATH], *p;

  if (Cmd_Argc() < 2)  // no map name given
  {
    if (CLS_GetState() == ca_dedicated) {
      if (SV_Active())
        Con_Printf("Current map: %s\n", sv.name);
      else
        Con_Printf("Server not active\n");
    } else if (CLS_GetState() == ca_connected) {
      Con_Printf("Current map: %s ( %s )\n", cl.levelname, cl.mapname);
    } else {
      Con_Printf("map <levelname>: start a new server\n");
    }
    return;
  }

  if (!IsSrcCommand()) return;

  CLS_StopDemoCycle();  // stop demo loop in case this fails

  CL_Disconnect();
  Host_ShutdownServer(false);

  if (CLS_GetState() != ca_dedicated) IN_Activate();
  SetKeyDest(key_game);  // remove console or menu
  SCR_BeginLoadingPlaque();

  SVS_SetServerFlags(0);  // haven't completed an episode yet
  q_strlcpy(name, Cmd_Argv(1), sizeof(name));
  // remove (any) trailing ".bsp" from mapname -- S.A.
  p = strstr(name, ".bsp");
  if (p && p[4] == '\0') *p = '\0';
  SV_SpawnServer(name);
  if (!SV_Active()) return;

  if (CLS_GetState() != ca_dedicated) {
    memset(cls.spawnparms, 0, MAX_MAPSTRING);
    for (i = 2; i < Cmd_Argc(); i++) {
      q_strlcat(cls.spawnparms, Cmd_Argv(i), MAX_MAPSTRING);
      q_strlcat(cls.spawnparms, " ", MAX_MAPSTRING);
    }

    Cmd_ExecuteString("connect local", src_command);
  }
}

/*
==================
Host_Changelevel_f

Goes to a new map, taking all clients along
==================
*/
void Host_Changelevel_f(void) {
  char level[MAX_QPATH];

  if (Cmd_Argc() != 2) {
    Con_Printf("changelevel <levelname> : continue game on a new level\n");
    return;
  }
  if (!SV_Active() || CLS_IsDemoPlayback()) {
    Con_Printf("Only the server may changelevel\n");
    return;
  }

  // johnfitz -- check for client having map before anything else
  q_snprintf(level, sizeof(level), "maps/%s.bsp", Cmd_Argv(1));
  if (!COM_FileExists(level)) Host_Error("cannot find map %s", level);
  // johnfitz

  if (CLS_GetState() != ca_dedicated) IN_Activate();  // -- S.A.
  SetKeyDest(key_game);                               // remove console or menu
  SV_SaveSpawnparms();
  q_strlcpy(level, Cmd_Argv(1), sizeof(level));
  SV_SpawnServer(level);
  // also issue an error if spawn failed -- O.S.
  if (!SV_Active()) Host_Error("cannot run map %s", level);
}

/*
==================
Host_Restart_f

Restarts the current server for a dead player
==================
*/
void Host_Restart_f(void) {
  char mapname[MAX_QPATH];

  if (CLS_IsDemoPlayback() || !SV_Active()) return;

  if (!IsSrcCommand()) return;
  q_strlcpy(mapname, sv.name,
            sizeof(mapname));  // mapname gets cleared in spawnserver
  SV_SpawnServer(mapname);
  if (!SV_Active()) Host_Error("cannot restart map %s", mapname);
}

/*
==================
Host_Reconnect_f

This command causes the client to wait for the signon messages again.
This is sent just before a server changes levels
==================
*/
void Host_Reconnect_f(void) {
  if (CLS_IsDemoPlayback())  // cross-map demo playback fix from Baker
    return;
  SCR_BeginLoadingPlaque();
  CLS_SetSignon(0);  // need new connection messages
}

/*
=====================
Host_Connect_f

User command to connect to server
=====================
*/
void Host_Connect_f(void) {
  char name[MAX_QPATH];

  CLS_StopDemoCycle();  // stop demo loop in case this fails
  if (CLS_IsDemoPlayback()) {
    CL_StopPlayback();
    CL_Disconnect();
  }
  q_strlcpy(name, Cmd_Argv(1), sizeof(name));
  CL_EstablishConnection(name);
  Host_Reconnect_f();
}

/*
===============================================================================

LOAD / SAVE GAME

===============================================================================
*/

#define SAVEGAME_VERSION 5

/*
===============
Host_SavegameComment

Writes a SAVEGAME_COMMENT_LENGTH character comment describing the current
===============
*/
void Host_SavegameComment(char *text) {
  int i;
  char kills[20];

  for (i = 0; i < SAVEGAME_COMMENT_LENGTH; i++) text[i] = ' ';
  memcpy(text, cl.levelname,
         q_min(strlen(cl.levelname), 22));  // johnfitz -- only copy 22 chars.
  sprintf(kills, "kills:%3i/%3i", CL_Stats(STAT_MONSTERS),
          CL_Stats(STAT_TOTALMONSTERS));
  memcpy(text + 22, kills, strlen(kills));
  // convert space to _ to make stdio happy
  for (i = 0; i < SAVEGAME_COMMENT_LENGTH; i++) {
    if (text[i] == ' ') text[i] = '_';
  }
  text[SAVEGAME_COMMENT_LENGTH] = '\0';
}

/*
===============
Host_Savegame_f
===============
*/
void Host_Savegame_f(void) {
  char name[MAX_OSPATH];
  FILE *f;
  int i;
  char comment[SAVEGAME_COMMENT_LENGTH + 1];

  if (!IsSrcCommand()) return;

  if (!SV_Active()) {
    Con_Printf("Not playing a local game.\n");
    return;
  }

  if (CL_Intermission()) {
    Con_Printf("Can't save in intermission.\n");
    return;
  }

  if (SVS_GetMaxClients() != 1) {
    Con_Printf("Can't save multiplayer games.\n");
    return;
  }

  if (Cmd_Argc() != 2) {
    Con_Printf("save <savename> : save a game\n");
    return;
  }

  if (strstr(Cmd_Argv(1), "..")) {
    Con_Printf("Relative pathnames are not allowed.\n");
    return;
  }

  // (therjak) Why this? SVS_GetMaxClients is 1
  for (i = 0; i < SVS_GetMaxClients(); i++) {
    if (GetClientActive(i) && (EdictV(SV_GetEdict(i))->health <= 0)) {
      Con_Printf("Can't savegame with a dead player\n");
      return;
    }
  }

  q_snprintf(name, sizeof(name), "%s/%s", Com_Gamedir(), Cmd_Argv(1));
  COM_AddExtension(name, ".sav", sizeof(name));

  Con_Printf("Saving game to %s...\n", name);
  f = fopen(name, "w");
  if (!f) {
    Con_Printf("ERROR: couldn't open.\n");
    return;
  }

  fprintf(f, "%i\n", SAVEGAME_VERSION);
  Host_SavegameComment(comment);
  fprintf(f, "%s\n", comment);
  for (i = 0; i < NUM_SPAWN_PARMS; i++)
    fprintf(f, "%f\n", GetClientSpawnParam(0, i));
  fprintf(f, "%d\n", current_skill);
  fprintf(f, "%s\n", sv.name);
  fprintf(f, "%f\n", SV_Time());

  // write the light styles

  for (i = 0; i < MAX_LIGHTSTYLES; i++) {
    if (sv.lightstyles[i])
      fprintf(f, "%s\n", sv.lightstyles[i]);
    else
      fprintf(f, "m\n");
  }

  ED_WriteGlobals(f);
  for (i = 0; i < SV_NumEdicts(); i++) {
    ED_Write(f, EDICT_NUM(i));
    fflush(f);
  }
  fclose(f);
  Con_Printf("done.\n");
}

/*
===============
Host_Loadgame_f
===============
*/
void Host_Loadgame_f(void) {
  char name[MAX_OSPATH];
  FILE *f;
  char mapname[MAX_QPATH];
  float time, tfloat;
  char str[32768];
  const char *start;
  int i, r;
  edict_t *ent;
  int entnum;
  int version;
  float spawn_parms[NUM_SPAWN_PARMS];

  if (!IsSrcCommand()) return;

  if (Cmd_Argc() != 2) {
    Con_Printf("load <savename> : load a game\n");
    return;
  }

  CLS_StopDemoCycle();  // stop demo loop in case this fails

  q_snprintf(name, sizeof(name), "%s/%s", Com_Gamedir(), Cmd_Argv(1));
  COM_AddExtension(name, ".sav", sizeof(name));

  // we can't call SCR_BeginLoadingPlaque, because too much stack space has
  // been used.  The menu calls it before stuffing loadgame command
  //	SCR_BeginLoadingPlaque ();

  Con_Printf("Loading game from %s...\n", name);
  f = fopen(name, "r");
  if (!f) {
    Con_Printf("ERROR: couldn't open.\n");
    return;
  }

  fscanf(f, "%i\n", &version);
  if (version != SAVEGAME_VERSION) {
    fclose(f);
    Con_Printf("Savegame is version %i, not %i\n", version, SAVEGAME_VERSION);
    return;
  }
  fscanf(f, "%s\n", str);
  for (i = 0; i < NUM_SPAWN_PARMS; i++) fscanf(f, "%f\n", &spawn_parms[i]);
  // this silliness is so we can load 1.06 save files, which have float skill
  // values
  fscanf(f, "%f\n", &tfloat);
  current_skill = (int)(tfloat + 0.1);
  Cvar_SetValue("skill", (float)current_skill);

  fscanf(f, "%s\n", mapname);
  fscanf(f, "%f\n", &time);

  CL_Disconnect_f();

  SV_SpawnServer(mapname);

  if (!SV_Active()) {
    fclose(f);
    Con_Printf("Couldn't load map\n");
    return;
  }
  SV_SetPaused(true);  // pause until all clients connect
  SV_SetLoadGame(true);

  // load the light styles

  for (i = 0; i < MAX_LIGHTSTYLES; i++) {
    fscanf(f, "%s\n", str);
    sv.lightstyles[i] = (const char *)Hunk_Strdup(str, "lightstyles");
  }

  // load the edicts out of the savegame file
  entnum = -1;  // -1 is the globals
  while (!feof(f)) {
    qboolean inside_string = false;
    for (i = 0; i < (int)sizeof(str) - 1; i++) {
      r = fgetc(f);
      if (r == EOF || !r) break;
      str[i] = r;
      if (r == '"') {
        inside_string = !inside_string;
      } else if (r == '}' && !inside_string)  // only handle } characters
                                              // outside of quoted strings
      {
        i++;
        break;
      }
    }
    if (i == (int)sizeof(str) - 1) {
      fclose(f);
      Go_Error("Loadgame buffer overflow");
    }
    str[i] = 0;
    start = str;
    start = COM_Parse(str);
    if (!com_token[0]) break;  // end of file
    if (strcmp(com_token, "{")) {
      fclose(f);
      Go_Error("First token isn't a brace");
    }

    if (entnum == -1) {  // parse the global vars
      ED_ParseGlobals(start);
    } else {  // parse an edict
      ent = EDICT_NUM(entnum);
      if (entnum < SV_NumEdicts()) {
        ent->free = false;
        memset(EdictV(ent), 0, progs->entityfields * 4);
      } else {
        memset(ent, 0, pr_edict_size);
      }
      ED_ParseEdict(start, ent);

      // link it into the bsp tree
      if (!ent->free) SV_LinkEdict(ent, false);
    }

    entnum++;
  }

  SV_SetNumEdicts(entnum);
  SV_SetTime(time);

  fclose(f);

  for (i = 0; i < NUM_SPAWN_PARMS; i++)
    SetClientSpawnParam(0, i, spawn_parms[i]);

  if (CLS_GetState() != ca_dedicated) {
    CL_EstablishConnection("local");
    Host_Reconnect_f();
  }
}

//============================================================================

/*
======================
Host_Name_f
======================
*/
void Host_Name_f(void) {
  char newName[32];

  if (Cmd_Argc() == 1) {
    Con_Printf("\"name\" is \"%s\"\n", Cvar_GetString(&cl_name));
    return;
  }
  if (Cmd_Argc() == 2)
    q_strlcpy(newName, Cmd_Argv(1), sizeof(newName));
  else
    q_strlcpy(newName, Cmd_Args(), sizeof(newName));
  newName[15] = 0;  // client_t structure actually says name[32].

  if (IsSrcCommand()) {
    if (Q_strcmp(Cvar_GetString(&cl_name), newName) == 0) return;
    Cvar_Set("_cl_name", newName);
    if (CLS_GetState() == ca_connected) Cmd_ForwardToServer();
    return;
  }

  char *name = GetClientName(HostClient());
  if (name[0] && strcmp(name, "unconnected")) {
    if (Q_strcmp(name, newName) != 0)
      Con_Printf("%s renamed to %s\n", name, newName);
  }
  SetClientName(HostClient(), newName);
  free(name);
  EdictV(SV_GetEdict(HostClient()))->netname = PR_SetEngineString(newName);

  // send notification to all clients

  SV_RD_WriteByte(svc_updatename);
  SV_RD_WriteByte(HostClient());
  SV_RD_WriteString(newName);
}

void Host_Say(qboolean teamonly) {
  int j;
  int save;
  const char *p;
  char text[MAXCMDLINE], *p2;
  qboolean quoted;
  qboolean fromServer = false;

  if (IsSrcCommand()) {
    if (CLS_GetState() != ca_dedicated) {
      Cmd_ForwardToServer();
      return;
    }
    fromServer = true;
    teamonly = false;
  }

  if (Cmd_Argc() < 2) return;

  save = HostClient();

  p = Cmd_Args();
  // remove quotes if present
  quoted = false;
  if (*p == '\"') {
    p++;
    quoted = true;
  }
  // turn on color set 1
  if (!fromServer) {
    char *name = GetClientName(save);
    q_snprintf(text, sizeof(text), "\001%s: %s", name, p);
    free(name);
  } else {
    q_snprintf(text, sizeof(text), "\001<%s> %s", Cvar_GetString(&hostname), p);
  }

  // check length & truncate if necessary
  j = (int)strlen(text);
  if (j >= (int)sizeof(text) - 1) {
    text[sizeof(text) - 2] = '\n';
    text[sizeof(text) - 1] = '\0';
  } else {
    p2 = text + j;
    while ((const char *)p2 > (const char *)text &&
           (p2[-1] == '\r' || p2[-1] == '\n' || (p2[-1] == '\"' && quoted))) {
      if (p2[-1] == '\"' && quoted) quoted = false;
      p2[-1] = '\0';
      p2--;
    }
    p2[0] = '\n';
    p2[1] = '\0';
  }

  for (j = 0; j < SVS_GetMaxClients(); j++) {
    if (!GetClientActive(j) || !GetClientSpawned(j)) continue;
    if (Cvar_GetValue(&teamplay) && teamonly &&
        EdictV(SV_GetEdict(j))->team != EdictV(SV_GetEdict(save))->team)
      continue;
    SV_ClientPrintf2(j, "%s", text);
  }
  host_client = save;

  if (CLS_GetState() == ca_dedicated) Sys_Print(&text[1]);
}

void Host_Say_f(void) { Host_Say(false); }

void Host_Say_Team_f(void) { Host_Say(true); }

void Host_Tell_f(void) {
  int j;
  int save;
  const char *p;
  char text[MAXCMDLINE], *p2;
  qboolean quoted;

  if (IsSrcCommand()) {
    Cmd_ForwardToServer();
    return;
  }

  if (Cmd_Argc() < 3) return;

  p = Cmd_Args();
  // remove quotes if present
  quoted = false;
  if (*p == '\"') {
    p++;
    quoted = true;
  }
  char *name = GetClientName(HostClient());
  q_snprintf(text, sizeof(text), "%s: %s", name, p);
  free(name);

  // check length & truncate if necessary
  j = (int)strlen(text);
  if (j >= (int)sizeof(text) - 1) {
    text[sizeof(text) - 2] = '\n';
    text[sizeof(text) - 1] = '\0';
  } else {
    p2 = text + j;
    while ((const char *)p2 > (const char *)text &&
           (p2[-1] == '\r' || p2[-1] == '\n' || (p2[-1] == '\"' && quoted))) {
      if (p2[-1] == '\"' && quoted) quoted = false;
      p2[-1] = '\0';
      p2--;
    }
    p2[0] = '\n';
    p2[1] = '\0';
  }

  save = HostClient();
  for (j = 0; j < SVS_GetMaxClients(); j++) {
    if (!GetClientActive(j) || !GetClientSpawned(j)) continue;
    char *name = GetClientName(j);
    if (q_strcasecmp(name, Cmd_Argv(1))) {
      free(name);
      continue;
    }
    free(name);
    SV_ClientPrintf2(j, "%s", text);
    break;
  }
  host_client = save;
}

/*
==================
Host_Color_f
==================
*/
void Host_Color_f(void) {
  int top, bottom;
  int playercolor;

  if (Cmd_Argc() == 1) {
    Con_Printf("\"color\" is \"%i %i\"\n", ((int)Cvar_GetValue(&cl_color)) >> 4,
               ((int)Cvar_GetValue(&cl_color)) & 0x0f);
    Con_Printf("color <0-13> [0-13]\n");
    return;
  }

  if (Cmd_Argc() == 2)
    top = bottom = Cmd_ArgvAsInt(1);
  else {
    top = Cmd_ArgvAsInt(1);
    bottom = Cmd_ArgvAsInt(2);
  }

  top &= 15;
  if (top > 13) top = 13;
  bottom &= 15;
  if (bottom > 13) bottom = 13;

  playercolor = top * 16 + bottom;

  if (IsSrcCommand()) {
    Cvar_SetValue("_cl_color", playercolor);
    if (CLS_GetState() == ca_connected) Cmd_ForwardToServer();
    return;
  }

  SetClientColors(HostClient(), playercolor);
  EdictV(SV_GetEdict(HostClient()))->team = bottom + 1;

  // send notification to all clients
  SV_RD_WriteByte(svc_updatecolors);
  SV_RD_WriteByte(HostClient());
  SV_RD_WriteByte(GetClientColors(HostClient()));
}

/*
==================
Host_Kill_f
==================
*/
void Host_Kill_f(void) {
  if (IsSrcCommand()) {
    Cmd_ForwardToServer();
    return;
  }

  if (EdictV(sv_player)->health <= 0) {
    SV_ClientPrintf2(HostClient(), "Can't suicide -- allready dead!\n");
    return;
  }

  pr_global_struct->time = SV_Time();
  pr_global_struct->self = EDICT_TO_PROG(sv_player);
  PR_ExecuteProgram(pr_global_struct->ClientKill);
}

/*
==================
Host_Pause_f
==================
*/
void Host_Pause_f(void) {
  // ericw -- demo pause support (inspired by MarkV)
  if (CLS_IsDemoPlayback()) {
    CLS_SetDemoPaused(!CLS_IsDemoPaused());
    CL_SetPaused(CLS_IsDemoPaused());
    return;
  }

  if (IsSrcCommand()) {
    Cmd_ForwardToServer();
    return;
  }
  if (!Cvar_GetValue(&pausable))
    SV_ClientPrintf2(HostClient(), "Pause not allowed.\n");
  else {
    SV_SetPaused(!SV_Paused());

    if (SV_Paused()) {
      SV_BroadcastPrintf("%s paused the game\n",
                         PR_GetString(EdictV(sv_player)->netname));
    } else {
      SV_BroadcastPrintf("%s unpaused the game\n",
                         PR_GetString(EdictV(sv_player)->netname));
    }

    // send notification to all clients
    SV_RD_WriteByte(svc_setpause);
    SV_RD_WriteByte(SV_Paused());
  }
}

//===========================================================================

/*
==================
Host_Spawn_f
==================
*/
void Host_Spawn_f(void) {
  int i;
  edict_t *ent;

  if (IsSrcCommand()) {
    Con_Printf("spawn is not valid from the console\n");
    return;
  }

  if (GetClientSpawned(HostClient())) {
    Con_Printf("Spawn not valid -- allready spawned\n");
    return;
  }

  // run the entrance script
  if (SV_LoadGame()) {  // loaded games are fully inited allready
    // if this is the last client to be connected, unpause
    SV_SetPaused(false);
  } else {
    // set up the edict
    ent = SV_GetEdict(HostClient());

    memset(EdictV(ent), 0, progs->entityfields * 4);
    EdictV(ent)->colormap = NUM_FOR_EDICT(ent);
    EdictV(ent)->team = (GetClientColors(HostClient()) & 15) + 1;
    // TODO(therjak): This is a memory leak!!!
    Sys_Print("Memory Leaking");
    char *name = GetClientName(HostClient());
    EdictV(ent)->netname = PR_SetEngineString(name);

    // copy spawn parms out of the client_t
    for (i = 0; i < NUM_SPAWN_PARMS; i++)
      (&pr_global_struct->parm1)[i] = GetClientSpawnParam(HostClient(), i);
    // call the spawn function
    pr_global_struct->time = SV_Time();
    pr_global_struct->self = EDICT_TO_PROG(sv_player);
    PR_ExecuteProgram(pr_global_struct->ClientConnect);

    if ((Sys_DoubleTime() - ClientConnectTime(HostClient())) <= SV_Time()) {
      char *name = GetClientName(HostClient());
      Sys_Print_S("%v entered the game\n", name);
      free(name);
    }

    PR_ExecuteProgram(pr_global_struct->PutClientInServer);
  }

  // send all current names, colors, and frag counts
  ClientClearMessage(HostClient());

  // send time of update
  ClientWriteByte(HostClient(), svc_time);
  ClientWriteFloat(HostClient(), SV_Time());

  for (i = 0; i < SVS_GetMaxClients(); i++) {
    ClientWriteByte(HostClient(), svc_updatename);
    ClientWriteByte(HostClient(), i);
    char *name = GetClientName(i);
    ClientWriteString(HostClient(), name);
    free(name);
    ClientWriteByte(HostClient(), svc_updatefrags);
    ClientWriteByte(HostClient(), i);
    ClientWriteShort(HostClient(), GetClientOldFrags(i));
    ClientWriteByte(HostClient(), svc_updatecolors);
    ClientWriteByte(HostClient(), i);
    ClientWriteByte(HostClient(), GetClientColors(i));
  }

  // send all current light styles
  for (i = 0; i < MAX_LIGHTSTYLES; i++) {
    ClientWriteByte(HostClient(), svc_lightstyle);
    ClientWriteByte(HostClient(), (char)i);
    ClientWriteString(HostClient(), sv.lightstyles[i]);
  }

  //
  // send some stats
  //
  ClientWriteByte(HostClient(), svc_updatestat);
  ClientWriteByte(HostClient(), STAT_TOTALSECRETS);
  ClientWriteLong(HostClient(), pr_global_struct->total_secrets);

  ClientWriteByte(HostClient(), svc_updatestat);
  ClientWriteByte(HostClient(), STAT_TOTALMONSTERS);
  ClientWriteLong(HostClient(), pr_global_struct->total_monsters);

  ClientWriteByte(HostClient(), svc_updatestat);
  ClientWriteByte(HostClient(), STAT_SECRETS);
  ClientWriteLong(HostClient(), pr_global_struct->found_secrets);

  ClientWriteByte(HostClient(), svc_updatestat);
  ClientWriteByte(HostClient(), STAT_MONSTERS);
  ClientWriteLong(HostClient(), pr_global_struct->killed_monsters);

  //
  // send a fixangle
  // Never send a roll angle, because savegames can catch the server
  // in a state where it is expecting the client to correct the angle
  // and it won't happen if the game was just loaded, so you wind up
  // with a permanent head tilt
  ent = EDICT_NUM(1 + (HostClient()));
  ClientWriteByte(HostClient(), svc_setangle);
  for (i = 0; i < 2; i++)
    ClientWriteAngle(HostClient(), EdictV(ent)->angles[i]);
  ClientWriteAngle(HostClient(), 0);
  {
    SV_MS_Clear();
    SV_MS_SetMaxLen(MAX_DATAGRAM);
    SV_WriteClientdataToMessage(sv_player);
    ClientWriteSVMSG(HostClient());
  }
  ClientWriteByte(HostClient(), svc_signonnum);
  ClientWriteByte(HostClient(), 3);
  SetClientSendSignon(HostClient(), true);
}

/*
==================
Host_Begin_f
==================
*/
void Host_Begin_f(void) {
  if (IsSrcCommand()) {
    Con_Printf("begin is not valid from the console\n");
    return;
  }

  SetClientSpawned(HostClient(), true);
}

//===========================================================================

/*
==================
Host_Kick_f

Kicks a user off of the server
==================
*/
void Host_Kick_f(void) {
  const char *who;
  const char *message = NULL;
  int save;
  int i;
  qboolean byNumber = false;

  if (IsSrcCommand()) {
    if (!SV_Active()) {
      Cmd_ForwardToServer();
      return;
    }
  } else if (pr_global_struct->deathmatch)
    return;

  save = HostClient();

  if (Cmd_Argc() > 2 && Q_strcmp(Cmd_Argv(1), "#") == 0) {
    i = Cmd_ArgvAsInt(2) - 1;
    if (i < 0 || i >= SVS_GetMaxClients()) return;
    if (!GetClientActive(i)) return;
    host_client = i;
    byNumber = true;
  } else {
    for (i = 0, host_client = 0; i < SVS_GetMaxClients();
         i++, host_client++) {
      if (!GetClientActive(HostClient())) continue;

      char *name = GetClientName(HostClient());
      if (q_strcasecmp(name, Cmd_Argv(1)) == 0) {
        free(name);
        break;
      }
      free(name);
    }
  }
  char *name = NULL;
  if (i < SVS_GetMaxClients()) {
    if (IsSrcCommand())
      if (CLS_GetState() == ca_dedicated)
        who = "Console";
      else
        who = Cvar_GetString(&cl_name);
    else {
      name = GetClientName(save);
      who = name;
    }

    // can't kick yourself!
    if (host_client == save) return;

    if (Cmd_Argc() > 2) {
      message = COM_Parse(Cmd_Args());
      if (byNumber) {
        message++;               // skip the #
        while (*message == ' ')  // skip white space
          message++;
        message += strlen(Cmd_Argv(2));  // skip the number
      }
      while (*message && *message == ' ') message++;
    }
    if (message)
      SV_ClientPrintf2(HostClient(), "Kicked by %s: %s\n", who, message);
    else
      SV_ClientPrintf2(HostClient(), "Kicked by %s\n", who);
    SV_DropClient(HostClient(), false);
  }
  free(name);

  host_client = save;
}

/*
===============================================================================

DEBUGGING TOOLS

===============================================================================
*/

/*
==================
Host_Give_f
==================
*/
void Host_Give_f(void) {
  const char *t;
  int v;
  eval_t *val;

  if (IsSrcCommand()) {
    Cmd_ForwardToServer();
    return;
  }

  if (pr_global_struct->deathmatch) return;

  t = Cmd_Argv(1);
  v = Cmd_ArgvAsInt(2);

  switch (t[0]) {
    case '0':
    case '1':
    case '2':
    case '3':
    case '4':
    case '5':
    case '6':
    case '7':
    case '8':
    case '9':
      // MED 01/04/97 added hipnotic give stuff
      if (CMLHipnotic()) {
        if (t[0] == '6') {
          if (t[1] == 'a')
            EdictV(sv_player)->items = 
              (int)EdictV(sv_player)->items | HIT_PROXIMITY_GUN;
          else
            EdictV(sv_player)->items = 
              (int)EdictV(sv_player)->items | IT_GRENADE_LAUNCHER;
        } else if (t[0] == '9')
          EdictV(sv_player)->items = 
            (int)EdictV(sv_player)->items | HIT_LASER_CANNON;
        else if (t[0] == '0')
          EdictV(sv_player)->items = 
            (int)EdictV(sv_player)->items | HIT_MJOLNIR;
        else if (t[0] >= '2')
          EdictV(sv_player)->items =
              (int)EdictV(sv_player)->items | (IT_SHOTGUN << (t[0] - '2'));
      } else {
        if (t[0] >= '2')
          EdictV(sv_player)->items =
              (int)EdictV(sv_player)->items | (IT_SHOTGUN << (t[0] - '2'));
      }
      break;

    case 's':
      if (CMLRogue()) {
        val = GetEdictFieldValue(sv_player, "ammo_shells1");
        if (val) val->_float = v;
      }
      EdictV(sv_player)->ammo_shells = v;
      break;

    case 'n':
      if (CMLRogue()) {
        val = GetEdictFieldValue(sv_player, "ammo_nails1");
        if (val) {
          val->_float = v;
          if (EdictV(sv_player)->weapon <= IT_LIGHTNING) 
            EdictV(sv_player)->ammo_nails = v;
        }
      } else {
        EdictV(sv_player)->ammo_nails = v;
      }
      break;

    case 'l':
      if (CMLRogue()) {
        val = GetEdictFieldValue(sv_player, "ammo_lava_nails");
        if (val) {
          val->_float = v;
          if (EdictV(sv_player)->weapon > IT_LIGHTNING) 
            EdictV(sv_player)->ammo_nails = v;
        }
      }
      break;

    case 'r':
      if (CMLRogue()) {
        val = GetEdictFieldValue(sv_player, "ammo_rockets1");
        if (val) {
          val->_float = v;
          if (EdictV(sv_player)->weapon <= IT_LIGHTNING)
            EdictV(sv_player)->ammo_rockets = v;
        }
      } else {
        EdictV(sv_player)->ammo_rockets = v;
      }
      break;

    case 'm':
      if (CMLRogue()) {
        val = GetEdictFieldValue(sv_player, "ammo_multi_rockets");
        if (val) {
          val->_float = v;
          if (EdictV(sv_player)->weapon > IT_LIGHTNING) 
            EdictV(sv_player)->ammo_rockets = v;
        }
      }
      break;

    case 'h':
      EdictV(sv_player)->health = v;
      break;

    case 'c':
      if (CMLRogue()) {
        val = GetEdictFieldValue(sv_player, "ammo_cells1");
        if (val) {
          val->_float = v;
          if (EdictV(sv_player)->weapon <= IT_LIGHTNING) 
            EdictV(sv_player)->ammo_cells = v;
        }
      } else {
        EdictV(sv_player)->ammo_cells = v;
      }
      break;

    case 'p':
      if (CMLRogue()) {
        val = GetEdictFieldValue(sv_player, "ammo_plasma");
        if (val) {
          val->_float = v;
          if (EdictV(sv_player)->weapon > IT_LIGHTNING) 
            EdictV(sv_player)->ammo_cells = v;
        }
      }
      break;

    // johnfitz -- give armour
    case 'a':
      if (v > 150) {
        EdictV(sv_player)->armortype = 0.8;
        EdictV(sv_player)->armorvalue = v;
        EdictV(sv_player)->items = EdictV(sv_player)->items -
                             ((int)(EdictV(sv_player)->items) &
                              (int)(IT_ARMOR1 | IT_ARMOR2 | IT_ARMOR3)) +
                             IT_ARMOR3;
      } else if (v > 100) {
        EdictV(sv_player)->armortype = 0.6;
        EdictV(sv_player)->armorvalue = v;
        EdictV(sv_player)->items = EdictV(sv_player)->items -
                             ((int)(EdictV(sv_player)->items) &
                              (int)(IT_ARMOR1 | IT_ARMOR2 | IT_ARMOR3)) +
                             IT_ARMOR2;
      } else if (v >= 0) {
        EdictV(sv_player)->armortype = 0.3;
        EdictV(sv_player)->armorvalue = v;
        EdictV(sv_player)->items = EdictV(sv_player)->items -
                             ((int)(EdictV(sv_player)->items) &
                              (int)(IT_ARMOR1 | IT_ARMOR2 | IT_ARMOR3)) +
                             IT_ARMOR1;
      }
      break;
      // johnfitz
  }

  // johnfitz -- update currentammo to match new ammo (so statusbar updates
  // correctly)
  switch ((int)(EdictV(sv_player)->weapon)) {
    case IT_SHOTGUN:
    case IT_SUPER_SHOTGUN:
      EdictV(sv_player)->currentammo = EdictV(sv_player)->ammo_shells;
      break;
    case IT_NAILGUN:
    case IT_SUPER_NAILGUN:
    case RIT_LAVA_SUPER_NAILGUN:
      EdictV(sv_player)->currentammo = EdictV(sv_player)->ammo_nails;
      break;
    case IT_GRENADE_LAUNCHER:
    case IT_ROCKET_LAUNCHER:
    case RIT_MULTI_GRENADE:
    case RIT_MULTI_ROCKET:
      EdictV(sv_player)->currentammo = EdictV(sv_player)->ammo_rockets;
      break;
    case IT_LIGHTNING:
    case HIT_LASER_CANNON:
    case HIT_MJOLNIR:
      EdictV(sv_player)->currentammo = EdictV(sv_player)->ammo_cells;
      break;
    case RIT_LAVA_NAILGUN:  // same as IT_AXE
      if (CMLRogue()) EdictV(sv_player)->currentammo = EdictV(sv_player)->ammo_nails;
      break;
    case RIT_PLASMA_GUN:  // same as HIT_PROXIMITY_GUN
      if (CMLRogue()) EdictV(sv_player)->currentammo = EdictV(sv_player)->ammo_cells;
      if (CMLHipnotic()) EdictV(sv_player)->currentammo = EdictV(sv_player)->ammo_rockets;
      break;
  }
  // johnfitz
}

edict_t *FindViewthing(void) {
  int i;
  edict_t *e;

  for (i = 0; i < SV_NumEdicts(); i++) {
    e = EDICT_NUM(i);
    if (!strcmp(PR_GetString(EdictV(e)->classname), "viewthing")) return e;
  }
  Con_Printf("No viewthing on map\n");
  return NULL;
}

/*
==================
Host_Viewmodel_f
==================
*/
void Host_Viewmodel_f(void) {
  edict_t *e;
  qmodel_t *m;

  e = FindViewthing();
  if (!e) return;

  m = Mod_ForName(Cmd_Argv(1), false);
  if (!m) {
    Con_Printf("Can't load %s\n", Cmd_Argv(1));
    return;
  }

  EdictV(e)->frame = 0;
  cl.model_precache[(int)EdictV(e)->modelindex] = m;
}

/*
==================
Host_Viewframe_f
==================
*/
void Host_Viewframe_f(void) {
  edict_t *e;
  int f;
  qmodel_t *m;

  e = FindViewthing();
  if (!e) return;
  m = cl.model_precache[(int)EdictV(e)->modelindex];

  f = Cmd_ArgvAsInt(1);
  if (f >= m->numframes) f = m->numframes - 1;

  EdictV(e)->frame = f;
}

void PrintFrameName(qmodel_t *m, int frame) {
  aliashdr_t *hdr;
  maliasframedesc_t *pframedesc;

  hdr = (aliashdr_t *)Mod_Extradata(m);
  if (!hdr) return;
  pframedesc = &hdr->frames[frame];

  Con_Printf("frame %i: %s\n", frame, pframedesc->name);
}

/*
==================
Host_Viewnext_f
==================
*/
void Host_Viewnext_f(void) {
  edict_t *e;
  qmodel_t *m;

  e = FindViewthing();
  if (!e) return;
  m = cl.model_precache[(int)EdictV(e)->modelindex];

  EdictV(e)->frame = EdictV(e)->frame + 1;
  if (EdictV(e)->frame >= m->numframes) 
    EdictV(e)->frame = m->numframes - 1;

  PrintFrameName(m, EdictV(e)->frame);
}

/*
==================
Host_Viewprev_f
==================
*/
void Host_Viewprev_f(void) {
  edict_t *e;
  qmodel_t *m;

  e = FindViewthing();
  if (!e) return;

  m = cl.model_precache[(int)EdictV(e)->modelindex];

  EdictV(e)->frame = EdictV(e)->frame - 1;
  if (EdictV(e)->frame < 0) EdictV(e)->frame = 0;

  PrintFrameName(m, EdictV(e)->frame);
}

/*
===============================================================================

DEMO LOOP CONTROL

===============================================================================
*/

/*
==================
Host_Startdemos_f
==================
*/
void Host_Startdemos_f(void) {
  int i, c;

  if (CLS_GetState() == ca_dedicated) return;

  c = Cmd_Argc() - 1;
  if (c > MAX_DEMOS) {
    Con_Printf("Max %i demos in demoloop\n", MAX_DEMOS);
    c = MAX_DEMOS;
  }
  Con_Printf("%i demo(s) in loop\n", c);

  for (i = 1; i < c + 1; i++)
    q_strlcpy(cls.demos[i - 1], Cmd_Argv(i), sizeof(cls.demos[0]));

  if (!SV_Active() && !CLS_IsDemoCycleStopped() && !CLS_IsDemoPlayback()) {
    CLS_StartDemoCycle();
    if (!CMLFitz()) { /* QuakeSpasm customization: */
      /* go straight to menu, no CL_NextDemo */
      CLS_StopDemoCycle();
      Cbuf_InsertText("menu_main\n");
      return;
    }
    CL_NextDemo();
  } else {
    CLS_StopDemoCycle();
  }
}

/*
==================
Host_Demos_f

Return to looping demos
==================
*/
void Host_Demos_f(void) {
  if (CLS_GetState() == ca_dedicated) return;
  if (CLS_IsDemoCycleStopped()) CLS_StartDemoCycle();
  CL_Disconnect_f();
  CL_NextDemo();
}

/*
==================
Host_Stopdemo_f

Return to looping demos
==================
*/
void Host_Stopdemo_f(void) {
  if (CLS_GetState() == ca_dedicated) return;
  if (!CLS_IsDemoPlayback()) return;
  CL_StopPlayback();
  CL_Disconnect();
}

//=============================================================================

/*
==================
Host_InitCommands
==================
*/
void Host_InitCommands(void) {
  Cmd_AddCommand("maps", Host_Maps_f);  // johnfitz
  Cmd_AddCommand("mods", Host_Mods_f);  // johnfitz
  Cmd_AddCommand("games",
                 Host_Mods_f);  // as an alias to "mods" -- S.A. / QuakeSpasm
  Cmd_AddCommand("mapname", Host_Mapname_f);  // johnfitz

  Cmd_AddCommand("status", Host_Status_f);
  Cmd_AddCommand("quit", Host_Quit_f);
  Cmd_AddCommand("god", Host_God_f);
  Cmd_AddCommand("notarget", Host_Notarget_f);
  Cmd_AddCommand("fly", Host_Fly_f);
  Cmd_AddCommand("map", Host_Map_f);
  Cmd_AddCommand("restart", Host_Restart_f);
  Cmd_AddCommand("changelevel", Host_Changelevel_f);
  Cmd_AddCommand("connect", Host_Connect_f);
  Cmd_AddCommand("reconnect", Host_Reconnect_f);
  Cmd_AddCommand("name", Host_Name_f);
  Cmd_AddCommand("noclip", Host_Noclip_f);
  Cmd_AddCommand("setpos", Host_SetPos_f);  // QuakeSpasm

  Cmd_AddCommand("say", Host_Say_f);
  Cmd_AddCommand("say_team", Host_Say_Team_f);
  Cmd_AddCommand("tell", Host_Tell_f);
  Cmd_AddCommand("color", Host_Color_f);
  Cmd_AddCommand("kill", Host_Kill_f);
  Cmd_AddCommand("pause", Host_Pause_f);
  Cmd_AddCommand("spawn", Host_Spawn_f);
  Cmd_AddCommand("begin", Host_Begin_f);
  Cmd_AddCommand("prespawn", Host_PreSpawn_f);
  Cmd_AddCommand("kick", Host_Kick_f);
  Cmd_AddCommand("ping", Host_Ping_f);
  Cmd_AddCommand("load", Host_Loadgame_f);
  Cmd_AddCommand("save", Host_Savegame_f);
  Cmd_AddCommand("give", Host_Give_f);

  Cmd_AddCommand("startdemos", Host_Startdemos_f);
  Cmd_AddCommand("demos", Host_Demos_f);
  Cmd_AddCommand("stopdemo", Host_Stopdemo_f);

  Cmd_AddCommand("viewmodel", Host_Viewmodel_f);
  Cmd_AddCommand("viewframe", Host_Viewframe_f);
  Cmd_AddCommand("viewnext", Host_Viewnext_f);
  Cmd_AddCommand("viewprev", Host_Viewprev_f);

  Cmd_AddCommand("mcache", Mod_Print);
}
