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

  Cmd_AddCommand("restart", Host_Restart_f);
  Cmd_AddCommand("changelevel", Host_Changelevel_f);

  Cmd_AddCommand("startdemos", Host_Startdemos_f);

  Cmd_AddCommand("mcache", Mod_Print);
}
