// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"fmt"
	"log"
	"log/slog"
	"runtime/debug"
	"time"

	"goquake/bsp"
	cmdl "goquake/commandline"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/math"
	"goquake/math/vec"
	"goquake/model"
	"goquake/net"
	"goquake/progs"
	"goquake/protocol"
	svc "goquake/protocol/server"
	"goquake/protos"

	"github.com/chewxy/math32"
	"google.golang.org/protobuf/proto"
)

const (
	SOLID_NOT = iota
	SOLID_TRIGGER
	SOLID_BBOX
	SOLID_SLIDEBOX
	SOLID_BSP
)

const (
	FL_FLY = 1 << iota
	FL_SWIM
	FL_CONVEYOR
	FL_CLIENT
	FL_INWATER
	FL_MONSTER
	FL_GODMODE
	FL_NOTARGET
	FL_ITEM
	FL_ONGROUND
	FL_PARTIALGROUND
	FL_WATERJUMP
	FL_JUMPRELEASED
)

const (
	NUM_SPAWN_PARMS = 16
)

type ServerStatic struct {
	maxClients      int
	maxClientsLimit int
	// stores data between map changes, while actually flags (aka as int) it is
	// only modified to anything but 0 inside qc. So no point in matching the
	// flag intent but better to match the use as qc number which is a float.
	serverFlags float32
}

type ServerState bool

const (
	ServerStateLoading = ServerState(false)
	ServerStateActive  = ServerState(true)
)

type Server struct {
	// active is used by the client to check if the server is local
	// needs to be changed to a client state which gets the state change by channel
	active   bool
	paused   bool
	loadGame bool

	time          float32
	lastCheck     int
	lastCheckTime float32

	datagram         net.Message
	reliableDatagram net.Message
	signon           net.Message

	numEdicts int
	maxEdicts int

	edicts []Edict

	protocol      int
	protocolFlags uint32

	state ServerState // some actions are only valid during load

	soundPrecache []string
	lightStyles   [64]string

	name string // map name

	// TODO(therjak): merge modelPrecache and models into a map[string]model.Model?
	//                we also need an index based access for entvars
	//                the empty 0 element can probably be dropped
	modelPrecache []string
	models        []model.Model
	worldModel    *bsp.Model
}

var (
	svs         = ServerStatic{}
	svTODO      = NewServer()
	host_client int
	progsdat    *progs.LoadedProg
)

var (
	msgBuf       = net.Message{}
	msgBufMaxLen = 0
)

func NewServer() *Server {
	s := &Server{
		models: make([]model.Model, 1),
	}
	cvars.ServerGravity.SetCallback(s.notifyCallback)
	cvars.ServerFriction.SetCallback(s.notifyCallback)
	cvars.ServerMaxSpeed.SetCallback(s.notifyCallback)
	cvars.TimeLimit.SetCallback(s.notifyCallback)
	cvars.FragLimit.SetCallback(s.notifyCallback)
	cvars.TeamPlay.SetCallback(s.notifyCallback)
	cvars.NoExit.SetCallback(s.notifyCallback)
	return s
}

func (s *Server) notifyCallback(cv *cvar.Cvar) {
	if !s.Active() {
		return
	}
	s.BroadcastPrintf("\"%s\" changed to \"%s\"\n", cv.Name(), cv.String())
}

func (s *Server) Active() bool {
	if s == nil {
		return false
	}
	return s.active
}

func (s *Server) StartParticle(org, dir vec.Vec3, color, count int) {
	if s.datagram.Len() > protocol.MaxDatagram-18 {
		return
	}
	p := protos.Particle_builder{
		Origin:    protos.Coord_builder{X: org[0], Y: org[1], Z: org[2]}.Build(),
		Direction: protos.Coord_builder{X: dir[0], Y: dir[1], Z: dir[2]}.Build(),
		Count:     int32(count),
		Color:     int32(color),
	}.Build()
	svc.WriteParticle(p, s.protocolFlags, &s.datagram)
}

func (s *Server) SendDatagram(sc *SVClient) (bool, error) {
	b := msgBuf.Bytes()
	// If there is space add the server datagram
	if len(b)+s.datagram.Len() < protocol.MaxDatagram {
		b = append(b, s.datagram.Bytes()...)
	}
	if sc.netConnection.SendUnreliableMessage(b) == -1 {
		if err := sc.Drop(true, s); err != nil {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func (s *Server) SendReliableDatagram() {
	b := s.reliableDatagram.Bytes()
	for _, cl := range sv_clients {
		if cl.active {
			cl.msg.WriteBytes(b)
		}
	}
	s.reliableDatagram.ClearMessage()
}

func (s *Server) sendReconnect() {
	SendReconnectToAll()
}

/*
Each entity can have eight independent sound sources, like voice,
weapon, feet, etc.

Channel 0 is an auto-allocate channel, the others override anything
already running on that entity/channel pair.

An attenuation of 0 will play full volume everywhere in the level.
Larger attenuations will drop off.  (max 4 attenuation)
*/
func (s *Server) StartSound(entity, channel, volume int, sample string, attenuation float32) error {
	if volume < 0 || volume > 255 {
		return fmt.Errorf("SV_StartSound: volume = %d", volume)
	}
	if attenuation < 0 || attenuation > 4 {
		return fmt.Errorf("SV_StartSound: attenuation = %f", attenuation)
	}
	if channel < 0 || channel > 7 {
		return fmt.Errorf("SV_StartSound: channel = %d", channel)
	}
	if s.datagram.Len() > protocol.MaxDatagram-21 {
		return nil
	}
	for soundnum, m := range s.soundPrecache {
		if m == sample {
			s.sendStartSound(entity, channel, volume, soundnum, attenuation)
			return nil
		}
	}
	slog.Warn("SV_StartSound: sound not precacheed", slog.String("sound", sample))
	return nil
}

func (s *Server) sendStartSound(entity, channel, volume, soundnum int, attenuation float32) {
	ev := entvars.Get(entity)
	snd := protos.Sound_builder{
		Entity:   int32(entity),
		SoundNum: int32(soundnum),
		Channel:  int32(channel),
		Origin: protos.Coord_builder{
			X: ev.Origin[0] + 0.5*(ev.Mins[0]+ev.Maxs[0]),
			Y: ev.Origin[1] + 0.5*(ev.Mins[1]+ev.Maxs[1]),
			Z: ev.Origin[2] + 0.5*(ev.Mins[2]+ev.Maxs[2]),
		}.Build(),
	}.Build()
	if volume != 255 {
		snd.SetVolume(int32(volume))
	}
	if attenuation != 1.0 {
		snd.SetAttenuation(int32(64 * attenuation))
	}
	svc.WriteSound(snd, s.protocol, s.protocolFlags, &s.datagram)
}

func (s *Server) CleanupEntvarEffects() {
	for i := 1; i < s.numEdicts; i++ {
		ev := entvars.Get(i)
		eff := int(ev.Effects)
		ev.Effects = float32(eff &^ svc.EffectMuzzleFlash)
	}
}

func (s *Server) WriteClientdataToMessage(player int) {
	e := entvars.Get(player)
	alpha := s.edicts[player].Alpha
	flags := s.protocolFlags
	if e.DmgTake != 0 || e.DmgSave != 0 {
		other := entvars.Get(int(e.DmgInflictor))
		p := protos.Coord_builder{
			X: other.Origin[0] + 0.5*(other.Mins[0]+other.Maxs[0]),
			Y: other.Origin[1] + 0.5*(other.Mins[1]+other.Maxs[1]),
			Z: other.Origin[2] + 0.5*(other.Mins[2]+other.Maxs[2]),
		}.Build()
		dmg := protos.Damage_builder{
			Armor:    int32(e.DmgSave),
			Blood:    int32(e.DmgTake),
			Position: p,
		}.Build()
		svc.WriteDamage(dmg, s.protocol, flags, &msgBuf)
		e.DmgTake = 0
		e.DmgSave = 0
	}

	// send the current viewpos offset from the view entity
	s.setIdealPitch(player) // how much to loop up/down ideally

	// a fixangle might get lost in a dropped packet.  Oh well.
	if e.FixAngle != 0 {
		a := protos.Coord_builder{
			X: e.Angles[0],
			Y: e.Angles[1],
			Z: e.Angles[2],
		}.Build()
		svc.WriteSetAngle(a, s.protocol, flags, &msgBuf)
		e.FixAngle = 0
	}

	clientData := &protos.ClientData{}
	clientData.SetPunchAngle(&protos.IntCoord{})
	clientData.SetVelocity(&protos.IntCoord{})

	if e.ViewOfs[2] != svc.DEFAULT_VIEWHEIGHT {
		clientData.SetViewHeight(int32(e.ViewOfs[2]))
	}
	clientData.SetIdealPitch(int32(e.IdealPitch))
	// stuff the sigil bits into the high bits of items for sbar, or else mix in items2
	items := func() int {
		/*
			  		v := GetEdictFieldValue(e, "items2")
						if v != 0 {
							return e.Items | v.float << 23
						}
		*/
		return int(e.Items) | int(progsdat.Globals.ServerFlags)<<28
	}()
	clientData.SetItems(uint32(items))
	if (int(e.Flags) & progs.FlagOnGround) != 0 {
		clientData.SetOnGround(true)
	}
	if e.WaterLevel >= 2 {
		clientData.SetInWater(true)
	}

	wmi := 0
	wms, err := progsdat.String(e.WeaponModel)
	if err == nil {
		wmi = s.ModelIndex(wms)
	}

	clientData.GetPunchAngle().SetX(int32(e.PunchAngle[0]))
	clientData.GetVelocity().SetX(int32(e.Velocity[0] / 16))
	clientData.GetPunchAngle().SetY(int32(e.PunchAngle[1]))
	clientData.GetVelocity().SetY(int32(e.Velocity[1] / 16))
	clientData.GetPunchAngle().SetZ(int32(e.PunchAngle[2]))
	clientData.GetVelocity().SetZ(int32(e.Velocity[2] / 16))

	clientData.SetWeaponFrame(int32(e.WeaponFrame))
	clientData.SetArmor(int32(e.ArmorValue))
	clientData.SetWeapon(int32(wmi))
	clientData.SetHealth(int32(e.Health))
	clientData.SetAmmo(int32(e.CurrentAmmo))
	clientData.SetShells(int32(e.AmmoShells))
	clientData.SetNails(int32(e.AmmoNails))
	clientData.SetRockets(int32(e.AmmoRockets))
	clientData.SetCells(int32(e.AmmoCells))

	if cmdl.Quoth() || cmdl.Rogue() || cmdl.Hipnotic() {
		for i := 0; i < 32; i++ {
			if int(e.Weapon)&(1<<uint(i)) != 0 {
				clientData.SetActiveWeapon(int32(i))
				break
			}
		}
	} else {
		clientData.SetActiveWeapon(int32(e.Weapon))
	}
	clientData.SetWeaponAlpha(int32(alpha))

	svc.WriteClientData(clientData, s.protocol, flags, &msgBuf)
}

// Initializes a client_t for a new net connection.  This will only be called
// once for a player each game, not once for each level change.
func (s *Server) connectClient(n int) error {
	old := sv_clients[n]
	newC := &SVClient{
		netConnection: old.netConnection,
		admin:         old.admin, // admin is a property of the connection
		edictId:       n + 1,
		id:            n,
		name:          "unconnected",
		active:        true,
		spawned:       false,
	}
	if s.loadGame {
		newC.spawnParams = old.spawnParams
	} else {
		if err := vm.ExecuteProgram(progsdat.Globals.SetNewParms, s); err != nil {
			return err
		}
		newC.spawnParams = progsdat.Globals.Parm
	}
	sv_clients[n] = newC
	newC.SendServerinfo(s)
	return nil
}

func (s *Server) SendClientDatagram(sc *SVClient) (bool, error) {
	msgBuf.ClearMessage()
	msgBufMaxLen = protocol.MaxDatagram
	if sc.Address() != net.LocalAddress {
		msgBufMaxLen = net.DATAGRAM_MTU
	}
	svc.WriteTime(s.time, s.protocol, s.protocolFlags, &msgBuf)

	s.WriteClientdataToMessage(sc.edictId)

	s.WriteEntitiesToClient(sc.edictId)

	return s.SendDatagram(sc)
}

func (s *Server) UpdateToReliableMessages() {
	b := s.reliableDatagram.Bytes()
	for _, sc := range sv_clients {
		newFrags := entvars.Get(sc.edictId).Frags
		if sc.active {
			// Does it actually matter to compare as float32?
			// These subtle C things...
			if float32(sc.oldFrags) != newFrags {
				uf := protos.UpdateFrags_builder{
					Player:   int32(sc.id),
					NewFrags: int32(newFrags),
				}.Build()
				svc.WriteUpdateFrags(uf, s.protocol, s.protocolFlags, &sc.msg)
			}
			sc.msg.WriteBytes(b)
		}
		sc.oldFrags = int(newFrags)
	}
	s.reliableDatagram.ClearMessage()
}

func (s *Server) impact(e1, e2 int) error {
	oldSelf := progsdat.Globals.Self
	oldOther := progsdat.Globals.Other

	progsdat.Globals.Time = s.time

	ent1 := entvars.Get(e1)
	ent2 := entvars.Get(e2)
	if ent1.Touch != 0 && ent1.Solid != SOLID_NOT {
		progsdat.Globals.Self = int32(e1)
		progsdat.Globals.Other = int32(e2)
		if err := vm.ExecuteProgram(ent1.Touch, s); err != nil {
			return err
		}
	}

	if ent2.Touch != 0 && ent2.Solid != SOLID_NOT {
		progsdat.Globals.Self = int32(e2)
		progsdat.Globals.Other = int32(e1)
		if err := vm.ExecuteProgram(ent2.Touch, s); err != nil {
			return err
		}
	}

	progsdat.Globals.Self = oldSelf
	progsdat.Globals.Other = oldOther
	return nil
}

func CheckVelocity(ent *progs.EntVars) {
	maxVelocity := cvars.ServerMaxVelocity.Value()
	for i := 0; i < 3; i++ {
		if ent.Velocity[i] != ent.Velocity[i] {
			s, _ := progsdat.String(ent.ClassName)
			slog.Warn("Got a NaN velocity", slog.String("class", s))
			ent.Velocity[i] = 0
		}
		if ent.Origin[i] != ent.Origin[i] {
			s, _ := progsdat.String(ent.ClassName)
			slog.Warn("Got a NaN origin", slog.String("class", s))
			ent.Origin[i] = 0
		}
		if ent.Velocity[i] > maxVelocity {
			ent.Velocity[i] = maxVelocity
		} else if ent.Velocity[i] < -maxVelocity {
			ent.Velocity[i] = -maxVelocity
		}
	}
}

func (s *Server) CreateBaseline() {
	for entnum := 0; entnum < s.numEdicts; entnum++ {
		e := &s.edicts[entnum]
		if e.Free {
			continue
		}
		sev := entvars.Get(entnum)
		if entnum > svs.maxClients && sev.ModelIndex == 0 {
			continue
		}

		e.Baseline.Origin = sev.Origin
		e.Baseline.Angles = sev.Angles

		e.Baseline.Frame = uint16(sev.Frame)
		e.Baseline.Skin = byte(sev.Skin)
		if entnum > 0 && entnum <= svs.maxClients {
			e.Baseline.ColorMap = byte(entnum)
			e.Baseline.ModelIndex = uint16(s.ModelIndex("progs/player.mdl"))
			e.Baseline.Alpha = svc.EntityAlphaDefault
		} else {
			e.Baseline.ColorMap = 0
			str, err := progsdat.String(sev.Model)
			if err != nil {
				log.Printf("Error in CreateBaseline: %v", err)
			}
			e.Baseline.ModelIndex = uint16(s.ModelIndex(str))
			e.Baseline.Alpha = e.Alpha
		}

		bits := 0
		mi := int(e.Baseline.ModelIndex)
		frame := int(e.Baseline.Frame)
		if s.protocol == protocol.NetQuake {
			if mi&0xFF00 != 0 {
				mi = 0
				e.Baseline.ModelIndex = 0
			}
			if frame&0xFF00 != 0 {
				frame = 0
				e.Baseline.Frame = 0
			}
			e.Baseline.Alpha = svc.EntityAlphaDefault
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
			s.signon.WriteByte(svc.SpawnBaseline2)
		} else {
			s.signon.WriteByte(svc.SpawnBaseline)
		}

		s.signon.WriteShort(entnum)
		if bits != 0 {
			s.signon.WriteByte(bits)
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

		s.signon.WriteByte(int(e.Baseline.ColorMap))
		s.signon.WriteByte(int(e.Baseline.Skin))
		for i := 0; i < 3; i++ {
			s.signon.WriteCoord(float32(e.Baseline.Origin[i]), s.protocolFlags)
			s.signon.WriteAngle(float32(e.Baseline.Angles[i]), s.protocolFlags)
		}

		if bits&svc.EntityBaselineAlpha != 0 {
			s.signon.WriteByte(int(e.Alpha))
		}
	}
}

func (s *Server) SendClientMessages() error {
	// update frags, names, etc
	s.UpdateToReliableMessages()

	// build individual updates
	for _, c := range sv_clients {
		if !c.active {
			continue
		}

		if c.spawned {
			if s, err := s.SendClientDatagram(c); err != nil {
				return err
			} else if !s {
				continue
			}
		} else {
			// the player isn't totally in the game yet
			// send small keepalive messages if too much time has passed
			// send a full message when the next signon stage has been requested
			// some other message data (name changes, etc) may accumulate
			// between signon stages
			if !c.sendSignon {
				if host.Time()-c.lastMessage > 5 {
					if err := c.SendNop(s); err != nil {
						return err
					}
				}
				// don't send out non-signon messages
				continue
			}
		}

		// check for an overflowed message.  Should only happen
		// on a very fucked up connection that backs up a lot, then
		// changes level
		if false { // GetClientOverflowed(i) {
			if err := c.Drop(true, s); err != nil {
				return err
			}
			// SetClientOverflowed(i, false)
			continue
		}

		if c.msg.HasMessage() {
			if !c.CanSendMessage() {
				continue
			}

			if c.SendMessage() == -1 {
				// if the message couldn't send, kick off
				if err := c.Drop(true, s); err != nil {
					return err
				}
			}
			c.msg.ClearMessage()
			c.lastMessage = host.Time()
			c.sendSignon = false
		}
	}

	// clear muzzle flashes
	s.CleanupEntvarEffects()
	return nil
}

// Runs thinking code if time.  There is some play in the exact time the think
// function will be called, because it is called before any movement is done
// in a frame.  Not used for pushmove objects, because they must be exact.
// Returns false if the entity removed itself.
func (s *Server) runThink(e int) (bool, error) {
	thinktime := entvars.Get(e).NextThink
	if thinktime <= 0 || thinktime > s.time+float32(host.FrameTime()) {
		return true, nil
	}

	if thinktime < s.time {
		// don't let things stay in the past.
		// it is possible to start that way
		// by a trigger with a local time.
		thinktime = s.time
	}

	oldframe := entvars.Get(e).Frame

	ev := entvars.Get(e)
	ev.NextThink = 0
	progsdat.Globals.Time = thinktime
	progsdat.Globals.Self = int32(e)
	progsdat.Globals.Other = 0
	if err := vm.ExecuteProgram(ev.Think, s); err != nil {
		return false, err
	}

	// capture interval to nextthink here and send it to client for better
	// lerp timing, but only if interval is not 0.1 (which client assumes)
	ed := &s.edicts[e]
	ed.SendInterval = false
	if !ed.Free && ev.NextThink != 0 &&
		(ev.MoveType == progs.MoveTypeStep || ev.Frame != oldframe) {
		i := math.Round((ev.NextThink - thinktime) * 255)
		// 25 and 26 are close enough to 0.1 to not send
		if i >= 0 && i < 256 && i != 25 && i != 26 {
			ed.SendInterval = true
		}
	}

	return !ed.Free, nil
}

// Does not change the entities velocity at all
func (s *Server) pushEntity(e int, push vec.Vec3) (bsp.Trace, error) {
	ev := entvars.Get(e)
	origin := ev.Origin
	mins := ev.Mins
	maxs := ev.Maxs
	end := vec.Add(origin, push)

	tr := func() bsp.Trace {
		if ev.MoveType == progs.MoveTypeFlyMissile {
			return svMove(origin, mins, maxs, end, MOVE_MISSILE, e, s)
		}
		if ev.Solid == SOLID_TRIGGER || ev.Solid == SOLID_NOT {
			// only clip against bmodels
			return svMove(origin, mins, maxs, end, MOVE_NOMONSTERS, e, s)
		}

		return svMove(origin, mins, maxs, end, MOVE_NORMAL, e, s)
	}()

	ev.Origin = tr.EndPos
	if err := vm.LinkEdict(e, true, s); err != nil {
		return bsp.Trace{}, err
	}

	if tr.EntPointer {
		if err := s.impact(e, tr.EntNumber); err != nil {
			return bsp.Trace{}, err
		}
	}

	return tr, nil
}

func (s *Server) setIdealPitch(player int) {
	const MAX_FORWARD = 6
	z := [MAX_FORWARD]float32{}
	ev := entvars.Get(player)
	if int(ev.Flags)&FL_ONGROUND == 0 {
		return
	}

	angleval := ev.Angles[1] * math32.Pi * 2 / 360 // YAW
	sinval := math32.Sin(angleval)
	cosval := math32.Cos(angleval)
	for i := 0; i < MAX_FORWARD; i++ {
		a := (i + 3) * 12
		top := vec.Vec3{
			ev.Origin[0] + cosval*float32(a),
			ev.Origin[1] + sinval*float32(a),
			ev.Origin[2] + ev.ViewOfs[2],
		}
		bottom := top
		bottom[2] -= 160

		tr := svMove(top, vec.Vec3{}, vec.Vec3{}, bottom, 1, player, s)
		if tr.AllSolid {
			// looking at a wall, leave ideal the way is was
			return
		}

		if tr.Fraction == 1 {
			// near a dropoff
			return
		}

		z[i] = top[2] + tr.Fraction*(bottom[2]-top[2])
	}

	// Original Quake has both dir and step as int but the code does not make any
	// sense with ints (the 0.1 parts or the last line)
	dir := float32(0)
	steps := 0
	for j := 1; j < MAX_FORWARD; j++ {
		step := z[j] - z[j-1]
		if step > -0.1 && step < 0.1 {
			continue
		}

		if dir != 0 && (step-dir > 0.1 || step-dir < -0.1) {
			return // mixed changes
		}

		steps++
		dir = step
	}

	if dir == 0 {
		ev.IdealPitch = 0
		return
	}

	if steps < 2 {
		return
	}
	ev.IdealPitch = -dir * cvars.ServerIdealPitchScale.Value()
}

const (
	MAX_ENT_LEAFS = 32
)

func (s *Server) WriteEntitiesToClient(clent int) {
	// TODO: this looks like the worst case for any branch prediction
	// probably worth to get a better implementation

	cev := entvars.Get(clent)
	org := vec.Add(cev.Origin, cev.ViewOfs)
	// find the client's PVS
	pvs := s.worldModel.FatPVS(org)

	// send over all entities (except the client) that touch the pvs
	for ent := 1; ent < s.numEdicts; ent++ {
		ev := entvars.Get(ent)
		edict := &s.edicts[ent]

		// check if we need to send this edict
		if ent != clent {
			// clent is ALLWAYS sent

			// ignore ents without visible models
			mn, err := progsdat.String(ev.Model)
			if ev.ModelIndex == 0 || err != nil || len(mn) == 0 {
				continue
			}

			// don't send model>255 entities if protocol is 15
			if s.protocol == protocol.NetQuake &&
				int(ev.ModelIndex)&0xFF00 != 0 {
				continue
			}

			// ignore if not touching a PV leaf
			i := 0
			for ; i < edict.num_leafs; i++ {
				if pvs[edict.leafnums[i]/8]&(1<<(uint(edict.leafnums[i])&7)) != 0 {
					break
				}
			}

			// if ent->num_leafs == MAX_ENT_LEAFS, the ent is visible from too many leafs
			// for us to say whether it's in the PVS, so don't try to vis cull it.
			// this commonly happens with rotators, because they often have huge bboxes
			// spanning the entire map, or really tall lifts, etc.
			if i == edict.num_leafs &&
				edict.num_leafs < MAX_ENT_LEAFS {
				continue // not visible
			}
		}

		// if (pr_alpha_supported) {
		// TODO: find a cleaner place to put this code
		//   s.updateEdictAlpha(ent);
		// }

		// don't send invisible entities unless they have effects
		if edict.Alpha == svc.EntityAlphaZero && ev.Effects == 0 {
			continue
		}

		// max size for protocol 15 is 18 bytes.
		// for protocol 85 the max size is 24 bytes.
		if msgBuf.Len()+24 > msgBufMaxLen {
			slog.Warn("Packet overflow!")
		}

		// send an update
		eu := &protos.EntityUpdate{}
		eu.SetEntity(int32(ent))

		if ev.ModelIndex != float32(edict.Baseline.ModelIndex) {
			eu.SetModel(int32(ev.ModelIndex))
		}
		if ev.Frame != float32(edict.Baseline.Frame) {
			eu.SetFrame(int32(ev.Frame))
		}
		if ev.ColorMap != float32(edict.Baseline.ColorMap) {
			eu.SetColorMap(int32(ev.ColorMap))
		}
		if ev.Skin != float32(edict.Baseline.Skin) {
			eu.SetSkin(int32(ev.Skin))
		}
		if ev.Effects != float32(edict.Baseline.Effects) {
			eu.SetEffects(int32(ev.Effects))
		}
		if miss := ev.Origin[0] - edict.Baseline.Origin[0]; miss < -0.1 || miss > 0.1 {
			eu.SetOriginX(ev.Origin[0])
		}
		if ev.Angles[0] != edict.Baseline.Angles[0] {
			eu.SetAngleX(ev.Angles[0])
		}
		if miss := ev.Origin[1] - edict.Baseline.Origin[1]; miss < -0.1 || miss > 0.1 {
			eu.SetOriginY(ev.Origin[1])
		}
		if ev.Angles[1] != edict.Baseline.Angles[1] {
			eu.SetAngleY(ev.Angles[1])
		}
		if miss := ev.Origin[2] - edict.Baseline.Origin[2]; miss < -0.1 || miss > 0.1 {
			eu.SetOriginZ(ev.Origin[2])
		}
		if ev.Angles[2] != edict.Baseline.Angles[2] {
			eu.SetAngleZ(ev.Angles[2])
		}
		// don't mess up the step animation
		eu.SetLerpMoveStep(ev.MoveType == progs.MoveTypeStep)

		if edict.Baseline.Alpha != edict.Alpha {
			eu.SetAlpha(int32(edict.Alpha))
		}
		if edict.SendInterval {
			eu.SetLerpFinish(int32(math.Round((ev.NextThink - s.time) * 255)))
		}
		svc.WriteEntityUpdate(eu, s.protocol, s.protocolFlags, &msgBuf)
	}
}

func init() {
	//if (Cvar_GetValue(&coop)) Cvar_Set("deathmatch", "0");
	cvars.Skill.SetCallback(func(cv *cvar.Cvar) {
		cs := float32(int(cv.Value() + 0.5))
		cs = math.Clamp(0, cs, 3)
		if cv.Value() != cs { // Break recursion
			cv.SetValue(cs)
		}
	})
}

// Grabs the current state of each client for saving across the
// transition to another level
func (s *Server) saveSpawnparms() error {
	svs.serverFlags = progsdat.Globals.ServerFlags

	for _, c := range sv_clients {
		if !c.active {
			continue
		}
		// call the progs to get default spawn parms for the new client
		progsdat.Globals.Self = int32(c.edictId)
		if err := vm.ExecuteProgram(progsdat.Globals.SetChangeParms, s); err != nil {
			return err
		}
		c.spawnParams = progsdat.Globals.Parm
	}
	return nil
}

func (s *Server) ChangeLevel(mapName string, pcl int) error {
	if err := s.saveSpawnparms(); err != nil {
		return err
	}
	if err := s.SpawnServer(mapName, pcl); err != nil {
		return fmt.Errorf("cannot run map %s: %w", mapName, err)
	}
	return nil
}

func (s *Server) SpawnSaveGameServer(data *protos.SaveGame, pcl int) error {
	cvars.Skill.SetValue(float32(data.GetCurrentSkill()))
	if err := s.SpawnServer(data.GetMapName(), pcl); err != nil {
		return err
	}
	// pause until all clients connect
	s.paused = true
	s.loadGame = true
	copy(s.lightStyles[:], data.GetLightStyles())
	vm.LoadGameGlobals(data.GetGlobals())
	if err := s.loadGameEdicts(data.GetEdicts()); err != nil {
		return err
	}
	s.time = data.GetMapTime()
	copy(sv_clients[0].spawnParams[:], data.GetSpawnParams())
	return nil
}

func (s *Server) ResetServer() error {
	if !s.active {
		return nil
	}
	mapname := s.name // s.name gets cleared in spawnserver
	pcl := s.protocol // s.protocol gets cleared in spawnserver
	if err := s.SpawnServer(mapname, pcl); err != nil {
		return fmt.Errorf("cannot restart map %s: %w", mapname, err)
	}
	return nil
}

func (s *Server) reset() {
	s.active = false
	s.paused = false
	s.loadGame = false
	s.time = 1.0
	s.lastCheck = 0
	s.lastCheckTime = 0
	s.datagram.Reset()
	s.reliableDatagram.Reset()
	s.signon.Reset()
	s.numEdicts = 0
	s.maxEdicts = 0
	s.edicts = s.edicts[:0]
	s.protocol = 0
	s.protocolFlags = 0
	s.state = ServerStateLoading
	s.soundPrecache = s.soundPrecache[:0]
	s.soundPrecache = append(s.soundPrecache, string([]byte{0, 0, 0, 0, 0, 0, 0, 0}))
	s.lightStyles = [64]string{}
	s.name = ""
	s.modelPrecache = s.modelPrecache[:0]
	s.modelPrecache = append(s.modelPrecache, string([]byte{0, 0, 0, 0, 0, 0, 0, 0}))
	s.models = s.models[:0]
	s.models = append(s.models, nil)
	s.worldModel = nil
}

// This is called at the start of each level
func (s *Server) SpawnServer(mapName string, pcl int) error {
	// let's not have any servers with no name
	if len(cvars.HostName.String()) == 0 {
		cvars.HostName.SetByString("UNNAMED")
	}

	slog.Debug("SpawnServer", slog.String("mapname", mapName))
	// now safe to issue another
	vm.changeLevelIssued = false

	// tell all connected clients that we are going to a new level
	if s.Active() {
		s.sendReconnect()
	}

	// set up the new server
	s.freeEdicts()
	s.reset()
	s.name = mapName
	s.protocol = pcl

	if s.protocol == protocol.RMQ {
		s.protocolFlags = protocol.PRFL_INT32COORD | protocol.PRFL_SHORTANGLE
	}

	// load progs to get entity field count
	slog.Info("LOADING PROGS")
	p, err := progs.LoadProgs()
	if err != nil {
		log.Fatalf("Failed to load progs.dat: %v", err)
	}
	progsdat = p
	vm.prog = p

	// allocate server memory
	s.maxEdicts = int(cvars.MaxEdicts.Value())
	s.allocEdicts()

	// leave slots at start for clients only
	s.numEdicts = svs.maxClients + 1
	for i := 0; i < s.numEdicts; i++ {
		s.clearEdict(i)
	}
	for i := 0; i < svs.maxClients; i++ {
		sv_clients[i].edictId = i + 1
	}

	modelName := fmt.Sprintf("maps/%s.bsp", mapName)

	log.Printf("New world: %s", modelName)
	sv_models = make(map[string]model.Model)
	mods, err := bsp.Load(modelName)
	if err != nil || len(mods) < 1 {
		slog.Warn("Couldn't spawn server", slog.String("modelname", modelName))
		return nil
	}
	s.worldModel = mods[0]
	for _, m := range mods {
		s.modelPrecache = append(s.modelPrecache, m.Name())
		s.models = append(s.models, m)
	}

	s.clearWorld()

	// load the rest of the entities
	entvars.Clear(0)
	s.edicts[0].Free = false
	ev := entvars.Get(0)
	ev.Model = progsdat.AddString(modelName)
	ev.ModelIndex = 1 // world model
	ev.Solid = SOLID_BSP
	ev.MoveType = progs.MoveTypePush

	if cvars.Coop.Bool() {
		progsdat.Globals.Coop = 1
	} else {
		progsdat.Globals.DeathMatch = cvars.DeathMatch.Value()
	}
	progsdat.Globals.MapName = progsdat.AddString(mapName)

	// serverflags are for cross level information (sigils)
	progsdat.Globals.ServerFlags = svs.serverFlags

	if err := s.loadEntities(s.worldModel.Entities, mapName); err != nil {
		return err
	}

	s.active = true

	// all setup is completed, any further precache statements are errors
	s.state = ServerStateActive

	// run two frames to allow everything to settle
	host.Reset()
	if err := s.runPhysics(); err != nil {
		s.active = false
		return err
	}
	if err := s.runPhysics(); err != nil {
		s.active = false
		return err
	}

	// create a baseline for more efficient communications
	s.CreateBaseline()

	// warn if signon buffer larger than standard server can handle
	if s.signon.Len() > 8000-2 {
		// max size that will fit into 8000-sized client->message buffer
		// with 2 extra bytes on the end
		slog.Debug("byte signon buffer exceeds standard limit of 7998.", slog.Int("Count", s.signon.Len()))
	}

	// send serverinfo to all connected clients
	for i := 0; i < svs.maxClients; i++ {
		if sv_clients[i].active {
			sv_clients[i].SendServerinfo(s)
		}
	}

	slog.Debug("Server spawned.")
	return nil
}

// This only happens at the end of a game, not between levels
func (s *Server) Shutdown() error {
	if !s.Active() {
		return nil
	}

	s.active = false

	// flush any pending messages - like the score!!!
	end := time.Now().Add(3 * time.Second)
	count := 1
	for count != 0 {
		count = 0
		for _, c := range sv_clients {
			if c.active && c.msg.HasMessage() {
				if c.CanSendMessage() {
					c.SendMessage()
					c.msg.ClearMessage()
				} else {
					if err := c.GetMessage(); err != nil {
						return err
					}
					count++
				}
			}
		}
		if time.Now().After(end) {
			break
		}
	}

	// make sure all the clients know we're disconnecting
	SendToAll([]byte{svc.Disconnect})

	for _, c := range sv_clients {
		if c.active {
			if err := c.Drop(false, s); err != nil {
				return nil
			}
		}
	}

	s.worldModel = nil

	CreateSVClients()
	return nil
}

func (s *Server) saveGameEdicts() []*protos.Edict {
	eds := make([]*protos.Edict, 0, s.numEdicts)
	for i := 0; i < s.numEdicts; i++ {
		if s.edicts[i].Free {
			eds = append(eds, &protos.Edict{})
			continue
		}
		e := vm.saveGameEntVars(i)

		if /*!pr_alpha_supported &&*/ s.edicts[i].Alpha != 0 {
			wa := s.edicts[i].Alpha
			if wa == 1 {
				e.SetAlpha(-1)
			} else {
				e.SetAlpha((float32(wa) - 1) / 254)
			}
		}

		eds = append(eds, e)
	}
	return eds
}

func (s *Server) loadGameEdicts(es []*protos.Edict) error {
	for i, e := range es {
		if proto.Equal(e, &protos.Edict{}) {
			s.edicts[i] = Edict{
				Free: true,
			}
			continue
		}
		a := byte(0)
		readA := e.GetAlpha()
		if readA != 0 {
			ta := (readA * 254) + 1
			ta = math.Clamp(1, ta, 255)
			a = byte(ta)
		}
		s.edicts[i] = Edict{
			Alpha: a,
		}

		vm.loadGameEntVars(i, e)
		if err := vm.LinkEdict(i, false, s); err != nil {
			return err
		}
	}
	s.numEdicts = len(es)
	return nil
}

func (s *Server) ModelIndex(n string) int {
	if len(n) == 0 {
		return 0
	}
	for i, m := range s.modelPrecache {
		if m == n {
			return i
		}
	}
	debug.PrintStack()
	log.Fatalf("SV_ModelIndex: model %v not precached", n)
	return 0
}
