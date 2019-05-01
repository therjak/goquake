package quakelib

import "C"

import (
	"fmt"
	"log"
	"math/rand"
	"quake/cbuf"
	"quake/cvars"
	"quake/math"
	"quake/math/vec"
	"quake/model"
	"quake/progs"
	"quake/protocol"
	"quake/protocol/server"
	"runtime"

	"github.com/chewxy/math32"
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

func setMinMaxSize(ev *progs.EntVars, min, max vec.Vec3) {
	if min.X > max.X || min.Y > max.Y || min.Z > max.Z {
		runError("backwards mins/maxs")
	}
	ev.Mins = min.Array()
	ev.Maxs = max.Array()
	s := vec.Sub(max, min)
	ev.Size = s.Array()
}

//export PF_setsize
func PF_setsize() {
	e := int(progsdat.RawGlobalsI[progs.OffsetParm0])
	min := vec.VFromA(*progsdat.Globals.Parm1f())
	max := vec.VFromA(*progsdat.Globals.Parm2f())
	setMinMaxSize(EntVars(e), min, max)
	LinkEdict(e, false)
}

//export PF_setmodel2
func PF_setmodel2() {

	e := int(progsdat.RawGlobalsI[progs.OffsetParm0])
	mi := progsdat.RawGlobalsI[progs.OffsetParm1]
	m := PRGetString(int(mi))
	if m == nil {
		runError("no precache: %d", mi)
		return
	}

	idx := -1
	for i, mp := range sv.modelPrecache {
		if mp == *m {
			idx = i
			break
		}
	}
	if idx == -1 {
		runError("no precache: %s", m)
		return
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
		setMinMaxSize(ev, vec.Vec3{}, vec.Vec3{})
	}
	LinkEdict(e, false)
}

//export PF_normalize
func PF_normalize() {
	v := vec.VFromA(*progsdat.Globals.Parm0f())
	vn := v.Normalize()
	*progsdat.Globals.Returnf() = vn.Array()
}

//export PF_vlen
func PF_vlen() {
	v := vec.VFromA(*progsdat.Globals.Parm0f())
	l := v.Length()
	progsdat.Globals.Returnf()[0] = l
}

//export PF_vectoyaw
func PF_vectoyaw() {
	v := vec.VFromA(*progsdat.Globals.Parm0f())
	yaw := func() float32 {
		if v.X == 0 && v.Y == 0 {
			return 0
		}
		y := (math32.Atan2(v.Y, v.X) * 180) / math32.Pi
		y = math32.Trunc(y)
		if y < 0 {
			y += 360
		}
		return y
	}()
	progsdat.Globals.Returnf()[0] = yaw
}

//export PF_vectoangles
func PF_vectoangles() {
	v := vec.VFromA(*progsdat.Globals.Parm0f())
	yaw, pitch := func() (float32, float32) {
		if v.X == 0 && v.Y == 0 {
			p := func() float32 {
				if v.Z > 0 {
					return 90
				}
				return 270
			}()
			return 0, p
		}
		y := (math32.Atan2(v.Y, v.X) * 180) / math32.Pi
		y = math32.Trunc(y)
		if y < 0 {
			y += 360
		}
		forward := math32.Sqrt(v.X*v.X + v.Y*v.Y)
		p := (math32.Atan2(v.Z, forward) * 180) / math32.Pi
		p = math32.Trunc(p)
		if p < 0 {
			p += 360
		}
		return y, p
	}()
	*progsdat.Globals.Returnf() = [3]float32{pitch, yaw, 0}
}

// Returns a number from 0 <= num < 1
//export PF_random
func PF_random() {
	progsdat.Globals.Returnf()[0] = rand.Float32()
}

//export PF_particle
func PF_particle() {
	org := vec.VFromA(*progsdat.Globals.Parm0f())
	dir := vec.VFromA(*progsdat.Globals.Parm1f())
	color := progsdat.RawGlobalsF[progs.OffsetParm2]
	count := progsdat.RawGlobalsF[progs.OffsetParm3]
	sv.StartParticle(org, dir, int(color), int(count))
}

//export PF_ambientsound
func PF_ambientsound() {
	large := false
	pos := vec.VFromA(*progsdat.Globals.Parm0f())
	sample := PRGetString(int(progsdat.RawGlobalsI[progs.OffsetParm1]))
	if sample == nil {
		conPrintf("no precache: %v\n", pos)
		return
	}
	volume := progsdat.RawGlobalsF[progs.OffsetParm2] * 255
	attenuation := progsdat.RawGlobalsF[progs.OffsetParm3] * 64

	// check to see if samp was properly precached
	soundnum := func() int {
		for i, m := range sv.soundPrecache {
			if m == *sample {
				return i
			}
		}
		return -1
	}()

	if soundnum == -1 {
		conPrintf("no precache: %v\n", sample)
		return
	}

	if soundnum > 255 {
		if sv.protocol == protocol.NetQuake {
			return // don't send any info protocol can't support
		} else {
			large = true
		}
	}

	// add an svc_spawnambient command to the level signon packet
	if large {
		sv.signon.WriteByte(server.SpawnStaticSound2)
	} else {
		sv.signon.WriteByte(server.SpawnStaticSound)
	}

	sv.signon.WriteCoord(pos.X, int(sv.protocolFlags))
	sv.signon.WriteCoord(pos.Y, int(sv.protocolFlags))
	sv.signon.WriteCoord(pos.Z, int(sv.protocolFlags))

	if large {
		sv.signon.WriteShort(soundnum)
	} else {
		sv.signon.WriteByte(soundnum)
	}

	sv.signon.WriteByte(int(volume))
	sv.signon.WriteByte(int(attenuation))
}

// Each entity can have eight independant sound sources, like voice,
// weapon, feet, etc.
// Channel 0 is an auto-allocate channel, the others override anything
// already running on that entity/channel pair.
// An attenuation of 0 will play full volume everywhere in the level.
// Larger attenuations will drop off.
//export PF_sound
func PF_sound() {
	entity := progsdat.RawGlobalsI[progs.OffsetParm0]
	channel := progsdat.RawGlobalsF[progs.OffsetParm1]
	sample := PRGetString(int(progsdat.RawGlobalsI[progs.OffsetParm2]))
	if sample == nil {
		runError("PF_sound: no sample")
		return
	}
	volume := progsdat.RawGlobalsF[progs.OffsetParm3] * 255
	attenuation := progsdat.RawGlobalsF[progs.OffsetParm4]

	if volume < 0 || volume > 255 {
		HostError("SV_StartSound: volume = %v", volume)
	}

	if attenuation < 0 || attenuation > 4 {
		HostError("SV_StartSound: attenuation = %v", attenuation)
	}

	if channel < 0 || channel > 7 {
		HostError("SV_StartSound: channel = %v", channel)
	}
	sv.StartSound(int(entity), int(channel), int(volume), *sample, attenuation)
}

//export PF_break
func PF_break() {
	conPrintf("break statement\n")
	runtime.Breakpoint()
}

// Used for use tracing and shot targeting
// Traces are blocked by bbox and exact bsp entityes, and also slide
// box entities if the tryents flag is set.
//export PF_traceline
func PF_traceline() {
	v1 := vec.VFromA(*progsdat.Globals.Parm0f())
	v2 := vec.VFromA(*progsdat.Globals.Parm1f())
	nomonsters := progsdat.RawGlobalsF[progs.OffsetParm2]
	ent := int(progsdat.RawGlobalsI[progs.OffsetParm3])

	// FIXME FIXME FIXME: Why do we hit this with certain progs.dat ??
	if cvars.Developer.Bool() {
		if !vec.Equal(v1, v1) || !vec.Equal(v2, v2) {
			conPrintf("NAN in traceline:\nv1(%v %v %v) v2(%v %v %v)\nentity %v\n",
				v1.X, v1.Y, v1.Z, v2.X, v2.Y, v2.Z, ent)
		}
	}

	if !vec.Equal(v1, v1) {
		v1 = vec.Vec3{}
	}
	if !vec.Equal(v2, v2) {
		v2 = vec.Vec3{}
	}

	trace := svMove(v1, vec.Vec3{}, vec.Vec3{}, v2, int(nomonsters), ent)

	progsdat.Globals.TraceAllSolid = float32(trace.allsolid)
	progsdat.Globals.TraceStartSolid = float32(trace.startsolid)
	progsdat.Globals.TraceFraction = float32(trace.fraction)
	progsdat.Globals.TraceInWater = float32(trace.inwater)
	progsdat.Globals.TraceInOpen = float32(trace.inopen)
	progsdat.Globals.TraceEndPos = [3]float32{
		float32(trace.endpos[0]),
		float32(trace.endpos[1]),
		float32(trace.endpos[2])}

	progsdat.Globals.TracePlaneNormal = [3]float32{
		float32(trace.plane.normal[0]),
		float32(trace.plane.normal[1]),
		float32(trace.plane.normal[2])}

	progsdat.Globals.TracePlaneDist = float32(trace.plane.dist)
	if trace.entp != 0 {
		progsdat.Globals.TraceEnt = int32(trace.entn)
	} else {
		progsdat.Globals.TraceEnt = 0
	}
}

/*
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

    if (edictNum(ent).free) continue;
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
  if (edictNum(ent).free || EVars(ent)->health <= 0) {
	  progsdat.RawGlobalsI[progs.OffsetReturn] = 0;
    return;
  }

  // if current entity can't possibly see the check entity, return 0
  self = Pr_global_struct_self();
  VectorAdd(EVars(self)->origin, EVars(self)->view_ofs, view);
  leaf = Mod_PointInLeaf(view, sv.worldmodel);
  l = (leaf - sv.worldmodel->leafs) - 1;
  if ((l < 0) || !(checkpvs[l >> 3] & (1 << (l & 7)))) {
    c_notvis++;
	  progsdat.RawGlobalsI[progs.OffsetReturn] = 0;
    return;
  }

  // might be able to see it
  c_invis++;
	progsdat.RawGlobalsI[progs.OffsetReturn] = int32(ent);
}
*/

// Sends text over to the client's execution buffer
//export PF_stuffcmd
func PF_stuffcmd() {
	entnum := int(progsdat.RawGlobalsI[progs.OffsetParm0])
	if entnum < 1 || entnum > svs.maxClients {
		runError("Parm 0 not a client")
		return
	}
	str := PRGetString(int(progsdat.RawGlobalsI[progs.OffsetParm1]))
	if str == nil {
		runError("stuffcmd: no string")
		return
	}

	c := sv_clients[entnum-1]
	c.msg.WriteByte(server.StuffText)
	c.msg.WriteString(*str)
}

// Sends text over to the client's execution buffer
//export PF_localcmd
func PF_localcmd() {
	str := PRGetString(int(progsdat.RawGlobalsI[progs.OffsetParm0]))
	if str == nil {
		runError("localcmd: no string")
		return
	}
	cbuf.AddText(*str)
}

//export PF_cvar
func PF_cvar() {
	str := PRGetString(int(progsdat.RawGlobalsI[progs.OffsetParm0]))
	if str == nil {
		runError("PF_cvar: no string")
		return
	}
	progsdat.Globals.Returnf()[0] = CvarVariableValue(*str)
}

//export PF_cvar_set
func PF_cvar_set() {
	name := PRGetString(int(progsdat.RawGlobalsI[progs.OffsetParm0]))
	if name == nil {
		runError("PF_cvar_set: no name string")
		return
	}
	val := PRGetString(int(progsdat.RawGlobalsI[progs.OffsetParm1]))
	if val == nil {
		runError("PF_cvar_set: no val string")
		return
	}
	cvarSet(*name, *val)
}

// Returns a chain of entities that have origins within a spherical area
//export PF_findradius
func PF_findradius() {
	chain := int32(0)
	org := vec.VFromA(*progsdat.Globals.Parm0f())
	rad := progsdat.RawGlobalsF[progs.OffsetParm1]

	for ent := 1; ent < sv.numEdicts; ent++ {
		if edictNum(ent).free != 0 {
			continue
		}
		ev := EntVars(ent)
		if ev.Solid == SOLID_NOT {
			continue
		}
		eo := vec.VFromA(ev.Origin)
		mins := vec.VFromA(ev.Mins)
		maxs := vec.VFromA(ev.Maxs)
		eorg := vec.Sub(org, vec.Add(eo, vec.Add(mins, maxs).Scale(0.5)))
		if eorg.Length() > rad {
			continue
		}

		ev.Chain = chain
		chain = int32(ent)
	}

	progsdat.RawGlobalsI[progs.OffsetReturn] = chain
}

/*
static void PF_dprint(void) {
	Con_DPrintf("%s", PF_VarString(0));
}
*/

//export PF_ftos
func PF_ftos() {
	v := progsdat.RawGlobalsF[progs.OffsetParm0]
	s := func() string {
		iv := int(v)
		if v == float32(iv) {
			return fmt.Sprintf("%d", iv)
		}
		return fmt.Sprintf("%5.1f", v)
	}()
	progsdat.RawGlobalsI[progs.OffsetReturn] = int32(PRSetEngineString(s))
}

//export PF_fabs
func PF_fabs() {
	f := progsdat.RawGlobalsF[progs.OffsetParm0]
	progsdat.Globals.Returnf()[0] = math32.Abs(f)
}

//export PF_vtos
func PF_vtos() {
	s := fmt.Sprintf("'%5.1f %5.1f %5.1f'",
		progsdat.RawGlobalsF[progs.OffsetParm0],
		progsdat.RawGlobalsF[progs.OffsetParm0+1],
		progsdat.RawGlobalsF[progs.OffsetParm0+2])
	progsdat.RawGlobalsI[progs.OffsetReturn] = int32(PRSetEngineString(s))
}

//export PF_Spawn
func PF_Spawn() {
	ed := edictAlloc()
	progsdat.RawGlobalsI[progs.OffsetReturn] = int32(ed)
}

//export PF_Remove
func PF_Remove() {
	ed := progsdat.RawGlobalsI[progs.OffsetParm0]
	edictFree(int(ed))
}

//export PF_Find
func PF_Find() {
	e := progsdat.RawGlobalsI[progs.OffsetParm0]
	f := progsdat.RawGlobalsI[progs.OffsetParm1]
	s := PRGetString(int(progsdat.RawGlobalsI[progs.OffsetParm2]))
	if s == nil {
		runError("PF_Find: bad search string")
		return
	}
	for e++; int(e) < sv.numEdicts; e++ {
		if edictNum(int(e)).free != 0 {
			continue
		}
		ti := RawEntVarsI(int(e), int(f))
		t := PRGetString(int(ti))
		if t == nil {
			continue
		}
		if *t == *s {
			progsdat.RawGlobalsI[progs.OffsetReturn] = int32(e)
			return
		}
	}
	progsdat.RawGlobalsI[progs.OffsetReturn] = 0
}

/*
// THERJAK
static void PR_CheckEmptyString(const char *s) {
  if (s[0] <= ' ') PR_RunError("Bad string");
}

// THERJAK
static void PF_precache_file(void) {  // precache_file is only used to copy
                                      // files with qcc, it does nothing
  Set_Pr_globalsi(OFS_RETURN, Pr_globalsi(OFS_PARM0));
}

// THERJAK
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
*/

/*
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
*/

//export PF_droptofloor
func PF_droptofloor() {
	ent := int(progsdat.Globals.Self)
	ev := EntVars(ent)
	start := vec.VFromA(ev.Origin)
	mins := vec.VFromA(ev.Mins)
	maxs := vec.VFromA(ev.Maxs)
	end := vec.VFromA(ev.Origin)
	end.Z -= 256

	trace := svMove(start, mins, maxs, end, 0, ent)

	if trace.fraction == 1 || trace.allsolid != 0 {
		progsdat.Globals.Returnf()[0] = 0
	} else {
		ev.Origin = [3]float32{
			float32(trace.endpos[0]),
			float32(trace.endpos[1]),
			float32(trace.endpos[2])}
		LinkEdict(ent, false)
		ev.Flags = float32(int(ev.Flags) | FL_ONGROUND)
		ev.GroundEntity = int32(trace.entn)
		progsdat.Globals.Returnf()[0] = 1
	}
}

/* TODO
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
*/

//export PF_rint
func PF_rint() {
	v := progsdat.RawGlobalsF[progs.OffsetParm0]
	progsdat.Globals.Returnf()[0] = math.RoundToEven(v)
}

//export PF_floor
func PF_floor() {
	v := progsdat.RawGlobalsF[progs.OffsetParm0]
	progsdat.Globals.Returnf()[0] = math32.Floor(v)
}

//export PF_ceil
func PF_ceil() {
	v := progsdat.RawGlobalsF[progs.OffsetParm0]
	progsdat.Globals.Returnf()[0] = math32.Ceil(v)
}

//export PF_checkbottom
func PF_checkbottom() {
	entnum := int(progsdat.RawGlobalsI[progs.OffsetParm0])
	f := float32(0)
	if checkBottom(entnum) {
		f = 1
	}
	progsdat.Globals.Returnf()[0] = f
}

//export PF_pointcontents
func PF_pointcontents() {
	v := vec.VFromA(*progsdat.Globals.Parm0f())
	pc := pointContents(v)
	progsdat.Globals.Returnf()[0] = float32(pc)
}

//export PF_nextent
func PF_nextent() {
	i := progsdat.RawGlobalsI[progs.OffsetParm0]
	for {
		i++
		if int(i) == sv.numEdicts {
			progsdat.RawGlobalsI[progs.OffsetReturn] = 0
			return
		}
		if edictNum(int(i)).free == 0 {
			progsdat.RawGlobalsI[progs.OffsetReturn] = i
			return
		}
	}
}

/*
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
*/

// This was a major timewaster in progs
//export PF_changeyaw
func PF_changeyaw() {
	ent := int(progsdat.Globals.Self)
	ev := EntVars(ent)
	current := math.AngleMod32(ev.Angles[1])
	ideal := ev.IdealYaw
	speed := ev.YawSpeed

	if current == ideal {
		return
	}
	move := ideal - current
	if ideal > current {
		if move >= 180 {
			move -= 360
		}
	} else {
		if move <= -180 {
			move += 360
		}
	}
	if move > 0 {
		if move > speed {
			move = speed
		}
	} else {
		if move < -speed {
			move = -speed
		}
	}
	ev.Angles[1] = math.AngleMod32(current + move)
}

const (
	MSG_BROADCAST = iota // unreliable to all
	MSG_ONE              // reliable to one
	MSG_ALL              // reliable to all
	MSG_INIT             // write to the init string
)

func writeClient() int {
	entnum := int(progsdat.Globals.MsgEntity)
	if entnum < 1 || entnum > svs.maxClients {
		runError("WriteDest: not a client")
	}
	return entnum - 1
}

//export PF_WriteByte
func PF_WriteByte() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[writeClient()].msg.WriteByte(int(msg))
	case MSG_INIT:
		sv.signon.WriteByte(int(msg))
	case MSG_BROADCAST:
		sv.datagram.WriteByte(int(msg))
	case MSG_ALL:
		sv.reliableDatagram.WriteByte(int(msg))
	default:
		runError("WriteDest: bad destination")
	}
}

//export PF_WriteChar
func PF_WriteChar() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[writeClient()].msg.WriteChar(int(msg))
	case MSG_INIT:
		sv.signon.WriteChar(int(msg))
	case MSG_BROADCAST:
		sv.datagram.WriteChar(int(msg))
	case MSG_ALL:
		sv.reliableDatagram.WriteChar(int(msg))
	default:
		runError("WriteDest: bad destination")
	}
}

//export PF_WriteShort
func PF_WriteShort() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[writeClient()].msg.WriteShort(int(msg))
	case MSG_INIT:
		sv.signon.WriteShort(int(msg))
	case MSG_BROADCAST:
		sv.datagram.WriteShort(int(msg))
	case MSG_ALL:
		sv.reliableDatagram.WriteShort(int(msg))
	default:
		runError("WriteDest: bad destination")
	}
}

//export PF_WriteLong
func PF_WriteLong() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[writeClient()].msg.WriteLong(int(msg))
	case MSG_INIT:
		sv.signon.WriteLong(int(msg))
	case MSG_BROADCAST:
		sv.datagram.WriteLong(int(msg))
	case MSG_ALL:
		sv.reliableDatagram.WriteLong(int(msg))
	default:
		runError("WriteDest: bad destination")
	}
}

//export PF_WriteAngle
func PF_WriteAngle() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[writeClient()].msg.WriteAngle(msg, int(sv.protocolFlags))
	case MSG_INIT:
		sv.signon.WriteAngle(msg, int(sv.protocolFlags))
	case MSG_BROADCAST:
		sv.datagram.WriteAngle(msg, int(sv.protocolFlags))
	case MSG_ALL:
		sv.reliableDatagram.WriteAngle(msg, int(sv.protocolFlags))
	default:
		runError("WriteDest: bad destination")
	}
}

//export PF_WriteCoord
func PF_WriteCoord() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[writeClient()].msg.WriteCoord(msg, int(sv.protocolFlags))
	case MSG_INIT:
		sv.signon.WriteCoord(msg, int(sv.protocolFlags))
	case MSG_BROADCAST:
		sv.datagram.WriteCoord(msg, int(sv.protocolFlags))
	case MSG_ALL:
		sv.reliableDatagram.WriteCoord(msg, int(sv.protocolFlags))
	default:
		runError("WriteDest: bad destination")
	}
}

//export PF_WriteString
func PF_WriteString() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	i := int(progsdat.RawGlobalsI[progs.OffsetParm1])
	msg := PRGetString(i)
	if msg == nil {
		runError("PF_WriteString: bad string")
		return
	}
	switch dest {
	case MSG_ONE:
		sv_clients[writeClient()].msg.WriteString(*msg)
	case MSG_INIT:
		sv.signon.WriteString(*msg)
	case MSG_BROADCAST:
		sv.datagram.WriteString(*msg)
	case MSG_ALL:
		sv.reliableDatagram.WriteString(*msg)
	default:
		runError("WriteDest: bad destination")
	}
}

//export PF_WriteEntity
func PF_WriteEntity() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[writeClient()].msg.WriteShort(int(msg))
	case MSG_INIT:
		sv.signon.WriteShort(int(msg))
	case MSG_BROADCAST:
		sv.datagram.WriteShort(int(msg))
	case MSG_ALL:
		sv.reliableDatagram.WriteShort(int(msg))
	default:
		runError("WriteDest: bad destination")
	}
}

/*
// THERJAK
static void PF_makestatic(void) {
  int ent;
  int i;
  int bits = 0;  // johnfitz -- PROTOCOL_FITZQUAKE

  ent = Pr_globalsi(OFS_PARM0);

  // johnfitz -- don't send invisible static entities
  if (edictNum(ent)->alpha == ENTALPHA_ZERO) {
    edictFree(ent);
    return;
  }
  // johnfitz

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (SV_Protocol() == PROTOCOL_NETQUAKE) {
    if (SV_ModelIndex(PR_GetString(EVars(ent)->model)) & 0xFF00 ||
        (int)(EVars(ent)->frame) & 0xFF00) {
      edictFree(ent);
      return;  // can't display the correct model & frame, so don't show it at
               // all
    }
  } else {
    if (SV_ModelIndex(PR_GetString(EVars(ent)->model)) & 0xFF00)
      bits |= B_LARGEMODEL;
    if ((int)(EVars(ent)->frame) & 0xFF00) bits |= B_LARGEFRAME;
    if (edictNum(ent)->alpha != ENTALPHA_DEFAULT) bits |= B_ALPHA;
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
  if (bits & B_ALPHA) SV_SO_WriteByte(edictNum(ent)->alpha);
  // johnfitz

  // throw the entity away now
  edictFree(ent);
}

// THERJAK
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
	s := PRGetString(i)
	if s == nil {
		runError("PF_changelevel: bad level name")
		return
	}
	cbuf.AddText(fmt.Sprintf("changelevel %s\n", *s))
}
