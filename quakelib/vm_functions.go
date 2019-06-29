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

// Dumps out self, then an error message.  The program is aborted and self is
// removed, but the level can continue.
func (v *virtualMachine) objError() {
	s := v.varString(0)
	fs := v.funcName()
	conlog.Printf("======OBJECT ERROR in %s:\n%s\n", fs, s)
	ed := int(v.prog.Globals.Self)
	edictPrint(ed)
	v.edictFree(ed)
}

// This is a TERMINAL error, which will kill off the entire server.
// Dumps self.
func (v *virtualMachine) terminalError() {
	s := v.varString(0)
	fs := v.funcName()
	conlog.Printf("======SERVER ERROR in %s:\n%s\n", fs, s)
	edictPrint(int(v.prog.Globals.Self))
	HostError("Program error")
}

func (v *virtualMachine) dprint() {
	s := v.varString(0)
	conlog.DPrintf(s)
}

// broadcast print to everyone on server
func (v *virtualMachine) bprint() {
	s := v.varString(0)
	SV_BroadcastPrint(s)
}

// single print to a specific client
func (v *virtualMachine) sprint() {
	e := int(v.prog.Globals.Parm0[0])
	s := v.varString(1)
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
func (v *virtualMachine) centerPrint() {
	e := int(v.prog.Globals.Parm0[0])
	s := v.varString(1)
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
func (v *virtualMachine) setOrigin() {
	e := int(v.prog.Globals.Parm0[0])
	ev := EntVars(e)
	ev.Origin = *v.prog.Globals.Parm1f()

	v.LinkEdict(e, false)
}

func setMinMaxSize(ev *progs.EntVars, min, max vec.Vec3) {
	if min[0] > max[0] || min[1] > max[1] || min[2] > max[2] {
		conlog.DPrintf("backwards mins/maxs")
	}
	ev.Mins = min
	ev.Maxs = max
	ev.Size = vec.Sub(max, min)
}

func (v *virtualMachine) setSize() {
	e := int(v.prog.Globals.Parm0[0])
	min := *v.prog.Globals.Parm1f()
	max := *v.prog.Globals.Parm2f()
	setMinMaxSize(EntVars(e), min, max)
	v.LinkEdict(e, false)
}

func (v *virtualMachine) setModel() {
	e := int(v.prog.Globals.Parm0[0])
	mi := v.prog.Globals.Parm1[0]
	m, err := v.prog.String(mi)
	if err != nil {
		v.runError("no precache: %d", mi)
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
		v.runError("no precache: %s", m)
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
	v.LinkEdict(e, false)
}

func (v *virtualMachine) normalize() {
	ve := vec.VFromA(*v.prog.Globals.Parm0f())
	*v.prog.Globals.Returnf() = ve.Normalize()
}

func (v *virtualMachine) vlen() {
	ve := vec.VFromA(*v.prog.Globals.Parm0f())
	l := ve.Length()
	v.prog.Globals.Returnf()[0] = l
}

func (v *virtualMachine) vecToYaw() {
	ve := vec.VFromA(*v.prog.Globals.Parm0f())
	yaw := func() float32 {
		if ve[0] == 0 && ve[1] == 0 {
			return 0
		}
		y := (math32.Atan2(ve[1], ve[0]) * 180) / math32.Pi
		y = math32.Trunc(y)
		if y < 0 {
			y += 360
		}
		return y
	}()
	v.prog.Globals.Returnf()[0] = yaw
}

func (v *virtualMachine) vecToAngles() {
	ve := vec.VFromA(*v.prog.Globals.Parm0f())
	yaw, pitch := func() (float32, float32) {
		if ve[0] == 0 && ve[1] == 0 {
			p := func() float32 {
				if ve[2] > 0 {
					return 90
				}
				return 270
			}()
			return 0, p
		}
		y := (math32.Atan2(ve[1], ve[0]) * 180) / math32.Pi
		y = math32.Trunc(y)
		if y < 0 {
			y += 360
		}
		forward := math32.Sqrt(ve[0]*ve[0] + ve[1]*ve[1])
		p := (math32.Atan2(ve[2], forward) * 180) / math32.Pi
		p = math32.Trunc(p)
		if p < 0 {
			p += 360
		}
		return y, p
	}()
	*v.prog.Globals.Returnf() = [3]float32{pitch, yaw, 0}
}

// Returns a number from 0 <= num < 1
func (v *virtualMachine) random() {
	v.prog.Globals.Returnf()[0] = rand.Float32()
}

func (v *virtualMachine) particle() {
	org := vec.VFromA(*v.prog.Globals.Parm0f())
	dir := vec.VFromA(*v.prog.Globals.Parm1f())
	color := v.prog.RawGlobalsF[progs.OffsetParm2]
	count := v.prog.RawGlobalsF[progs.OffsetParm3]
	sv.StartParticle(org, dir, int(color), int(count))
}

func (v *virtualMachine) ambientSound() {
	large := false
	pos := vec.VFromA(*v.prog.Globals.Parm0f())
	sample, err := v.prog.String(v.prog.Globals.Parm1[0])
	if err != nil {
		conlog.Printf("no precache: %v\n", pos)
		return
	}
	volume := v.prog.RawGlobalsF[progs.OffsetParm2] * 255
	attenuation := v.prog.RawGlobalsF[progs.OffsetParm3] * 64

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
func (v *virtualMachine) sound() {
	entity := v.prog.Globals.Parm0[0]
	channel := v.prog.RawGlobalsF[progs.OffsetParm1]
	sample, err := v.prog.String(v.prog.Globals.Parm2[0])
	if err != nil {
		v.runError("PF_sound: no sample")
		return
	}
	volume := v.prog.RawGlobalsF[progs.OffsetParm3] * 255
	attenuation := v.prog.RawGlobalsF[progs.OffsetParm4]

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

func (v *virtualMachine) doBreak() {
	conlog.Printf("break statement\n")
	runtime.Breakpoint()
}

// Used for use tracing and shot targeting
// Traces are blocked by bbox and exact bsp entityes, and also slide
// box entities if the tryents flag is set.
func (v *virtualMachine) traceline() {
	v1 := vec.VFromA(*v.prog.Globals.Parm0f())
	v2 := vec.VFromA(*v.prog.Globals.Parm1f())
	nomonsters := v.prog.RawGlobalsF[progs.OffsetParm2]
	ent := int(v.prog.Globals.Parm3[0])

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

	b2f := func(b bool) float32 {
		if b {
			return 1
		}
		return 0
	}
	v.prog.Globals.TraceAllSolid = b2f(trace.AllSolid)
	v.prog.Globals.TraceStartSolid = b2f(trace.StartSolid)
	v.prog.Globals.TraceFraction = trace.Fraction
	v.prog.Globals.TraceInWater = b2f(trace.InWater)
	v.prog.Globals.TraceInOpen = b2f(trace.InOpen)
	v.prog.Globals.TraceEndPos = trace.EndPos
	v.prog.Globals.TracePlaneNormal = trace.Plane.Normal
	v.prog.Globals.TracePlaneDist = trace.Plane.Distance
	if trace.EntPointer {
		v.prog.Globals.TraceEnt = int32(trace.EntNumber)
	} else {
		v.prog.Globals.TraceEnt = 0
	}
}

var (
	checkpvs []byte // to cache results
)

func init() {
	checkpvs = make([]byte, model.MAX_MAP_LEAFS/8)
}

func (v *virtualMachine) newcheckclient(check int) int {
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
func (v *virtualMachine) checkClient() {
	// find a new check if on a new frame
	if sv.time-sv.lastCheckTime >= 0.1 {
		sv.lastCheck = v.newcheckclient(sv.lastCheck)
		sv.lastCheckTime = sv.time
	}

	// return check if it might be visible
	ent := sv.lastCheck
	if edictNum(ent).Free || EntVars(ent).Health <= 0 {
		v.prog.Globals.Return[0] = 0
		return
	}

	// if current entity can't possibly see the check entity, return 0
	self := int(v.prog.Globals.Self)
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
		v.prog.Globals.Return[0] = 0
		return
	}

	// might be able to see it
	v.prog.Globals.Return[0] = int32(ent)
}

// Sends text over to the client's execution buffer
func (v *virtualMachine) stuffCmd() {
	entnum := int(v.prog.Globals.Parm0[0])
	if entnum < 1 || entnum > svs.maxClients {
		v.runError("Parm 0 not a client")
		return
	}
	str, err := v.prog.String(v.prog.Globals.Parm1[0])
	if err != nil {
		v.runError("stuffcmd: no string")
		return
	}

	c := sv_clients[entnum-1]
	c.msg.WriteByte(server.StuffText)
	c.msg.WriteString(str)
}

// Sends text over to the client's execution buffer
func (v *virtualMachine) localCmd() {
	str, err := v.prog.String(v.prog.Globals.Parm0[0])
	if err != nil {
		v.runError("localcmd: no string")
		return
	}
	cbuf.AddText(str)
}

func (v *virtualMachine) cvar() {
	str, err := v.prog.String(v.prog.Globals.Parm0[0])
	if err != nil {
		v.runError("PF_cvar: no string")
		return
	}
	v.prog.Globals.Returnf()[0] = CvarVariableValue(str)
}

func (v *virtualMachine) cvarSet() {
	name, err := v.prog.String(v.prog.Globals.Parm0[0])
	if err != nil {
		v.runError("PF_cvar_set: no name string")
		return
	}
	val, err := v.prog.String(v.prog.Globals.Parm1[0])
	if err != nil {
		v.runError("PF_cvar_set: no val string")
		return
	}
	cvarSet(name, val)
}

// Returns a chain of entities that have origins within a spherical area
func (v *virtualMachine) findRadius() {
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
		eorg := vec.Sub(org, vec.Scale(0.5, vec.Add(eo, vec.Add(mins, maxs))))
		if eorg.Length() > rad {
			continue
		}

		ev.Chain = chain
		chain = int32(ent)
	}

	progsdat.Globals.Return[0] = chain
}

func (v *virtualMachine) ftos() {
	ve := progsdat.RawGlobalsF[progs.OffsetParm0]
	s := func() string {
		iv := int(ve)
		if ve == float32(iv) {
			return fmt.Sprintf("%d", iv)
		}
		return fmt.Sprintf("%5.1f", ve)
	}()
	progsdat.Globals.Return[0] = progsdat.AddString(s)
}

func (v *virtualMachine) fabs() {
	f := progsdat.RawGlobalsF[progs.OffsetParm0]
	progsdat.Globals.Returnf()[0] = math32.Abs(f)
}

func (v *virtualMachine) vtos() {
	p := *progsdat.Globals.Parm0f()
	s := fmt.Sprintf("'%5.1f %5.1f %5.1f'", p[0], p[1], p[2])
	progsdat.Globals.Return[0] = progsdat.AddString(s)
}

func (v *virtualMachine) spawn() {
	ed := edictAlloc()
	progsdat.Globals.Return[0] = int32(ed)
}

func (v *virtualMachine) remove() {
	ed := progsdat.Globals.Parm0[0]
	v.edictFree(int(ed))
}

func (v *virtualMachine) find() {
	e := progsdat.Globals.Parm0[0]
	f := progsdat.Globals.Parm1[0]
	s, err := progsdat.String(progsdat.Globals.Parm2[0])
	if err != nil {
		v.runError("PF_Find: bad search string")
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
func (v *virtualMachine) precacheFile() {
	progsdat.Globals.Return[0] = progsdat.Globals.Parm0[0]
}

func (v *virtualMachine) precacheSound() {
	if sv.state != ServerStateLoading {
		v.runError("PF_Precache_*: Precache can only be done in spawn functions")
		return
	}

	si := progsdat.Globals.Parm0[0]
	progsdat.Globals.Return[0] = si
	s, err := progsdat.String(si)
	if err != nil {
		v.runError("Bad string")
		return
	}

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
		v.runError("PF_precache_sound: overflow")
		return
	}
	sv.soundPrecache = append(sv.soundPrecache, s)
}

func (v *virtualMachine) precacheModel() {
	if sv.state != ServerStateLoading {
		v.runError("PF_Precache_*: Precache can only be done in spawn functions")
		return
	}

	si := progsdat.Globals.Parm0[0]
	progsdat.Globals.Return[0] = si
	s, err := progsdat.String(si)
	if err != nil {
		v.runError("Bad string")
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
		v.runError("PF_precache_sound: overflow")
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

func (v *virtualMachine) coredump() {
	edictPrintEdicts()
}

func (v *virtualMachine) eprint() {
	edictPrint(int(progsdat.Globals.Parm0[0]))
}

func (v *virtualMachine) traceOn() {
	v.trace = true
}

func (v *virtualMachine) traceOff() {
	v.trace = false
}

func (v *virtualMachine) walkMove() {
	ent := int(v.prog.Globals.Self)
	yaw := v.prog.Globals.Parm0f()[0]
	dist := v.prog.Globals.Parm1f()[0]
	ev := EntVars(ent)

	if int(ev.Flags)&(FL_ONGROUND|FL_FLY|FL_SWIM) == 0 {
		(*(v.prog.Globals.Returnf()))[0] = 0
		return
	}

	yaw = yaw * math32.Pi * 2 / 360

	s, c := math32.Sincos(yaw)
	move := vec.Vec3{c * dist, s * dist, 0}

	// save program state, because monsterMoveStep may call other progs
	oldf := v.xfunction
	oldself := v.prog.Globals.Self

	r := v.monsterMoveStep(ent, move, true)
	if r {
		(*(v.prog.Globals.Returnf()))[0] = 1
	} else {
		(*(v.prog.Globals.Returnf()))[0] = 0
	}

	// restore program state
	v.xfunction = oldf
	v.prog.Globals.Self = oldself
}

func (v *virtualMachine) dropToFloor() {
	ent := int(progsdat.Globals.Self)
	ev := EntVars(ent)
	start := vec.VFromA(ev.Origin)
	mins := vec.VFromA(ev.Mins)
	maxs := vec.VFromA(ev.Maxs)
	end := vec.VFromA(ev.Origin)
	end[2] -= 256

	trace := svMove(start, mins, maxs, end, MOVE_NORMAL, ent)

	if trace.Fraction == 1 || trace.AllSolid {
		progsdat.Globals.Returnf()[0] = 0
	} else {
		ev.Origin = trace.EndPos
		v.LinkEdict(ent, false)
		ev.Flags = float32(int(ev.Flags) | FL_ONGROUND)
		ev.GroundEntity = int32(trace.EntNumber)
		progsdat.Globals.Returnf()[0] = 1
	}
}

func (v *virtualMachine) lightStyle() {
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

func (v *virtualMachine) rint() {
	f := progsdat.RawGlobalsF[progs.OffsetParm0]
	progsdat.Globals.Returnf()[0] = math.RoundToEven(f)
}

func (v *virtualMachine) floor() {
	f := progsdat.RawGlobalsF[progs.OffsetParm0]
	progsdat.Globals.Returnf()[0] = math32.Floor(f)
}

func (v *virtualMachine) ceil() {
	f := progsdat.RawGlobalsF[progs.OffsetParm0]
	progsdat.Globals.Returnf()[0] = math32.Ceil(f)
}

func (v *virtualMachine) checkBottom() {
	entnum := int(progsdat.Globals.Parm0[0])
	f := float32(0)
	if checkBottom(entnum) {
		f = 1
	}
	progsdat.Globals.Returnf()[0] = f
}

// Writes new values for v_forward, v_up, and v_right based on angles makevectors(vector)
func (v *virtualMachine) makeVectors() {
	ve := vec.VFromA(*progsdat.Globals.Parm0f())
	f, r, u := vec.AngleVectors(ve)
	progsdat.Globals.VForward = f
	progsdat.Globals.VRight = r
	progsdat.Globals.VUp = u
}

func (v *virtualMachine) pointContents() {
	ve := vec.VFromA(*progsdat.Globals.Parm0f())
	pc := pointContents(ve)
	progsdat.Globals.Returnf()[0] = float32(pc)
}

func (v *virtualMachine) nextEnt() {
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
func (v *virtualMachine) aim() {
	const DAMAGE_AIM = 2
	ent := int(progsdat.Globals.Parm0[0])
	ev := EntVars(ent)
	// variable set but not used
	// speed := progsdat.RawGlobalsF[progs.OffsetParm1]

	start := vec.VFromA(ev.Origin)
	start[2] += 20

	// try sending a trace straight
	dir := vec.VFromA(progsdat.Globals.VForward)
	end := vec.Add(start, vec.Scale(2048, dir))
	tr := svMove(start, vec.Vec3{}, vec.Vec3{}, end, MOVE_NORMAL, ent)
	if tr.EntPointer {
		tev := EntVars(int(tr.EntNumber))
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
		vforward := progsdat.Globals.VForward
		dist := vec.Dot(dir, vforward)
		if dist < bestdist {
			// to far to turn
			continue
		}
		tr := svMove(start, vec.Vec3{}, vec.Vec3{}, end, MOVE_NORMAL, ent)
		if tr.EntNumber == check {
			// can shoot at this one
			bestdist = dist
			bestent = check
		}
	}

	if bestent >= 0 {
		bev := EntVars(bestent)
		borigin := bev.Origin
		eorigin := ev.Origin
		dir := vec.Sub(borigin, eorigin)
		vforward := vec.Vec3(progsdat.Globals.VForward)
		dist := vec.Dot(dir, vforward)
		end := vec.Scale(dist, vforward)
		end[2] = dir[2]
		end = end.Normalize()
		*progsdat.Globals.Returnf() = end
	} else {
		*progsdat.Globals.Returnf() = bestdir
	}
}

// This was a major timewaster in progs
func (v *virtualMachine) changeYaw() {
	ent := int(progsdat.Globals.Self)
	changeYaw(ent)
}

const (
	MSG_BROADCAST = iota // unreliable to all
	MSG_ONE              // reliable to one
	MSG_ALL              // reliable to all
	MSG_INIT             // write to the init string
)

func (v *virtualMachine) writeClient() int {
	entnum := int(progsdat.Globals.MsgEntity)
	if entnum < 1 || entnum > svs.maxClients {
		v.runError("WriteDest: not a client")
	}
	return entnum - 1
}

func (v *virtualMachine) writeByte() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[v.writeClient()].msg.WriteByte(int(msg))
	case MSG_INIT:
		sv.signon.WriteByte(int(msg))
	case MSG_BROADCAST:
		sv.datagram.WriteByte(int(msg))
	case MSG_ALL:
		sv.reliableDatagram.WriteByte(int(msg))
	default:
		v.runError("WriteDest: bad destination")
	}
}

func (v *virtualMachine) writeChar() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[v.writeClient()].msg.WriteChar(int(msg))
	case MSG_INIT:
		sv.signon.WriteChar(int(msg))
	case MSG_BROADCAST:
		sv.datagram.WriteChar(int(msg))
	case MSG_ALL:
		sv.reliableDatagram.WriteChar(int(msg))
	default:
		v.runError("WriteDest: bad destination")
	}
}

func (v *virtualMachine) writeShort() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[v.writeClient()].msg.WriteShort(int(msg))
	case MSG_INIT:
		sv.signon.WriteShort(int(msg))
	case MSG_BROADCAST:
		sv.datagram.WriteShort(int(msg))
	case MSG_ALL:
		sv.reliableDatagram.WriteShort(int(msg))
	default:
		v.runError("WriteDest: bad destination")
	}
}

func (v *virtualMachine) writeLong() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[v.writeClient()].msg.WriteLong(int(msg))
	case MSG_INIT:
		sv.signon.WriteLong(int(msg))
	case MSG_BROADCAST:
		sv.datagram.WriteLong(int(msg))
	case MSG_ALL:
		sv.reliableDatagram.WriteLong(int(msg))
	default:
		v.runError("WriteDest: bad destination")
	}
}

func (v *virtualMachine) writeAngle() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[v.writeClient()].msg.WriteAngle(msg, sv.protocolFlags)
	case MSG_INIT:
		sv.signon.WriteAngle(msg, sv.protocolFlags)
	case MSG_BROADCAST:
		sv.datagram.WriteAngle(msg, sv.protocolFlags)
	case MSG_ALL:
		sv.reliableDatagram.WriteAngle(msg, sv.protocolFlags)
	default:
		v.runError("WriteDest: bad destination")
	}
}

func (v *virtualMachine) writeCoord() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[v.writeClient()].msg.WriteCoord(msg, sv.protocolFlags)
	case MSG_INIT:
		sv.signon.WriteCoord(msg, sv.protocolFlags)
	case MSG_BROADCAST:
		sv.datagram.WriteCoord(msg, sv.protocolFlags)
	case MSG_ALL:
		sv.reliableDatagram.WriteCoord(msg, sv.protocolFlags)
	default:
		v.runError("WriteDest: bad destination")
	}
}

func (v *virtualMachine) writeString() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	i := progsdat.Globals.Parm1[0]
	msg, err := progsdat.String(i)
	if err != nil {
		v.runError("PF_WriteString: bad string")
		return
	}
	switch dest {
	case MSG_ONE:
		sv_clients[v.writeClient()].msg.WriteString(msg)
	case MSG_INIT:
		sv.signon.WriteString(msg)
	case MSG_BROADCAST:
		sv.datagram.WriteString(msg)
	case MSG_ALL:
		sv.reliableDatagram.WriteString(msg)
	default:
		v.runError("WriteDest: bad destination")
	}
}

func (v *virtualMachine) writeEntity() {
	dest := int(progsdat.RawGlobalsF[progs.OffsetParm0])
	msg := progsdat.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		sv_clients[v.writeClient()].msg.WriteShort(int(msg))
	case MSG_INIT:
		sv.signon.WriteShort(int(msg))
	case MSG_BROADCAST:
		sv.datagram.WriteShort(int(msg))
	case MSG_ALL:
		sv.reliableDatagram.WriteShort(int(msg))
	default:
		v.runError("WriteDest: bad destination")
	}
}

func (v *virtualMachine) makeStatic() {
	bits := 0

	ent := int(progsdat.Globals.Parm0[0])
	e := edictNum(ent)

	// don't send invisible static entities
	if e.Alpha == server.EntityAlphaZero {
		v.edictFree(ent)
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
			v.edictFree(ent)
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
	v.edictFree(ent)
}

func (v *virtualMachine) setSpawnParms() {
	i := int(progsdat.Globals.Parm0[0])
	if i < 1 || i > svs.maxClients {
		v.runError("Entity is not a client")
		return
	}

	// copy spawn parms out of the client_t
	client := sv_clients[i-1]

	for i := 0; i < NUM_SPAWN_PARMS; i++ {
		progsdat.Globals.Parm[i] = client.spawnParams[i]
	}
}

func (v *virtualMachine) fixme() {
	v.runError("unimplemented builtin")
}

func (v *virtualMachine) changeLevel() {
	// make sure we don't issue two changelevels
	if svs.changeLevelIssued {
		return
	}
	svs.changeLevelIssued = true

	i := progsdat.Globals.Parm0[0]
	s, err := progsdat.String(i)
	if err != nil {
		v.runError("PF_changelevel: bad level name")
		return
	}
	cbuf.AddText(fmt.Sprintf("changelevel %s\n", s))
}

func (v *virtualMachine) moveToGoal() {
	v.monsterMoveToGoal()
}
