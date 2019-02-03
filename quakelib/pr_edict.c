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
// sv_edict.c -- entity dictionary

#include "_cgo_export.h"
//
#include "quakedef.h"

dprograms_t *progs;
dfunction_t *pr_functions;

static char *pr_strings;
static int pr_stringssize;
static const char **pr_knownstrings;
static int pr_maxknownstrings;
static int pr_numknownstrings;
static ddef_t *pr_fielddefs;
static ddef_t *pr_globaldefs;

qboolean pr_alpha_supported;  // johnfitz

dstatement_t *pr_statements;
globalvars_t *pr_global_struct;
float *pr_globals;  // same as pr_global_struct

unsigned short pr_crc;

int type_size[8] = {
    1,  // ev_void
    1,  // sizeof(GoInt32) / 4		// ev_string
    1,  // ev_float
    3,  // ev_vector
    1,  // ev_entity
    1,  // ev_field
    1,  // sizeof(GoInt32) / 4		// ev_function
    1   // sizeof(void *) / 4		// ev_pointer
};

static ddef_t *ED_FieldAtOfs(int ofs);
static qboolean ED_ParseEpair(void *base, ddef_t *key, const char *s);

#define MAX_FIELD_LEN 64
#define GEFV_CACHESIZE 2

typedef struct {
  ddef_t *pcache;
  char field[MAX_FIELD_LEN];
} gefv_cache;

static gefv_cache gefvCache[GEFV_CACHESIZE] = {{NULL, ""}, {NULL, ""}};

cvar_t nomonsters;
cvar_t gamecfg;
cvar_t scratch1;
cvar_t scratch2;
cvar_t scratch3;
cvar_t scratch4;
cvar_t savedgamecfg;
cvar_t saved1;
cvar_t saved2;
cvar_t saved3;
cvar_t saved4;

/*
=================
ED_ClearEdict

Sets everything to NULL
=================
*/
void ED_ClearEdict(edict_t *e) {
  TT_ClearEntVars(EdictV(e));
  e->free = false;
}

/*
=================
ED_Alloc

Either finds a free edict, or allocates a new one.
Try to avoid reusing an entity that was recently freed, because it
can cause the client to think the entity morphed into something else
instead of being removed and recreated, which can cause interpolated
angles and bad trails.
=================
*/
edict_t *ED_Alloc(void) {
  int i;
  edict_t *e;

  for (i = SVS_GetMaxClients() + 1; i < SV_NumEdicts(); i++) {
    e = EDICT_NUM(i);
    // the first couple seconds of server time can involve a lot of
    // freeing and allocating, so relax the replacement policy
    if (e->free && (e->freetime < 2 || SV_Time() - e->freetime > 0.5)) {
      ED_ClearEdict(e);
      return e;
    }
  }

  if (i == SV_MaxEdicts())  // johnfitz -- use sv.max_edicts instead of
                            // MAX_EDICTS
    Host_Error("ED_Alloc: no free edicts (max_edicts is %i)", SV_MaxEdicts());

  SV_SetNumEdicts(SV_NumEdicts() + 1);
  e = EDICT_NUM(i);
  TT_ClearEdict(e);
  // ericw -- switched sv.edicts to malloc(), so
  // we are accessing uninitialized memory and
  // must fully zero it, not just ED_ClearEdict

  return e;
}

/*
=================
ED_Free

Marks the edict as free
FIXME: walk all entities and NULL out references to this entity
=================
*/
void ED_Free(edict_t *ed) {
  SV_UnlinkEdict(ed);  // unlink from world bsp

  ed->free = true;
  EdictV(ed)->model = 0;
  EdictV(ed)->takedamage = 0;
  EdictV(ed)->modelindex = 0;
  EdictV(ed)->colormap = 0;
  EdictV(ed)->skin = 0;
  EdictV(ed)->frame = 0;
  VectorCopy(vec3_origin, EdictV(ed)->origin);
  VectorCopy(vec3_origin, EdictV(ed)->angles);
  EdictV(ed)->nextthink = -1;
  EdictV(ed)->solid = 0;
  ed->alpha = ENTALPHA_DEFAULT;  // johnfitz -- reset alpha for next entity

  ed->freetime = SV_Time();
}

//===========================================================================

/*
============
ED_GlobalAtOfs
============
*/
static ddef_t *ED_GlobalAtOfs(int ofs) {
  ddef_t *def;
  int i;

  for (i = 0; i < progs->numglobaldefs; i++) {
    def = &pr_globaldefs[i];
    if (def->ofs == ofs) return def;
  }
  return NULL;
}

/*
============
ED_FieldAtOfs
============
*/
static ddef_t *ED_FieldAtOfs(int ofs) {
  ddef_t *def;
  int i;

  for (i = 0; i < progs->numfielddefs; i++) {
    def = &pr_fielddefs[i];
    if (def->ofs == ofs) return def;
  }
  return NULL;
}

/*
============
ED_FindField
============
*/
static ddef_t *ED_FindField(const char *name) {
  ddef_t *def;
  int i;

  for (i = 0; i < progs->numfielddefs; i++) {
    def = &pr_fielddefs[i];
    if (!strcmp(PR_GetString(def->s_name), name)) return def;
  }
  return NULL;
}

/*
============
ED_FindGlobal
============
*/
static ddef_t *ED_FindGlobal(const char *name) {
  ddef_t *def;
  int i;

  for (i = 0; i < progs->numglobaldefs; i++) {
    def = &pr_globaldefs[i];
    if (!strcmp(PR_GetString(def->s_name), name)) return def;
  }
  return NULL;
}

/*
============
ED_FindFunction
============
*/
static dfunction_t *ED_FindFunction(const char *fn_name) {
  dfunction_t *func;
  int i;

  for (i = 0; i < progs->numfunctions; i++) {
    func = &pr_functions[i];
    if (!strcmp(PR_GetString(func->s_name), fn_name)) return func;
  }
  return NULL;
}

/*
============
GetEdictFieldValue
============
*/
eval_t *GetEdictFieldValue(entvars_t *ev, const char *field) {
  ddef_t *def = NULL;
  int i;
  static int rep = 0;

  for (i = 0; i < GEFV_CACHESIZE; i++) {
    if (!strcmp(field, gefvCache[i].field)) {
      def = gefvCache[i].pcache;
      goto Done;
    }
  }

  def = ED_FindField(field);

  if (strlen(field) < MAX_FIELD_LEN) {
    gefvCache[rep].pcache = def;
    strcpy(gefvCache[rep].field, field);
    rep ^= 1;
  }

Done:
  if (!def) return NULL;

  return (eval_t *)((char *)ev + def->ofs * 4);
}

/*
============
PR_ValueString
(etype_t type, eval_t *val)

Returns a string describing *data in a type specific manner
=============
*/
static const char *PR_ValueString(int type, eval_t *val) {
  static char line[512];
  ddef_t *def;
  dfunction_t *f;

  type &= ~DEF_SAVEGLOBAL;

  switch (type) {
    case ev_string:
      sprintf(line, "%s", PR_GetString(val->string));
      break;
    case ev_entity:
      sprintf(line, "entity %i", NUM_FOR_EDICT(EDICT_NUM(val->edict)));
      break;
    case ev_function:
      f = pr_functions + val->function;
      sprintf(line, "%s()", PR_GetString(f->s_name));
      break;
    case ev_field:
      def = ED_FieldAtOfs(val->_int);
      sprintf(line, ".%s", PR_GetString(def->s_name));
      break;
    case ev_void:
      sprintf(line, "void");
      break;
    case ev_float:
      sprintf(line, "%5.1f", val->_float);
      break;
    case ev_vector:
      sprintf(line, "'%5.1f %5.1f %5.1f'", val->vector[0], val->vector[1],
              val->vector[2]);
      break;
    case ev_pointer:
      sprintf(line, "pointer");
      break;
    default:
      sprintf(line, "bad type %i", type);
      break;
  }

  return line;
}

/*
============
PR_UglyValueString
(etype_t type, eval_t *val)

Returns a string describing *data in a type specific manner
Easier to parse than PR_ValueString
=============
*/
static const char *PR_UglyValueString(int type, eval_t *val) {
  static char line[512];
  ddef_t *def;
  dfunction_t *f;

  type &= ~DEF_SAVEGLOBAL;

  switch (type) {
    case ev_string:
      sprintf(line, "%s", PR_GetString(val->string));
      break;
    case ev_entity:
      sprintf(line, "%i", NUM_FOR_EDICT(EDICT_NUM(val->edict)));
      break;
    case ev_function:
      f = pr_functions + val->function;
      sprintf(line, "%s", PR_GetString(f->s_name));
      break;
    case ev_field:
      def = ED_FieldAtOfs(val->_int);
      sprintf(line, "%s", PR_GetString(def->s_name));
      break;
    case ev_void:
      sprintf(line, "void");
      break;
    case ev_float:
      sprintf(line, "%f", val->_float);
      break;
    case ev_vector:
      sprintf(line, "%f %f %f", val->vector[0], val->vector[1], val->vector[2]);
      break;
    default:
      sprintf(line, "bad type %i", type);
      break;
  }

  return line;
}

/*
============
PR_GlobalString

Returns a string with a description and the contents of a global,
padded to 20 field width
============
*/
const char *PR_GlobalString(int ofs) {
  static char line[512];
  const char *s;
  int i;
  ddef_t *def;
  void *val;

  val = (void *)&pr_globals[ofs];
  def = ED_GlobalAtOfs(ofs);
  if (!def)
    sprintf(line, "%i(?)", ofs);
  else {
    s = PR_ValueString(def->type, (eval_t *)val);
    sprintf(line, "%i(%s)%s", ofs, PR_GetString(def->s_name), s);
  }

  i = strlen(line);
  for (; i < 20; i++) strcat(line, " ");
  strcat(line, " ");

  return line;
}

const char *PR_GlobalStringNoContents(int ofs) {
  static char line[512];
  int i;
  ddef_t *def;

  def = ED_GlobalAtOfs(ofs);
  if (!def)
    sprintf(line, "%i(?)", ofs);
  else
    sprintf(line, "%i(%s)", ofs, PR_GetString(def->s_name));

  i = strlen(line);
  for (; i < 20; i++) strcat(line, " ");
  strcat(line, " ");

  return line;
}

/*
=============
ED_Print

For debugging
=============
*/
void ED_Print(edict_t *ed) {
  ddef_t *d;
  int *v;
  int i, j, l;
  const char *name;
  int type;

  if (ed->free) {
    Con_Printf("FREE\n");
    return;
  }

  Con_SafePrintf("\nEDICT %i:\n",
                 NUM_FOR_EDICT(ed));  // johnfitz -- was Con_Printf
  for (i = 1; i < progs->numfielddefs; i++) {
    d = &pr_fielddefs[i];
    name = PR_GetString(d->s_name);
    l = strlen(name);
    if (l > 1 && name[l - 2] == '_') continue;  // skip _x, _y, _z vars

    v = (int *)((char *)EdictV(ed) + d->ofs * 4);

    // if the value is still all 0, skip the field
    type = d->type & ~DEF_SAVEGLOBAL;

    for (j = 0; j < type_size[type]; j++) {
      if (v[j]) break;
    }
    if (j == type_size[type]) continue;

    Con_SafePrintf("%s", name);            // johnfitz -- was Con_Printf
    while (l++ < 15) Con_SafePrintf(" ");  // johnfitz -- was Con_Printf

    Con_SafePrintf(
        "%s\n",
        PR_ValueString(d->type, (eval_t *)v));  // johnfitz -- was Con_Printf
  }
}

/*
=============
ED_Write

For savegames
=============
*/
void ED_Write(FILE *f, edict_t *ed) {
  ddef_t *d;
  int *v;
  int i, j;
  const char *name;
  int type;

  fprintf(f, "{\n");

  if (ed->free) {
    fprintf(f, "}\n");
    return;
  }

  for (i = 1; i < progs->numfielddefs; i++) {
    d = &pr_fielddefs[i];
    name = PR_GetString(d->s_name);
    j = strlen(name);
    if (j > 1 && name[j - 2] == '_') continue;  // skip _x, _y, _z vars

    v = (int *)((char *)EdictV(ed) + d->ofs * 4);

    // if the value is still all 0, skip the field
    type = d->type & ~DEF_SAVEGLOBAL;
    for (j = 0; j < type_size[type]; j++) {
      if (v[j]) break;
    }
    if (j == type_size[type]) continue;

    fprintf(f, "\"%s\" ", name);
    fprintf(f, "\"%s\"\n", PR_UglyValueString(d->type, (eval_t *)v));
  }

  // johnfitz -- save entity alpha manually when progs.dat doesn't know about
  // alpha
  if (!pr_alpha_supported && ed->alpha != ENTALPHA_DEFAULT)
    fprintf(f, "\"alpha\" \"%f\"\n", ENTALPHA_TOSAVE(ed->alpha));
  // johnfitz

  fprintf(f, "}\n");
}

void ED_PrintNum(int ent) { ED_Print(EDICT_NUM(ent)); }

/*
=============
ED_PrintEdicts

For debugging, prints all the entities in the current server
=============
*/
void ED_PrintEdicts(void) {
  int i;

  if (!SV_Active()) return;

  Con_Printf("%i entities\n", SV_NumEdicts());
  for (i = 0; i < SV_NumEdicts(); i++) ED_PrintNum(i);
}

/*
=============
ED_PrintEdict_f

For debugging, prints a single edicy
=============
*/
static void ED_PrintEdict_f(void) {
  int i;

  if (!SV_Active()) return;

  i = Cmd_ArgvAsInt(1);
  if (i < 0 || i >= SV_NumEdicts()) {
    Con_Printf("Bad edict number\n");
    return;
  }
  ED_PrintNum(i);
}

/*
=============
ED_Count

For debugging
=============
*/
static void ED_Count(void) {
  edict_t *ent;
  int i, active, models, solid, step;

  if (!SV_Active()) return;

  active = models = solid = step = 0;
  for (i = 0; i < SV_NumEdicts(); i++) {
    ent = EDICT_NUM(i);
    if (ent->free) continue;
    active++;
    if (EdictV(ent)->solid) solid++;
    if (EdictV(ent)->model) models++;
    if (EdictV(ent)->movetype == MOVETYPE_STEP) step++;
  }

  Con_Printf("num_edicts:%3i\n", SV_NumEdicts());
  Con_Printf("active    :%3i\n", active);
  Con_Printf("view      :%3i\n", models);
  Con_Printf("touch     :%3i\n", solid);
  Con_Printf("step      :%3i\n", step);
}

/*
==============================================================================

ARCHIVING GLOBALS

FIXME: need to tag constants, doesn't really work
==============================================================================
*/

/*
=============
ED_WriteGlobals
=============
*/
void ED_WriteGlobals(FILE *f) {
  ddef_t *def;
  int i;
  const char *name;
  int type;

  fprintf(f, "{\n");
  for (i = 0; i < progs->numglobaldefs; i++) {
    def = &pr_globaldefs[i];
    type = def->type;
    if (!(def->type & DEF_SAVEGLOBAL)) continue;
    type &= ~DEF_SAVEGLOBAL;

    if (type != ev_string && type != ev_float && type != ev_entity) continue;

    name = PR_GetString(def->s_name);
    fprintf(f, "\"%s\" ", name);
    fprintf(f, "\"%s\"\n",
            PR_UglyValueString(type, (eval_t *)&pr_globals[def->ofs]));
  }
  fprintf(f, "}\n");
}

/*
=============
ED_ParseGlobals
=============
*/
void ED_ParseGlobals(const char *data) {
  char keyname[64];
  ddef_t *key;

  while (1) {
    // parse key
    data = COM_Parse(data);
    if (com_token[0] == '}') break;
    if (!data) Host_Error("ED_ParseEntity: EOF without closing brace");

    strcpy(keyname, com_token);

    // parse value
    data = COM_Parse(data);
    if (!data) Host_Error("ED_ParseEntity: EOF without closing brace");

    if (com_token[0] == '}')
      Host_Error("ED_ParseEntity: closing brace without data");

    key = ED_FindGlobal(keyname);
    if (!key) {
      Con_Printf("'%s' is not a global\n", keyname);
      continue;
    }

    if (!ED_ParseEpair((void *)pr_globals, key, com_token))
      Host_Error("ED_ParseGlobals: parse error");
  }
}

//============================================================================

/*
=============
ED_NewString
=============
*/
static GoInt32 ED_NewString(const char *string) {
  char *new_p;
  int i, l;
  GoInt32 num;

  l = strlen(string) + 1;
  num = PR_AllocString(l, &new_p);

  for (i = 0; i < l; i++) {
    if (string[i] == '\\' && i < l - 1) {
      i++;
      if (string[i] == 'n')
        *new_p++ = '\n';
      else
        *new_p++ = '\\';
    } else
      *new_p++ = string[i];
  }

  return num;
}

/*
=============
ED_ParseEval

Can parse either fields or globals
returns false if error
=============
*/
static qboolean ED_ParseEpair(void *base, ddef_t *key, const char *s) {
  int i;
  char string[128];
  ddef_t *def;
  char *v, *w;
  void *d;
  dfunction_t *func;

  d = (void *)((int *)base + key->ofs);

  switch (key->type & ~DEF_SAVEGLOBAL) {
    case ev_string:
      *(GoInt32 *)d = ED_NewString(s);
      break;

    case ev_float:
      *(float *)d = atof(s);
      break;

    case ev_vector:
      strcpy(string, s);
      v = string;
      w = string;
      for (i = 0; i < 3; i++) {
        while (*v && *v != ' ') v++;
        *v = 0;
        ((float *)d)[i] = atof(w);
        w = v = v + 1;
      }
      break;

    case ev_entity:
      *(int *)d = (atoi(s));
      break;

    case ev_field:
      def = ED_FindField(s);
      if (!def) {
        // johnfitz -- HACK -- suppress error becuase fog/sky fields might not
        // be mentioned in defs.qc
        if (strncmp(s, "sky", 3) && strcmp(s, "fog"))
          Con_DPrintf("Can't find field %s\n", s);
        return false;
      }
      *(int *)d = G_INT(def->ofs);
      break;

    case ev_function:
      func = ED_FindFunction(s);
      if (!func) {
        Con_Printf("Can't find function %s\n", s);
        return false;
      }
      *(GoInt32 *)d = func - pr_functions;
      break;

    default:
      break;
  }
  return true;
}

/*
====================
ED_ParseEdict

Parses an edict out of the given string, returning the new position
ed should be a properly initialized empty edict.
Used for initial level load and for savegames.
====================
*/
const char *ED_ParseEdict(const char *data, edict_t *ent) {
  ddef_t *key;
  char keyname[256];
  qboolean anglehack, init;
  int n;

  init = false;

  // clear it
  if (ent != sv.edicts)  // hack
    TT_ClearEntVars(EdictV(ent));

  // go through all the dictionary pairs
  while (1) {
    // parse key
    data = COM_Parse(data);
    if (com_token[0] == '}') break;
    if (!data) Host_Error("ED_ParseEntity: EOF without closing brace");

    // anglehack is to allow QuakeEd to write single scalar angles
    // and allow them to be turned into vectors. (FIXME...)
    if (!strcmp(com_token, "angle")) {
      strcpy(com_token, "angles");
      anglehack = true;
    } else
      anglehack = false;

    // FIXME: change light to _light to get rid of this hack
    if (!strcmp(com_token, "light"))
      strcpy(com_token, "light_lev");  // hack for single light def

    strcpy(keyname, com_token);

    // another hack to fix keynames with trailing spaces
    n = strlen(keyname);
    while (n && keyname[n - 1] == ' ') {
      keyname[n - 1] = 0;
      n--;
    }

    // parse value
    data = COM_Parse(data);
    if (!data) Host_Error("ED_ParseEntity: EOF without closing brace");

    if (com_token[0] == '}')
      Host_Error("ED_ParseEntity: closing brace without data");

    init = true;

    // keynames with a leading underscore are used for utility comments,
    // and are immediately discarded by quake
    if (keyname[0] == '_') continue;

    // johnfitz -- hack to support .alpha even when progs.dat doesn't know about
    // it
    if (!strcmp(keyname, "alpha"))
      ent->alpha = ENTALPHA_ENCODE(atof(com_token));
    // johnfitz

    key = ED_FindField(keyname);
    if (!key) {
      // johnfitz -- HACK -- suppress error becuase fog/sky/alpha fields might
      // not be mentioned in defs.qc
      if (strncmp(keyname, "sky", 3) && strcmp(keyname, "fog") &&
          strcmp(keyname, "alpha"))
        Con_DPrintf("\"%s\" is not a field\n",
                    keyname);  // johnfitz -- was Con_Printf
      continue;
    }

    if (anglehack) {
      char temp[32];
      strcpy(temp, com_token);
      sprintf(com_token, "0 %s 0", temp);
    }

    if (!ED_ParseEpair((void *)EdictV(ent), key, com_token))
      Host_Error("ED_ParseEdict: parse error");
  }

  if (!init) ent->free = true;

  return data;
}

/*
================
ED_LoadFromFile

The entities are directly placed in the array, rather than allocated with
ED_Alloc, because otherwise an error loading the map would have entity
number references out of order.

Creates a server's entity / program execution context by
parsing textual entity definitions out of an ent file.

Used for both fresh maps and savegame loads.  A fresh map would also need
to call ED_CallSpawnFunctions () to let the objects initialize themselves.
================
*/
void ED_LoadFromFile(const char *data) {
  dfunction_t *func;
  edict_t *ent = NULL;
  int inhibit = 0;

  pr_global_struct->time = SV_Time();

  // parse ents
  while (1) {
    // parse the opening brace
    data = COM_Parse(data);
    if (!data) break;
    if (com_token[0] != '{')
      Host_Error("ED_LoadFromFile: found %s when expecting {", com_token);

    if (!ent)
      ent = EDICT_NUM(0);
    else
      ent = ED_Alloc();
    data = ED_ParseEdict(data, ent);

    // remove things from different skill levels or deathmatch
    if (Cvar_GetValue(&deathmatch)) {
      if (((int)EdictV(ent)->spawnflags & SPAWNFLAG_NOT_DEATHMATCH)) {
        ED_Free(ent);
        inhibit++;
        continue;
      }
    } else if ((current_skill == 0 &&
                ((int)EdictV(ent)->spawnflags & SPAWNFLAG_NOT_EASY)) ||
               (current_skill == 1 &&
                ((int)EdictV(ent)->spawnflags & SPAWNFLAG_NOT_MEDIUM)) ||
               (current_skill >= 2 &&
                ((int)EdictV(ent)->spawnflags & SPAWNFLAG_NOT_HARD))) {
      ED_Free(ent);
      inhibit++;
      continue;
    }

    //
    // immediately call spawn function
    //
    if (!EdictV(ent)->classname) {
      Con_SafePrintf("No classname for:\n");  // johnfitz -- was Con_Printf
      ED_Print(ent);
      ED_Free(ent);
      continue;
    }

    // look for the spawn function
    func = ED_FindFunction(PR_GetString(EdictV(ent)->classname));

    if (!func) {
      Con_SafePrintf("No spawn function for:\n");  // johnfitz -- was Con_Printf
      ED_Print(ent);
      ED_Free(ent);
      continue;
    }

    pr_global_struct->self = NUM_FOR_EDICT(ent);
    PR_ExecuteProgram(func - pr_functions);
  }

  Con_DPrintf("%i entities inhibited\n", inhibit);
}

/*
===============
PR_LoadProgs
===============
*/
void PR_LoadProgs(void) {
  int length = 0;
  int i;

  // flush the non-C variable lookup cache
  for (i = 0; i < GEFV_CACHESIZE; i++) gefvCache[i].field[0] = 0;

  progs = (dprograms_t *)COM_LoadFileGo("progs.dat", &length);
  if (!progs) Host_Error("PR_LoadProgs: couldn't load progs.dat");
  Con_DPrintf("Programs occupy %iK.\n", length / 1024);

  pr_crc = CRC_Block(((byte *)progs), length);

  // byte swap the header
  for (i = 0; i < (int)sizeof(*progs) / 4; i++)
    ((int *)progs)[i] = LittleLong(((int *)progs)[i]);

  if (progs->version != PROG_VERSION)
    Host_Error("progs.dat has wrong version number (%i should be %i)",
               progs->version, PROG_VERSION);
  if (progs->crc != PROGHEADER_CRC)
    Host_Error(
        "progs.dat system vars have been modified, progdefs.h is out of date");

  pr_functions = (dfunction_t *)((byte *)progs + progs->ofs_functions);
  pr_strings = (char *)progs + progs->ofs_strings;
  if (progs->ofs_strings + progs->numstrings >= length)
    Host_Error("progs.dat strings go past end of file\n");

  // initialize the strings
  pr_numknownstrings = 0;
  pr_maxknownstrings = 0;
  pr_stringssize = progs->numstrings;
  if (pr_knownstrings) Z_Free((void *)pr_knownstrings);
  pr_knownstrings = NULL;
  PR_SetEngineString("");

  pr_globaldefs = (ddef_t *)((byte *)progs + progs->ofs_globaldefs);
  pr_fielddefs = (ddef_t *)((byte *)progs + progs->ofs_fielddefs);
  pr_statements = (dstatement_t *)((byte *)progs + progs->ofs_statements);

  pr_global_struct = (globalvars_t *)((byte *)progs + progs->ofs_globals);
  pr_globals = (float *)pr_global_struct;

  // byte swap the lumps
  for (i = 0; i < progs->numstatements; i++) {
    pr_statements[i].op = LittleShort(pr_statements[i].op);
    pr_statements[i].a = LittleShort(pr_statements[i].a);
    pr_statements[i].b = LittleShort(pr_statements[i].b);
    pr_statements[i].c = LittleShort(pr_statements[i].c);
  }

  for (i = 0; i < progs->numfunctions; i++) {
    pr_functions[i].first_statement =
        LittleLong(pr_functions[i].first_statement);
    pr_functions[i].parm_start = LittleLong(pr_functions[i].parm_start);
    pr_functions[i].s_name = LittleLong(pr_functions[i].s_name);
    pr_functions[i].s_file = LittleLong(pr_functions[i].s_file);
    pr_functions[i].numparms = LittleLong(pr_functions[i].numparms);
    pr_functions[i].locals = LittleLong(pr_functions[i].locals);
  }

  for (i = 0; i < progs->numglobaldefs; i++) {
    pr_globaldefs[i].type = LittleShort(pr_globaldefs[i].type);
    pr_globaldefs[i].ofs = LittleShort(pr_globaldefs[i].ofs);
    pr_globaldefs[i].s_name = LittleLong(pr_globaldefs[i].s_name);
  }

  pr_alpha_supported = false;  // johnfitz

  for (i = 0; i < progs->numfielddefs; i++) {
    pr_fielddefs[i].type = LittleShort(pr_fielddefs[i].type);
    if (pr_fielddefs[i].type & DEF_SAVEGLOBAL)
      Host_Error("PR_LoadProgs: pr_fielddefs[i].type & DEF_SAVEGLOBAL");
    pr_fielddefs[i].ofs = LittleShort(pr_fielddefs[i].ofs);
    pr_fielddefs[i].s_name = LittleLong(pr_fielddefs[i].s_name);

    // johnfitz -- detect alpha support in progs.dat
    if (!strcmp(pr_strings + pr_fielddefs[i].s_name, "alpha"))
      pr_alpha_supported = true;
    // johnfitz
  }

  for (i = 0; i < progs->numglobals; i++)
    ((int *)pr_globals)[i] = LittleLong(((int *)pr_globals)[i]);
}

void TT_ClearEdict(edict_t *e) {
  memset(e, 0, sizeof(edict_t));
  TT_ClearEntVars(EdictV(e));
}

edict_t *NEXT_EDICT(edict_t *e) {
  return ((edict_t *)((byte *)e + sizeof(edict_t)));
}

edict_t *EDICT_NUM(int n) {
  if (n < 0 || n >= SV_MaxEdicts()) Host_Error("EDICT_NUM: bad number %i", n);
  return (edict_t *)((byte *)sv.edicts + (n) * sizeof(edict_t));
}

int NUM_FOR_EDICT(edict_t *e) {
  int b;

  b = (byte *)e - (byte *)sv.edicts;
  b = b / sizeof(edict_t);

  if (b < 0 || b >= SV_NumEdicts()) Host_Error("NUM_FOR_EDICT: bad pointer");
  return b;
}

edict_t *AllocEdicts() {
  AllocEntvars(SV_MaxEdicts(), progs->entityfields);
  return (edict_t *)malloc(SV_MaxEdicts() * sizeof(edict_t));
}

void FreeEdicts(edict_t *e) {
  FreeEntvars();
  free(e);
}

edict_t *G_EDICT(int o) { return EDICT_NUM(*(int *)&pr_globals[o]); }

entvars_t *EdictV(edict_t *e) {
  int n = NUM_FOR_EDICT(e);
  return EVars(n);
}

/*
===============
PR_Init
===============
*/
void PR_Init(void) {
  Cmd_AddCommand("edict", ED_PrintEdict_f);
  Cmd_AddCommand("edicts", ED_PrintEdicts);
  Cmd_AddCommand("edictcount", ED_Count);
  Cmd_AddCommand("profile", PR_Profile_f);
  Cvar_FakeRegister(&nomonsters, "nomonsters");
  Cvar_FakeRegister(&gamecfg, "gamecfg");
  Cvar_FakeRegister(&scratch1, "scratch1");
  Cvar_FakeRegister(&scratch2, "scratch2");
  Cvar_FakeRegister(&scratch3, "scratch3");
  Cvar_FakeRegister(&scratch4, "scratch4");
  Cvar_FakeRegister(&savedgamecfg, "savedgamecfg");
  Cvar_FakeRegister(&saved1, "saved1");
  Cvar_FakeRegister(&saved2, "saved2");
  Cvar_FakeRegister(&saved3, "saved3");
  Cvar_FakeRegister(&saved4, "saved4");
}

//===========================================================================

#define PR_STRING_ALLOCSLOTS 256

static void PR_AllocStringSlots(void) {
  pr_maxknownstrings += PR_STRING_ALLOCSLOTS;
  Con_DPrintf2("PR_AllocStringSlots: realloc'ing for %d slots\n",
               pr_maxknownstrings);
  pr_knownstrings = (const char **)Z_Realloc(
      (void *)pr_knownstrings, pr_maxknownstrings * sizeof(char *));
}

const char *PR_GetString(int num) {
  // positive numbers are strings in progs.dat
  // negative ones new ones from SetEngineString
  if (num >= 0 && num < pr_stringssize)
    return pr_strings + num;
  else if (num < 0 && num >= -pr_numknownstrings) {
    if (!pr_knownstrings[-1 - num]) {
      Host_Error("PR_GetString: attempt to get a non-existant string %d\n",
                 num);
      return "";
    }
    return pr_knownstrings[-1 - num];
  } else {
    Host_Error("PR_GetString: invalid string offset %d\n", num);
    return "";
  }
}

int PR_SetEngineString(const char *s) {
  int i;

  if (!s) return 0;
#if 0 /* can't: sv.model_precache & sv.sound_precache points to pr_strings */
	if (s >= pr_strings && s <= pr_strings + pr_stringssize)
		Host_Error("PR_SetEngineString: \"%s\" in pr_strings area\n", s);
#else
  if (s >= pr_strings && s <= pr_strings + pr_stringssize - 2)
    return (int)(s - pr_strings);
#endif
  for (i = 0; i < pr_numknownstrings; i++) {
    if (pr_knownstrings[i] == s) return -1 - i;
  }
// new unknown engine string
// Con_DPrintf ("PR_SetEngineString: new engine string %p\n", s);
#if 0
	for (i = 0; i < pr_numknownstrings; i++)
	{
		if (!pr_knownstrings[i])
			break;
	}
#endif
  //	if (i >= pr_numknownstrings)
  //	{
  if (i >= pr_maxknownstrings) PR_AllocStringSlots();
  pr_numknownstrings++;
  //	}
  pr_knownstrings[i] = s;
  return -1 - i;
}

int PR_AllocString(int size, char **ptr) {
  int i;

  if (!size) return 0;
  for (i = 0; i < pr_numknownstrings; i++) {
    if (!pr_knownstrings[i]) break;
  }
  //	if (i >= pr_numknownstrings)
  //	{
  if (i >= pr_maxknownstrings) PR_AllocStringSlots();
  pr_numknownstrings++;
  //	}
  pr_knownstrings[i] = (char *)Hunk_AllocName(size, "string");
  if (ptr) *ptr = (char *)pr_knownstrings[i];
  return -1 - i;
}
