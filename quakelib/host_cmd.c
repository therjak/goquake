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
    Con_Printf("\"mapname\" is \"%s\"\n", SV_Name());
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

// THERJAK
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
  print_fn("map:     %s\n", SV_Name());
  int active = 0;
  for (j = 0; j < SVS_GetMaxClients(); j++) {
    if (GetClientActive(j)) {
      active++;
    }
  }
  print_fn("players: %i active (%i max)\n\n", active, SVS_GetMaxClients());
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
             (int)EVars(GetClientEdictId(j))->frags, hours, minutes, seconds);
    free(name);
    print_fn("   %s\n", NET_QSocketGetAddressString(j));
  }
}

qboolean noclip_anglehack;

/*
==================
Host_Noclip_f
==================
*/
// THERJAK just ignore noclip_anglehack
void Host_Noclip_f(void) {
  if (IsSrcCommand()) {
    Cmd_ForwardToServer();
    return;
  }

  if (Pr_global_struct_deathmatch()) return;
  entvars_t *pent = EVars(SV_Player());
  float *movetype = &pent->movetype;

  // johnfitz -- allow user to explicitly set noclip to on or off
  switch (Cmd_Argc()) {
    case 1:
      if (*movetype != MOVETYPE_NOCLIP) {
        noclip_anglehack = true;
        *movetype = MOVETYPE_NOCLIP;
        SV_ClientPrintf2(HostClient(), "noclip ON\n");
      } else {
        noclip_anglehack = false;
        *movetype = MOVETYPE_WALK;
        SV_ClientPrintf2(HostClient(), "noclip OFF\n");
      }
      break;
    case 2:
      if (Cmd_ArgvAsInt(1)) {
        noclip_anglehack = true;
        *movetype = MOVETYPE_NOCLIP;
        SV_ClientPrintf2(HostClient(), "noclip ON\n");
      } else {
        noclip_anglehack = false;
        *movetype = MOVETYPE_WALK;
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
// THERJAK just igroner noclip_anglehack
void Host_SetPos_f(void) {
  if (IsSrcCommand()) {
    Cmd_ForwardToServer();
    return;
  }

  if (Pr_global_struct_deathmatch()) return;
  entvars_t *pent = EVars(SV_Player());

  if (Cmd_Argc() != 7 && Cmd_Argc() != 4) {
    SV_ClientPrintf2(HostClient(), "usage:\n");
    SV_ClientPrintf2(HostClient(), "   setpos <x> <y> <z>\n");
    SV_ClientPrintf2(HostClient(),
                     "   setpos <x> <y> <z> <pitch> <yaw> <roll>\n");
    SV_ClientPrintf2(HostClient(), "current values:\n");
    SV_ClientPrintf2(HostClient(), "   %i %i %i %i %i %i\n",
                     (int)pent->origin[0], (int)pent->origin[1],
                     (int)pent->origin[2], (int)pent->v_angle[0],
                     (int)pent->v_angle[1], (int)pent->v_angle[2]);
    return;
  }

  if (pent->movetype != MOVETYPE_NOCLIP) {
    noclip_anglehack = true;
    pent->movetype = MOVETYPE_NOCLIP;
    SV_ClientPrintf2(HostClient(), "noclip ON\n");
  }

  // make sure they're not going to whizz away from it
  pent->velocity[0] = 0;
  pent->velocity[1] = 0;
  pent->velocity[2] = 0;

  pent->origin[0] = Cmd_ArgvAsDouble(1);
  pent->origin[1] = Cmd_ArgvAsDouble(2);
  pent->origin[2] = Cmd_ArgvAsDouble(3);

  if (Cmd_Argc() == 7) {
    pent->angles[0] = Cmd_ArgvAsDouble(4);
    pent->angles[1] = Cmd_ArgvAsDouble(5);
    pent->angles[2] = Cmd_ArgvAsDouble(6);
    pent->fixangle = 1;
  }

  SV_LinkEdict(SV_Player(), false);
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
        Con_Printf("Current map: %s\n", SV_Name());
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
  q_strlcpy(mapname, SV_Name(),
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
    if (GetClientActive(i) && (EVars(GetClientEdictId(i))->health <= 0)) {
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
  fprintf(f, "%s\n", SV_Name());
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
    ED_Write(f, i);
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
    SetSVLightStyles(i, sv.lightstyles[i]);
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
      if (entnum < SV_NumEdicts()) {
        EDICT_NUM(entnum)->free = false;
        TT_ClearEntVars(EVars(entnum));
      } else {
        TT_ClearEdict(entnum);
      }
      ED_ParseEdict(start, entnum);

      // link it into the bsp tree
      if (!EDICT_NUM(entnum)->free) SV_LinkEdict(entnum, false);
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
// THERJAK
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
  EVars(GetClientEdictId(HostClient()))->netname = PR_SetEngineString(newName);

  // send notification to all clients

  SV_RD_WriteByte(svc_updatename);
  SV_RD_WriteByte(HostClient());
  SV_RD_WriteString(newName);
}

/*
==================
Host_Kill_f
==================
*/
// THERJAK
void Host_Kill_f(void) {
  if (IsSrcCommand()) {
    Cmd_ForwardToServer();
    return;
  }

  if (EVars(SV_Player())->health <= 0) {
    SV_ClientPrintf2(HostClient(), "Can't suicide -- allready dead!\n");
    return;
  }

  Set_pr_global_struct_time(SV_Time());
  Set_pr_global_struct_self(SV_Player());
  PR_ExecuteProgram(Pr_global_struct_ClientKill());
}

/*
==================
Host_Kick_f

Kicks a user off of the server
==================
*/
// THERJAK
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
  } else if (Pr_global_struct_deathmatch())
    return;

  save = HostClient();

  if (Cmd_Argc() > 2 && Q_strcmp(Cmd_Argv(1), "#") == 0) {
    i = Cmd_ArgvAsInt(2) - 1;
    if (i < 0 || i >= SVS_GetMaxClients()) return;
    if (!GetClientActive(i)) return;
    SetHost_Client(i);
    byNumber = true;
  } else {
    for (i = 0; i < SVS_GetMaxClients(); i++) {
      SetHost_Client(i);
      if (!GetClientActive(HostClient())) continue;

      char *name = GetClientName(HostClient());
      if (q_strcasecmp(name, Cmd_Argv(1)) == 0) {
        free(name);
        break;
      }
      free(name);
    }
    SetHost_Client(i);
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
    if (HostClient() == save) return;

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

  SetHost_Client(save);
}

/*
===============================================================================

DEBUGGING TOOLS

===============================================================================
*/
// THERJAK
entvars_t *FindViewthingEV(void) {
  int i;
  entvars_t *ev;

  for (i = 0; i < SV_NumEdicts(); i++) {
    ev = EVars(i);
    if (!strcmp(PR_GetString(ev->classname), "viewthing")) {
      return ev;
    }
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
  entvars_t *e;
  qmodel_t *m;

  e = FindViewthingEV();
  if (!e) return;

  m = Mod_ForName(Cmd_Argv(1), false);
  if (!m) {
    Con_Printf("Can't load %s\n", Cmd_Argv(1));
    return;
  }

  e->frame = 0;
  cl.model_precache[(int)e->modelindex] = m;
}

/*
==================
Host_Viewframe_f
==================
*/
void Host_Viewframe_f(void) {
  entvars_t *e;
  int f;
  qmodel_t *m;

  e = FindViewthingEV();
  if (!e) return;
  m = cl.model_precache[(int)e->modelindex];

  f = Cmd_ArgvAsInt(1);
  if (f >= m->numframes) f = m->numframes - 1;

  e->frame = f;
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
  entvars_t *e;
  qmodel_t *m;

  e = FindViewthingEV();
  if (!e) return;
  m = cl.model_precache[(int)e->modelindex];

  e->frame = e->frame + 1;
  if (e->frame >= m->numframes) {
    e->frame = m->numframes - 1;
  }

  PrintFrameName(m, e->frame);
}

/*
==================
Host_Viewprev_f
==================
*/
void Host_Viewprev_f(void) {
  entvars_t *e;
  qmodel_t *m;

  e = FindViewthingEV();
  if (!e) return;

  m = cl.model_precache[(int)e->modelindex];

  e->frame = e->frame - 1;
  if (e->frame < 0) e->frame = 0;

  PrintFrameName(m, e->frame);
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
  Cmd_AddCommand("map", Host_Map_f);
  Cmd_AddCommand("restart", Host_Restart_f);
  Cmd_AddCommand("changelevel", Host_Changelevel_f);
  Cmd_AddCommand("connect", Host_Connect_f);
  Cmd_AddCommand("reconnect", Host_Reconnect_f);
  Cmd_AddCommand("name", Host_Name_f);
  Cmd_AddCommand("noclip", Host_Noclip_f);
  Cmd_AddCommand("setpos", Host_SetPos_f);  // QuakeSpasm

  Cmd_AddCommand("kill", Host_Kill_f);
  Cmd_AddCommand("prespawn", Host_PreSpawn_f);
  Cmd_AddCommand("kick", Host_Kick_f);
  Cmd_AddCommand("load", Host_Loadgame_f);
  Cmd_AddCommand("save", Host_Savegame_f);

  Cmd_AddCommand("startdemos", Host_Startdemos_f);
  Cmd_AddCommand("demos", Host_Demos_f);
  Cmd_AddCommand("stopdemo", Host_Stopdemo_f);

  Cmd_AddCommand("viewmodel", Host_Viewmodel_f);
  Cmd_AddCommand("viewframe", Host_Viewframe_f);
  Cmd_AddCommand("viewnext", Host_Viewnext_f);
  Cmd_AddCommand("viewprev", Host_Viewprev_f);

  Cmd_AddCommand("mcache", Mod_Print);
}
