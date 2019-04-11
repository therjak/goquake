package quakelib

import "C"

import (
	"fmt"
	"log"
	"quake/cbuf"
	"quake/math"
	"quake/model"
	"quake/progs"
)

func runError(format string, v ...interface{}) {
	// TODO: see PR_RunError
	conPrintf(format, v...)
}

/*
This is the only valid way to move an object without using the physics
of the world (setting velocity and waiting).  Directly changing origin
will not set internal links correctly, so clipping would be messed up.

This should be called when an object is spawned, and then only if it is
teleported.
*/
//export PF_setorigin
func PF_setorigin() {
	e := int(progsdat.RawGlobalsI[progs.OffsetParm0])
	ev := EntVars(e)
	ev.Origin = *progsdat.Globals.Parm1f()

	LinkEdict(e, false)
}

func setMinMaxSize(ev *progs.EntVars, min, max math.Vec3) {
	if min.X > max.X || min.Y > max.Y || min.Z > max.Z {
		runError("backwards mins/maxs")
	}
	ev.Mins = min.Array()
	ev.Maxs = max.Array()
	s := math.Sub(max, min)
	ev.Size = s.Array()
}

//export PF_setsize
func PF_setsize() {
	e := int(progsdat.RawGlobalsI[progs.OffsetParm0])
	min := math.VFromA(*progsdat.Globals.Parm1f())
	max := math.VFromA(*progsdat.Globals.Parm2f())
	setMinMaxSize(EntVars(e), min, max)
	LinkEdict(e, false)
}

//export PF_setmodel2
func PF_setmodel2() {

	e := int(progsdat.RawGlobalsI[progs.OffsetParm0])
	mi := progsdat.RawGlobalsI[progs.OffsetParm1]
	m := PR_GetStringWrap(int(mi))

	idx := -1
	for i, mp := range sv.modelPrecache {
		if mp == m {
			idx = i
			break
		}
	}
	if idx == -1 {
		runError("no precache: %s", m)
	}

	ev := EntVars(e)
	ev.Model = mi
	ev.ModelIndex = float32(idx)

	mod := sv.models[idx]
	if mod != nil {
		if mod.Type == model.ModBrush {
			log.Printf("ModBrush")
			log.Printf("mins: %v, maxs: %v", mod.ClipMins, mod.ClipMaxs)
			setMinMaxSize(ev, mod.ClipMins, mod.ClipMaxs)
		} else {
			log.Printf("!!!ModBrush")
			setMinMaxSize(ev, mod.Mins, mod.Maxs)
		}
	} else {
		log.Printf("No Mod")
		setMinMaxSize(ev, math.Vec3{}, math.Vec3{})
	}
	LinkEdict(e, false)
}

//export PF_normalize
func PF_normalize() {
	v := math.VFromA(*progsdat.Globals.Parm0f())
	vn := v.Normalize()
	*progsdat.Globals.Returnf() = vn.Array()
}

//export PF_vlen
func PF_vlen() {
	v := math.VFromA(*progsdat.Globals.Parm0f())
	l := v.Length()
	progsdat.Globals.Returnf()[0] = l
}

/*
*
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

// Returns a number from 0 <= num < 1
static void PF_random(void) {
  float num;

  num = (rand() & 0x7fff) / ((float)0x7fff);

  Set_Pr_globalsf(OFS_RETURN, num);
}

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

// Each entity can have eight independant sound sources, like voice,
// weapon, feet, etc.
// Channel 0 is an auto-allocate channel, the others override anything
// already running on that entity/channel pair.
// An attenuation of 0 will play full volume everywhere in the level.
// Larger attenuations will drop off.
static void PF_sound(void) {
  const char *sample;
  int channel;
  int entity;
  int volume;
  float attenuation;

  entity = Pr_globalsi(OFS_PARM0);
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

static void PF_break(void) {
  Con_Printf("break statement\n");
  *(int *)-4 = 0;  // dump to debugger
  //	PR_RunError ("break statement");
}

// Used for use tracing and shot targeting
// Traces are blocked by bbox and exact bsp entityes, and also slide
// box entities if the tryents flag is set.
static void PF_traceline(void) {
  vec3_t v1, v2;
  trace_t trace;
  int nomonsters;
  int ent;

  v1[0] = Pr_globalsf(OFS_PARM0);
  v1[1] = Pr_globalsf(OFS_PARM0 + 1);
  v1[2] = Pr_globalsf(OFS_PARM0 + 2);
  v2[0] = Pr_globalsf(OFS_PARM1);
  v2[1] = Pr_globalsf(OFS_PARM1 + 1);
  v2[2] = Pr_globalsf(OFS_PARM1 + 2);
  nomonsters = Pr_globalsf(OFS_PARM2);
  ent = Pr_globalsi(OFS_PARM3);

  // FIXME FIXME FIXME: Why do we hit this with certain progs.dat ??
  if (Cvar_GetValue(&developer)) {
    if (IS_NAN(v1[0]) || IS_NAN(v1[1]) || IS_NAN(v1[2]) || IS_NAN(v2[0]) ||
        IS_NAN(v2[1]) || IS_NAN(v2[2])) {
      Con_Warning("NAN in traceline:\nv1(%f %f %f) v2(%f %f %f)\nentity %d\n",
                  v1[0], v1[1], v1[2], v2[0], v2[1], v2[2], ent);
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
  if (trace.entp)
    Set_pr_global_struct_trace_ent(trace.entn);
  else
    Set_pr_global_struct_trace_ent(0);
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

// Returns a client (or object that has a client enemy) that would be a
// valid target.
// If there are more than one valid options, they are cycled each frame
// If (self.origin + self.viewofs) is not in the PVS of the current target,
// it is not returned at all.
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

// Sends text over to the client's execution buffer
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

// Sends text over to the client's execution buffer
static void PF_localcmd(void) {
  const char *str;

  str = PR_GetString(Pr_globalsi(OFS_PARM0));
  Cbuf_AddText(str);
}

static void PF_cvar(void) {
  const char *str;

  str = PR_GetString(Pr_globalsi(OFS_PARM0));

  Set_Pr_globalsf(OFS_RETURN, Cvar_VariableValue(str));
}

static void PF_cvar_set(void) {
  const char *var, *val;

  var = PR_GetString(Pr_globalsi(OFS_PARM0));
  val = PR_GetString(Pr_globalsi(OFS_PARM1));

  Cvar_Set(var, val);
}

// Returns a chain of entities that have origins within a spherical area
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

static void PF_droptofloor(void) {
  int ent;
  vec3_t end;
  trace_t trace;

  ent = Pr_global_struct_self();

  VectorCopy(EVars(ent)->origin, end);
  end[2] -= 256;

  trace = SV_Move(EVars(ent)->origin, EVars(ent)->mins, EVars(ent)->maxs, end,
                  false, ent);

  if (trace.fraction == 1 || trace.allsolid)
    Set_Pr_globalsf(OFS_RETURN, 0);
  else {
    VectorCopy(trace.endpos, EVars(ent)->origin);
    SV_LinkEdict(ent, false);
    EVars(ent)->flags = (int)EVars(ent)->flags | FL_ONGROUND;
    EVars(ent)->groundentity = trace.entn;
    Set_Pr_globalsf(OFS_RETURN, 1);
  }
}

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

static void PF_checkbottom(void) {
  int ent;

  ent = Pr_globalsi(OFS_PARM0);

  Set_Pr_globalsf(OFS_RETURN, SV_CheckBottom(ent));
}

static void PF_pointcontents(void) {
  vec3_t v;

  v[0] = Pr_globalsf(OFS_PARM0);
  v[1] = Pr_globalsf(OFS_PARM0 + 1);
  v[2] = Pr_globalsf(OFS_PARM0 + 2);

  Set_Pr_globalsf(OFS_RETURN, SV_PointContents(v));
}

static void PF_nextent(void) {
  int i;

  i = Pr_globalsi(OFS_PARM0);
  while (1) {
    i++;
    if (i == SV_NumEdicts()) {
      RETURN_EDICT(0);
      return;
    }
    if (!EDICT_NUM(i)->free) {
      RETURN_EDICT(i);
      return;
    }
  }
}

// Pick a vector for the player to shoot along
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
  (void)speed; // variable set but not used

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

// This was a major timewaster in progs, so it was converted to C
void PF_changeyaw(void) {
  int ent;
  float ideal, current, move, speed;

  ent = Pr_global_struct_self();
  current = anglemod(EVars(ent)->angles[1]);
  ideal = EVars(ent)->ideal_yaw;
  speed = EVars(ent)->yaw_speed;

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

  EVars(ent)->angles[1] = anglemod(current + move);
}

static int WriteClient() {
  int entnum;

  entnum = Pr_global_struct_msg_entity();
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
*/

//export PF_changelevel
func PF_changelevel() {
	// make sure we don't issue two changelevels
	if svs.changeLevelIssued {
		return
	}
	svs.changeLevelIssued = true

	i := int(progsdat.RawGlobalsI[progs.OffsetParm0])
	s := PR_GetStringWrap(i)
	cbuf.AddText(fmt.Sprintf("changelevel %s\n", s))
}
