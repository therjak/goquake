// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

//void Host_ClearMemory(void);
import "C"

import (
	"fmt"
	"log"

	"goquake/bsp"
	"goquake/cmd"
	cmdl "goquake/commandline"
	"goquake/conlog"
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
	maxClients        int
	maxClientsLimit   int
	serverFlags       int // TODO: is int the correct way?
	changeLevelIssued bool
}

type ServerState bool

const (
	ServerStateLoading = ServerState(false)
	ServerStateActive  = ServerState(true)
)

type Server struct {
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

	modelPrecache []string
	soundPrecache []string
	lightStyles   [64]string

	name      string // map name
	modelName string // maps/<name>.bsp, for model_precache[0]

	models     []model.Model
	worldModel *bsp.Model
}

var (
	svs = ServerStatic{}
	sv  = Server{
		models: make([]model.Model, 1),
	}
	sv_protocol int
	host_client int
)

func svProtocol(args []cmd.QArg, _ int) {
	switch len(args) {
	default:
		conlog.SafePrintf("usage: sv_protocol <protocol>\n")
	case 0:
		conlog.Printf(`"sv_protocol" is "%v"`+"\n", sv_protocol)
	case 1:
		i := args[0].Int()
		switch i {
		case protocol.NetQuake, protocol.FitzQuake, protocol.RMQ:
			sv_protocol = i
			if sv.active {
				conlog.Printf("changes will not take effect until the next level load.\n")
			}
		default:
			conlog.Printf("sv_protocol must be %v or %v or %v\n",
				protocol.NetQuake, protocol.FitzQuake, protocol.RMQ)
		}
	}
}

func init() {
	Must(cmd.AddCommand("sv_protocol", svProtocol))
}

func serverInit() {
	sv_protocol = cmdl.Protocol()
	switch sv_protocol {
	case protocol.NetQuake:
		log.Printf("Server using protocol %v (NetQuake)\n", sv_protocol)
	case protocol.FitzQuake:
		log.Printf("Server using protocol %v (FitzQuake)\n", sv_protocol)
	case protocol.RMQ:
		log.Printf("Server using protocol %v (RMQ)\n", sv_protocol)
	case protocol.GoQuake:
		log.Printf("Server using protocol %v (GoQuake)\n", sv_protocol)
	default:
		Error("Bad protocol version request %v. Accepted values: %v, %v, %v.",
			sv_protocol, protocol.NetQuake, protocol.FitzQuake, protocol.RMQ)
		log.Printf("Server using protocol %v (Unknown)\n", sv_protocol)
	}
}

func SV_NumEdicts() C.int {
	return C.int(sv.numEdicts)
}

var (
	msgBuf       = net.Message{}
	msgBufMaxLen = 0
)

func (s *Server) StartParticle(org, dir vec.Vec3, color, count int) {
	if s.datagram.Len()+16 > net.MAX_DATAGRAM {
		return
	}
	s.datagram.WriteByte(svc.Particle)
	s.datagram.WriteCoord(org[0], s.protocolFlags)
	s.datagram.WriteCoord(org[1], s.protocolFlags)
	s.datagram.WriteCoord(org[2], s.protocolFlags)
	df := func(d float32) int {
		v := d * 16
		if v > 127 {
			return 127
		}
		if v < -128 {
			return -128
		}
		return int(v)
	}
	s.datagram.WriteChar(df(dir[0]))
	s.datagram.WriteChar(df(dir[1]))
	s.datagram.WriteChar(df(dir[2]))
	s.datagram.WriteByte(count)
	s.datagram.WriteByte(color)
}

func (s *Server) SendDatagram(c *SVClient) bool {
	b := msgBuf.Bytes()
	// If there is space add the server datagram
	if len(b)+s.datagram.Len() < protocol.MaxDatagram {
		b = append(b, s.datagram.Bytes()...)
	}
	if c.netConnection.SendUnreliableMessage(b) == -1 {
		c.Drop(true)
		return false
	}
	return true
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
	if !cmdl.Dedicated() {
		clientReconnect()
	}
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
	if s.datagram.Len() > net.MAX_DATAGRAM-16 {
		return nil
	}
	for soundnum, m := range s.soundPrecache {
		if m == sample {
			s.sendStartSound(entity, channel, volume, soundnum, attenuation)
			return nil
		}
	}
	conlog.Printf("SV_StartSound: %s not precacheed\n", sample)
	return nil
}

func (s *Server) sendStartSound(entity, channel, volume, soundnum int, attenuation float32) {
	fieldMask := 0
	if volume != 255 {
		fieldMask |= svc.SoundVolume
	}
	if attenuation != 1.0 {
		fieldMask |= svc.SoundAttenuation
	}
	if entity >= 8192 {
		if s.protocol == protocol.NetQuake {
			return // protocol does not support this info
		}
		fieldMask |= svc.SoundLargeEntity
	}
	if soundnum >= 256 || channel >= 8 {
		if s.protocol == protocol.NetQuake {
			return
		}
		fieldMask |= svc.SoundLargeSound
	}
	s.datagram.WriteByte(svc.Sound)
	s.datagram.WriteByte(fieldMask)
	if fieldMask&svc.SoundVolume != 0 {
		s.datagram.WriteByte(volume)
	}
	if fieldMask&svc.SoundAttenuation != 0 {
		s.datagram.WriteByte(int(attenuation * 64))
	}
	if fieldMask&svc.SoundLargeEntity != 0 {
		s.datagram.WriteShort(entity)
		s.datagram.WriteByte(channel)
	} else {
		s.datagram.WriteShort((entity << 3) | channel)
	}
	if fieldMask&svc.SoundLargeSound != 0 {
		s.datagram.WriteShort(soundnum)
	} else {
		s.datagram.WriteByte(soundnum)
	}
	ev := EntVars(entity)
	flags := s.protocolFlags
	s.datagram.WriteCoord(ev.Origin[0]+0.5*(ev.Mins[0]+ev.Maxs[0]), flags)
	s.datagram.WriteCoord(ev.Origin[1]+0.5*(ev.Mins[1]+ev.Maxs[1]), flags)
	s.datagram.WriteCoord(ev.Origin[2]+0.5*(ev.Mins[2]+ev.Maxs[2]), flags)
}

func (s *Server) CleanupEntvarEffects() {
	for i := 1; i < s.numEdicts; i++ {
		ev := EntVars(i)
		eff := int(ev.Effects)
		ev.Effects = float32(eff &^ svc.EffectMuzzleFlash)
	}
}

func (s *Server) WriteClientdataToMessage(player int) {
	e := EntVars(player)
	alpha := s.edicts[player].Alpha
	flags := s.protocolFlags
	if e.DmgTake != 0 || e.DmgSave != 0 {
		other := EntVars(int(e.DmgInflictor))
		msgBuf.WriteByte(svc.Damage)
		msgBuf.WriteByte(int(e.DmgSave))
		msgBuf.WriteByte(int(e.DmgTake))
		msgBuf.WriteCoord(other.Origin[0]+0.5*(other.Mins[0]+other.Maxs[0]), flags)
		msgBuf.WriteCoord(other.Origin[1]+0.5*(other.Mins[1]+other.Maxs[1]), flags)
		msgBuf.WriteCoord(other.Origin[2]+0.5*(other.Mins[2]+other.Maxs[2]), flags)
		e.DmgTake = 0
		e.DmgSave = 0
	}

	// send the current viewpos offset from the view entity
	SV_SetIdealPitch(player) // how much to loop up/down ideally

	// a fixangle might get lost in a dropped packet.  Oh well.
	if e.FixAngle != 0 {
		msgBuf.WriteByte(svc.SetAngle)
		msgBuf.WriteAngle(e.Angles[0], flags)
		msgBuf.WriteAngle(e.Angles[1], flags)
		msgBuf.WriteAngle(e.Angles[2], flags)
		e.FixAngle = 0
	}

	bits := 0
	if e.ViewOfs[2] != svc.DEFAULT_VIEWHEIGHT {
		bits |= svc.SU_VIEWHEIGHT
	}
	if e.IdealPitch != 0 {
		bits |= svc.SU_IDEALPITCH
	}
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
	bits |= svc.SU_ITEMS
	if (int(e.Flags) & progs.FlagOnGround) != 0 {
		bits |= svc.SU_ONGROUND
	}
	if e.WaterLevel >= 2 {
		bits |= svc.SU_INWATER
	}
	if e.PunchAngle[0] != 0 {
		bits |= svc.SU_PUNCH1
	}
	if e.PunchAngle[1] != 0 {
		bits |= svc.SU_PUNCH2
	}
	if e.PunchAngle[2] != 0 {
		bits |= svc.SU_PUNCH3
	}
	if e.Velocity[0] != 0 {
		bits |= svc.SU_VELOCITY1
	}
	if e.Velocity[1] != 0 {
		bits |= svc.SU_VELOCITY2
	}
	if e.Velocity[2] != 0 {
		bits |= svc.SU_VELOCITY3
	}
	if e.WeaponFrame != 0 {
		bits |= svc.SU_WEAPONFRAME
	}
	if e.ArmorValue != 0 {
		bits |= svc.SU_ARMOR
	}
	bits |= svc.SU_WEAPON

	wmi := 0
	wms, err := progsdat.String(e.WeaponModel)
	if err == nil {
		wmi = s.ModelIndex(wms)
	}

	if s.protocol != protocol.NetQuake {
		if (wmi & 0xFF00) != 0 {
			bits |= svc.SU_WEAPON2
		}
		if (int(e.ArmorValue) & 0xFF00) != 0 {
			bits |= svc.SU_ARMOR2
		}
		if (int(e.CurrentAmmo) & 0xFF00) != 0 {
			bits |= svc.SU_AMMO2
		}
		if (int(e.AmmoShells) & 0xFF00) != 0 {
			bits |= svc.SU_SHELLS2
		}
		if (int(e.AmmoNails) & 0xFF00) != 0 {
			bits |= svc.SU_NAILS2
		}
		if (int(e.AmmoRockets) & 0xFF00) != 0 {
			bits |= svc.SU_ROCKETS2
		}
		if (int(e.AmmoCells) & 0xFF00) != 0 {
			bits |= svc.SU_CELLS2
		}
		if (bits&svc.SU_WEAPONFRAME != 0) &&
			(int(e.WeaponFrame)&0xFF00) != 0 {
			bits |= svc.SU_WEAPONFRAME2
		}
		if alpha != 0 {
			bits |= svc.SU_WEAPONALPHA
		}
		if bits >= 65536 {
			bits |= svc.SU_EXTEND1
		}
		if bits >= 16777216 {
			bits |= svc.SU_EXTEND2
		}
	}
	msgBuf.WriteByte(svc.ClientData)
	msgBuf.WriteShort(bits)
	if (bits & svc.SU_EXTEND1) != 0 {
		msgBuf.WriteByte(bits >> 16)
	}
	if (bits & svc.SU_EXTEND2) != 0 {
		msgBuf.WriteByte(bits >> 24)
	}
	if (bits & svc.SU_VIEWHEIGHT) != 0 {
		msgBuf.WriteChar(int(e.ViewOfs[2]))
	}
	if (bits & svc.SU_IDEALPITCH) != 0 {
		msgBuf.WriteChar(int(e.IdealPitch))
	}
	if (bits & (svc.SU_PUNCH1)) != 0 {
		msgBuf.WriteChar(int(e.PunchAngle[0]))
	}
	if (bits & (svc.SU_VELOCITY1)) != 0 {
		msgBuf.WriteChar(int(e.Velocity[0] / 16))
	}
	if (bits & (svc.SU_PUNCH2)) != 0 {
		msgBuf.WriteChar(int(e.PunchAngle[1]))
	}
	if (bits & (svc.SU_VELOCITY2)) != 0 {
		msgBuf.WriteChar(int(e.Velocity[1] / 16))
	}
	if (bits & (svc.SU_PUNCH3)) != 0 {
		msgBuf.WriteChar(int(e.PunchAngle[2]))
	}
	if (bits & (svc.SU_VELOCITY3)) != 0 {
		msgBuf.WriteChar(int(e.Velocity[2] / 16))
	}

	msgBuf.WriteLong(items)

	if (bits & (svc.SU_WEAPONFRAME)) != 0 {
		msgBuf.WriteByte(int(e.WeaponFrame))
	}
	if (bits & (svc.SU_ARMOR)) != 0 {
		msgBuf.WriteByte(int(e.ArmorValue))
	}
	msgBuf.WriteByte(wmi)
	msgBuf.WriteShort(int(e.Health))
	msgBuf.WriteByte(int(e.CurrentAmmo))
	msgBuf.WriteByte(int(e.AmmoShells))
	msgBuf.WriteByte(int(e.AmmoNails))
	msgBuf.WriteByte(int(e.AmmoRockets))
	msgBuf.WriteByte(int(e.AmmoCells))

	if cmdl.Quoth() || cmdl.Rogue() || cmdl.Hipnotic() {
		for i := 0; i < 32; i++ {
			if int(e.Weapon)&(1<<uint(i)) != 0 {
				msgBuf.WriteByte(i)
				break
			}
		}
	} else {
		msgBuf.WriteByte(int(e.Weapon))
	}
	if (bits & (svc.SU_WEAPON2)) != 0 {
		msgBuf.WriteByte(wmi >> 8)
	}
	if (bits & (svc.SU_ARMOR2)) != 0 {
		msgBuf.WriteByte(int(e.ArmorValue) >> 8)
	}
	if (bits & (svc.SU_AMMO2)) != 0 {
		msgBuf.WriteByte(int(e.CurrentAmmo) >> 8)
	}
	if (bits & (svc.SU_SHELLS2)) != 0 {
		msgBuf.WriteByte(int(e.AmmoShells) >> 8)
	}
	if (bits & (svc.SU_NAILS2)) != 0 {
		msgBuf.WriteByte(int(e.AmmoNails) >> 8)
	}
	if (bits & (svc.SU_NAILS2)) != 0 {
		msgBuf.WriteByte(int(e.AmmoNails) >> 8)
	}
	if (bits & (svc.SU_ROCKETS2)) != 0 {
		msgBuf.WriteByte(int(e.AmmoRockets) >> 8)
	}
	if (bits & (svc.SU_CELLS2)) != 0 {
		msgBuf.WriteByte(int(e.AmmoCells) >> 8)
	}
	if (bits & (svc.SU_WEAPONFRAME2)) != 0 {
		msgBuf.WriteByte(int(e.WeaponFrame) >> 8)
	}
	if (bits & (svc.SU_WEAPONALPHA)) != 0 {
		msgBuf.WriteByte(int(alpha))
	}
}

// Initializes a client_t for a new net connection.  This will only be called
//once for a player each game, not once for each level change.
func ConnectClient(n int) {
	old := sv_clients[n]
	new := &SVClient{
		netConnection: old.netConnection,
		edictId:       n + 1,
		id:            n,
		name:          "unconnected",
		active:        true,
		spawned:       false,
	}
	if sv.loadGame {
		new.spawnParams = old.spawnParams
	} else {
		if err := vm.ExecuteProgram(progsdat.Globals.SetNewParms); err != nil {
			HostError(err)
		}
		new.spawnParams = progsdat.Globals.Parm
	}
	sv_clients[n] = new
	new.SendServerinfo()
}

func (s *Server) SendClientDatagram(c *SVClient) bool {
	msgBuf.ClearMessage()
	msgBufMaxLen = net.MAX_DATAGRAM
	if c.Address() != "LOCAL" {
		msgBufMaxLen = net.DATAGRAM_MTU
	}
	msgBuf.WriteByte(svc.Time)
	msgBuf.WriteFloat(s.time)

	s.WriteClientdataToMessage(c.edictId)

	s.WriteEntitiesToClient(c.edictId)

	return s.SendDatagram(c)
}

func (s *Server) UpdateToReliableMessages() {
	b := s.reliableDatagram.Bytes()
	for _, cl := range sv_clients {
		newFrags := EntVars(cl.edictId).Frags
		if cl.active {
			// Does it actually matter to compare as float32?
			// These subtle C things...
			if float32(cl.oldFrags) != newFrags {
				cl.msg.WriteByte(svc.UpdateFrags)
				cl.msg.WriteByte(cl.id)
				cl.msg.WriteShort(int(newFrags))
			}
			cl.msg.WriteBytes(b)
		}
		cl.oldFrags = int(newFrags)
	}
	s.reliableDatagram.ClearMessage()
}

func (s *Server) Impact(e1, e2 int) {
	oldSelf := progsdat.Globals.Self
	oldOther := progsdat.Globals.Other

	progsdat.Globals.Time = s.time

	ent1 := EntVars(e1)
	ent2 := EntVars(e2)
	if ent1.Touch != 0 && ent1.Solid != SOLID_NOT {
		progsdat.Globals.Self = int32(e1)
		progsdat.Globals.Other = int32(e2)
		if err := vm.ExecuteProgram(ent1.Touch); err != nil {
			HostError(err)
		}
	}

	if ent2.Touch != 0 && ent2.Solid != SOLID_NOT {
		progsdat.Globals.Self = int32(e2)
		progsdat.Globals.Other = int32(e1)
		if err := vm.ExecuteProgram(ent2.Touch); err != nil {
			HostError(err)
		}
	}

	progsdat.Globals.Self = oldSelf
	progsdat.Globals.Other = oldOther
}

func CheckVelocity(ent *progs.EntVars) {
	maxVelocity := cvars.ServerMaxVelocity.Value()
	for i := 0; i < 3; i++ {
		if ent.Velocity[i] != ent.Velocity[i] {
			s, _ := progsdat.String(ent.ClassName)
			conlog.Printf("Got a NaN velocity on %s\n", s)
			ent.Velocity[i] = 0
		}
		if ent.Origin[i] != ent.Origin[i] {
			s, _ := progsdat.String(ent.ClassName)
			conlog.Printf("Got a NaN origin on %s\n", s)
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
		sev := EntVars(entnum)
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

//Grabs the current state of each client for saving across the
//transition to another level
func SV_SaveSpawnparms() {
	svs.serverFlags = int(progsdat.Globals.ServerFlags)

	for _, c := range sv_clients {
		if !c.active {
			continue
		}
		// call the progs to get default spawn parms for the new client
		progsdat.Globals.Self = int32(c.edictId)
		if err := vm.ExecuteProgram(progsdat.Globals.SetChangeParms); err != nil {
			HostError(err)
		}
		c.spawnParams = progsdat.Globals.Parm
	}
}

func (s *Server) SendClientMessages() {
	// update frags, names, etc
	s.UpdateToReliableMessages()

	// build individual updates
	for _, c := range sv_clients {
		if !c.active {
			continue
		}

		if c.spawned {
			if !s.SendClientDatagram(c) {
				continue
			}
		} else {
			// the player isn't totally in the game yet
			// send small keepalive messages if too much time has passed
			// send a full message when the next signon stage has been requested
			// some other message data (name changes, etc) may accumulate
			// between signon stages
			if !c.sendSignon {
				if host.time-c.lastMessage > 5 {
					c.SendNop()
				}
				// don't send out non-signon messages
				continue
			}
		}

		// check for an overflowed message.  Should only happen
		// on a very fucked up connection that backs up a lot, then
		// changes level
		if false { // GetClientOverflowed(i) {
			c.Drop(true)
			// SetClientOverflowed(i, false)
			continue
		}

		if c.msg.HasMessage() {
			if !c.CanSendMessage() {
				continue
			}

			if c.SendMessage() == -1 {
				// if the message couldn't send, kick off
				c.Drop(true)
			}
			c.msg.ClearMessage()
			c.lastMessage = host.time
			c.sendSignon = false
		}
	}

	// clear muzzle flashes
	s.CleanupEntvarEffects()
}

// Runs thinking code if time.  There is some play in the exact time the think
// function will be called, because it is called before any movement is done
// in a frame.  Not used for pushmove objects, because they must be exact.
// Returns false if the entity removed itself.
func runThink(e int) bool {
	thinktime := EntVars(e).NextThink
	if thinktime <= 0 || thinktime > sv.time+float32(host.frameTime) {
		return true
	}

	if thinktime < sv.time {
		// don't let things stay in the past.
		// it is possible to start that way
		// by a trigger with a local time.
		thinktime = sv.time
	}

	oldframe := EntVars(e).Frame

	ev := EntVars(e)
	ev.NextThink = 0
	progsdat.Globals.Time = thinktime
	progsdat.Globals.Self = int32(e)
	progsdat.Globals.Other = 0
	if err := vm.ExecuteProgram(ev.Think); err != nil {
		HostError(err)
	}

	// capture interval to nextthink here and send it to client for better
	// lerp timing, but only if interval is not 0.1 (which client assumes)
	ed := &sv.edicts[e]
	ed.SendInterval = false
	if !ed.Free && ev.NextThink != 0 &&
		(ev.MoveType == progs.MoveTypeStep || ev.Frame != oldframe) {
		i := math.Round((ev.NextThink - thinktime) * 255)
		// 25 and 26 are close enough to 0.1 to not send
		if i >= 0 && i < 256 && i != 25 && i != 26 {
			ed.SendInterval = true
		}
	}

	return !ed.Free
}

//Does not change the entities velocity at all
func pushEntity(e int, push vec.Vec3) trace {
	// trace_t trace;
	// vec3_t end;
	ev := EntVars(e)
	origin := ev.Origin
	mins := ev.Mins
	maxs := ev.Maxs
	end := vec.Add(origin, push)

	tr := func() trace {
		if ev.MoveType == progs.MoveTypeFlyMissile {
			return svMove(origin, mins, maxs, end, MOVE_MISSILE, e)
		}
		if ev.Solid == SOLID_TRIGGER || ev.Solid == SOLID_NOT {
			// only clip against bmodels
			return svMove(origin, mins, maxs, end, MOVE_NOMONSTERS, e)
		}

		return svMove(origin, mins, maxs, end, MOVE_NORMAL, e)
	}()

	ev.Origin = tr.EndPos
	vm.LinkEdict(e, true)

	if tr.EntPointer {
		sv.Impact(e, tr.EntNumber)
	}

	return tr
}

func SV_SetIdealPitch(player int) {
	const MAX_FORWARD = 6
	z := [MAX_FORWARD]float32{}
	ev := EntVars(player)
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

		tr := svMove(top, vec.Vec3{}, vec.Vec3{}, bottom, 1, player)
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

	cev := EntVars(clent)
	org := vec.Add(cev.Origin, cev.ViewOfs)
	// find the client's PVS
	pvs := s.worldModel.FatPVS(org)

	// send over all entities (except the client) that touch the pvs
	for ent := 1; ent < s.numEdicts; ent++ {
		ev := EntVars(ent)
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
		// max size for protocol 15 is 18 bytes.
		// for protocol 85 the max size is 24 bytes.
		if msgBuf.Len()+24 > msgBufMaxLen {
			conlog.Printf("Packet overflow!\n")
		}

		// send an update
		bits := 0

		for i := uint32(0); i < 3; i++ {
			miss := ev.Origin[i] - edict.Baseline.Origin[i]
			if miss < -0.1 || miss > 0.1 {
				bits |= svc.U_ORIGIN1 << i
			}
		}

		if ev.Angles[0] != edict.Baseline.Angles[0] {
			bits |= svc.U_ANGLE1
		}

		if ev.Angles[1] != edict.Baseline.Angles[1] {
			bits |= svc.U_ANGLE2
		}

		if ev.Angles[2] != edict.Baseline.Angles[2] {
			bits |= svc.U_ANGLE3
		}

		if ev.MoveType == progs.MoveTypeStep {
			bits |= svc.U_STEP // don't mess up the step animation
		}

		if ev.ColorMap != float32(edict.Baseline.ColorMap) {
			bits |= svc.U_COLORMAP
		}

		if ev.Skin != float32(edict.Baseline.Skin) {
			bits |= svc.U_SKIN
		}

		if ev.Frame != float32(edict.Baseline.Frame) {
			bits |= svc.U_FRAME
		}

		if ev.Effects != float32(edict.Baseline.Effects) {
			bits |= svc.U_EFFECTS
		}

		if ev.ModelIndex != float32(edict.Baseline.ModelIndex) {
			bits |= svc.U_MODEL
		}

		//     if (pr_alpha_supported) {
		// TODO: find a cleaner place to put this code
		//       UpdateEdictAlpha(ent);
		//     }

		// don't send invisible entities unless they have effects
		if edict.Alpha == svc.EntityAlphaZero && ev.Effects == 0 {
			continue
		}

		// fitzquake
		if s.protocol != protocol.NetQuake {
			if edict.Baseline.Alpha != edict.Alpha {
				bits |= svc.U_ALPHA
			}
			if bits&svc.U_FRAME != 0 &&
				int(ev.Frame)&0xFF00 != 0 {
				bits |= svc.U_FRAME2
			}
			if bits&svc.U_MODEL != 0 &&
				int(ev.ModelIndex)&0xFF00 != 0 {
				bits |= svc.U_MODEL2
			}
			if edict.SendInterval {
				bits |= svc.U_LERPFINISH
			}
			if bits >= 65536 {
				bits |= svc.U_EXTEND1
			}
			if bits >= 16777216 {
				bits |= svc.U_EXTEND2
			}
		}

		if ent >= 256 {
			bits |= svc.U_LONGENTITY
		}

		if bits >= 256 {
			bits |= svc.U_MOREBITS
		}

		// write the message
		msgBuf.WriteByte(bits | svc.U_SIGNAL)

		if bits&svc.U_MOREBITS != 0 {
			msgBuf.WriteByte(bits >> 8)
		}

		if bits&svc.U_EXTEND1 != 0 {
			msgBuf.WriteByte(bits >> 16)
		}
		if bits&svc.U_EXTEND2 != 0 {
			msgBuf.WriteByte(bits >> 24)
		}

		if bits&svc.U_LONGENTITY != 0 {
			msgBuf.WriteShort(ent)
		} else {
			msgBuf.WriteByte(ent)
		}

		if bits&svc.U_MODEL != 0 {
			msgBuf.WriteByte(int(ev.ModelIndex))
		}
		if bits&svc.U_FRAME != 0 {
			msgBuf.WriteByte(int(ev.Frame))
		}
		if bits&svc.U_COLORMAP != 0 {
			msgBuf.WriteByte(int(ev.ColorMap))
		}
		if bits&svc.U_SKIN != 0 {
			msgBuf.WriteByte(int(ev.Skin))
		}
		if bits&svc.U_EFFECTS != 0 {
			msgBuf.WriteByte(int(ev.Effects))
		}
		if bits&svc.U_ORIGIN1 != 0 {
			msgBuf.WriteCoord(ev.Origin[0], s.protocolFlags)
		}
		if bits&svc.U_ANGLE1 != 0 {
			msgBuf.WriteAngle(ev.Angles[0], s.protocolFlags)
		}
		if bits&svc.U_ORIGIN2 != 0 {
			msgBuf.WriteCoord(ev.Origin[1], s.protocolFlags)
		}
		if bits&svc.U_ANGLE2 != 0 {
			msgBuf.WriteAngle(ev.Angles[1], s.protocolFlags)
		}
		if bits&svc.U_ORIGIN3 != 0 {
			msgBuf.WriteCoord(ev.Origin[2], s.protocolFlags)
		}
		if bits&svc.U_ANGLE3 != 0 {
			msgBuf.WriteAngle(ev.Angles[2], s.protocolFlags)
		}

		if bits&svc.U_ALPHA != 0 {
			msgBuf.WriteByte(int(edict.Alpha))
		}
		if bits&svc.U_FRAME2 != 0 {
			msgBuf.WriteByte(int(ev.Frame) >> 8)
		}
		if bits&svc.U_MODEL2 != 0 {
			msgBuf.WriteByte(int(ev.ModelIndex) >> 8)
		}
		if bits&svc.U_LERPFINISH != 0 {
			msgBuf.WriteByte(int(math.Round((ev.NextThink - sv.time) * 255)))
		}
	}
}

func init() {
	//if (Cvar_GetValue(&coop)) Cvar_Set("deathmatch", "0");
	cvars.Skill.SetCallback(func(cv *cvar.Cvar) {
		cs := float32(int(cv.Value() + 0.5))
		cs = math.Clamp32(0, cs, 3)
		if cv.Value() != cs { // Break recursion
			cv.SetValue(cs)
		}
	})
}

//This is called at the start of each level
func (s *Server) SpawnServer(name string) error {
	// let's not have any servers with no name
	if len(cvars.HostName.String()) == 0 {
		cvars.HostName.SetByString("UNNAMED")
	}

	conlog.DPrintf("SpawnServer: %s\n", name)
	// now safe to issue another
	svs.changeLevelIssued = false

	// tell all connected clients that we are going to a new level
	if s.active {
		s.sendReconnect()
	}

	// set up the new server
	C.Host_ClearMemory()
	freeEdicts()
	sv = Server{
		models:   make([]model.Model, 1),
		name:     name,
		protocol: sv_protocol,
	}
	s = &sv

	if s.protocol == protocol.RMQ {
		s.protocolFlags = protocol.PRFL_INT32COORD | protocol.PRFL_SHORTANGLE
	} else {
		s.protocolFlags = 0
	}

	// load progs to get entity field count
	LoadProgs()

	// allocate server memory
	// Host_ClearMemory() called above already cleared the whole sv structure
	s.maxEdicts = math.ClampI(MIN_EDICTS, int(cvars.MaxEdicts.Value()), MAX_EDICTS)
	AllocEdicts()

	// leave slots at start for clients only
	s.numEdicts = svs.maxClients + 1
	for i := 0; i < s.numEdicts; i++ {
		ClearEdict(i)
	}
	for i := 0; i < svs.maxClients; i++ {
		sv_clients[i].edictId = i + 1
	}

	s.state = ServerStateLoading
	s.paused = false
	s.time = 1.0
	s.modelName = fmt.Sprintf("maps/%s.bsp", name)

	log.Printf("New world: %s", s.modelName)
	s.worldModel = nil
	s.modelPrecache = s.modelPrecache[:0]
	s.soundPrecache = s.soundPrecache[:0]
	s.models = append(s.models, nil)
	s.models = s.models[:1]
	mods, err := bsp.Load(s.modelName)
	if err != nil || len(mods) < 1 {
		conlog.Printf("Couldn't spawn server %s\n", s.modelName)
		s.active = false
		return nil
	}
	s.worldModel = mods[0]
	s.modelPrecache = append(s.modelPrecache, string([]byte{0, 0, 0, 0, 0, 0, 0, 0}))
	s.soundPrecache = append(s.soundPrecache, string([]byte{0, 0, 0, 0, 0, 0, 0, 0}))
	for _, m := range mods {
		s.modelPrecache = append(s.modelPrecache, m.Name())
		s.models = append(s.models, m)
	}

	clearWorld()

	// load the rest of the entities
	ClearEntVars(0)
	sv.edicts[0].Free = false
	ev := EntVars(0)
	ev.Model = progsdat.AddString(s.modelName)
	ev.ModelIndex = 1 // world model
	ev.Solid = SOLID_BSP
	ev.MoveType = progs.MoveTypePush

	if cvars.Coop.Bool() {
		progsdat.Globals.Coop = 1
	} else {
		progsdat.Globals.DeathMatch = cvars.DeathMatch.Value()
	}
	progsdat.Globals.MapName = progsdat.AddString(name)

	// serverflags are for cross level information (sigils)
	progsdat.Globals.ServerFlags = float32(svs.serverFlags)

	if err := loadEntities(sv.worldModel.Entities); err != nil {
		return err
	}

	s.active = true

	// all setup is completed, any further precache statements are errors
	s.state = ServerStateActive

	// run two frames to allow everything to settle
	host.frameTime = 0.1
	if err := RunPhysics(); err != nil {
		return err
	}
	if err := RunPhysics(); err != nil {
		return err
	}

	// create a baseline for more efficient communications
	s.CreateBaseline()

	// warn if signon buffer larger than standard server can handle
	if s.signon.Len() > 8000-2 {
		// max size that will fit into 8000-sized client->message buffer
		// with 2 extra bytes on the end
		conlog.DWarning("%d byte signon buffer exceeds standard limit of 7998.\n", s.signon.Len())
	}

	// send serverinfo to all connected clients
	for i := 0; i < svs.maxClients; i++ {
		if sv_clients[i].active {
			sv_clients[i].SendServerinfo()
		}
	}

	conlog.DPrintf("Server spawned.\n")
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
				e.Alpha = -1
			} else {
				e.Alpha = (float32(wa) - 1) / 254
			}
		}

		eds = append(eds, e)
	}
	return eds
}

func (s *Server) loadGameEdicts(es []*protos.Edict) {
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
			ta = math.Clamp32(1, ta, 255)
			a = byte(ta)
		}
		s.edicts[i] = Edict{
			Alpha: a,
		}

		vm.loadGameEntVars(i, e)
		vm.LinkEdict(i, false)
	}
	s.numEdicts = len(es)
}
