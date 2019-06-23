package quakelib

import "C"

import (
	"fmt"
	"log"
	"math/rand"
	"quake/cbuf"
	"quake/conlog"
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
	conlog.Printf(format, v...)
}

// Dumps out self, then an error message.  The program is aborted and self is
// removed, but the level can continue.
//export PF_objerror
func PF_objerror() {
	s := vmVarString(0)
	fs := vmFuncName()
	conlog.Printf("======OBJECT ERROR in %s:\n%s\n", fs, s)
	ed := int(progsdat.Globals.Self)
	edictPrint(ed)
	edictFree(ed)
}

// This is a TERMINAL error, which will kill off the entire server.
// Dumps self.
//export PF_error
func PF_error() {
	s := vmVarString(0)
	fs := vmFuncName()
	conlog.Printf("======SERVER ERROR in %s:\n%s\n", fs, s)
	edictPrint(int(progsdat.Globals.Self))
	HostError("Program error")
}

//export PF_dprint
func PF_dprint() {
	s := vmVarString(0)
	conlog.DPrintf(s)
}

// broadcast print to everyone on server
//export PF_bprint
func PF_bprint() {
	s := vmVarString(0)
	SV_BroadcastPrint(s)
}

// single print to a specific client
//export PF_sprint
func PF_sprint() {
	e := int(progsdat.Globals.Parm0[0])
	s := vmVarString(1)
	if e < 1 || e > svs.maxClients {
		conlog.Printf("tried to sprint to a non-client\n")
		return
	}
	e--
	c := sv_clients[e]
	c.msg.WriteChar(server.Print)
	c.msg.WriteString(s)
}

// single print to a specific client
//export PF_centerprint
func PF_centerprint() {
	e := int(progsdat.Globals.Parm0[0])
	s := vmVarString(1)
	if e < 1 || e > svs.maxClients {
		conlog.Printf("tried to sprint to a non-client\n")
		return
	}
	e--
	c := sv_clients[e]
	c.msg.WriteChar(server.CenterPrint)
	c.msg.WriteString(s)
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
	e := int(progsdat.Globals.Parm0[0])
	ev := EntVars(e)
	ev.Origin = *progsdat.Globals.Parm1f()

	LinkEdict(e, false)
}

func setMinMaxSize(ev *progs.EntVars, min, max vec.Vec3) {
	if min[0] > max[0] || min[1] > max[1] || min[2] > max[2] {
		conlog.DPrintf("backwards mins/maxs")
	}
	ev.Mins = min
	ev.Maxs = max
	ev.Size = vec.Sub(max, min)
}

//export PF_setsize
func PF_setsize() {
	e := int(progsdat.Globals.Parm0[0])
	min := vec.VFromA(*progsdat.Globals.Parm1f())
	max := vec.VFromA(*progsdat.Globals.Parm2f())
	setMinMaxSize(EntVars(e), min, max)
	LinkEdict(e, false)
}

//export PF_setmodel
func PF_setmodel() {

	e := int(progsdat.Globals.Parm0[0])
	mi := progsdat.Globals.Parm1[0]
	m, err := progsdat.String(mi)
	if err != nil {
		runError("no precache: %d", mi)
		return
	}

	idx := -1
	for i, mp := range sv.modelPrecache {
		if mp == m {
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
			// log.Printf("ModBrush")
			// log.Printf("mins: %v, maxs: %v", mod.ClipMins, mod.ClipMaxs)
			setMinMaxSize(ev, mod.ClipMins, mod.ClipMaxs)
		} else {
			// log.Printf("!!!ModBrush")
			setMinMaxSize(ev, mod.Mins, mod.Maxs)
		}
	} else {
		// log.Printf("No Mod")
		setMinMaxSize(ev, vec.Vec3{}, vec.Vec3{})
	}
	LinkEdict(e, false)
}

//export PF_normalize
func PF_normalize() {
	v := vec.VFromA(*progsdat.Globals.Parm0f())
	*progsdat.Globals.Returnf() = v.Normalize()
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
		if v[0] == 0 && v[1] == 0 {
			return 0
		}
		y := (math32.Atan2(v[1], v[0]) * 180) / math32.Pi
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
		if v[0] == 0 && v[1] == 0 {
			p := func() float32 {
				if v[2] > 0 {
					return 90
				}
				return 270
			}()
			return 0, p
		}
		y := (math32.Atan2(v[1], v[0]) * 180) / math32.Pi
		y = math32.Trunc(y)
		if y < 0 {
			y += 360
		}
		forward := math32.Sqrt(v[0]*v[0] + v[1]*v[1])
		p := (math32.Atan2(v[2], forward) * 180) / math32.Pi
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
	sample, err := progsdat.String(progsdat.Globals.Parm1[0])
	if err != nil {
		conlog.Printf("no precache: %v\n", pos)
		return
	}
	volume := progsdat.RawGlobalsF[progs.OffsetParm2] * 255
	attenuation := progsdat.RawGlobalsF[progs.OffsetParm3] * 64

	// check to see if samp was properly precached
	soundnum := func() int {
		for i, m := range sv.soundPrecache {
			if m == sample {
				return i
			}
		}
		return -1
	}()

	if soundnum == -1 {
		conlog.Printf("no precache: %v\n", sample)
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

	sv.signon.WriteCoord(pos[0], sv.protocolFlags)
	sv.signon.WriteCoord(pos[1], sv.protocolFlags)
	sv.signon.WriteCoord(pos[2], sv.protocolFlags)

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
	entity := progsdat.Globals.Parm0[0]
	channel := progsdat.RawGlobalsF[progs.OffsetParm1]
	sample, err := progsdat.String(progsdat.Globals.Parm2[0])
	if err != nil {
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
	sv.StartSound(int(entity), int(channel), int(volume), sample, attenuation)
}

//export PF_break
func PF_break() {
	conlog.Printf("break statement\n")
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
	ent := int(progsdat.Globals.Parm3[0])

	// FIXME FIXME FIXME: Why do we hit this with certain progs.dat ??
	if cvars.Developer.Bool() {
		if v1 != v1 || v2 != v2 {
			conlog.Printf("NAN in traceline:\nv1(%v %v %v) v2(%v %v %v)\nentity %v\n",
				v1[0], v1[1], v1[2], v2[0], v2[1], v2[2], ent)
		}
	}

	if v1 != v1 {
		v1 = vec.Vec3{}
	}
	if v2 != v2 {
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

var (
	checkpvs []byte // to cache results
)

func init() {
	checkpvs = make([]byte, model.MAX_MAP_LEAFS/8)
}

func PF_newcheckclient(check int) int {
	// cycle to the next one
	if check < 1 {
		check = 1
	}
	if check > svs.maxClients {
		check = svs.maxClients
	}

	i := check + 1
	if check == svs.maxClients {
		i = 1
	}
	ent := 0

	for ; ; i++ {
		if i == svs.maxClients+1 {
			i = 1
		}

		ent = i

		if i == check {
			// didn't find anything else
			break
		}

		if edictNum(ent).Free {
			continue
		}
		ev := EntVars(ent)
		if ev.Health <= 0 {
			continue
		}
		if int(ev.Flags)&FL_NOTARGET != 0 {
			continue
		}

		// anything that is a client, or has a client as an enemy
		break
	}

	ev := EntVars(ent)
	// get the PVS for the entity
	org := vec.Add(ev.Origin, ev.ViewOfs)
	leaf, _ := sv.worldModel.PointInLeaf(org)
	pvs := sv.worldModel.LeafPVS(leaf)

	// we care only about the first (len(sv.worldModel.Leafs)+7)/8 bytes
	copy(checkpvs, pvs)
	return i
}

// Returns a client (or object that has a client enemy) that would be a
// valid target.
// If there are more than one valid options, they are cycled each frame
// If (self.origin + self.viewofs) is not in the PVS of the current target,
// it is not returned at all.
//export PF_checkclient
func PF_checkclient() {
	// find a new check if on a new frame
	if sv.time-sv.lastCheckTime >= 0.1 {
		sv.lastCheck = PF_newcheckclient(sv.lastCheck)
		sv.lastCheckTime = sv.time
	}

	// return check if it might be visible
	ent := sv.lastCheck
	if edictNum(ent).Free || EntVars(ent).Health <= 0 {
		progsdat.Globals.Return[0] = 0
		return
	}

	// if current entity can't possibly see the check entity, return 0
	self := int(progsdat.Globals.Self)
	es := EntVars(self)
	view := vec.Add(es.Origin, es.ViewOfs)
	leaf, _ := sv.worldModel.PointInLeaf(view)
	leafNum := -2
	for i, l := range sv.worldModel.Leafs {
		if l == leaf {
			leafNum = i - 1 // -1 to remove the solid 0 leaf
		}
	}
	if leafNum == -2 {
		log.Printf("checkclient: Got leafnum -2, len(leafs)= %d", len(sv.worldModel.Leafs))
	}

	if (leafNum < 0) || (checkpvs[leafNum/8]&(1<<(uint(leafNum)&7)) == 0) {
		progsdat.Globals.Return[0] = 0
		return
	}

	// might be able to see it
	progsdat.Globals.Return[0] = int32(ent)
}

// Sends text over to the client's execution buffer
//export PF_stuffcmd
func PF_stuffcmd() {
	entnum := int(progsdat.Globals.Parm0[0])
	if entnum < 1 || entnum > svs.maxClients {
		runError("Parm 0 not a client")
		return
	}
	str, err := progsdat.String(progsdat.Globals.Parm1[0])
	if err != nil {
		runError("stuffcmd: no string")
		return
	}

	c := sv_clients[entnum-1]
	c.msg.WriteByte(server.StuffText)
	c.msg.WriteString(str)
}

// Sends text over to the client's execution buffer
//export PF_localcmd
func PF_localcmd() {
	str, err := progsdat.String(progsdat.Globals.Parm0[0])
	if err != nil {
		runError("localcmd: no string")
		return
	}
	cbuf.AddText(str)
}

//export PF_cvar
func PF_cvar() {
	str, err := progsdat.String(progsdat.Globals.Parm0[0])
	if err != nil {
		runError("PF_cvar: no string")
		return
	}
	progsdat.Globals.Returnf()[0] = CvarVariableValue(str)
}

//export PF_cvar_set
func PF_cvar_set() {
	name, err := progsdat.String(progsdat.Globals.Parm0[0])
	if err != nil {
		runError("PF_cvar_set: no name string")
		return
	}
	val, err := progsdat.String(progsdat.Globals.Parm1[0])
	if err != nil {
		runError("PF_cvar_set: no val string")
		return
	}
	cvarSet(name, val)
}

// Returns a chain of entities that have origins within a spherical area
//export PF_findradius
func PF_findradius() {
	chain := int32(0)
	org := vec.VFromA(*progsdat.Globals.Parm0f())
	rad := progsdat.RawGlobalsF[progs.OffsetParm1]

	for ent := 1; ent < sv.numEdicts; ent++ {
		if edictNum(ent).Free {
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

	progsdat.Globals.Return[0] = chain
}

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
	progsdat.Globals.Return[0] = progsdat.AddString(s)
}

//export PF_fabs
func PF_fabs() {
	f := progsdat.RawGlobalsF[progs.OffsetParm0]
	progsdat.Globals.Returnf()[0] = math32.Abs(f)
}

//export PF_vtos
func PF_vtos() {
	p := *progsdat.Globals.Parm0f()
	s := fmt.Sprintf("'%5.1f %5.1f %5.1f'", p[0], p[1], p[2])
	progsdat.Globals.Return[0] = progsdat.AddString(s)
}

//export PF_Spawn
func PF_Spawn() {
	ed := edictAlloc()
	progsdat.Globals.Return[0] = int32(ed)
}

//export PF_Remove
func PF_Remove() {
	ed := progsdat.Globals.Parm0[0]
	edictFree(int(ed))
}

//export PF_Find
func PF_Find() {
	e := progsdat.Globals.Parm0[0]
	f := progsdat.Globals.Parm1[0]
	s, err := progsdat.String(progsdat.Globals.Parm2[0])
	if err != nil {
		runError("PF_Find: bad search string")
		return
	}
	for e++; int(e) < sv.numEdicts; e++ {
		if edictNum(int(e)).Free {
			continue
		}
		ti := RawEntVarsI(int(e), int(f))
		t, err := progsdat.String(ti)
		if err != nil {
			continue
		}
		if t == s {
			progsdat.Globals.Return[0] = int32(e)
			return
		}
	}
	progsdat.Globals.Return[0] = 0
}

// precache_file is only used to copy  files with qcc, it does nothing
//export PF_precache_file
func PF_precache_file() {
	progsdat.Globals.Return[0] = progsdat.Globals.Parm0[0]
}

/*
// THERJAK
static void PR_CheckEmptyString(const char *s) {
  if (s[0] <= ' ') PR_RunError("Bad string");
}
*/

//export PF_precache_sound
func PF_precache_sound() {
	if sv.state != ServerStateLoading {
		runError("PF_Precache_*: Precache can only be done in spawn functions")
		return
	}

	si := progsdat.Globals.Parm0[0]
	progsdat.Globals.Return[0] = si
	s, err := progsdat.String(si)
	if err != nil {
		// same result as PR_CheckEmptyString
	}
	//PR_CheckEmptyString(s);

	exist := func(s string) bool {
		for _, e := range sv.soundPrecache {
			if e == s {
				return true
			}
		}
		return false
	}
	if exist(s) {
		return
	}
	if len(sv.soundPrecache) >= 2048 {
		runError("PF_precache_sound: overflow")
		return
	}
	sv.soundPrecache = append(sv.soundPrecache, s)
}

//export PF_precache_model
func PF_precache_model() {
	if sv.state != ServerStateLoading {
		runError("PF_Precache_*: Precache can only be done in spawn functions")
		return
	}

	si := progsdat.Globals.Parm0[0]
	progsdat.Globals.Return[0] = si
	s, err := progsdat.String(si)
	if err != nil {
		runError("Bad string")
		return
	}

	exist := func(s string) bool {
		for _, e := range sv.modelPrecache {
			if e == s {
				return true
			}
		}
		return false
	}
	if exist(s) {
		return
	}
	if len(sv.modelPrecache) >= 2048 {
		runError("PF_precache_sound: overflow")
		return
	}
	sv.modelPrecache = append(sv.modelPrecache, s)
	m, ok := models[s]
	if !ok {
		loadModel(s)
		m, ok = models[s]
		if !ok {
			log.Printf("Model could not be loaded: %s", s)
			return
		}
	}
	sv.models = append(sv.models, m)
}

//export PF_coredump
func PF_coredump() {
	edictPrintEdicts()
}

//export PF_eprint
func PF_eprint() {
	edictPrint(int(progsdat.Globals.Parm0[0]))
}

//export PF_traceon
func PF_traceon() {
	vmTraceOn()
}

//export PF_traceoff
func PF_traceoff() {
	vmTraceOff()
}

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
	end[2] -= 256

	trace := svMove(start, mins, maxs, end, MOVE_NORMAL, ent)

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

//export PF_lightstyle
func PF_lightstyle() {
	style := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	vi := progsdat.Globals.Parm1[0]
	val, err := progsdat.String(vi)
	if err != nil {
		log.Printf("Invalid light style: %v", err)
		return
	}

	sv.lightStyles[style] = val

	// send message to all clients on this server
	if sv.state != ServerStateActive {
		return
	}

	for _, c := range sv_clients {
		if c.active || c.spawned {
			c.msg.WriteChar(server.LightStyle)
			c.msg.WriteChar(style)
			c.msg.WriteString(val)
		}
	}
}

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
	entnum := int(progsdat.Globals.Parm0[0])
	f := float32(0)
	if checkBottom(entnum) {
		f = 1
	}
	progsdat.Globals.Returnf()[0] = f
}

// Writes new values for v_forward, v_up, and v_right based on angles makevectors(vector)
//export PF_makevectors
func PF_makevectors() {
	v := vec.VFromA(*progsdat.Globals.Parm0f())
	f, r, u := vec.AngleVectors(v)
	progsdat.Globals.VForward = f
	progsdat.Globals.VRight = r
	progsdat.Globals.VUp = u
}

//export PF_pointcontents
func PF_pointcontents() {
	v := vec.VFromA(*progsdat.Globals.Parm0f())
	pc := pointContents(v)
	progsdat.Globals.Returnf()[0] = float32(pc)
}

//export PF_nextent
func PF_nextent() {
	i := progsdat.Globals.Parm0[0]
	for {
		i++
		if int(i) == sv.numEdicts {
			progsdat.Globals.Return[0] = 0
			return
		}
		if edictNum(int(i)).Free {
			progsdat.Globals.Return[0] = i
			return
		}
	}
}

// Pick a vector for the player to shoot along
//export PF_aim
func PF_aim() {
	const DAMAGE_AIM = 2
	ent := int(progsdat.Globals.Parm0[0])
	ev := EntVars(ent)
	// variable set but not used
	// speed := progsdat.RawGlobalsF[progs.OffsetParm1]

	start := vec.VFromA(ev.Origin)
	start[2] += 20

	// try sending a trace straight
	dir := vec.VFromA(progsdat.Globals.VForward)
	end := vec.Add(start, dir.Scale(2048))
	tr := svMove(start, vec.Vec3{}, vec.Vec3{}, end, MOVE_NORMAL, ent)
	if tr.entp != 0 {
		tev := EntVars(int(tr.entn))
		if tev.TakeDamage == DAMAGE_AIM &&
			(!cvars.TeamPlay.Bool() || tev.Team <= 0 || ev.Team != tev.Team) {
			*progsdat.Globals.Returnf() = progsdat.Globals.VForward
			return
		}
	}

	// try all possible entities
	bestdir := dir
	bestdist := cvars.ServerAim.Value()
	bestent := -1

	for check := 1; check < sv.numEdicts; check++ {
		cev := EntVars(check)
		if cev.TakeDamage != DAMAGE_AIM {
			continue
		}
		if check == ent {
			continue
		}
		if cvars.TeamPlay.Bool() && ev.Team > 0 && ev.Team == cev.Team {
			// don't aim at teammate
			continue
		}
		end := vec.Vec3{
			cev.Origin[0] + 0.5*(cev.Mins[0]+cev.Maxs[0]),
			cev.Origin[1] + 0.5*(cev.Mins[1]+cev.Maxs[1]),
			cev.Origin[2] + 0.5*(cev.Mins[2]+cev.Maxs[2]),
		}
		dir = vec.Sub(end, start)
		dir = dir.Normalize()
		vforward := vec.VFromA(progsdat.Globals.VForward)
		dist := vec.Dot(dir, vforward)
		if dist < bestdist {
			// to far to turn
			continue
		}
		tr := svMove(start, vec.Vec3{}, vec.Vec3{}, end, MOVE_NORMAL, ent)
		if int(tr.entn) == check {
			// can shoot at this one
			bestdist = dist
			bestent = check
		}
	}

	if bestent >= 0 {
		bev := EntVars(bestent)
		borigin := vec.VFromA(bev.Origin)
		eorigin := vec.VFromA(ev.Origin)
		dir := vec.Sub(borigin, eorigin)
		vforward := vec.VFromA(progsdat.Globals.VForward)
		dist := vec.Dot(dir, vforward)
		end := vforward.Scale(dist)
		end[2] = dir[2]
		end = end.Normalize()
		*progsdat.Globals.Returnf() = end
	} else {
		*progsdat.Globals.Returnf() = bestdir
	}
}

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
		sv_clients[writeClient()].msg.WriteAngle(msg, sv.protocolFlags)
	case MSG_INIT:
		sv.signon.WriteAngle(msg, sv.protocolFlags)
	case MSG_BROADCAST:
		sv.datagram.WriteAngle(msg, sv.protocolFlags)
	case MSG_ALL:
		sv.reliableDatagram.WriteAngle(msg, sv.protocolFlags)
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
		sv_clients[writeClient()].msg.WriteCoord(msg, sv.protocolFlags)
	case MSG_INIT:
		sv.signon.WriteCoord(msg, sv.protocolFlags)
	case MSG_BROADCAST:
		sv.datagram.WriteCoord(msg, sv.protocolFlags)
	case MSG_ALL:
		sv.reliableDatagram.WriteCoord(msg, sv.protocolFlags)
	default:
		runError("WriteDest: bad destination")
	}
}

//export PF_WriteString
func PF_WriteString() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	i := progsdat.Globals.Parm1[0]
	msg, err := progsdat.String(i)
	if err != nil {
		runError("PF_WriteString: bad string")
		return
	}
	switch dest {
	case MSG_ONE:
		sv_clients[writeClient()].msg.WriteString(msg)
	case MSG_INIT:
		sv.signon.WriteString(msg)
	case MSG_BROADCAST:
		sv.datagram.WriteString(msg)
	case MSG_ALL:
		sv.reliableDatagram.WriteString(msg)
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

//export PF_makestatic
func PF_makestatic() {
	bits := 0

	ent := int(progsdat.Globals.Parm0[0])
	e := edictNum(ent)

	// don't send invisible static entities
	if e.Alpha == server.EntityAlphaZero {
		edictFree(ent)
		return
	}
	ev := EntVars(ent)

	m, err := progsdat.String(ev.Model)
	if err != nil {
		log.Printf("Error in PF_makstatic: %v", err)
		return
	}
	mi := sv.ModelIndex(m)
	frame := int(ev.Frame)
	if sv.protocol == protocol.NetQuake {
		if mi&0xFF00 != 0 ||
			frame&0xFF00 != 0 {
			edictFree(ent)
			// can't display the correct model & frame, so don't show it at all
			return
		}
	} else {
		if mi&0xFF00 != 0 {
			bits |= server.EntityBaselineLargeModel
		}
		if frame&0xFF00 != 0 {
			bits |= server.EntityBaselineLargeFrame
		}
		if e.Alpha != server.EntityAlphaDefault {
			bits |= server.EntityBaselineAlpha
		}
	}

	if bits != 0 {
		sv.signon.WriteByte(server.SpawnStatic2)
		sv.signon.WriteByte(bits)
	} else {
		sv.signon.WriteByte(server.SpawnStatic)
	}

	if bits&server.EntityBaselineLargeModel != 0 {
		sv.signon.WriteShort(mi)
	} else {
		sv.signon.WriteByte(mi)
	}

	if bits&server.EntityBaselineLargeFrame != 0 {
		sv.signon.WriteShort(frame)
	} else {
		sv.signon.WriteByte(frame)
	}

	sv.signon.WriteByte(int(ev.ColorMap))
	sv.signon.WriteByte(int(ev.Skin))
	for i := 0; i < 3; i++ {
		sv.signon.WriteCoord(ev.Origin[i], sv.protocolFlags)
		sv.signon.WriteAngle(ev.Angles[i], sv.protocolFlags)
	}

	if bits&server.EntityBaselineAlpha != 0 {
		sv.signon.WriteByte(int(e.Alpha))
	}

	// throw the entity away now
	edictFree(ent)
}

//export PF_setspawnparms
func PF_setspawnparms() {
	i := int(progsdat.Globals.Parm0[0])
	if i < 1 || i > svs.maxClients {
		runError("Entity is not a client")
		return
	}

	// copy spawn parms out of the client_t
	client := sv_clients[i-1]

	for i := 0; i < NUM_SPAWN_PARMS; i++ {
		progsdat.Globals.Parm[i] = client.spawnParams[i]
	}
}

//export PF_Fixme
func PF_Fixme() {
	runError("unimplemented builtin")
}

//export PF_changelevel
func PF_changelevel() {
	// make sure we don't issue two changelevels
	if svs.changeLevelIssued {
		return
	}
	svs.changeLevelIssued = true

	i := progsdat.Globals.Parm0[0]
	s, err := progsdat.String(i)
	if err != nil {
		runError("PF_changelevel: bad level name")
		return
	}
	cbuf.AddText(fmt.Sprintf("changelevel %s\n", s))
}
