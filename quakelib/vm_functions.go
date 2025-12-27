// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"fmt"
	"log/slog"
	gmath "math"
	"runtime"
	"strings"

	"goquake/bsp"
	"goquake/cbuf"
	"goquake/cvars"
	"goquake/math"
	"goquake/math/vec"
	"goquake/model"
	"goquake/progs"
	"goquake/protocol"
	svc "goquake/protocol/server"
	"goquake/protos"

	"github.com/chewxy/math32"
)

const (
	saveGlobal = (1 << 15)
)

func (v *virtualMachine) LoadGameGlobals(g *protos.Globals) {
	for _, st := range g.GetStrings() {
		def, err := v.prog.FindGlobalDef(st.GetId())
		if err != nil {
			continue
		}
		id := v.prog.NewString(st.GetValue())
		v.prog.RawGlobalsI[def.Offset] = id
	}
	for _, fl := range g.GetFloats() {
		def, err := v.prog.FindGlobalDef(fl.GetId())
		if err != nil {
			continue
		}
		v.prog.RawGlobalsF[def.Offset] = fl.GetValue()
	}
	for _, ent := range g.GetEntities() {
		def, err := v.prog.FindGlobalDef(ent.GetId())
		if err != nil {
			continue
		}
		v.prog.RawGlobalsI[def.Offset] = ent.GetValue()
	}
}

func (v *virtualMachine) saveGlobalString(name string, offset uint16) *protos.StringDef {
	val := v.prog.RawGlobalsI[offset]
	s, _ := v.prog.String(val)
	return protos.StringDef_builder{
		Id:    name,
		Value: s,
	}.Build()
}

func (v *virtualMachine) saveGlobalFloat(name string, offset uint16) *protos.FloatDef {
	val := v.prog.RawGlobalsF[offset]
	return protos.FloatDef_builder{
		Id:    name,
		Value: val,
	}.Build()
}

func (v *virtualMachine) saveGlobalEntity(name string, offset uint16) *protos.EntityDef {
	val := v.prog.RawGlobalsI[offset]
	return protos.EntityDef_builder{
		Id:    name,
		Value: val,
	}.Build()
}

func (v *virtualMachine) SaveGameGlobals() *protos.Globals {
	entities := []*protos.EntityDef{}
	floats := []*protos.FloatDef{}
	ostrings := []*protos.StringDef{}
	for _, d := range v.prog.GlobalDefs {
		t := d.Type
		if t&saveGlobal == 0 {
			continue
		}
		t &^= saveGlobal
		name, _ := v.prog.String(d.SName)
		offset := d.Offset
		switch t {
		case progs.EV_String:
			ostrings = append(ostrings, v.saveGlobalString(name, offset))
		case progs.EV_Float:
			floats = append(floats, v.saveGlobalFloat(name, offset))
		case progs.EV_Entity:
			entities = append(entities, v.saveGlobalEntity(name, offset))
		default:
			// progs.EV_Vector, progs.EV_Field, progs.EV_Function, progs.EV_Void:
			// progs.EV_Pointer:
			// progs.EV_Bad:
			continue
		}
	}
	return protos.Globals_builder{
		Entities: entities,
		Floats:   floats,
		Strings:  ostrings,
	}.Build()
}

func (v *virtualMachine) loadGameEntVars(idx int, e *protos.Edict) {
	entvars.Clear(idx)
	// TODO: keyname == "alpha"
	for _, st := range e.GetStrings() {
		def, err := v.prog.FindFieldDef(st.GetId())
		if err != nil {
			slog.Warn("No string", slog.String("ID", st.GetId()))
			continue
		}
		id := v.prog.NewString(st.GetValue())
		entvars.SetRawI(int32(idx), int32(def.Offset), id)
	}
	for _, fl := range e.GetFloats() {
		def, err := v.prog.FindFieldDef(fl.GetId())
		if err != nil {
			slog.Warn("No float", slog.String("ID", fl.GetId()))
			continue
		}
		entvars.SetRawF(int32(idx), int32(def.Offset), fl.GetValue())
	}
	for _, ent := range e.GetEntities() {
		def, err := v.prog.FindFieldDef(ent.GetId())
		if err != nil {
			slog.Warn("No field", slog.String("ID", ent.GetId()))
			continue
		}
		entvars.SetRawI(int32(idx), int32(def.Offset), ent.GetValue())
	}
	for _, fnc := range e.GetFunctions() {
		def, err := v.prog.FindFieldDef(fnc.GetId())
		if err != nil {
			continue
		}
		fidx, err := v.prog.FindFunction(fnc.GetValue())
		if err != nil {
			continue
		}
		entvars.SetRawI(int32(idx), int32(def.Offset), int32(fidx))
	}
	for _, field := range e.GetFields() {
		def, err := v.prog.FindFieldDef(field.GetId())
		if err != nil {
			continue
		}
		vdef, err := v.prog.FindFieldDef(field.GetValue())
		if err != nil {
			continue
		}
		entvars.SetRawI(int32(idx), int32(def.Offset), int32(vdef.Offset))
	}
	for _, vector := range e.GetVectors() {
		def, err := v.prog.FindFieldDef(vector.GetId())
		if err != nil {
			continue
		}
		val := vector.GetValue()
		entvars.SetRawF(int32(idx), int32(def.Offset), val.GetX())
		entvars.SetRawF(int32(idx), int32(def.Offset+1), val.GetY())
		entvars.SetRawF(int32(idx), int32(def.Offset+2), val.GetZ())
	}

}

func (v *virtualMachine) saveEVString(idx int, name string, offset uint16) (*protos.StringDef, bool) {
	val := entvars.RawI(int32(idx), int32(offset))
	if val == 0 {
		return nil, false
	}
	s, _ := v.prog.String(val)
	return protos.StringDef_builder{
		Id:    name,
		Value: s,
	}.Build(), true
}

func (v *virtualMachine) saveEVFloat(idx int, name string, offset uint16) (*protos.FloatDef, bool) {
	val := entvars.RawF(int32(idx), int32(offset))
	if val == 0 {
		return nil, false
	}
	return protos.FloatDef_builder{
		Id:    name,
		Value: val,
	}.Build(), true
}

func (v *virtualMachine) saveEVEntity(idx int, name string, offset uint16) (*protos.EntityDef, bool) {
	val := entvars.RawI(int32(idx), int32(offset))
	if val == 0 {
		return nil, false
	}
	return protos.EntityDef_builder{
		Id:    name,
		Value: val,
	}.Build(), true
}

func (v *virtualMachine) saveEVVector(idx int, name string, offset uint16) (*protos.VectorDef, bool) {
	x := entvars.RawF(int32(idx), int32(offset))
	y := entvars.RawF(int32(idx), int32(offset+1))
	z := entvars.RawF(int32(idx), int32(offset+2))
	if x == 0 && y == 0 && z == 0 {
		return nil, false
	}
	vec := protos.Vector_builder{X: x, Y: y, Z: z}.Build()
	return protos.VectorDef_builder{
		Id:    name,
		Value: vec,
	}.Build(), true
}

func (v *virtualMachine) saveEVField(idx int, name string, offset uint16) (*protos.FieldDef, bool) {
	s := ""
	val := entvars.RawI(int32(idx), int32(offset))
	if val == 0 {
		return nil, false
	}
	for _, f := range v.prog.FieldDefs {
		if int32(f.Offset) == val {
			s, _ = v.prog.String(f.SName)
			break
		}
	}
	return protos.FieldDef_builder{
		Id:    name,
		Value: s,
	}.Build(), true
}

func (v *virtualMachine) saveEVFunction(idx int, name string, offset uint16) (*protos.FunctionDef, bool) {
	val := entvars.RawI(int32(idx), int32(offset))
	if val == 0 {
		return nil, false
	}
	sname := v.prog.Functions[val].SName
	s, _ := v.prog.String(sname)
	return protos.FunctionDef_builder{
		Id:    name,
		Value: s,
	}.Build(), true
}

func (v *virtualMachine) saveGameEntVars(idx int) *protos.Edict {
	entities := []*protos.EntityDef{}
	fields := []*protos.FieldDef{}
	floats := []*protos.FloatDef{}
	functions := []*protos.FunctionDef{}
	ostrings := []*protos.StringDef{}
	vectors := []*protos.VectorDef{}
	for _, d := range v.prog.FieldDefs[1:] {
		t := d.Type
		t &^= saveGlobal
		name, _ := v.prog.String(d.SName)
		if strings.HasPrefix(name, "_") {
			// skip _x, _y, _z vars
			continue
		}
		offset := d.Offset
		switch t {
		case progs.EV_String:
			if s, ok := v.saveEVString(idx, name, offset); ok {
				ostrings = append(ostrings, s)
			}
		case progs.EV_Float:
			if f, ok := v.saveEVFloat(idx, name, offset); ok {
				floats = append(floats, f)
			}
		case progs.EV_Entity:
			if e, ok := v.saveEVEntity(idx, name, offset); ok {
				entities = append(entities, e)
			}
		case progs.EV_Vector:
			if ve, ok := v.saveEVVector(idx, name, offset); ok {
				vectors = append(vectors, ve)
			}
		case progs.EV_Field:
			if f, ok := v.saveEVField(idx, name, offset); ok {
				fields = append(fields, f)
			}
		case progs.EV_Function:
			if f, ok := v.saveEVFunction(idx, name, offset); ok {
				functions = append(functions, f)
			}
		default:
			// progs.EV_Void: // this was written but never read
			// progs.EV_Pointer:
			// progs.EV_Bad:
			continue
		}
	}
	// TODO: alpha
	return protos.Edict_builder{
		Entities:  entities,
		Fields:    fields,
		Floats:    floats,
		Functions: functions,
		Strings:   ostrings,
		Vectors:   vectors,
	}.Build()
}

// Dumps out self, then an error message.  The program is aborted and self is
// removed, but the level can continue.
func (v *virtualMachine) objError(s *Server) error {
	st := v.varString(0)
	fs := v.funcName()
	slog.Error("======OBJECT ERROR======", slog.String("function", fs), slog.String("var", st))
	ed := int(v.prog.Globals.Self)
	s.edictPrint(ed)
	v.edictFree(ed, s)
	return nil
}

// This is a TERMINAL error, which will kill off the entire server.
// Dumps self.
func (v *virtualMachine) terminalError(s *Server) error {
	st := v.varString(0)
	fs := v.funcName()
	slog.Error("======SERVER ERROR======", slog.String("function", fs), slog.String("var", st))
	s.edictPrint(int(v.prog.Globals.Self))
	return fmt.Errorf("Program error")
}

func (v *virtualMachine) dprint(s *Server) error {
	st := v.varString(0)
	slog.Debug(st)
	return nil
}

// broadcast print to everyone on server
func (v *virtualMachine) bprint(s *Server) error {
	st := v.varString(0)
	s.BroadcastPrint(st)
	return nil
}

// single print to a specific client
func (v *virtualMachine) sprint(s *Server) error {
	e := int(v.prog.Globals.Parm0[0])
	st := v.varString(1)
	if e < 1 || e > svs.maxClients {
		slog.Error("tried to sprint to a non-client", slog.Int("client", e))
		return nil
	}
	e--
	c := sv_clients[e]
	c.msg.WriteChar(svc.Print)
	c.msg.WriteString(st)
	return nil
}

// single print to a specific client
func (v *virtualMachine) centerPrint(s *Server) error {
	e := int(v.prog.Globals.Parm0[0])
	st := v.varString(1)
	if e < 1 || e > svs.maxClients {
		slog.Error("tried to sprint to a non-client", slog.Int("client", e))
		return nil
	}
	e--
	c := sv_clients[e]
	c.msg.WriteChar(svc.CenterPrint)
	c.msg.WriteString(st)
	return nil
}

/*
This is the only valid way to move an object without using the physics
of the world (setting velocity and waiting).  Directly changing origin
will not set internal links correctly, so clipping would be messed up.

This should be called when an object is spawned, and then only if it is
teleported.
*/
func (v *virtualMachine) setOrigin(s *Server) error {
	e := int(v.prog.Globals.Parm0[0])
	ev := entvars.Get(e)
	ev.Origin = *v.prog.Globals.Parm1f()

	if err := v.LinkEdict(e, false, s); err != nil {
		return err
	}
	return nil
}

// TODO
func setMinMaxSize(ev *progs.EntVars, min, max vec.Vec3) {
	if min[0] > max[0] || min[1] > max[1] || min[2] > max[2] {
		slog.Warn("backwards mins/maxs")
	}
	ev.Mins = min
	ev.Maxs = max
	ev.Size = vec.Sub(max, min)
}

func (v *virtualMachine) setSize(s *Server) error {
	e := int(v.prog.Globals.Parm0[0])
	min := *v.prog.Globals.Parm1f()
	max := *v.prog.Globals.Parm2f()
	setMinMaxSize(entvars.Get(e), min, max)
	if err := v.LinkEdict(e, false, s); err != nil {
		return err
	}
	return nil
}

func (v *virtualMachine) setModel(s *Server) error {
	e := int(v.prog.Globals.Parm0[0])
	mi := v.prog.Globals.Parm1[0]
	m, err := v.prog.String(mi)
	if err != nil {
		slog.Error("no precache", slog.Int("model", int(mi)))
		v.abort()
		return errProgram
	}

	idx := -1
	for i, mp := range s.modelPrecache {
		if mp == m {
			idx = i
			break
		}
	}
	if idx == -1 {
		slog.Error("no precache", slog.String("model", m))
		v.abort()
		return errProgram
	}

	ev := entvars.Get(e)
	ev.Model = mi
	ev.ModelIndex = float32(idx)

	mod := s.models[idx]
	if mod != nil {
		switch qm := mod.(type) {
		case *bsp.Model:
			// log.Printf("ModBrush")
			// log.Printf("mins: %v, maxs: %v", mod.ClipMins, mod.ClipMaxs)
			setMinMaxSize(ev, qm.ClipMins, qm.ClipMaxs)
		default:
			// log.Printf("!!!ModBrush")
			setMinMaxSize(ev, mod.Mins(), mod.Maxs())
		}
	} else {
		// log.Printf("No Mod")
		setMinMaxSize(ev, vec.Vec3{}, vec.Vec3{})
	}
	if err := v.LinkEdict(e, false, s); err != nil {
		return err
	}
	return nil
}

// TODO
func (v *virtualMachine) normalize(s *Server) error {
	ve := vec.VFromA(*v.prog.Globals.Parm0f())
	l := 1 / gmath.Sqrt(vec.DoublePrecDot(ve, ve))

	*v.prog.Globals.Returnf() = vec.Vec3{
		float32(float64(ve[0]) * l),
		float32(float64(ve[1]) * l),
		float32(float64(ve[2]) * l),
	}
	return nil
}

// TODO
func (v *virtualMachine) vlen(s *Server) error {
	ve := vec.VFromA(*v.prog.Globals.Parm0f())
	l := gmath.Sqrt(vec.DoublePrecDot(ve, ve))
	v.prog.Globals.Returnf()[0] = float32(l)
	return nil
}

// TODO
func (v *virtualMachine) vecToYaw(s *Server) error {
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
	return nil
}

// TODO
func (v *virtualMachine) vecToAngles(s *Server) error {
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
		y := (gmath.Atan2(float64(ve[1]), float64(ve[0])) * 180) / gmath.Pi
		y = gmath.Trunc(y)
		if y < 0 {
			y += 360
		}
		forward := gmath.Sqrt(float64(ve[0])*float64(ve[0]) + float64(ve[1])*float64(ve[1]))
		p := (gmath.Atan2(float64(ve[2]), forward) * 180) / gmath.Pi
		p = gmath.Trunc(p)
		if p < 0 {
			p += 360
		}
		return float32(y), float32(p)
	}()
	*v.prog.Globals.Returnf() = [3]float32{pitch, yaw, 0}
	return nil
}

// TODO
// Returns a number from 0 <= num < 1
func (v *virtualMachine) random(s *Server) error {
	v.prog.Globals.Returnf()[0] = sRand.Float32()
	return nil
}

func (v *virtualMachine) particle(s *Server) error {
	org := vec.VFromA(*v.prog.Globals.Parm0f())
	dir := vec.VFromA(*v.prog.Globals.Parm1f())
	color := v.prog.RawGlobalsF[progs.OffsetParm2]
	count := v.prog.RawGlobalsF[progs.OffsetParm3]
	s.StartParticle(org, dir, int(color), int(count))
	return nil
}

func (v *virtualMachine) ambientSound(s *Server) error {
	large := false
	pos := vec.VFromA(*v.prog.Globals.Parm0f())
	sample, err := v.prog.String(v.prog.Globals.Parm1[0])
	if err != nil {
		slog.Error("ambientSound: no precache", slog.Any("pos", pos))
		return nil
	}
	volume := v.prog.RawGlobalsF[progs.OffsetParm2] * 255
	attenuation := v.prog.RawGlobalsF[progs.OffsetParm3] * 64

	// check to see if samp was properly precached
	soundnum := func() int {
		for i, m := range s.soundPrecache {
			if m == sample {
				return i
			}
		}
		return -1
	}()

	if soundnum == -1 {
		slog.Error("ambientSound: no precache", slog.Any("pos", pos), slog.String("sample", sample))
		return nil
	}

	if soundnum > 255 {
		if s.protocol == protocol.NetQuake {
			return nil // don't send any info protocol can't support
		} else {
			large = true
		}
	}

	// add an svc_spawnambient command to the level signon packet
	if large {
		s.signon.WriteByte(svc.SpawnStaticSound2)
	} else {
		s.signon.WriteByte(svc.SpawnStaticSound)
	}

	s.signon.WriteCoord(pos[0], s.protocolFlags)
	s.signon.WriteCoord(pos[1], s.protocolFlags)
	s.signon.WriteCoord(pos[2], s.protocolFlags)

	if large {
		s.signon.WriteShort(soundnum)
	} else {
		s.signon.WriteByte(soundnum)
	}

	s.signon.WriteByte(int(volume))
	s.signon.WriteByte(int(attenuation))
	return nil
}

// Each entity can have eight independent sound sources, like voice,
// weapon, feet, etc.
// Channel 0 is an auto-allocate channel, the others override anything
// already running on that entity/channel pair.
// An attenuation of 0 will play full volume everywhere in the level.
// Larger attenuations will drop off.
func (v *virtualMachine) sound(s *Server) error {
	entity := v.prog.Globals.Parm0[0]
	channel := v.prog.RawGlobalsF[progs.OffsetParm1]
	sample, err := v.prog.String(v.prog.Globals.Parm2[0])
	if err != nil {
		slog.Error("PF_sound: no sample")
		v.abort()
		return errProgram
	}
	volume := v.prog.RawGlobalsF[progs.OffsetParm3] * 255
	attenuation := v.prog.RawGlobalsF[progs.OffsetParm4]

	if volume < 0 || volume > 255 {
		return fmt.Errorf("SV_StartSound: volume = %v", volume)
	}

	if attenuation < 0 || attenuation > 4 {
		return fmt.Errorf("SV_StartSound: attenuation = %v", attenuation)
	}

	if channel < 0 || channel > 7 {
		return fmt.Errorf("SV_StartSound: channel = %v", channel)
	}
	if err := s.StartSound(int(entity), int(channel), int(volume), sample, attenuation); err != nil {
		return err
	}
	return nil
}

func (v *virtualMachine) doBreak(s *Server) error {
	slog.Info("break statement")
	runtime.Breakpoint()
	return nil
}

// Used for use tracing and shot targeting
// Traces are blocked by bbox and exact bsp entityes, and also slide
// box entities if the tryents flag is set.
func (v *virtualMachine) traceline(s *Server) error {
	v1 := vec.VFromA(*v.prog.Globals.Parm0f())
	v2 := vec.VFromA(*v.prog.Globals.Parm1f())
	nomonsters := v.prog.RawGlobalsF[progs.OffsetParm2]
	ent := int(v.prog.Globals.Parm3[0])

	// FIXME FIXME FIXME: Why do we hit this with certain progs.dat ??
	if cvars.Developer.Bool() {
		if v1 != v1 || v2 != v2 {
			slog.Error("NAN in traceline",
				slog.Any("v1", v1),
				slog.Any("v2", v2),
				slog.Int("entity", ent))
		}
	}

	if v1 != v1 {
		v1 = vec.Vec3{}
	}
	if v2 != v2 {
		v2 = vec.Vec3{}
	}

	t := svMove(v1, vec.Vec3{}, vec.Vec3{}, v2, int(nomonsters), ent, s)

	b2f := func(b bool) float32 {
		if b {
			return 1
		}
		return 0
	}
	v.prog.Globals.TraceAllSolid = b2f(t.AllSolid)
	v.prog.Globals.TraceStartSolid = b2f(t.StartSolid)
	v.prog.Globals.TraceFraction = t.Fraction
	v.prog.Globals.TraceInWater = b2f(t.InWater)
	v.prog.Globals.TraceInOpen = b2f(t.InOpen)
	v.prog.Globals.TraceEndPos = t.EndPos
	v.prog.Globals.TracePlaneNormal = t.Plane.Normal
	v.prog.Globals.TracePlaneDist = t.Plane.Distance
	if t.EntPointer {
		v.prog.Globals.TraceEnt = int32(t.EntNumber)
	} else {
		v.prog.Globals.TraceEnt = 0
	}
	return nil
}

var (
	checkpvs []byte // to cache results
)

func init() {
	checkpvs = make([]byte, bsp.MaxMapLeafs/8)
}

func (v *virtualMachine) newcheckclient(check int, s *Server) int {
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

		if s.edicts[ent].Free {
			continue
		}
		ev := entvars.Get(ent)
		if ev.Health <= 0 {
			continue
		}
		if int(ev.Flags)&FL_NOTARGET != 0 {
			continue
		}

		// anything that is a client, or has a client as an enemy
		break
	}

	ev := entvars.Get(ent)
	// get the PVS for the entity
	org := vec.Add(ev.Origin, ev.ViewOfs)
	leaf, _ := s.worldModel.PointInLeaf(org)
	pvs := s.worldModel.LeafPVS(leaf)

	// we care only about the first (len(s.worldModel.Leafs)+7)/8 bytes
	copy(checkpvs, pvs)
	return i
}

// Returns a client (or object that has a client enemy) that would be a
// valid target.
// If there are more than one valid options, they are cycled each frame
// If (self.origin + self.viewofs) is not in the PVS of the current target,
// it is not returned at all.
func (v *virtualMachine) checkClient(s *Server) error {
	// find a new check if on a new frame
	if s.time-s.lastCheckTime >= 0.1 {
		s.lastCheck = v.newcheckclient(s.lastCheck, s)
		s.lastCheckTime = s.time
	}

	// return check if it might be visible
	ent := s.lastCheck
	if s.edicts[ent].Free || entvars.Get(ent).Health <= 0 {
		v.prog.Globals.Return[0] = 0
		return nil
	}

	// if current entity can't possibly see the check entity, return 0
	self := int(v.prog.Globals.Self)
	es := entvars.Get(self)
	view := vec.Add(es.Origin, es.ViewOfs)
	leaf, _ := s.worldModel.PointInLeaf(view)
	leafNum := -2
	for i, l := range s.worldModel.Leafs {
		if l == leaf {
			leafNum = i - 1 // -1 to remove the solid 0 leaf
		}
	}
	if leafNum == -2 {
		slog.Warn("checkclient: Got leafnum -2", slog.Int("len(leafs)", len(s.worldModel.Leafs)))
	}

	if (leafNum < 0) || (checkpvs[leafNum/8]&(1<<(uint(leafNum)&7)) == 0) {
		v.prog.Globals.Return[0] = 0
		return nil
	}

	// might be able to see it
	v.prog.Globals.Return[0] = int32(ent)
	return nil
}

// Sends text over to the client's execution buffer
func (v *virtualMachine) stuffCmd(s *Server) error {
	entnum := int(v.prog.Globals.Parm0[0])
	if entnum < 1 || entnum > svs.maxClients {
		slog.Error("Parm 0 not a client")
		v.abort()
		return errProgram
	}
	str, err := v.prog.String(v.prog.Globals.Parm1[0])
	if err != nil {
		slog.Error("stuffcmd: no string")
		v.abort()
		return errProgram
	}

	c := sv_clients[entnum-1]
	c.msg.WriteByte(svc.StuffText)
	c.msg.WriteString(str)
	return nil
}

// Sends text over to the client's execution buffer
func (v *virtualMachine) localCmd(s *Server) error {
	str, err := v.prog.String(v.prog.Globals.Parm0[0])
	if err != nil {
		slog.Error("localcmd: no string")
		v.abort()
		return errProgram
	}
	cbuf.AddText(str)
	return nil
}

func (v *virtualMachine) cvar(s *Server) error {
	str, err := v.prog.String(v.prog.Globals.Parm0[0])
	if err != nil {
		slog.Error("PF_cvar: no string")
		v.abort()
		return errProgram
	}
	f := func(n string) float32 {
		if cv, ok := (*commandVars)[n]; ok {
			return cv.Value()
		}
		return 0
	}
	v.prog.Globals.Returnf()[0] = f(str)
	return nil
}

func (v *virtualMachine) cvarSet(s *Server) error {
	name, err := v.prog.String(v.prog.Globals.Parm0[0])
	if err != nil {
		slog.Error("PF_cvar_set: no name string")
		v.abort()
		return errProgram
	}
	val, err := v.prog.String(v.prog.Globals.Parm1[0])
	if err != nil {
		slog.Error("PF_cvar_set: no val string")
		v.abort()
		return errProgram
	}
	if cv, ok := (*commandVars)[name]; ok {
		cv.SetByString(val)
	} else {
		slog.Warn("Cvar_Set: variable not found", slog.String("name", name))
	}
	return nil
}

// Returns a chain of entities that have origins within a spherical area
func (v *virtualMachine) findRadius(s *Server) error {
	chain := int32(0)
	org := vec.VFromA(*v.prog.Globals.Parm0f())
	rad := v.prog.RawGlobalsF[progs.OffsetParm1]

	for ent := 1; ent < s.numEdicts; ent++ {
		if s.edicts[ent].Free {
			continue
		}
		ev := entvars.Get(ent)
		if ev.Solid == SOLID_NOT {
			continue
		}
		eo := vec.VFromA(ev.Origin)
		mins := vec.VFromA(ev.Mins)
		maxs := vec.VFromA(ev.Maxs)
		eorg := vec.Sub(org, vec.Add(eo, vec.Scale(0.5, vec.Add(mins, maxs))))
		if eorg.Length() > rad {
			continue
		}

		ev.Chain = chain
		chain = int32(ent)
	}

	v.prog.Globals.Return[0] = chain
	return nil
}

// TODO
func (v *virtualMachine) ftos(s *Server) error {
	ve := v.prog.RawGlobalsF[progs.OffsetParm0]
	st := func() string {
		iv := int(ve)
		if ve == float32(iv) {
			return fmt.Sprintf("%d", iv)
		}
		return fmt.Sprintf("%5.1f", ve)
	}()
	v.prog.Globals.Return[0] = v.prog.AddString(st)
	return nil
}

// TODO
func (v *virtualMachine) fabs(s *Server) error {
	f := v.prog.RawGlobalsF[progs.OffsetParm0]
	v.prog.Globals.Returnf()[0] = math32.Abs(f)
	return nil
}

// TODO
func (v *virtualMachine) vtos(s *Server) error {
	p := *v.prog.Globals.Parm0f()
	st := fmt.Sprintf("'%5.1f %5.1f %5.1f'", p[0], p[1], p[2])
	v.prog.Globals.Return[0] = v.prog.AddString(st)
	return nil
}

func (v *virtualMachine) spawn(s *Server) error {
	ed, err := s.edictAlloc()
	if err != nil {
		return err
	}
	v.prog.Globals.Return[0] = int32(ed)
	return nil
}

func (v *virtualMachine) remove(s *Server) error {
	ed := v.prog.Globals.Parm0[0]
	v.edictFree(int(ed), s)
	return nil
}

func (v *virtualMachine) find(s *Server) error {
	e := v.prog.Globals.Parm0[0]
	f := v.prog.Globals.Parm1[0]
	st, err := v.prog.String(v.prog.Globals.Parm2[0])
	if err != nil {
		slog.Error("PF_Find: bad search string")
		v.abort()
		return errProgram
	}
	for e++; int(e) < s.numEdicts; e++ {
		if s.edicts[e].Free {
			continue
		}
		ti := entvars.RawI(e, f)
		t, err := v.prog.String(ti)
		if err != nil {
			continue
		}
		if t == st {
			v.prog.Globals.Return[0] = int32(e)
			return nil
		}
	}
	v.prog.Globals.Return[0] = 0
	return nil
}

func (v *virtualMachine) finaleFinished(s *Server) error {
	// Used by 2021 release
	// Expected to return a bool
	v.prog.Globals.Return[0] = 0
	return nil
}

// precache_file is only used to copy  files with qcc, it does nothing
func (v *virtualMachine) precacheFile(s *Server) error {
	v.prog.Globals.Return[0] = v.prog.Globals.Parm0[0]
	return nil
}

func (v *virtualMachine) precacheSound(s *Server) error {
	if s.state != ServerStateLoading {
		slog.Error("PF_Precache_*: Precache can only be done in spawn functions")
		v.abort()
		return errProgram
	}

	si := v.prog.Globals.Parm0[0]
	v.prog.Globals.Return[0] = si
	st, err := v.prog.String(si)
	if err != nil {
		slog.Error("precacheSound: Bad string", slog.Any("err", err))
		v.abort()
		return errProgram
	}

	exist := func(st string) bool {
		for _, e := range s.soundPrecache {
			if e == st {
				return true
			}
		}
		return false
	}
	if exist(st) {
		return nil
	}
	if len(s.soundPrecache) >= 2048 {
		slog.Error("PF_precache_sound: overflow")
		v.abort()
		return errProgram
	}
	s.soundPrecache = append(s.soundPrecache, st)
	return nil
}

func (v *virtualMachine) precacheModel(s *Server) error {
	if s.state != ServerStateLoading {
		slog.Error("PF_Precache_*: Precache can only be done in spawn functions")
		v.abort()
		return errProgram
	}

	si := v.prog.Globals.Parm0[0]
	v.prog.Globals.Return[0] = si
	st, err := v.prog.String(si)
	if err != nil {
		slog.Error("precacheModel: Bad string", slog.Any("err", err))
		v.abort()
		return errProgram
	}

	exist := func(st string) bool {
		for _, e := range s.modelPrecache {
			if e == st {
				return true
			}
		}
		return false
	}
	if exist(st) {
		return nil
	}
	if len(s.modelPrecache) >= 2048 {
		slog.Error("PF_precache_sound: overflow")
		v.abort()
		return errProgram
	}
	s.modelPrecache = append(s.modelPrecache, st)

	m, err := svLoadModel(st)
	if err != nil {
		slog.Error("Model could not be loaded", slog.String("model", st), slog.Any("err", err))
		return nil
	}
	s.models = append(s.models, m)
	return nil
}

// create in SpawnServer
var sv_models map[string]model.Model

func svLoadModel(name string) (model.Model, error) {
	m, ok := sv_models[name]
	if ok {
		return m, nil
	}
	mods, err := model.Load(name)
	if err != nil {
		return nil, err
	}
	for _, m := range mods {
		sv_models[m.Name()] = m
	}
	m, ok = sv_models[name]
	if ok {
		return m, nil
	}
	return nil, fmt.Errorf("not found")
}

func (v *virtualMachine) coredump(s *Server) error {
	s.edictPrintEdicts()
	return nil
}

func (v *virtualMachine) eprint(s *Server) error {
	s.edictPrint(int(v.prog.Globals.Parm0[0]))
	return nil
}

func (v *virtualMachine) traceOn(s *Server) error {
	v.trace = true
	return nil
}

func (v *virtualMachine) traceOff(s *Server) error {
	v.trace = false
	return nil
}

func (v *virtualMachine) walkMove(s *Server) error {
	ent := int(v.prog.Globals.Self)
	yaw := v.prog.Globals.Parm0f()[0]
	dist := v.prog.Globals.Parm1f()[0]
	ev := entvars.Get(ent)

	if int(ev.Flags)&(FL_ONGROUND|FL_FLY|FL_SWIM) == 0 {
		(*(v.prog.Globals.Returnf()))[0] = 0
		return nil
	}

	yaw = yaw * math32.Pi * 2 / 360

	si, co := math32.Sincos(yaw)
	move := vec.Vec3{co * dist, si * dist, 0}

	// save program state, because monsterMoveStep may call other progs
	oldf := v.xfunction
	oldself := v.prog.Globals.Self

	r, err := v.monsterMoveStep(ent, move, true, s)
	if err != nil {
		return err
	}
	if r {
		(*(v.prog.Globals.Returnf()))[0] = 1
	} else {
		(*(v.prog.Globals.Returnf()))[0] = 0
	}

	// restore program state
	v.xfunction = oldf
	v.prog.Globals.Self = oldself
	return nil
}

func (v *virtualMachine) dropToFloor(s *Server) error {
	ent := int(v.prog.Globals.Self)
	ev := entvars.Get(ent)
	start := vec.VFromA(ev.Origin)
	mins := vec.VFromA(ev.Mins)
	maxs := vec.VFromA(ev.Maxs)
	end := vec.VFromA(ev.Origin)
	end[2] -= 256

	t := svMove(start, mins, maxs, end, MOVE_NORMAL, ent, s)

	if t.Fraction == 1 || t.AllSolid {
		v.prog.Globals.Returnf()[0] = 0
	} else {
		ev.Origin = t.EndPos
		if err := v.LinkEdict(ent, false, s); err != nil {
			return err
		}
		ev.Flags = float32(int(ev.Flags) | FL_ONGROUND)
		ev.GroundEntity = int32(t.EntNumber)
		v.prog.Globals.Returnf()[0] = 1
	}
	return nil
}

func (v *virtualMachine) lightStyle(s *Server) error {
	style := int(v.prog.RawGlobalsF[progs.OffsetParm0])
	vi := v.prog.Globals.Parm1[0]
	val, err := v.prog.String(vi)
	if err != nil {
		slog.Warn("Invalid light style", slog.Any("err", err))
		return nil
	}

	s.lightStyles[style] = val

	// send message to all clients on this server
	if s.state != ServerStateActive {
		return nil
	}

	for _, c := range sv_clients {
		if c.active || c.spawned {
			c.msg.WriteChar(svc.LightStyle)
			c.msg.WriteChar(style)
			c.msg.WriteString(val)
		}
	}
	return nil
}

// TODO
func (v *virtualMachine) rint(s *Server) error {
	f := v.prog.RawGlobalsF[progs.OffsetParm0]
	v.prog.Globals.Returnf()[0] = math.RoundToEven(f)
	return nil
}

// TODO
func (v *virtualMachine) floor(s *Server) error {
	f := v.prog.RawGlobalsF[progs.OffsetParm0]
	v.prog.Globals.Returnf()[0] = math32.Floor(f)
	return nil
}

// TODO
func (v *virtualMachine) ceil(s *Server) error {
	f := v.prog.RawGlobalsF[progs.OffsetParm0]
	v.prog.Globals.Returnf()[0] = math32.Ceil(f)
	return nil
}

func (v *virtualMachine) checkBottom(s *Server) error {
	entnum := int(v.prog.Globals.Parm0[0])
	f := float32(0)
	if checkBottom(entnum, s) {
		f = 1
	}
	v.prog.Globals.Returnf()[0] = f
	return nil
}

// TODO
// Writes new values for v_forward, v_up, and v_right based on angles makevectors(vector)
func (v *virtualMachine) makeVectors(s *Server) error {
	ve := vec.VFromA(*v.prog.Globals.Parm0f())
	f, r, u := vec.AngleVectors(ve)
	v.prog.Globals.VForward = f
	v.prog.Globals.VRight = r
	v.prog.Globals.VUp = u
	return nil
}

func (v *virtualMachine) pointContents(s *Server) error {
	ve := vec.VFromA(*v.prog.Globals.Parm0f())
	pc := pointContents(ve, s.worldModel)
	v.prog.Globals.Returnf()[0] = float32(pc)
	return nil
}

func (v *virtualMachine) nextEnt(s *Server) error {
	i := v.prog.Globals.Parm0[0]
	for {
		i++
		if int(i) == s.numEdicts {
			v.prog.Globals.Return[0] = 0
			return nil
		}
		if s.edicts[i].Free {
			v.prog.Globals.Return[0] = i
			return nil
		}
	}
}

// Pick a vector for the player to shoot along
func (v *virtualMachine) aim(s *Server) error {
	const DAMAGE_AIM = 2
	ent := int(v.prog.Globals.Parm0[0])
	ev := entvars.Get(ent)
	// variable set but not used
	// speed := v.prog.RawGlobalsF[progs.OffsetParm1]

	start := vec.VFromA(ev.Origin)
	start[2] += 20

	// try sending a trace straight
	dir := vec.VFromA(v.prog.Globals.VForward)
	end := vec.Add(start, vec.Scale(2048, dir))
	tr := svMove(start, vec.Vec3{}, vec.Vec3{}, end, MOVE_NORMAL, ent, s)
	if tr.EntPointer {
		tev := entvars.Get(int(tr.EntNumber))
		if tev.TakeDamage == DAMAGE_AIM &&
			(!cvars.TeamPlay.Bool() || tev.Team <= 0 || ev.Team != tev.Team) {
			*v.prog.Globals.Returnf() = v.prog.Globals.VForward
			return nil
		}
	}

	// try all possible entities
	bestdir := dir
	bestdist := cvars.ServerAim.Value()
	bestent := -1

	for check := 1; check < s.numEdicts; check++ {
		cev := entvars.Get(check)
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
		vforward := v.prog.Globals.VForward
		dist := vec.Dot(dir, vforward)
		if dist < bestdist {
			// to far to turn
			continue
		}
		tr := svMove(start, vec.Vec3{}, vec.Vec3{}, end, MOVE_NORMAL, ent, s)
		if tr.EntNumber == check {
			// can shoot at this one
			bestdist = dist
			bestent = check
		}
	}

	if bestent >= 0 {
		bev := entvars.Get(bestent)
		borigin := bev.Origin
		eorigin := ev.Origin
		dir := vec.Sub(borigin, eorigin)
		vforward := vec.Vec3(v.prog.Globals.VForward)
		dist := vec.Dot(dir, vforward)
		end := vec.Scale(dist, vforward)
		end[2] = dir[2]
		end = end.Normalize()
		*v.prog.Globals.Returnf() = end
	} else {
		*v.prog.Globals.Returnf() = bestdir
	}
	return nil
}

// This was a major timewaster in progs
func (v *virtualMachine) changeYaw(s *Server) error {
	ent := int(v.prog.Globals.Self)
	changeYaw(ent)
	return nil
}

const (
	MSG_BROADCAST = iota // unreliable to all
	MSG_ONE              // reliable to one
	MSG_ALL              // reliable to all
	MSG_INIT             // write to the init string
)

func (v *virtualMachine) writeClient() (*SVClient, error) {
	entnum := int(v.prog.Globals.MsgEntity)
	if entnum < 1 || entnum > svs.maxClients {
		slog.Error("WriteDest: not a client")
		v.abort()
		return nil, errProgram
	}
	return sv_clients[entnum-1], nil
}

func (v *virtualMachine) writeByte(s *Server) error {
	dest := int(v.prog.RawGlobalsF[progs.OffsetParm0])
	msg := v.prog.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		if c, err := v.writeClient(); err != nil {
			return err
		} else {
			c.msg.WriteByte(int(msg))
		}
	case MSG_INIT:
		s.signon.WriteByte(int(msg))
	case MSG_BROADCAST:
		s.datagram.WriteByte(int(msg))
	case MSG_ALL:
		s.reliableDatagram.WriteByte(int(msg))
	default:
		slog.Error("WriteDest: bad destination")
		v.abort()
		return errProgram
	}
	return nil
}

func (v *virtualMachine) writeChar(s *Server) error {
	dest := int(v.prog.RawGlobalsF[progs.OffsetParm0])
	msg := v.prog.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		if c, err := v.writeClient(); err != nil {
			return err
		} else {
			c.msg.WriteChar(int(msg))
		}
	case MSG_INIT:
		s.signon.WriteChar(int(msg))
	case MSG_BROADCAST:
		s.datagram.WriteChar(int(msg))
	case MSG_ALL:
		s.reliableDatagram.WriteChar(int(msg))
	default:
		slog.Error("WriteDest: bad destination")
		v.abort()
		return errProgram
	}
	return nil
}

func (v *virtualMachine) writeShort(s *Server) error {
	dest := int(v.prog.RawGlobalsF[progs.OffsetParm0])
	msg := v.prog.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		if c, err := v.writeClient(); err != nil {
			return err
		} else {
			c.msg.WriteShort(int(msg))
		}
	case MSG_INIT:
		s.signon.WriteShort(int(msg))
	case MSG_BROADCAST:
		s.datagram.WriteShort(int(msg))
	case MSG_ALL:
		s.reliableDatagram.WriteShort(int(msg))
	default:
		slog.Error("WriteDest: bad destination")
		v.abort()
		return errProgram
	}
	return nil
}

func (v *virtualMachine) writeLong(s *Server) error {
	dest := int(v.prog.RawGlobalsF[progs.OffsetParm0])
	msg := v.prog.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		if c, err := v.writeClient(); err != nil {
			return err
		} else {
			c.msg.WriteLong(int(msg))
		}
	case MSG_INIT:
		s.signon.WriteLong(int(msg))
	case MSG_BROADCAST:
		s.datagram.WriteLong(int(msg))
	case MSG_ALL:
		s.reliableDatagram.WriteLong(int(msg))
	default:
		slog.Error("WriteDest: bad destination")
		v.abort()
		return errProgram
	}
	return nil
}

func (v *virtualMachine) writeAngle(s *Server) error {
	dest := int(v.prog.RawGlobalsF[progs.OffsetParm0])
	msg := v.prog.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		if c, err := v.writeClient(); err != nil {
			return err
		} else {
			c.msg.WriteAngle(msg, s.protocolFlags)
		}
	case MSG_INIT:
		s.signon.WriteAngle(msg, s.protocolFlags)
	case MSG_BROADCAST:
		s.datagram.WriteAngle(msg, s.protocolFlags)
	case MSG_ALL:
		s.reliableDatagram.WriteAngle(msg, s.protocolFlags)
	default:
		slog.Error("WriteDest: bad destination")
		v.abort()
		return errProgram
	}
	return nil
}

func (v *virtualMachine) writeCoord(s *Server) error {
	dest := int(v.prog.RawGlobalsF[progs.OffsetParm0])
	msg := v.prog.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		if c, err := v.writeClient(); err != nil {
			return err
		} else {
			c.msg.WriteCoord(msg, s.protocolFlags)
		}
	case MSG_INIT:
		s.signon.WriteCoord(msg, s.protocolFlags)
	case MSG_BROADCAST:
		s.datagram.WriteCoord(msg, s.protocolFlags)
	case MSG_ALL:
		s.reliableDatagram.WriteCoord(msg, s.protocolFlags)
	default:
		slog.Error("WriteDest: bad destination")
		v.abort()
		return errProgram
	}
	return nil
}

func (v *virtualMachine) writeString(s *Server) error {
	dest := int(v.prog.RawGlobalsF[progs.OffsetParm0])
	i := v.prog.Globals.Parm1[0]
	msg, err := v.prog.String(i)
	if err != nil {
		slog.Error("PF_WriteString: bad string")
		v.abort()
		return errProgram
	}
	switch dest {
	case MSG_ONE:
		if c, err := v.writeClient(); err != nil {
			return err
		} else {
			c.msg.WriteString(msg)
		}
	case MSG_INIT:
		s.signon.WriteString(msg)
	case MSG_BROADCAST:
		s.datagram.WriteString(msg)
	case MSG_ALL:
		s.reliableDatagram.WriteString(msg)
	default:
		slog.Error("WriteDest: bad destination")
		v.abort()
		return errProgram
	}
	return nil
}

func (v *virtualMachine) writeEntity(s *Server) error {
	dest := int(v.prog.RawGlobalsF[progs.OffsetParm0])
	msg := v.prog.RawGlobalsF[progs.OffsetParm1]
	switch dest {
	case MSG_ONE:
		if c, err := v.writeClient(); err != nil {
			return err
		} else {
			c.msg.WriteShort(int(msg))
		}
	case MSG_INIT:
		s.signon.WriteShort(int(msg))
	case MSG_BROADCAST:
		s.datagram.WriteShort(int(msg))
	case MSG_ALL:
		s.reliableDatagram.WriteShort(int(msg))
	default:
		slog.Error("WriteDest: bad destination")
		v.abort()
		return errProgram
	}
	return nil
}

func (v *virtualMachine) makeStatic(s *Server) error {
	bits := 0

	ent := int(v.prog.Globals.Parm0[0])
	e := &s.edicts[ent]

	// don't send invisible static entities
	if e.Alpha == svc.EntityAlphaZero {
		v.edictFree(ent, s)
		return nil
	}
	ev := entvars.Get(ent)

	m, err := v.prog.String(ev.Model)
	if err != nil {
		slog.Warn("Error in PF_makstatic", slog.Any("err", err))
		return nil
	}
	mi := s.ModelIndex(m)
	frame := int(ev.Frame)
	if s.protocol == protocol.NetQuake {
		if mi&0xFF00 != 0 ||
			frame&0xFF00 != 0 {
			v.edictFree(ent, s)
			// can't display the correct model & frame, so don't show it at all
			return nil
		}
	} else {
		if mi&0xFF00 != 0 {
			bits |= svc.EntityBaselineLargeModel
		}
		if frame&0xFF00 != 0 {
			bits |= svc.EntityBaselineLargeFrame
		}
		if e.Alpha != svc.EntityAlphaDefault {
			bits |= svc.EntityBaselineAlpha
		}
	}

	if bits != 0 {
		s.signon.WriteByte(svc.SpawnStatic2)
		s.signon.WriteByte(bits)
	} else {
		s.signon.WriteByte(svc.SpawnStatic)
	}

	if bits&svc.EntityBaselineLargeModel != 0 {
		s.signon.WriteShort(mi)
	} else {
		s.signon.WriteByte(mi)
	}

	if bits&svc.EntityBaselineLargeFrame != 0 {
		s.signon.WriteShort(frame)
	} else {
		s.signon.WriteByte(frame)
	}

	s.signon.WriteByte(int(ev.ColorMap))
	s.signon.WriteByte(int(ev.Skin))
	for i := 0; i < 3; i++ {
		s.signon.WriteCoord(ev.Origin[i], s.protocolFlags)
		s.signon.WriteAngle(ev.Angles[i], s.protocolFlags)
	}

	if bits&svc.EntityBaselineAlpha != 0 {
		s.signon.WriteByte(int(e.Alpha))
	}

	// throw the entity away now
	v.edictFree(ent, s)
	return nil
}

func (v *virtualMachine) setSpawnParms(s *Server) error {
	i := int(v.prog.Globals.Parm0[0])
	if i < 1 || i > svs.maxClients {
		slog.Error("Entity is not a client")
		v.abort()
		return errProgram
	}

	// copy spawn parms out of the client_t
	client := sv_clients[i-1]

	for i := 0; i < NUM_SPAWN_PARMS; i++ {
		v.prog.Globals.Parm[i] = client.spawnParams[i]
	}
	return nil
}

func (v *virtualMachine) fixme(s *Server) error {
	slog.Error("unimplemented builtin")
	v.abort()
	return errProgram
}

func (v *virtualMachine) changeLevel(s *Server) error {
	// make sure we don't issue two changelevels
	if v.changeLevelIssued {
		return nil
	}
	v.changeLevelIssued = true

	i := v.prog.Globals.Parm0[0]
	st, err := v.prog.String(i)
	if err != nil {
		slog.Error("PF_changelevel: bad level name")
		v.abort()
		return errProgram
	}
	cbuf.AddText(fmt.Sprintf("changelevel %s\n", st))
	return nil
}

func (v *virtualMachine) moveToGoal(s *Server) error {
	if err := v.monsterMoveToGoal(s); err != nil {
		return err
	}
	return nil
}
