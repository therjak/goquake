#include "quakedef.h"

#define STRINGTEMP_BUFFERS 16
#define STRINGTEMP_LENGTH 1024
static char pr_string_temp[STRINGTEMP_BUFFERS][STRINGTEMP_LENGTH];
static byte pr_string_tempindex = 0;

static char *PR_GetTempString(void) {
  return pr_string_temp[(STRINGTEMP_BUFFERS - 1) & ++pr_string_tempindex];
}

#define RETURN_EDICT(e) (Set_Pr_globalsi(OFS_RETURN, e))

/*
===============================================================================

        BUILT-IN FUNCTIONS

===============================================================================
*/

static char *PF_VarString(int first) {
  int i;
  static char out[1024];
  size_t s;

  out[0] = 0;
  s = 0;
  for (i = first; i < pr_argc; i++) {
    s = q_strlcat(out, PR_GetString(Pr_globalsi(OFS_PARM0 + i * 3)),
                  sizeof(out));
    if (s >= sizeof(out)) {
      Con_Warning("PF_VarString: overflow (string truncated)\n");
      return out;
    }
  }
  if (s > 255)
    Con_DWarning("PF_VarString: %i characters exceeds standard limit of 255.\n",
                 (int)s);
  return out;
}

/*
=================
PF_error

This is a TERMINAL error, which will kill off the entire server.
Dumps self.

error(value)
=================
*/
static void PF_error(void) {
  char *s;

  s = PF_VarString(0);
  Con_Printf("======SERVER ERROR in %s:\n%s\n",
             PR_GetString(pr_xfunction->s_name), s);
  ED_PrintNum(Pr_global_struct_self());

  Host_Error("Program error");
}

/*
=================
PF_objerror

Dumps out self, then an error message.  The program is aborted and self is
removed, but the level can continue.

objerror(value)
=================
*/
static void PF_objerror(void) {
  char *s;
  int ed;

  s = PF_VarString(0);
  Con_Printf("======OBJECT ERROR in %s:\n%s\n",
             PR_GetString(pr_xfunction->s_name), s);
  ed = Pr_global_struct_self();
  ED_PrintNum(ed);
  ED_Free(ed);

  // Host_Error ("Program error"); //johnfitz -- by design, this should not be
  // fatal
}

/*
==============
PF_makevectors

Writes new values for v_forward, v_up, and v_right based on angles
makevectors(vector)
==============
*/
static void PF_makevectors(void) {
  vec3_t b, f, r, u;
  b[0] = Pr_globalsf(OFS_PARM0);
  b[1] = Pr_globalsf(OFS_PARM0 + 1);
  b[2] = Pr_globalsf(OFS_PARM0 + 2);
  AngleVectors(b, f, r, u);
  Set_pr_global_struct_v_forward(f[0], f[1], f[2]);
  Set_pr_global_struct_v_right(r[0], r[1], r[2]);
  Set_pr_global_struct_v_up(u[0], u[1], u[2]);
}

static void PF_setmodel(void) {
  int i;
  int mi;
  const char *m, **check;
  qmodel_t *mod;
  int e;
  PF_setmodel2();

  e = Pr_globalsi(OFS_PARM0);
  mi = Pr_globalsi(OFS_PARM1);
  m = PR_GetString(mi);

  // check to see if model was properly precached
  i = ElementOfSVModelPrecache(m);
  if (i == -1) {
    PR_RunError("no precache: %s", m);
  }
  mod = sv.models[i];  // Mod_ForName (m, true);
  Sys_Print_S("sm1 nn: ", mod->name);

  if (mod) {
    if (mod->Type == mod_brush) {
      Sys_Print_F("mms n: ", mod->clipmins[0]);
      Sys_Print_F("mms n: ", mod->clipmins[1]);
      Sys_Print_F("mms n: ", mod->clipmins[2]);
      Sys_Print_F("mms n: ", mod->clipmaxs[0]);
      Sys_Print_F("mms n: ", mod->clipmaxs[1]);
      Sys_Print_F("mms n: ", mod->clipmaxs[2]);
    } else {
      /*
  Sys_Print_F("mms n: ", minvec[0]);
  Sys_Print_F("mms m: ", minvec[1]);
  Sys_Print_F("mms m: ", minvec[2]);
  Sys_Print_F("mms m: ", maxvec[0]);
  Sys_Print_F("mms m: ", maxvec[1]);
  Sys_Print_F("mms m: ", maxvec[2]);
  */
    }
  }
}

/*
=================
PF_bprint

broadcast print to everyone on server

bprint(value)
=================
*/
static void PF_bprint(void) {
  char *s;

  s = PF_VarString(0);
  SV_BroadcastPrintf("%s", s);
}

/*
=================
PF_sprint

single print to a specific client

sprint(clientent, value)
=================
*/
static void PF_sprint(void) {
  char *s;
  int client;
  int entnum;

  entnum = Pr_globalsi(OFS_PARM0);
  s = PF_VarString(1);

  if (entnum < 1 || entnum > SVS_GetMaxClients()) {
    Con_Printf("tried to sprint to a non-client\n");
    return;
  }

  client = entnum - 1;

  ClientWriteChar(client, svc_print);
  ClientWriteString(client, s);
}

/*
=================
PF_centerprint

single print to a specific client

centerprint(clientent, value)
=================
*/
static void PF_centerprint(void) {
  char *s;
  int client;
  int entnum;

  entnum = Pr_globalsi(OFS_PARM0);
  s = PF_VarString(1);

  if (entnum < 1 || entnum > SVS_GetMaxClients()) {
    Con_Printf("tried to sprint to a non-client\n");
    return;
  }

  client = entnum - 1;

  ClientWriteChar(client, svc_centerprint);
  ClientWriteString(client, s);
}

static byte checkpvs[MAX_MAP_LEAFS / 8];

static int PF_newcheckclient(int check) {
  int i;
  byte *pvs;
  int ent;
  mleaf_t *leaf;
  vec3_t org;

  // cycle to the next one

  if (check < 1) check = 1;
  if (check > SVS_GetMaxClients()) check = SVS_GetMaxClients();

  if (check == SVS_GetMaxClients()) {
    i = 1;
  } else {
    i = check + 1;
  }

  for (;; i++) {
    if (i == SVS_GetMaxClients() + 1) i = 1;

    ent = i;

    if (i == check) break;  // didn't find anything else

    if (EDICT_NUM(ent)->free) continue;
    if (EVars(ent)->health <= 0) continue;
    if ((int)EVars(ent)->flags & FL_NOTARGET) continue;

    // anything that is a client, or has a client as an enemy
    break;
  }

  // get the PVS for the entity
  VectorAdd(EVars(ent)->origin, EVars(ent)->view_ofs, org);
  leaf = Mod_PointInLeaf(org, sv.worldmodel);
  pvs = Mod_LeafPVS(leaf, sv.worldmodel);
  memcpy(checkpvs, pvs, (sv.worldmodel->numleafs + 7) >> 3);

  return i;
}

/*
=================
PF_checkclient

Returns a client (or object that has a client enemy) that would be a
valid target.

If there are more than one valid options, they are cycled each frame

If (self.origin + self.viewofs) is not in the PVS of the current target,
it is not returned at all.

name checkclient ()
=================
*/
#define MAX_CHECK 16
static int c_invis, c_notvis;
static void PF_checkclient(void) {
  int ent;
  int self;
  mleaf_t *leaf;
  int l;
  vec3_t view;

  // find a new check if on a new frame
  if (SV_Time() - SV_LastCheckTime() >= 0.1) {
    SV_SetLastCheck(PF_newcheckclient(SV_LastCheck()));
    SV_SetLastCheckTime(SV_Time());
  }

  // return check if it might be visible
  ent = SV_LastCheck();
  if (EDICT_NUM(ent)->free || EVars(ent)->health <= 0) {
    RETURN_EDICT(0);
    return;
  }

  // if current entity can't possibly see the check entity, return 0
  self = Pr_global_struct_self();
  VectorAdd(EVars(self)->origin, EVars(self)->view_ofs, view);
  leaf = Mod_PointInLeaf(view, sv.worldmodel);
  l = (leaf - sv.worldmodel->leafs) - 1;
  if ((l < 0) || !(checkpvs[l >> 3] & (1 << (l & 7)))) {
    c_notvis++;
    RETURN_EDICT(0);
    return;
  }

  // might be able to see it
  c_invis++;
  RETURN_EDICT(ent);
}

/*
=================
PF_findradius

Returns a chain of entities that have origins within a spherical area

findradius (origin, radius)
=================
*/
static void PF_findradius(void) {
  int ent;
  int chain;
  float rad;
  vec3_t org;
  vec3_t eorg;
  int i, j;

  chain = 0;

  org[0] = Pr_globalsf(OFS_PARM0);
  org[1] = Pr_globalsf(OFS_PARM0 + 1);
  org[2] = Pr_globalsf(OFS_PARM0 + 2);
  rad = Pr_globalsf(OFS_PARM1);

  ent = 1;
  for (i = 1; i < SV_NumEdicts(); i++, ent++) {
    if (EDICT_NUM(ent)->free) continue;
    if (EVars(ent)->solid == SOLID_NOT) continue;
    for (j = 0; j < 3; j++)
      eorg[j] = org[j] - (EVars(ent)->origin[j] +
                          (EVars(ent)->mins[j] + EVars(ent)->maxs[j]) * 0.5);
    if (VectorLength(eorg) > rad) continue;

    EVars(ent)->chain = chain;
    chain = ent;
  }

  RETURN_EDICT(chain);
}

/*
=========
PF_dprint
=========
*/
static void PF_dprint(void) { Con_DPrintf("%s", PF_VarString(0)); }

static void PF_ftos(void) {
  float v;
  char *s;

  v = Pr_globalsf(OFS_PARM0);
  s = PR_GetTempString();
  if (v == (int)v)
    sprintf(s, "%d", (int)v);
  else
    sprintf(s, "%5.1f", v);
  Set_Pr_globalsi(OFS_RETURN, PR_SetEngineString(s));
}

static void PF_vtos(void) {
  char *s;

  s = PR_GetTempString();
  sprintf(s, "'%5.1f %5.1f %5.1f'", Pr_globalsf(OFS_PARM0),
          Pr_globalsf(OFS_PARM0 + 1), Pr_globalsf(OFS_PARM0 + 2));
  Set_Pr_globalsi(OFS_RETURN, PR_SetEngineString(s));
}

static void PF_Spawn(void) {
  int ed;

  ed = ED_Alloc();

  RETURN_EDICT(ed);
}

static void PF_Remove(void) {
  int ed;

  ed = Pr_globalsi(OFS_PARM0);
  ED_Free(ed);
}

// entity (entity start, .string field, string match) find = #5;
static void PF_Find(void) {
  int e;
  int f;
  const char *s, *t;
  int ed;

  e = Pr_globalsi(OFS_PARM0);
  f = Pr_globalsi(OFS_PARM1);
  s = PR_GetString(Pr_globalsi(OFS_PARM2));
  if (!s) PR_RunError("PF_Find: bad search string");

  for (e++; e < SV_NumEdicts(); e++) {
    ed = e;
    if (EDICT_NUM(ed)->free) continue;
    t = (PR_GetString(*(GoInt32 *)&((float *)EVars(ed))[f]));
    if (!t) continue;
    s = PR_GetString(Pr_globalsi(OFS_PARM2));
    if (!strcmp(t, s)) {
      RETURN_EDICT(ed);
      return;
    }
  }

  RETURN_EDICT(0);
}

static void PR_CheckEmptyString(const char *s) {
  if (s[0] <= ' ') PR_RunError("Bad string");
}

static void PF_precache_file(void) {  // precache_file is only used to copy
                                      // files with qcc, it does nothing
  Set_Pr_globalsi(OFS_RETURN, Pr_globalsi(OFS_PARM0));
}

static void PF_precache_sound(void) {
  const char *s;
  int i;

  if (SV_State() != ss_loading)
    PR_RunError("PF_Precache_*: Precache can only be done in spawn functions");

  s = PR_GetString(Pr_globalsi(OFS_PARM0));
  Set_Pr_globalsi(OFS_RETURN, Pr_globalsi(OFS_PARM0));
  PR_CheckEmptyString(s);

  if (ElementOfSVSoundPrecache(s) != -1) {
    return;
  }
  for (i = 0; i < MAX_SOUNDS; i++) {
    if (!ExistSVSoundPrecache(i)) {
      SetSVSoundPrecache(i, s);
      return;
    }
  }
  PR_RunError("PF_precache_sound: overflow");
}

static void PF_precache_model(void) {
  const char *s;
  int i;

  if (SV_State() != ss_loading)
    PR_RunError("PF_Precache_*: Precache can only be done in spawn functions");

  s = PR_GetString(Pr_globalsi(OFS_PARM0));
  Set_Pr_globalsi(OFS_RETURN, Pr_globalsi(OFS_PARM0));
  PR_CheckEmptyString(s);

  if (ElementOfSVModelPrecache(s) != -1) {
    return;
  }
  for (i = 0; i < MAX_MODELS; i++) {
    if (!ExistSVModelPrecache(i)) {
      SetSVModelPrecache(i, s);
      sv.models[i] = Mod_ForName(s, true);
      SVSetModel(sv.models[i], i, false);
      return;
    }
  }
  PR_RunError("PF_precache_model: overflow");
}

static void PF_coredump(void) { ED_PrintEdicts(); }

static void PF_traceon(void) { pr_trace = true; }

static void PF_traceoff(void) { pr_trace = false; }

static void PF_eprint(void) { ED_PrintNum(Pr_globalsi(OFS_PARM0)); }

/*
===============
PF_walkmove

float(float yaw, float dist) walkmove
===============
*/
static void PF_walkmove(void) {
  int ent;
  float yaw, dist;
  vec3_t move;
  dfunction_t *oldf;
  int oldself;

  ent = Pr_global_struct_self();
  yaw = Pr_globalsf(OFS_PARM0);
  dist = Pr_globalsf(OFS_PARM1);

  if (!((int)EVars(ent)->flags & (FL_ONGROUND | FL_FLY | FL_SWIM))) {
    Set_Pr_globalsf(OFS_RETURN, 0);
    return;
  }

  yaw = yaw * M_PI * 2 / 360;

  move[0] = cos(yaw) * dist;
  move[1] = sin(yaw) * dist;
  move[2] = 0;

  // save program state, because SV_movestep may call other progs
  oldf = pr_xfunction;
  oldself = Pr_global_struct_self();

  Set_Pr_globalsf(OFS_RETURN, SV_movestep(ent, move, true));

  // restore program state
  pr_xfunction = oldf;
  Set_pr_global_struct_self(oldself);
}

/*
===============
PF_lightstyle

void(float style, string value) lightstyle
===============
*/
static void PF_lightstyle(void) {
  int style;
  const char *val;
  int j;

  style = Pr_globalsf(OFS_PARM0);
  val = PR_GetString(Pr_globalsi(OFS_PARM1));

  // change the string in sv
  sv.lightstyles[style] = val;

  // send message to all clients on this server
  if (SV_State() != ss_active) return;

  for (j = 0; j < SVS_GetMaxClients(); j++) {
    if (GetClientActive(j) || GetClientSpawned(j)) {
      ClientWriteChar(j, svc_lightstyle);
      ClientWriteChar(j, style);
      ClientWriteString(j, val);
    }
  }
}

/*
=============
PF_checkbottom
=============
*/
static void PF_checkbottom(void) {
  int ent;

  ent = Pr_globalsi(OFS_PARM0);

  Set_Pr_globalsf(OFS_RETURN, SV_CheckBottom(ent));
}

/*
=============
PF_aim

Pick a vector for the player to shoot along
vector aim(entity, missilespeed)
=============
*/
cvar_t sv_aim;  // = {"sv_aim", "1", CVAR_NONE};  // ericw -- turn autoaim off
                // by default. was 0.93
static void PF_aim(void) {
  int ent;
  int check;
  int bestent;
  vec3_t start, dir, end, bestdir;
  int i, j;
  trace_t tr;
  float dist, bestdist;
  float speed;

  ent = Pr_globalsi(OFS_PARM0);
  speed = Pr_globalsf(OFS_PARM1);
  (void)speed; /* variable set but not used */

  VectorCopy(EVars(ent)->origin, start);
  start[2] += 20;

  // try sending a trace straight
  Pr_global_struct_v_forward(&dir[0], &dir[1], &dir[2]);
  VectorMA(start, 2048, dir, end);
  tr = SV_Move(start, vec3_origin, vec3_origin, end, false, ent);
  if (tr.entp && EVars(tr.entn)->takedamage == DAMAGE_AIM &&
      (!Cvar_GetValue(&teamplay) || EVars(ent)->team <= 0 ||
       EVars(ent)->team != EVars(tr.entn)->team)) {
    vec3_t r;
    Pr_global_struct_v_forward(&r[0], &r[1], &r[2]);
    Set_Pr_globalsf(OFS_RETURN, r[0]);
    Set_Pr_globalsf(OFS_RETURN + 1, r[1]);
    Set_Pr_globalsf(OFS_RETURN + 2, r[2]);
    return;
  }

  // try all possible entities
  VectorCopy(dir, bestdir);
  bestdist = Cvar_GetValue(&sv_aim);
  bestent = -1;

  check = 1;
  for (i = 1; i < SV_NumEdicts(); i++, check++) {
    if (EVars(check)->takedamage != DAMAGE_AIM) continue;
    if (check == ent) continue;
    if (Cvar_GetValue(&teamplay) && EVars(ent)->team > 0 &&
        EVars(ent)->team == EVars(check)->team)
      continue;  // don't aim at teammate
    for (j = 0; j < 3; j++)
      end[j] = EVars(check)->origin[j] +
               0.5 * (EVars(check)->mins[j] + EVars(check)->maxs[j]);
    VectorSubtract(end, start, dir);
    VectorNormalize(dir);
    vec3_t vforward;
    Pr_global_struct_v_forward(&vforward[0], &vforward[1], &vforward[2]);
    dist = DotProduct(dir, vforward);
    if (dist < bestdist) continue;  // to far to turn
    tr = SV_Move(start, vec3_origin, vec3_origin, end, false, ent);
    if (tr.entn == check) {  // can shoot at this one
      bestdist = dist;
      bestent = check;
    }
  }

  if (bestent >= 0) {
    VectorSubtract(EVars(bestent)->origin, EVars(ent)->origin, dir);
    vec3_t vforward;
    Pr_global_struct_v_forward(&vforward[0], &vforward[1], &vforward[2]);
    dist = DotProduct(dir, vforward);
    VectorScale(vforward, dist, end);
    end[2] = dir[2];
    VectorNormalize(end);
    Set_Pr_globalsf(OFS_RETURN, end[0]);
    Set_Pr_globalsf(OFS_RETURN + 1, end[1]);
    Set_Pr_globalsf(OFS_RETURN + 2, end[2]);
  } else {
    Set_Pr_globalsf(OFS_RETURN, bestdir[0]);
    Set_Pr_globalsf(OFS_RETURN + 1, bestdir[1]);
    Set_Pr_globalsf(OFS_RETURN + 2, bestdir[2]);
  }
}

//=============================================================================

static void PF_makestatic(void) {
  int ent;
  int i;
  int bits = 0;  // johnfitz -- PROTOCOL_FITZQUAKE

  ent = Pr_globalsi(OFS_PARM0);

  // johnfitz -- don't send invisible static entities
  if (EDICT_NUM(ent)->alpha == ENTALPHA_ZERO) {
    ED_Free(ent);
    return;
  }
  // johnfitz

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (SV_Protocol() == PROTOCOL_NETQUAKE) {
    if (SV_ModelIndex(PR_GetString(EVars(ent)->model)) & 0xFF00 ||
        (int)(EVars(ent)->frame) & 0xFF00) {
      ED_Free(ent);
      return;  // can't display the correct model & frame, so don't show it at
               // all
    }
  } else {
    if (SV_ModelIndex(PR_GetString(EVars(ent)->model)) & 0xFF00)
      bits |= B_LARGEMODEL;
    if ((int)(EVars(ent)->frame) & 0xFF00) bits |= B_LARGEFRAME;
    if (EDICT_NUM(ent)->alpha != ENTALPHA_DEFAULT) bits |= B_ALPHA;
  }

  if (bits) {
    SV_SO_WriteByte(svc_spawnstatic2);
    SV_SO_WriteByte(bits);
  } else
    SV_SO_WriteByte(svc_spawnstatic);

  if (bits & B_LARGEMODEL)
    SV_SO_WriteShort(SV_ModelIndex(PR_GetString(EVars(ent)->model)));
  else
    SV_SO_WriteByte(SV_ModelIndex(PR_GetString(EVars(ent)->model)));

  if (bits & B_LARGEFRAME)
    SV_SO_WriteShort(EVars(ent)->frame);
  else
    SV_SO_WriteByte(EVars(ent)->frame);
  // johnfitz

  SV_SO_WriteByte(EVars(ent)->colormap);
  SV_SO_WriteByte(EVars(ent)->skin);
  for (i = 0; i < 3; i++) {
    SV_SO_WriteCoord(EVars(ent)->origin[i]);
    SV_SO_WriteAngle(EVars(ent)->angles[i]);
  }

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (bits & B_ALPHA) SV_SO_WriteByte(EDICT_NUM(ent)->alpha);
  // johnfitz

  // throw the entity away now
  ED_Free(ent);
}

//=============================================================================

/*
==============
PF_setspawnparms
==============
*/
static void PF_setspawnparms(void) {
  int i;
  int client;

  i = Pr_globalsi(OFS_PARM0);
  if (i < 1 || i > SVS_GetMaxClients()) PR_RunError("Entity is not a client");

  // copy spawn parms out of the client_t
  client = i - 1;

  for (i = 0; i < NUM_SPAWN_PARMS; i++)
    Set_pr_global_struct_parm(i, GetClientSpawnParam(client, i));
}

static void PF_Fixme(void) { PR_RunError("unimplemented builtin"); }

static builtin_t pr_builtin[] = {
    PF_Fixme,
    PF_makevectors,  // void(entity e) makevectors		= #1
    PF_setorigin,    // void(entity e, vector o) setorigin	= #2
    PF_setmodel,     // void(entity e, string m) setmodel	= #3
    PF_setsize,      // void(entity e, vector min, vector max) setsize	= #4
    PF_Fixme,        // void(entity e, vector min, vector max) setabssize	= #5
    PF_break,        // void() break				= #6
    PF_random,       // float() random			= #7
    PF_sound,        // void(entity e, float chan, string samp) sound	= #8
    PF_normalize,    // vector(vector v) normalize		= #9
    PF_error,        // void(string e) error			= #10
    PF_objerror,     // void(string e) objerror		= #11
    PF_vlen,         // float(vector v) vlen			= #12
    PF_vectoyaw,     // float(vector v) vectoyaw		= #13
    PF_Spawn,        // entity() spawn			= #14
    PF_Remove,       // void(entity e) remove		= #15
    PF_traceline,    // float(vector v1, vector v2, float tryents) traceline
                     // =
                     // #16
    PF_checkclient,  // entity() clientlist			= #17
    PF_Find,  // entity(entity start, .string fld, string match) find	= #18
    PF_precache_sound,  // void(string s) precache_sound	= #19
    PF_precache_model,  // void(string s) precache_model	= #20
    PF_stuffcmd,        // void(entity client, string s)stuffcmd	= #21
    PF_findradius,      // entity(vector org, float rad) findradius	= #22
    PF_bprint,          // void(string s) bprint		= #23
    PF_sprint,          // void(entity client, string s) sprint	= #24
    PF_dprint,          // void(string s) dprint		= #25
    PF_ftos,            // void(string s) ftos			= #26
    PF_vtos,            // void(string s) vtos			= #27
    PF_coredump, PF_traceon, PF_traceoff,
    PF_eprint,    // void(entity e) debug print an entire entity
    PF_walkmove,  // float(float yaw, float dist) walkmove
    PF_Fixme,     // float(float yaw, float dist) walkmove
    PF_droptofloor, PF_lightstyle, PF_rint, PF_floor, PF_ceil, PF_Fixme,
    PF_checkbottom, PF_pointcontents, PF_Fixme, PF_fabs, PF_aim, PF_cvar,
    PF_localcmd, PF_nextent, PF_particle, PF_changeyaw, PF_Fixme,
    PF_vectoangles,

    PF_WriteByte, PF_WriteChar, PF_WriteShort, PF_WriteLong, PF_WriteCoord,
    PF_WriteAngle, PF_WriteString, PF_WriteEntity,

    PF_Fixme, PF_Fixme, PF_Fixme, PF_Fixme, PF_Fixme, PF_Fixme, PF_Fixme,

    SV_MoveToGoal, PF_precache_file, PF_makestatic,

    PF_changelevel, PF_Fixme,

    PF_cvar_set, PF_centerprint,

    PF_ambientsound,

    PF_precache_model,
    PF_precache_sound,  // precache_sound2 is different only for qcc
    PF_precache_file,

    PF_setspawnparms};

builtin_t *pr_builtins = pr_builtin;
int pr_numbuiltins = sizeof(pr_builtin) / sizeof(pr_builtin[0]);
