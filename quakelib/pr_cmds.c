#include "quakedef.h"

#define RETURN_EDICT(e) (Set_Pr_globalsi(OFS_RETURN, e))

/*
===============================================================================

        BUILT-IN FUNCTIONS

===============================================================================
*/

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
    RETURN_EDICT(0);
    return;
  }

  // might be able to see it
  RETURN_EDICT(ent);
}

static void PR_CheckEmptyString(const char *s) {
  if (s[0] <= ' ') PR_RunError("Bad string");
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

static void PF_traceon(void) { pr_trace = true; }

static void PF_traceoff(void) { pr_trace = false; }

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

//=============================================================================

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
