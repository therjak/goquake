#include "quakedef.h"
#ifndef _WIN32
#include <dirent.h>
#endif

int current_skill;

void Mod_Print(void);

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

qboolean noclip_anglehack;

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
        EDICT_SETFREE(entnum, false);
        TT_ClearEntVars(EVars(entnum));
      } else {
        TT_ClearEdict(entnum);
      }
      ED_ParseEdict(start, entnum);

      // link it into the bsp tree
      if (!EDICT_FREE(entnum)) SV_LinkEdict(entnum, false);
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

/*
===============================================================================

DEBUGGING TOOLS

===============================================================================
*/

void PrintFrameName(qmodel_t *m, int frame) {
  aliashdr_t *hdr;
  maliasframedesc_t *pframedesc;

  hdr = (aliashdr_t *)Mod_Extradata(m);
  if (!hdr) return;
  pframedesc = &hdr->frames[frame];

  Con_Printf("frame %i: %s\n", frame, pframedesc->name);
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

  Cmd_AddCommand("map", Host_Map_f);
  Cmd_AddCommand("restart", Host_Restart_f);
  Cmd_AddCommand("changelevel", Host_Changelevel_f);

  Cmd_AddCommand("load", Host_Loadgame_f);
  Cmd_AddCommand("save", Host_Savegame_f);

  Cmd_AddCommand("startdemos", Host_Startdemos_f);

  Cmd_AddCommand("mcache", Mod_Print);
}
