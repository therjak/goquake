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
float *pr_globals;

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
int PR_AllocString(int size, char **ptr);

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
void ED_ClearEdict(int e) {
  TT_ClearEntVars(EVars(e));
  EDICT_NUM(e)->free = false;
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
int ED_Alloc(void) {
  int i;
  edict_t *e;

  for (i = SVS_GetMaxClients() + 1; i < SV_NumEdicts(); i++) {
    e = EDICT_NUM(i);
    // the first couple seconds of server time can involve a lot of
    // freeing and allocating, so relax the replacement policy
    if (e->free && (e->freetime < 2 || SV_Time() - e->freetime > 0.5)) {
      ED_ClearEdict(i);
      return i;
    }
  }

  if (i == SV_MaxEdicts())  // johnfitz -- use sv.max_edicts instead of
                            // MAX_EDICTS
    Host_Error("ED_Alloc: no free edicts (max_edicts is %i)", SV_MaxEdicts());

  SV_SetNumEdicts(SV_NumEdicts() + 1);
  TT_ClearEdict(i);
  // ericw -- switched sv.edicts to malloc(), so
  // we are accessing uninitialized memory and
  // must fully zero it, not just ED_ClearEdict

  return i;
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
  static char line[1024];
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

  def = ED_GlobalAtOfs(ofs);
  if (!def)
    sprintf(line, "%i(?)", ofs);
  else {
    eval_t v;
    v.vector[0] = Pr_globalsf(ofs);
    v.vector[1] = Pr_globalsf(ofs + 1);
    v.vector[2] = Pr_globalsf(ofs + 2);
    s = PR_ValueString(def->type, &v);
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
ED_Write

For savegames
=============
*/
void ED_Write(FILE *f, int ed) {
  ddef_t *d;
  int *v;
  int i, j;
  const char *name;
  int type;

  fprintf(f, "{\n");

  if (EDICT_NUM(ed)->free) {
    fprintf(f, "}\n");
    return;
  }

  for (i = 1; i < progs->numfielddefs; i++) {
    d = &pr_fielddefs[i];
    name = PR_GetString(d->s_name);
    j = strlen(name);
    if (j > 1 && name[j - 2] == '_') continue;  // skip _x, _y, _z vars

    v = (int *)((char *)EVars(ed) + d->ofs * 4);

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
  if (!pr_alpha_supported && EDICT_NUM(ed)->alpha != ENTALPHA_DEFAULT)
    fprintf(f, "\"alpha\" \"%f\"\n", ENTALPHA_TOSAVE(EDICT_NUM(ed)->alpha));
  // johnfitz

  fprintf(f, "}\n");
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
    eval_t v;
    v.vector[0] = Pr_globalsf(def->ofs);
    v.vector[1] = Pr_globalsf(def->ofs + 1);
    v.vector[2] = Pr_globalsf(def->ofs + 2);
    fprintf(f, "\"%s\"\n", PR_UglyValueString(type, &v));
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

    // Sys_Print("ParseGlobals");
    if (!ED_ParseEpair((void *)pr_globals, key, com_token))
      Host_Error("ED_ParseGlobals: parse error");
  }
}

//============================================================================

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
      *(int *)d = Pr_globalsi(def->ofs);
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
const char *ED_ParseEdict(const char *data, int ent) {
  ddef_t *key;
  char keyname[256];
  qboolean anglehack, init;
  int n;

  init = false;

  // clear it
  if (ent != 0)  // hack
    TT_ClearEntVars(EVars(ent));

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
      EDICT_NUM(ent)->alpha = ENTALPHA_ENCODE(atof(com_token));
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

    if (!ED_ParseEpair((void *)EVars(ent), key, com_token))
      Host_Error("ED_ParseEdict: parse error");
  }

  if (!init) EDICT_NUM(ent)->free = true;

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
  int inhibit = 0;
  int edictnum = -1;

  Set_pr_global_struct_time(SV_Time());

  // parse ents
  while (1) {
    // parse the opening brace
    data = COM_Parse(data);
    if (!data) break;
    if (com_token[0] != '{')
      Host_Error("ED_LoadFromFile: found %s when expecting {", com_token);

    if (edictnum < 0)
      edictnum = 0;
    else
      edictnum = ED_Alloc();
    data = ED_ParseEdict(data, edictnum);

    // remove things from different skill levels or deathmatch
    if (Cvar_GetValue(&deathmatch)) {
      if (((int)EVars(edictnum)->spawnflags & SPAWNFLAG_NOT_DEATHMATCH)) {
        ED_Free(edictnum);
        inhibit++;
        continue;
      }
    } else if ((current_skill == 0 &&
                ((int)EVars(edictnum)->spawnflags & SPAWNFLAG_NOT_EASY)) ||
               (current_skill == 1 &&
                ((int)EVars(edictnum)->spawnflags & SPAWNFLAG_NOT_MEDIUM)) ||
               (current_skill >= 2 &&
                ((int)EVars(edictnum)->spawnflags & SPAWNFLAG_NOT_HARD))) {
      ED_Free(edictnum);
      inhibit++;
      continue;
    }

    //
    // immediately call spawn function
    //
    if (!EVars(edictnum)->classname) {
      Con_SafePrintf("No classname for:\n");  // johnfitz -- was Con_Printf
      ED_PrintNum(edictnum);
      ED_Free(edictnum);
      continue;
    }

    // look for the spawn function
    func = ED_FindFunction(PR_GetString(EVars(edictnum)->classname));

    if (!func) {
      Con_SafePrintf("No spawn function for:\n");  // johnfitz -- was Con_Printf
      ED_PrintNum(edictnum);
      ED_Free(edictnum);
      continue;
    }

    Set_pr_global_struct_self(edictnum);
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

  pr_globals = (float *)((byte *)progs + progs->ofs_globals);

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
}

void TT_ClearEdict(int e) {
  edict_t *ent = EDICT_NUM(e);
  memset(ent, 0, sizeof(edict_t));
  TT_ClearEntVars(EVars(e));
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

unsigned char EAlpha(int num) { return EDICT_NUM(num)->alpha; }

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
