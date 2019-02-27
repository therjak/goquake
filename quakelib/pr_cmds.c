#include "quakedef.h"

#define STRINGTEMP_BUFFERS 16
#define STRINGTEMP_LENGTH 1024
static char pr_string_temp[STRINGTEMP_BUFFERS][STRINGTEMP_LENGTH];
static byte pr_string_tempindex = 0;

static char *PR_GetTempString(void) {
  return pr_string_temp[(STRINGTEMP_BUFFERS - 1) & ++pr_string_tempindex];
}

#define RETURN_EDICT(e) (Set_Pr_globalsi(OFS_RETURN, NUM_FOR_EDICT(e)))

#define MSG_BROADCAST 0  // unreliable to all
#define MSG_ONE 1        // reliable to one (msg_entity)
#define MSG_ALL 2        // reliable to all
#define MSG_INIT 3       // write to the init string

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
  edict_t *ed;

  s = PF_VarString(0);
  Con_Printf("======SERVER ERROR in %s:\n%s\n",
             PR_GetString(pr_xfunction->s_name), s);
  ed = EDICT_NUM(Pr_global_struct_self());
  ED_Print(ed);

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
  edict_t *ed;

  s = PF_VarString(0);
  Con_Printf("======OBJECT ERROR in %s:\n%s\n",
             PR_GetString(pr_xfunction->s_name), s);
  ed = EDICT_NUM(Pr_global_struct_self());
  ED_Print(ed);
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

/*
=================
PF_setorigin

This is the only valid way to move an object without using the physics
of the world (setting velocity and waiting).  Directly changing origin
will not set internal links correctly, so clipping would be messed up.

This should be called when an object is spawned, and then only if it is
teleported.

setorigin (entity, origin)
=================
*/
static void PF_setorigin(void) {
  edict_t *e = EDICT_NUM(Pr_globalsi(OFS_PARM0));
  float *origin = EdictV(e)->origin;

  *(origin) = Pr_globalsf(OFS_PARM1);
  *(origin + 1) = Pr_globalsf(OFS_PARM1 + 1);
  *(origin + 2) = Pr_globalsf(OFS_PARM1 + 2);

  SV_LinkEdict(e, false);
}

static void SetMinMaxSize(edict_t *e, float *minvec, float *maxvec,
                          qboolean rotate) {
  float *angles;
  vec3_t rmin, rmax;
  float bounds[2][3];
  float xvector[2], yvector[2];
  float a;
  vec3_t base, transformed;
  int i, j, k, l;

  for (i = 0; i < 3; i++)
    if (minvec[i] > maxvec[i]) PR_RunError("backwards mins/maxs");

  rotate = false;  // FIXME: implement rotation properly again

  if (!rotate) {
    VectorCopy(minvec, rmin);
    VectorCopy(maxvec, rmax);
  } else {
    // find min / max for rotations
    angles = EdictV(e)->angles;

    a = angles[1] / 180 * M_PI;

    xvector[0] = cos(a);
    xvector[1] = sin(a);
    yvector[0] = -sin(a);
    yvector[1] = cos(a);

    VectorCopy(minvec, bounds[0]);
    VectorCopy(maxvec, bounds[1]);

    rmin[0] = rmin[1] = rmin[2] = 9999;
    rmax[0] = rmax[1] = rmax[2] = -9999;

    for (i = 0; i <= 1; i++) {
      base[0] = bounds[i][0];
      for (j = 0; j <= 1; j++) {
        base[1] = bounds[j][1];
        for (k = 0; k <= 1; k++) {
          base[2] = bounds[k][2];

          // transform the point
          transformed[0] = xvector[0] * base[0] + yvector[0] * base[1];
          transformed[1] = xvector[1] * base[0] + yvector[1] * base[1];
          transformed[2] = base[2];

          for (l = 0; l < 3; l++) {
            if (transformed[l] < rmin[l]) rmin[l] = transformed[l];
            if (transformed[l] > rmax[l]) rmax[l] = transformed[l];
          }
        }
      }
    }
  }

  // set derived values
  VectorCopy(rmin, EdictV(e)->mins);
  VectorCopy(rmax, EdictV(e)->maxs);
  VectorSubtract(maxvec, minvec, EdictV(e)->size);

  SV_LinkEdict(e, false);
}

/*
=================
PF_setsize

the size box is rotated by the current angle

setsize (entity, minvector, maxvector)
=================
*/
static void PF_setsize(void) {
  edict_t *e;
  vec3_t minvec, maxvec;

  e = EDICT_NUM(Pr_globalsi(OFS_PARM0));
  minvec[0] = Pr_globalsf(OFS_PARM1);
  minvec[1] = Pr_globalsf(OFS_PARM1 + 1);
  minvec[2] = Pr_globalsf(OFS_PARM1 + 2);
  maxvec[0] = Pr_globalsf(OFS_PARM2);
  maxvec[1] = Pr_globalsf(OFS_PARM2 + 1);
  maxvec[2] = Pr_globalsf(OFS_PARM2 + 2);
  SetMinMaxSize(e, minvec, maxvec, false);
}

/*
=================
PF_setmodel

setmodel(entity, model)
=================
*/
static void PF_setmodel(void) {
  int i;
  int mi;
  const char *m, **check;
  qmodel_t *mod;
  edict_t *e;

  e = EDICT_NUM(Pr_globalsi(OFS_PARM0));
  mi = Pr_globalsi(OFS_PARM1);
  m = PR_GetString(mi);

  // check to see if model was properly precached
  i = ElementOfSVModelPrecache(m);
  if (i == -1) {
    PR_RunError("no precache: %s", m);
  }
  EdictV(e)->model = mi;
  EdictV(e)->modelindex = i;  // SV_ModelIndex (m);

  mod = sv.models[(int)EdictV(e)->modelindex];  // Mod_ForName (m, true);

  if (mod)
  // johnfitz -- correct physics cullboxes for bmodels
  {
    if (mod->type == mod_brush)
      SetMinMaxSize(e, mod->clipmins, mod->clipmaxs, true);
    else
      SetMinMaxSize(e, mod->mins, mod->maxs, true);
  }
  // johnfitz
  else
    SetMinMaxSize(e, vec3_origin, vec3_origin, true);
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

/*
=================
PF_normalize

vector normalize(vector)
=================
*/
static void PF_normalize(void) {
  vec3_t newvalue;
  float new_temp;
  float x, y, z;

  x = Pr_globalsf(OFS_PARM0);
  new_temp = x * x;
  y = Pr_globalsf(OFS_PARM0 + 1);
  new_temp += y * y;
  z = Pr_globalsf(OFS_PARM0 + 2);
  new_temp += z * z;
  new_temp = sqrt(new_temp);

  if (new_temp == 0)
    newvalue[0] = newvalue[1] = newvalue[2] = 0;
  else {
    new_temp = 1 / new_temp;
    newvalue[0] = x * new_temp;
    newvalue[1] = y * new_temp;
    newvalue[2] = z * new_temp;
  }

  Set_Pr_globalsf(OFS_RETURN, newvalue[0]);
  Set_Pr_globalsf(OFS_RETURN + 1, newvalue[1]);
  Set_Pr_globalsf(OFS_RETURN + 2, newvalue[2]);
}

/*
=================
PF_vlen

scalar vlen(vector)
=================
*/
static void PF_vlen(void) {
  float new_temp;
  float t;

  t = Pr_globalsf(OFS_PARM0);
  new_temp = t * t;
  t = Pr_globalsf(OFS_PARM0 + 1);
  new_temp += t * t;
  t = Pr_globalsf(OFS_PARM0 + 2);
  new_temp += t * t;
  new_temp = sqrt(new_temp);

  Set_Pr_globalsf(OFS_RETURN, new_temp);
}

/*
=================
PF_vectoyaw

float vectoyaw(vector)
=================
*/
static void PF_vectoyaw(void) {
  float yaw;
  float x = Pr_globalsf(OFS_PARM0);
  float y = Pr_globalsf(OFS_PARM0 + 1);

  if (y == 0 && x == 0)
    yaw = 0;
  else {
    yaw = (int)(atan2(y, x) * 180 / M_PI);
    if (yaw < 0) yaw += 360;
  }

  Set_Pr_globalsf(OFS_RETURN, yaw);
}

/*
=================
PF_vectoangles

vector vectoangles(vector)
=================
*/
static void PF_vectoangles(void) {
  float forward;
  float yaw, pitch;

  float x = Pr_globalsf(OFS_PARM0);
  float y = Pr_globalsf(OFS_PARM0 + 1);
  float z = Pr_globalsf(OFS_PARM0 + 2);

  if (y == 0 && x == 0) {
    yaw = 0;
    if (z > 0)
      pitch = 90;
    else
      pitch = 270;
  } else {
    yaw = (int)(atan2(y, x) * 180 / M_PI);
    if (yaw < 0) yaw += 360;

    forward = sqrt(x * x + y * y);
    pitch = (int)(atan2(z, forward) * 180 / M_PI);
    if (pitch < 0) pitch += 360;
  }

  Set_Pr_globalsf(OFS_RETURN + 0, pitch);
  Set_Pr_globalsf(OFS_RETURN + 1, yaw);
  Set_Pr_globalsf(OFS_RETURN + 2, 0);
}

/*
=================
PF_Random

Returns a number from 0 <= num < 1

random()
=================
*/
static void PF_random(void) {
  float num;

  num = (rand() & 0x7fff) / ((float)0x7fff);

  Set_Pr_globalsf(OFS_RETURN, num);
}

/*
=================
PF_particle

particle(origin, color, count)
=================
*/
static void PF_particle(void) {
  float color;
  float count;
  vec3_t org, dir;

  org[0] = Pr_globalsf(OFS_PARM0);
  org[1] = Pr_globalsf(OFS_PARM0 + 1);
  org[2] = Pr_globalsf(OFS_PARM0 + 2);
  dir[0] = Pr_globalsf(OFS_PARM1);
  dir[1] = Pr_globalsf(OFS_PARM1 + 1);
  dir[2] = Pr_globalsf(OFS_PARM1 + 2);
  color = Pr_globalsf(OFS_PARM2);
  count = Pr_globalsf(OFS_PARM3);
  SV_StartParticle(org, dir, color, count);
}

/*
=================
PF_ambientsound

=================
*/
static void PF_ambientsound(void) {
  const char *samp, **check;
  vec3_t pos;
  float vol, attenuation;
  int i, soundnum;
  int large = false;  // johnfitz -- PROTOCOL_FITZQUAKE

  pos[0] = Pr_globalsf(OFS_PARM0);
  pos[1] = Pr_globalsf(OFS_PARM0 + 1);
  pos[2] = Pr_globalsf(OFS_PARM0 + 2);
  samp = PR_GetString(Pr_globalsi(OFS_PARM1));
  vol = Pr_globalsf(OFS_PARM2);
  attenuation = Pr_globalsf(OFS_PARM3);

  // check to see if samp was properly precached
  soundnum = ElementOfSVSoundPrecache(samp);

  if (soundnum == -1) {
    Con_Printf("no precache: %s\n", samp);
    return;
  }

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (soundnum > 255) {
    if (SV_Protocol() == PROTOCOL_NETQUAKE)
      return;  // don't send any info protocol can't support
    else
      large = true;
  }
  // johnfitz

  // add an svc_spawnambient command to the level signon packet

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (large)
    SV_SO_WriteByte(svc_spawnstaticsound2);
  else
    SV_SO_WriteByte(svc_spawnstaticsound);
  // johnfitz

  for (i = 0; i < 3; i++) SV_SO_WriteCoord(pos[i]);

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (large)
    SV_SO_WriteShort(soundnum);
  else
    SV_SO_WriteByte(soundnum);
  // johnfitz

  SV_SO_WriteByte(vol * 255);
  SV_SO_WriteByte(attenuation * 64);
}

/*
=================
PF_sound

Each entity can have eight independant sound sources, like voice,
weapon, feet, etc.

Channel 0 is an auto-allocate channel, the others override anything
already running on that entity/channel pair.

An attenuation of 0 will play full volume everywhere in the level.
Larger attenuations will drop off.

=================
*/
static void PF_sound(void) {
  const char *sample;
  int channel;
  edict_t *entity;
  int volume;
  float attenuation;

  entity = EDICT_NUM(Pr_globalsi(OFS_PARM0));
  channel = Pr_globalsf(OFS_PARM1);
  sample = PR_GetString(Pr_globalsi(OFS_PARM2));
  volume = Pr_globalsf(OFS_PARM3) * 255;
  attenuation = Pr_globalsf(OFS_PARM4);

  if (volume < 0 || volume > 255)
    Host_Error("SV_StartSound: volume = %i", volume);

  if (attenuation < 0 || attenuation > 4)
    Host_Error("SV_StartSound: attenuation = %f", attenuation);

  if (channel < 0 || channel > 7)
    Host_Error("SV_StartSound: channel = %i", channel);

  SV_StartSound(entity, channel, sample, volume, attenuation);
}

/*
=================
PF_break

break()
=================
*/
static void PF_break(void) {
  Con_Printf("break statement\n");
  *(int *)-4 = 0;  // dump to debugger
  //	PR_RunError ("break statement");
}

/*
=================
PF_traceline

Used for use tracing and shot targeting
Traces are blocked by bbox and exact bsp entityes, and also slide box entities
if the tryents flag is set.

traceline (vector1, vector2, tryents)
=================
*/
static void PF_traceline(void) {
  vec3_t v1, v2;
  trace_t trace;
  int nomonsters;
  edict_t *ent;

  v1[0] = Pr_globalsf(OFS_PARM0);
  v1[1] = Pr_globalsf(OFS_PARM0 + 1);
  v1[2] = Pr_globalsf(OFS_PARM0 + 2);
  v2[0] = Pr_globalsf(OFS_PARM1);
  v2[1] = Pr_globalsf(OFS_PARM1 + 1);
  v2[2] = Pr_globalsf(OFS_PARM1 + 2);
  nomonsters = Pr_globalsf(OFS_PARM2);
  ent = EDICT_NUM(Pr_globalsi(OFS_PARM3));

  /* FIXME FIXME FIXME: Why do we hit this with certain progs.dat ?? */
  if (Cvar_GetValue(&developer)) {
    if (IS_NAN(v1[0]) || IS_NAN(v1[1]) || IS_NAN(v1[2]) || IS_NAN(v2[0]) ||
        IS_NAN(v2[1]) || IS_NAN(v2[2])) {
      Con_Warning("NAN in traceline:\nv1(%f %f %f) v2(%f %f %f)\nentity %d\n",
                  v1[0], v1[1], v1[2], v2[0], v2[1], v2[2], NUM_FOR_EDICT(ent));
    }
  }

  if (IS_NAN(v1[0]) || IS_NAN(v1[1]) || IS_NAN(v1[2]))
    v1[0] = v1[1] = v1[2] = 0;
  if (IS_NAN(v2[0]) || IS_NAN(v2[1]) || IS_NAN(v2[2]))
    v2[0] = v2[1] = v2[2] = 0;

  trace = SV_Move(v1, vec3_origin, vec3_origin, v2, nomonsters, ent);

  Set_pr_global_struct_trace_allsolid(trace.allsolid);
  Set_pr_global_struct_trace_startsolid(trace.startsolid);
  Set_pr_global_struct_trace_fraction(trace.fraction);
  Set_pr_global_struct_trace_inwater(trace.inwater);
  Set_pr_global_struct_trace_inopen(trace.inopen);
  Set_pr_global_struct_trace_endpos(trace.endpos[0], trace.endpos[1],
                                    trace.endpos[2]);
  Set_pr_global_struct_trace_plane_normal(
      trace.plane.normal[0], trace.plane.normal[1], trace.plane.normal[2]);
  Set_pr_global_struct_trace_plane_dist(trace.plane.dist);
  if (trace.ent)
    Set_pr_global_struct_trace_ent(NUM_FOR_EDICT(trace.ent));
  else
    Set_pr_global_struct_trace_ent(NUM_FOR_EDICT(sv.edicts));
}

/*
=================
PF_checkpos

Returns true if the given entity can move to the given position from it's
current position by walking or rolling.
FIXME: make work...
scalar checkpos (entity, vector)
=================
*/
#if 0
static void PF_checkpos (void)
{
}
#endif

//============================================================================

static byte checkpvs[MAX_MAP_LEAFS / 8];

static int PF_newcheckclient(int check) {
  int i;
  byte *pvs;
  edict_t *ent;
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

    ent = EDICT_NUM(i);

    if (i == check) break;  // didn't find anything else

    if (ent->free) continue;
    if (EdictV(ent)->health <= 0) continue;
    if ((int)EdictV(ent)->flags & FL_NOTARGET) continue;

    // anything that is a client, or has a client as an enemy
    break;
  }

  // get the PVS for the entity
  VectorAdd(EdictV(ent)->origin, EdictV(ent)->view_ofs, org);
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
  edict_t *ent, *self;
  mleaf_t *leaf;
  int l;
  vec3_t view;

  // find a new check if on a new frame
  if (SV_Time() - SV_LastCheckTime() >= 0.1) {
    SV_SetLastCheck(PF_newcheckclient(SV_LastCheck()));
    SV_SetLastCheckTime(SV_Time());
  }

  // return check if it might be visible
  ent = EDICT_NUM(SV_LastCheck());
  if (ent->free || EdictV(ent)->health <= 0) {
    RETURN_EDICT(sv.edicts);
    return;
  }

  // if current entity can't possibly see the check entity, return 0
  self = EDICT_NUM(Pr_global_struct_self());
  VectorAdd(EdictV(self)->origin, EdictV(self)->view_ofs, view);
  leaf = Mod_PointInLeaf(view, sv.worldmodel);
  l = (leaf - sv.worldmodel->leafs) - 1;
  if ((l < 0) || !(checkpvs[l >> 3] & (1 << (l & 7)))) {
    c_notvis++;
    RETURN_EDICT(sv.edicts);
    return;
  }

  // might be able to see it
  c_invis++;
  RETURN_EDICT(ent);
}

//============================================================================

/*
=================
PF_stuffcmd

Sends text over to the client's execution buffer

stuffcmd (clientent, value)
=================
*/
static void PF_stuffcmd(void) {
  int entnum;
  const char *str;

  entnum = Pr_globalsi(OFS_PARM0);
  if (entnum < 1 || entnum > SVS_GetMaxClients()) {
    PR_RunError("Parm 0 not a client");
  }
  str = PR_GetString(Pr_globalsi(OFS_PARM1));

  Host_ClientCommands(entnum - 1, str);
}

/*
=================
PF_localcmd

Sends text over to the client's execution buffer

localcmd (string)
=================
*/
static void PF_localcmd(void) {
  const char *str;

  str = PR_GetString(Pr_globalsi(OFS_PARM0));
  Cbuf_AddText(str);
}

/*
=================
PF_cvar

float cvar (string)
=================
*/
static void PF_cvar(void) {
  const char *str;

  str = PR_GetString(Pr_globalsi(OFS_PARM0));

  Set_Pr_globalsf(OFS_RETURN, Cvar_VariableValue(str));
}

/*
=================
PF_cvar_set

float cvar (string)
=================
*/
static void PF_cvar_set(void) {
  const char *var, *val;

  var = PR_GetString(Pr_globalsi(OFS_PARM0));
  val = PR_GetString(Pr_globalsi(OFS_PARM1));

  Cvar_Set(var, val);
}

/*
=================
PF_findradius

Returns a chain of entities that have origins within a spherical area

findradius (origin, radius)
=================
*/
static void PF_findradius(void) {
  edict_t *ent, *chain;
  float rad;
  vec3_t org;
  vec3_t eorg;
  int i, j;

  chain = (edict_t *)sv.edicts;

  org[0] = Pr_globalsf(OFS_PARM0);
  org[1] = Pr_globalsf(OFS_PARM0 + 1);
  org[2] = Pr_globalsf(OFS_PARM0 + 2);
  rad = Pr_globalsf(OFS_PARM1);

  ent = NEXT_EDICT(sv.edicts);
  for (i = 1; i < SV_NumEdicts(); i++, ent = NEXT_EDICT(ent)) {
    if (ent->free) continue;
    if (EdictV(ent)->solid == SOLID_NOT) continue;
    for (j = 0; j < 3; j++)
      eorg[j] = org[j] - (EdictV(ent)->origin[j] +
                          (EdictV(ent)->mins[j] + EdictV(ent)->maxs[j]) * 0.5);
    if (VectorLength(eorg) > rad) continue;

    EdictV(ent)->chain = NUM_FOR_EDICT(chain);
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

static void PF_fabs(void) {
  float v;
  v = Pr_globalsf(OFS_PARM0);
  Set_Pr_globalsf(OFS_RETURN, fabs(v));
}

static void PF_vtos(void) {
  char *s;

  s = PR_GetTempString();
  sprintf(s, "'%5.1f %5.1f %5.1f'", Pr_globalsf(OFS_PARM0),
          Pr_globalsf(OFS_PARM0 + 1), Pr_globalsf(OFS_PARM0 + 2));
  Set_Pr_globalsi(OFS_RETURN, PR_SetEngineString(s));
}

static void PF_Spawn(void) {
  edict_t *ed;

  ed = ED_Alloc();

  RETURN_EDICT(ed);
}

static void PF_Remove(void) {
  edict_t *ed;

  ed = EDICT_NUM(Pr_globalsi(OFS_PARM0));
  ED_Free(ed);
}

// entity (entity start, .string field, string match) find = #5;
static void PF_Find(void) {
  int e;
  int f;
  const char *s, *t;
  edict_t *ed;

  e = Pr_globalsi(OFS_PARM0);
  f = Pr_globalsi(OFS_PARM1);
  s = PR_GetString(Pr_globalsi(OFS_PARM2));
  if (!s) PR_RunError("PF_Find: bad search string");

  for (e++; e < SV_NumEdicts(); e++) {
    ed = EDICT_NUM(e);
    if (ed->free) continue;
    t = (PR_GetString(*(GoInt32 *)&((float *)EdictV(ed))[f]));
    if (!t) continue;
    s = PR_GetString(Pr_globalsi(OFS_PARM2));
    if (!strcmp(t, s)) {
      RETURN_EDICT(ed);
      return;
    }
  }

  RETURN_EDICT(sv.edicts);
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
  edict_t *ent;
  float yaw, dist;
  vec3_t move;
  dfunction_t *oldf;
  int oldself;

  ent = EDICT_NUM(Pr_global_struct_self());
  yaw = Pr_globalsf(OFS_PARM0);
  dist = Pr_globalsf(OFS_PARM1);

  if (!((int)EdictV(ent)->flags & (FL_ONGROUND | FL_FLY | FL_SWIM))) {
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
PF_droptofloor

void() droptofloor
===============
*/
static void PF_droptofloor(void) {
  edict_t *ent;
  vec3_t end;
  trace_t trace;

  ent = EDICT_NUM(Pr_global_struct_self());

  VectorCopy(EdictV(ent)->origin, end);
  end[2] -= 256;

  trace = SV_Move(EdictV(ent)->origin, EdictV(ent)->mins, EdictV(ent)->maxs,
                  end, false, ent);

  if (trace.fraction == 1 || trace.allsolid)
    Set_Pr_globalsf(OFS_RETURN, 0);
  else {
    VectorCopy(trace.endpos, EdictV(ent)->origin);
    SV_LinkEdict(ent, false);
    EdictV(ent)->flags = (int)EdictV(ent)->flags | FL_ONGROUND;
    EdictV(ent)->groundentity = NUM_FOR_EDICT(trace.ent);
    Set_Pr_globalsf(OFS_RETURN, 1);
  }
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

static void PF_rint(void) {
  float f;
  f = Pr_globalsf(OFS_PARM0);
  if (f > 0)
    Set_Pr_globalsf(OFS_RETURN, (int)(f + 0.5));
  else
    Set_Pr_globalsf(OFS_RETURN, (int)(f - 0.5));
}

static void PF_floor(void) {
  Set_Pr_globalsf(OFS_RETURN, floor(Pr_globalsf(OFS_PARM0)));
}

static void PF_ceil(void) {
  Set_Pr_globalsf(OFS_RETURN, ceil(Pr_globalsf(OFS_PARM0)));
}

/*
=============
PF_checkbottom
=============
*/
static void PF_checkbottom(void) {
  edict_t *ent;

  ent = EDICT_NUM(Pr_globalsi(OFS_PARM0));

  Set_Pr_globalsf(OFS_RETURN, SV_CheckBottom(ent));
}

/*
=============
PF_pointcontents
=============
*/
static void PF_pointcontents(void) {
  vec3_t v;

  v[0] = Pr_globalsf(OFS_PARM0);
  v[1] = Pr_globalsf(OFS_PARM0 + 1);
  v[2] = Pr_globalsf(OFS_PARM0 + 2);

  Set_Pr_globalsf(OFS_RETURN, SV_PointContents(v));
}

/*
=============
PF_nextent

entity nextent(entity)
=============
*/
static void PF_nextent(void) {
  int i;
  edict_t *ent;

  i = Pr_globalsi(OFS_PARM0);
  while (1) {
    i++;
    if (i == SV_NumEdicts()) {
      RETURN_EDICT(sv.edicts);
      return;
    }
    ent = EDICT_NUM(i);
    if (!ent->free) {
      RETURN_EDICT(ent);
      return;
    }
  }
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
  edict_t *ent, *check, *bestent;
  vec3_t start, dir, end, bestdir;
  int i, j;
  trace_t tr;
  float dist, bestdist;
  float speed;

  ent = EDICT_NUM(Pr_globalsi(OFS_PARM0));
  speed = Pr_globalsf(OFS_PARM1);
  (void)speed; /* variable set but not used */

  VectorCopy(EdictV(ent)->origin, start);
  start[2] += 20;

  // try sending a trace straight
  Pr_global_struct_v_forward(&dir[0], &dir[1], &dir[2]);
  VectorMA(start, 2048, dir, end);
  tr = SV_Move(start, vec3_origin, vec3_origin, end, false, ent);
  if (tr.ent && EdictV(tr.ent)->takedamage == DAMAGE_AIM &&
      (!Cvar_GetValue(&teamplay) || EdictV(ent)->team <= 0 ||
       EdictV(ent)->team != EdictV(tr.ent)->team)) {
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
  bestent = NULL;

  check = NEXT_EDICT(sv.edicts);
  for (i = 1; i < SV_NumEdicts(); i++, check = NEXT_EDICT(check)) {
    if (EdictV(check)->takedamage != DAMAGE_AIM) continue;
    if (check == ent) continue;
    if (Cvar_GetValue(&teamplay) && EdictV(ent)->team > 0 &&
        EdictV(ent)->team == EdictV(check)->team)
      continue;  // don't aim at teammate
    for (j = 0; j < 3; j++)
      end[j] = EdictV(check)->origin[j] +
               0.5 * (EdictV(check)->mins[j] + EdictV(check)->maxs[j]);
    VectorSubtract(end, start, dir);
    VectorNormalize(dir);
    vec3_t vforward;
    Pr_global_struct_v_forward(&vforward[0], &vforward[1], &vforward[2]);
    dist = DotProduct(dir, vforward);
    if (dist < bestdist) continue;  // to far to turn
    tr = SV_Move(start, vec3_origin, vec3_origin, end, false, ent);
    if (tr.ent == check) {  // can shoot at this one
      bestdist = dist;
      bestent = check;
    }
  }

  if (bestent) {
    VectorSubtract(EdictV(bestent)->origin, EdictV(ent)->origin, dir);
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

/*
==============
PF_changeyaw

This was a major timewaster in progs, so it was converted to C
==============
*/
void PF_changeyaw(void) {
  edict_t *ent;
  float ideal, current, move, speed;

  ent = EDICT_NUM(Pr_global_struct_self());
  current = anglemod(EdictV(ent)->angles[1]);
  ideal = EdictV(ent)->ideal_yaw;
  speed = EdictV(ent)->yaw_speed;

  if (current == ideal) return;
  move = ideal - current;
  if (ideal > current) {
    if (move >= 180) move = move - 360;
  } else {
    if (move <= -180) move = move + 360;
  }
  if (move > 0) {
    if (move > speed) move = speed;
  } else {
    if (move < -speed) move = -speed;
  }

  EdictV(ent)->angles[1] = anglemod(current + move);
}

/*
===============================================================================

MESSAGE WRITING

===============================================================================
*/

static int WriteClient() {
  int entnum;
  edict_t *ent;

  ent = EDICT_NUM(Pr_global_struct_msg_entity());
  entnum = NUM_FOR_EDICT(ent);
  if (entnum < 1 || entnum > SVS_GetMaxClients())
    PR_RunError("WriteDest: not a client");
  return entnum - 1;
}

static void PF_WriteByte(void) {
  int dest = Pr_globalsf(OFS_PARM0);
  float msg = Pr_globalsf(OFS_PARM1);
  if (dest == MSG_ONE) {
    ClientWriteByte(WriteClient(), msg);
  } else if (dest == MSG_INIT) {
    SV_SO_WriteByte(msg);
  } else if (dest == MSG_BROADCAST) {
    SV_DG_WriteByte(msg);
  } else if (dest == MSG_ALL) {
    SV_RD_WriteByte(msg);
  } else {
    PR_RunError("WriteDest: bad destination");
  }
}

static void PF_WriteChar(void) {
  int dest = Pr_globalsf(OFS_PARM0);
  float msg = Pr_globalsf(OFS_PARM1);
  if (dest == MSG_ONE) {
    ClientWriteChar(WriteClient(), msg);
  } else if (dest == MSG_INIT) {
    SV_SO_WriteChar(msg);
  } else if (dest == MSG_BROADCAST) {
    SV_DG_WriteChar(msg);
  } else if (dest == MSG_ALL) {
    SV_RD_WriteChar(msg);
  } else {
    PR_RunError("WriteDest: bad destination");
  }
}

static void PF_WriteShort(void) {
  int dest = Pr_globalsf(OFS_PARM0);
  float msg = Pr_globalsf(OFS_PARM1);
  if (dest == MSG_ONE) {
    ClientWriteShort(WriteClient(), msg);
  } else if (dest == MSG_INIT) {
    SV_SO_WriteShort(msg);
  } else if (dest == MSG_BROADCAST) {
    SV_DG_WriteShort(msg);
  } else if (dest == MSG_ALL) {
    SV_RD_WriteShort(msg);
  } else {
    PR_RunError("WriteDest: bad destination");
  }
}

static void PF_WriteLong(void) {
  int dest = Pr_globalsf(OFS_PARM0);
  float msg = Pr_globalsf(OFS_PARM1);
  if (dest == MSG_ONE) {
    ClientWriteLong(WriteClient(), msg);
  } else if (dest == MSG_INIT) {
    SV_SO_WriteLong(msg);
  } else if (dest == MSG_BROADCAST) {
    SV_DG_WriteLong(msg);
  } else if (dest == MSG_ALL) {
    SV_RD_WriteLong(msg);
  } else {
    PR_RunError("WriteDest: bad destination");
  }
}

static void PF_WriteAngle(void) {
  int dest = Pr_globalsf(OFS_PARM0);
  float msg = Pr_globalsf(OFS_PARM1);
  if (dest == MSG_ONE) {
    ClientWriteAngle(WriteClient(), msg);
  } else if (dest == MSG_INIT) {
    SV_SO_WriteAngle(msg);
  } else if (dest == MSG_BROADCAST) {
    SV_DG_WriteAngle(msg);
  } else if (dest == MSG_ALL) {
    SV_RD_WriteAngle(msg);
  } else {
    PR_RunError("WriteDest: bad destination");
  }
}

static void PF_WriteCoord(void) {
  int dest = Pr_globalsf(OFS_PARM0);
  float msg = Pr_globalsf(OFS_PARM1);
  if (dest == MSG_ONE) {
    ClientWriteCoord(WriteClient(), msg);
  } else if (dest == MSG_INIT) {
    SV_SO_WriteCoord(msg);
  } else if (dest == MSG_BROADCAST) {
    SV_DG_WriteCoord(msg);
  } else if (dest == MSG_ALL) {
    SV_RD_WriteCoord(msg);
  } else {
    PR_RunError("WriteDest: bad destination");
  }
}

static void PF_WriteString(void) {
  int dest = Pr_globalsf(OFS_PARM0);
  const char *msg = PR_GetString(Pr_globalsi(OFS_PARM1));
  if (dest == MSG_ONE) {
    ClientWriteString(WriteClient(), msg);
  } else if (dest == MSG_INIT) {
    SV_SO_WriteString(msg);
  } else if (dest == MSG_BROADCAST) {
    SV_DG_WriteString(msg);
  } else if (dest == MSG_ALL) {
    SV_RD_WriteString(msg);
  } else {
    PR_RunError("WriteDest: bad destination");
  }
}

static void PF_WriteEntity(void) {
  int dest = Pr_globalsf(OFS_PARM0);
  float msg = Pr_globalsi(OFS_PARM1);
  if (dest == MSG_ONE) {
    ClientWriteShort(WriteClient(), msg);
  } else if (dest == MSG_INIT) {
    SV_SO_WriteShort(msg);
  } else if (dest == MSG_BROADCAST) {
    SV_DG_WriteShort(msg);
  } else if (dest == MSG_ALL) {
    SV_RD_WriteShort(msg);
  } else {
    PR_RunError("WriteDest: bad destination");
  }
}

//=============================================================================

static void PF_makestatic(void) {
  edict_t *ent;
  int i;
  int bits = 0;  // johnfitz -- PROTOCOL_FITZQUAKE

  ent = EDICT_NUM(Pr_globalsi(OFS_PARM0));

  // johnfitz -- don't send invisible static entities
  if (ent->alpha == ENTALPHA_ZERO) {
    ED_Free(ent);
    return;
  }
  // johnfitz

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (SV_Protocol() == PROTOCOL_NETQUAKE) {
    if (SV_ModelIndex(PR_GetString(EdictV(ent)->model)) & 0xFF00 ||
        (int)(EdictV(ent)->frame) & 0xFF00) {
      ED_Free(ent);
      return;  // can't display the correct model & frame, so don't show it at
               // all
    }
  } else {
    if (SV_ModelIndex(PR_GetString(EdictV(ent)->model)) & 0xFF00)
      bits |= B_LARGEMODEL;
    if ((int)(EdictV(ent)->frame) & 0xFF00) bits |= B_LARGEFRAME;
    if (ent->alpha != ENTALPHA_DEFAULT) bits |= B_ALPHA;
  }

  if (bits) {
    SV_SO_WriteByte(svc_spawnstatic2);
    SV_SO_WriteByte(bits);
  } else
    SV_SO_WriteByte(svc_spawnstatic);

  if (bits & B_LARGEMODEL)
    SV_SO_WriteShort(SV_ModelIndex(PR_GetString(EdictV(ent)->model)));
  else
    SV_SO_WriteByte(SV_ModelIndex(PR_GetString(EdictV(ent)->model)));

  if (bits & B_LARGEFRAME)
    SV_SO_WriteShort(EdictV(ent)->frame);
  else
    SV_SO_WriteByte(EdictV(ent)->frame);
  // johnfitz

  SV_SO_WriteByte(EdictV(ent)->colormap);
  SV_SO_WriteByte(EdictV(ent)->skin);
  for (i = 0; i < 3; i++) {
    SV_SO_WriteCoord(EdictV(ent)->origin[i]);
    SV_SO_WriteAngle(EdictV(ent)->angles[i]);
  }

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (bits & B_ALPHA) SV_SO_WriteByte(ent->alpha);
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
  edict_t *ent;
  int i;
  int client;

  ent = EDICT_NUM(Pr_globalsi(OFS_PARM0));
  i = NUM_FOR_EDICT(ent);
  if (i < 1 || i > SVS_GetMaxClients()) PR_RunError("Entity is not a client");

  // copy spawn parms out of the client_t
  client = i - 1;

  for (i = 0; i < NUM_SPAWN_PARMS; i++)
    Set_pr_global_struct_parm(i, GetClientSpawnParam(client, i));
}

/*
==============
PF_changelevel
==============
*/
static void PF_changelevel(void) {
  const char *s;

  // make sure we don't issue two changelevels
  if (SVS_IsChangeLevelIssued()) return;
  SVS_SetChangeLevelIssued(true);

  s = PR_GetString(Pr_globalsi(OFS_PARM0));
  Cbuf_AddText(va("changelevel %s\n", s));
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
