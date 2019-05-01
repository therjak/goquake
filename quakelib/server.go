package quakelib

// void SV_DropClient(int,int);
// void SV_SetIdealPitch();
// void PR_ExecuteProgram(int p);
// void SV_WriteEntitiesToClient(int clent);
import "C"

import (
	"fmt"
	"log"
	"quake/cmd"
	cmdl "quake/commandline"
	"quake/conlog"
	"quake/cvars"
	"quake/execute"
	"quake/math/vec"
	"quake/model"
	"quake/net"
	"quake/progs"
	"quake/protocol"
	"quake/protocol/server"
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

	protocol      uint16
	protocolFlags uint16

	state ServerState // some actions are only valid during load

	modelPrecache []string
	soundPrecache []string
	lightStyles   []string

	name      string // map name
	modelName string // maps/<name>.bsp, for model_precache[0]

	models     []*model.QModel
	worldModel *model.QModel
}

var (
	svs = ServerStatic{}
	sv  = Server{
		models: make([]*model.QModel, 1),
	}
	sv_protocol int
	sv_player   int
)

//export SV_NameInt
func SV_NameInt() *C.char {
	return C.CString(sv.name)
}

//export SV_SetName
func SV_SetName(n *C.char) {
	sv.name = C.GoString(n)
}

//export SV_ModelNameInt
func SV_ModelNameInt() *C.char {
	return C.CString(sv.modelName)
}

//export SV_SetModelName
func SV_SetModelName(n *C.char, s *C.char) {
	sv.modelName = fmt.Sprintf(C.GoString(n), C.GoString(s))
}

//export SV_State
func SV_State() int {
	if sv.state == ServerStateLoading {
		return 0
	}
	return 1
}

//export SV_SetState
func SV_SetState(s C.int) {
	if s == 0 {
		sv.state = ServerStateLoading
	} else {
		sv.state = ServerStateActive
	}
}

//export SV_Player
func SV_Player() int {
	return sv_player
}

//export Set_SV_Player
func Set_SV_Player(p int) {
	sv_player = p
}

func svProtocol(args []cmd.QArg) {
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
	cmd.AddCommand("sv_protocol", svProtocol)
}

//export SV_Init_Go
func SV_Init_Go() {
	sv_protocol = cmdl.Protocol()
	switch sv_protocol {
	case protocol.NetQuake:
		log.Printf("Server using protocol %v (NetQuake)\n", sv_protocol)
	case protocol.FitzQuake:
		log.Printf("Server using protocol %v (FitzQuake)\n", sv_protocol)
	case protocol.RMQ:
		log.Printf("Server using protocol %v (RMQ)\n", sv_protocol)
	default:
		Error("Bad protocol version request %v. Accepted values: %v, %v, %v.",
			sv_protocol, protocol.NetQuake, protocol.FitzQuake, protocol.RMQ)
		log.Printf("Server using protocol %v (Unknown)\n", sv_protocol)
	}
}

//export SV_LastCheck
func SV_LastCheck() C.int {
	return C.int(sv.lastCheck)
}

//export SV_SetLastCheck
func SV_SetLastCheck(c C.int) {
	sv.lastCheck = int(c)
}

//export SV_Time
func SV_Time() C.float {
	return C.float(sv.time)
}

//export SV_SetTime
func SV_SetTime(t C.float) {
	sv.time = float32(t)
}

//export SV_LastCheckTime
func SV_LastCheckTime() C.float {
	return C.float(sv.lastCheckTime)
}

//export SV_SetLastCheckTime
func SV_SetLastCheckTime(t C.float) {
	sv.lastCheckTime = float32(t)
}

//export SV_NumEdicts
func SV_NumEdicts() C.int {
	return C.int(sv.numEdicts)
}

//export SV_SetNumEdicts
func SV_SetNumEdicts(n C.int) {
	sv.numEdicts = int(n)
}

//export SV_MaxEdicts
func SV_MaxEdicts() C.int {
	return C.int(sv.maxEdicts)
}

//export SV_SetMaxEdicts
func SV_SetMaxEdicts(n C.int) {
	sv.maxEdicts = int(n)
}

//export SV_SetProtocol
func SV_SetProtocol() {
	sv.protocol = uint16(sv_protocol)
}

//export SV_Protocol
func SV_Protocol() C.ushort {
	return C.ushort(sv.protocol)
}

//export SV_SetProtocolFlags
func SV_SetProtocolFlags(flags C.ushort) {
	sv.protocolFlags = uint16(flags)
}

//export SV_ProtocolFlags
func SV_ProtocolFlags() C.ushort {
	return C.ushort(sv.protocolFlags)
}

//export SV_Paused
func SV_Paused() C.int {
	return b2i(sv.paused)
}

//export SV_SetPaused
func SV_SetPaused(b C.int) {
	sv.paused = (b != 0)
}

//export SV_LoadGame
func SV_LoadGame() C.int {
	return b2i(sv.loadGame)
}

//export SV_SetLoadGame
func SV_SetLoadGame(b C.int) {
	sv.loadGame = (b != 0)
}

//export SV_Clear
func SV_Clear() {
	sv = Server{
		models: make([]*model.QModel, 1),
	}
}

//export SV_Active
func SV_Active() C.int {
	return b2i(sv.active)
}

//export SV_SetActive
func SV_SetActive(v C.int) {
	sv.active = (v != 0)
}

var (
	msgBuf       = net.Message{}
	msgBufMaxLen = 0
)

//export SV_MS_Clear
func SV_MS_Clear() {
	msgBuf.ClearMessage()
}

//export SV_MS_SetMaxLen
func SV_MS_SetMaxLen(v C.int) {
	msgBufMaxLen = int(v)
}

//export SV_MS_MaxLen
func SV_MS_MaxLen() C.int {
	return C.int(msgBufMaxLen)
}

//export SV_MS_WriteByte
func SV_MS_WriteByte(v C.int) {
	msgBuf.WriteByte(int(v))
}

//export SV_MS_WriteChar
func SV_MS_WriteChar(v C.int) {
	msgBuf.WriteChar(int(v))
}

//export SV_MS_WriteShort
func SV_MS_WriteShort(v C.int) {
	msgBuf.WriteShort(int(v))
}

//export SV_MS_WriteLong
func SV_MS_WriteLong(v C.int) {
	msgBuf.WriteLong(int(v))
}

//export SV_MS_WriteFloat
func SV_MS_WriteFloat(f C.float) {
	msgBuf.WriteFloat(float32(f))
}

//export SV_MS_WriteAngle
func SV_MS_WriteAngle(v C.float) {
	msgBuf.WriteAngle(float32(v), int(sv.protocolFlags))
}

//export SV_MS_WriteCoord
func SV_MS_WriteCoord(v C.float) {
	msgBuf.WriteCoord(float32(v), int(sv.protocolFlags))
}

//export SV_MS_WriteString
func SV_MS_WriteString(s *C.char) {
	msgBuf.WriteString(C.GoString(s))
}

//export SV_MS_Len
func SV_MS_Len() C.int {
	return C.int(msgBuf.Len())
}

//export SV_SO_WriteByte
func SV_SO_WriteByte(v C.int) {
	sv.signon.WriteByte(int(v))
}

//export SV_SO_WriteChar
func SV_SO_WriteChar(v C.int) {
	sv.signon.WriteChar(int(v))
}

//export SV_SO_WriteShort
func SV_SO_WriteShort(v C.int) {
	sv.signon.WriteShort(int(v))
}

//export SV_SO_WriteLong
func SV_SO_WriteLong(v C.int) {
	sv.signon.WriteLong(int(v))
}

//export SV_SO_WriteAngle
func SV_SO_WriteAngle(v C.float) {
	sv.signon.WriteAngle(float32(v), int(sv.protocolFlags))
}

//export SV_SO_WriteCoord
func SV_SO_WriteCoord(v C.float) {
	sv.signon.WriteCoord(float32(v), int(sv.protocolFlags))
}

//export SV_SO_WriteString
func SV_SO_WriteString(s *C.char) {
	sv.signon.WriteString(C.GoString(s))
}

//export SV_SO_Len
func SV_SO_Len() C.int {
	return C.int(sv.signon.Len())
}

func (s *Server) StartParticle(org, dir vec.Vec3, color, count int) {
	if s.datagram.Len()+16 > net.MAX_DATAGRAM {
		return
	}
	s.datagram.WriteByte(server.Particle)
	s.datagram.WriteCoord(org.X, int(s.protocolFlags))
	s.datagram.WriteCoord(org.Y, int(s.protocolFlags))
	s.datagram.WriteCoord(org.Z, int(s.protocolFlags))
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
	s.datagram.WriteChar(df(dir.X))
	s.datagram.WriteChar(df(dir.Y))
	s.datagram.WriteChar(df(dir.Z))
	s.datagram.WriteByte(count)
	s.datagram.WriteByte(color)
}

//export SV_ClearDatagram
func SV_ClearDatagram() {
	sv.datagram.ClearMessage()
}

func (s *Server) SendDatagram(c *SVClient) bool {
	b := msgBuf.Bytes()
	// If there is space add the server datagram
	if len(b)+s.datagram.Len() < 32000 {
		b = append(b, s.datagram.Bytes()...)
	}
	if c.netConnection.SendUnreliableMessage(b) == -1 {
		C.SV_DropClient(C.int(c.id), 1)
		return false
	}
	return true
}

//export SV_RD_WriteByte
func SV_RD_WriteByte(v C.int) {
	sv.reliableDatagram.WriteByte(int(v))
}

//export SV_RD_WriteString
func SV_RD_WriteString(s *C.char) {
	sv.reliableDatagram.WriteString(C.GoString(s))
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

//export SVS_GetServerFlags
func SVS_GetServerFlags() C.int {
	return C.int(svs.serverFlags)
}

//export SVS_SetServerFlags
func SVS_SetServerFlags(flags C.int) {
	svs.serverFlags = int(flags)
}

//export SVS_IsChangeLevelIssued
func SVS_IsChangeLevelIssued() C.int {
	return b2i(svs.changeLevelIssued)
}

//export SVS_SetChangeLevelIssued
func SVS_SetChangeLevelIssued(s C.int) {
	if s == 0 {
		svs.changeLevelIssued = false
		return
	}
	svs.changeLevelIssued = true
}

//export SVS_GetMaxClients
func SVS_GetMaxClients() C.int {
	return C.int(svs.maxClients)
}

//export SVS_SetMaxClients
func SVS_SetMaxClients(n C.int) {
	svs.maxClients = int(n)
}

//export SVS_GetMaxClientsLimit
func SVS_GetMaxClientsLimit() C.int {
	return C.int(svs.maxClientsLimit)
}

//export SVS_SetMaxClientsLimit
func SVS_SetMaxClientsLimit(n C.int) {
	svs.maxClientsLimit = int(n)
}

//export SV_SendReconnect
func SV_SendReconnect() {
	SendReconnectToAll()
	if !cmdl.Dedicated() {
		execute.Execute("reconnect\n", execute.Command)
	}
}

/*
Each entity can have eight independant sound sources, like voice,
weapon, feet, etc.

Channel 0 is an auto-allocate channel, the others override anything
allready running on that entity/channel pair.

An attenuation of 0 will play full volume everywhere in the level.
Larger attenuations will drop off.  (max 4 attenuation)
*/
//export SV_StartSound
func SV_StartSound(entity, channel C.int, sample *C.char, volume C.int,
	attenuation C.float) {
	sv.StartSound(int(entity), int(channel), int(volume), C.GoString(sample), float32(attenuation))
}

func (s *Server) StartSound(entity, channel, volume int, sample string, attenuation float32) {
	if volume < 0 || volume > 255 {
		HostError("SV_StartSound: volume = %d", volume)
	}
	if attenuation < 0 || attenuation > 4 {
		HostError("SV_StartSound: attenuation = %f", attenuation)
	}
	if channel < 0 || channel > 7 {
		HostError("SV_StartSound: channel = %d", channel)
	}
	if s.datagram.Len() > net.MAX_DATAGRAM-16 {
		return
	}
	for soundnum, m := range s.soundPrecache {
		if m == sample {
			s.sendStartSound(entity, channel, volume, soundnum, attenuation)
			return
		}
	}
	conlog.Printf("SV_StartSound: %s not precacheed", sample)
}

func (s *Server) sendStartSound(entity, channel, volume, soundnum int, attenuation float32) {
	fieldMask := 0
	if volume != 255 {
		fieldMask |= server.SoundVolume
	}
	if attenuation != 1.0 {
		fieldMask |= server.SoundAttenuation
	}
	if entity >= 8192 {
		if s.protocol == protocol.NetQuake {
			return // protocol does not support this info
		}
		fieldMask |= server.SoundLargeEntity
	}
	if soundnum >= 256 || channel >= 8 {
		if s.protocol == protocol.NetQuake {
			return
		}
		fieldMask |= server.SoundLargeSound
	}
	s.datagram.WriteByte(server.Sound)
	s.datagram.WriteByte(fieldMask)
	if fieldMask&server.SoundVolume != 0 {
		s.datagram.WriteByte(volume)
	}
	if fieldMask&server.SoundAttenuation != 0 {
		s.datagram.WriteByte(int(attenuation * 64))
	}
	if fieldMask&server.SoundLargeEntity != 0 {
		s.datagram.WriteShort(entity)
		s.datagram.WriteByte(channel)
	} else {
		s.datagram.WriteShort((entity << 3) | channel)
	}
	if fieldMask&server.SoundLargeSound != 0 {
		s.datagram.WriteShort(soundnum)
	} else {
		s.datagram.WriteByte(soundnum)
	}
	ev := EntVars(entity)
	flags := int(s.protocolFlags)
	s.datagram.WriteCoord(ev.Origin[0]+0.5*(ev.Mins[0]+ev.Maxs[0]), flags)
	s.datagram.WriteCoord(ev.Origin[1]+0.5*(ev.Mins[1]+ev.Maxs[1]), flags)
	s.datagram.WriteCoord(ev.Origin[2]+0.5*(ev.Mins[2]+ev.Maxs[2]), flags)
}

//export SV_CleanupEnts
func SV_CleanupEnts() {
	sv.CleanupEntvarEffects()
}

func (s *Server) CleanupEntvarEffects() {
	for i := 1; i < s.numEdicts; i++ {
		ev := EntVars(i)
		eff := int(ev.Effects)
		ev.Effects = float32(eff &^ server.EffectMuzzleFlash)
	}
}

//export SV_WriteClientdataToMessage
func SV_WriteClientdataToMessage(ent C.int) {
	sv.WriteClientdataToMessage(EntVars(int(ent)), EntityAlpha(int(ent)))
}

func (s *Server) WriteClientdataToMessage(e *progs.EntVars, alpha byte) {
	flags := int(s.protocolFlags)
	if e.DmgTake != 0 || e.DmgSave != 0 {
		other := EntVars(int(e.DmgInflictor))
		msgBuf.WriteByte(server.Damage)
		msgBuf.WriteByte(int(e.DmgSave))
		msgBuf.WriteByte(int(e.DmgTake))
		msgBuf.WriteCoord(other.Origin[0]+0.5*(other.Mins[0]+other.Maxs[0]), flags)
		msgBuf.WriteCoord(other.Origin[1]+0.5*(other.Mins[1]+other.Maxs[1]), flags)
		msgBuf.WriteCoord(other.Origin[2]+0.5*(other.Mins[2]+other.Maxs[2]), flags)
		e.DmgTake = 0
		e.DmgSave = 0
	}

	// send the current viewpos offset from the view entity
	C.SV_SetIdealPitch() // how much to loop up/down ideally

	// a fixangle might get lost in a dropped packet.  Oh well.
	if e.FixAngle != 0 {
		msgBuf.WriteByte(server.SetAngle)
		msgBuf.WriteAngle(e.Angles[0], flags)
		msgBuf.WriteAngle(e.Angles[1], flags)
		msgBuf.WriteAngle(e.Angles[2], flags)
		e.FixAngle = 0
	}

	bits := 0
	if e.ViewOfs[2] != server.DEFAULT_VIEWHEIGHT {
		bits |= server.SU_VIEWHEIGHT
	}
	if e.IdealPitch != 0 {
		bits |= server.SU_IDEALPITCH
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
	bits |= server.SU_ITEMS
	if (int(e.Flags) & progs.FlagOnGround) != 0 {
		bits |= server.SU_ONGROUND
	}
	if e.WaterLevel >= 2 {
		bits |= server.SU_INWATER
	}
	if e.PunchAngle[0] != 0 {
		bits |= server.SU_PUNCH1
	}
	if e.PunchAngle[1] != 0 {
		bits |= server.SU_PUNCH2
	}
	if e.PunchAngle[2] != 0 {
		bits |= server.SU_PUNCH3
	}
	if e.Velocity[0] != 0 {
		bits |= server.SU_VELOCITY1
	}
	if e.Velocity[1] != 0 {
		bits |= server.SU_VELOCITY2
	}
	if e.Velocity[2] != 0 {
		bits |= server.SU_VELOCITY3
	}
	if e.WeaponFrame != 0 {
		bits |= server.SU_WEAPONFRAME
	}
	if e.ArmorValue != 0 {
		bits |= server.SU_ARMOR
	}
	bits |= server.SU_WEAPON

	wmi := 0
	wms := PRGetString(int(e.WeaponModel))
	if wms != nil {
		wmi = s.ModelIndex(*wms)
	}

	if s.protocol != protocol.NetQuake {
		if (wmi & 0xFF00) != 0 {
			bits |= server.SU_WEAPON2
		}
		if (int(e.ArmorValue) & 0xFF00) != 0 {
			bits |= server.SU_ARMOR2
		}
		if (int(e.CurrentAmmo) & 0xFF00) != 0 {
			bits |= server.SU_AMMO2
		}
		if (int(e.AmmoShells) & 0xFF00) != 0 {
			bits |= server.SU_SHELLS2
		}
		if (int(e.AmmoNails) & 0xFF00) != 0 {
			bits |= server.SU_NAILS2
		}
		if (int(e.AmmoRockets) & 0xFF00) != 0 {
			bits |= server.SU_ROCKETS2
		}
		if (int(e.AmmoCells) & 0xFF00) != 0 {
			bits |= server.SU_CELLS2
		}
		if (bits&server.SU_WEAPONFRAME != 0) &&
			(int(e.WeaponFrame)&0xFF00) != 0 {
			bits |= server.SU_WEAPONFRAME2
		}
		if alpha != 0 {
			bits |= server.SU_WEAPONALPHA
		}
		if bits >= 65536 {
			bits |= server.SU_EXTEND1
		}
		if bits >= 16777216 {
			bits |= server.SU_EXTEND2
		}
	}
	msgBuf.WriteByte(server.ClientData)
	msgBuf.WriteShort(bits)
	if (bits & server.SU_EXTEND1) != 0 {
		msgBuf.WriteByte(bits >> 16)
	}
	if (bits & server.SU_EXTEND2) != 0 {
		msgBuf.WriteByte(bits >> 24)
	}
	if (bits & server.SU_VIEWHEIGHT) != 0 {
		msgBuf.WriteChar(int(e.ViewOfs[2]))
	}
	if (bits & server.SU_IDEALPITCH) != 0 {
		msgBuf.WriteChar(int(e.IdealPitch))
	}
	if (bits & (server.SU_PUNCH1)) != 0 {
		msgBuf.WriteChar(int(e.PunchAngle[0]))
	}
	if (bits & (server.SU_PUNCH2)) != 0 {
		msgBuf.WriteChar(int(e.PunchAngle[1]))
	}
	if (bits & (server.SU_PUNCH3)) != 0 {
		msgBuf.WriteChar(int(e.PunchAngle[2]))
	}
	if (bits & (server.SU_VELOCITY1)) != 0 {
		msgBuf.WriteChar(int(e.Velocity[0] / 16))
	}
	if (bits & (server.SU_VELOCITY2)) != 0 {
		msgBuf.WriteChar(int(e.Velocity[1] / 16))
	}
	if (bits & (server.SU_VELOCITY3)) != 0 {
		msgBuf.WriteChar(int(e.Velocity[2] / 16))
	}

	msgBuf.WriteLong(items)

	if (bits & (server.SU_WEAPONFRAME)) != 0 {
		msgBuf.WriteByte(int(e.WeaponFrame))
	}
	if (bits & (server.SU_ARMOR)) != 0 {
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
	if (bits & (server.SU_WEAPON2)) != 0 {
		msgBuf.WriteByte(wmi >> 8)
	}
	if (bits & (server.SU_ARMOR2)) != 0 {
		msgBuf.WriteByte(int(e.ArmorValue) >> 8)
	}
	if (bits & (server.SU_AMMO2)) != 0 {
		msgBuf.WriteByte(int(e.CurrentAmmo) >> 8)
	}
	if (bits & (server.SU_SHELLS2)) != 0 {
		msgBuf.WriteByte(int(e.AmmoShells) >> 8)
	}
	if (bits & (server.SU_NAILS2)) != 0 {
		msgBuf.WriteByte(int(e.AmmoNails) >> 8)
	}
	if (bits & (server.SU_NAILS2)) != 0 {
		msgBuf.WriteByte(int(e.AmmoNails) >> 8)
	}
	if (bits & (server.SU_ROCKETS2)) != 0 {
		msgBuf.WriteByte(int(e.AmmoRockets) >> 8)
	}
	if (bits & (server.SU_CELLS2)) != 0 {
		msgBuf.WriteByte(int(e.AmmoCells) >> 8)
	}
	if (bits & (server.SU_WEAPONFRAME2)) != 0 {
		msgBuf.WriteByte(int(e.WeaponFrame) >> 8)
	}
	if (bits & (server.SU_WEAPONALPHA)) != 0 {
		msgBuf.WriteByte(int(alpha))
	}
}

/*
Initializes a client_t for a new net connection.  This will only be called
once for a player each game, not once for each level change.
*/
//export SV_ConnectClient
func SV_ConnectClient(n C.int) {
	ConnectClient(int(n))
}

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
		C.PR_ExecuteProgram(C.int(progsdat.Globals.SetNewParms))
		new.spawnParams = progsdat.Globals.Parm
	}
	sv_clients[n] = new
	new.SendServerinfo()
}

//export SV_SendClientDatagram
func SV_SendClientDatagram(c C.int) C.int {
	return b2i(sv.SendClientDatagram(sv_clients[int(c)]))
}

func (s *Server) SendClientDatagram(c *SVClient) bool {
	msgBuf.ClearMessage()
	msgBufMaxLen = net.MAX_DATAGRAM
	if c.Address() != "LOCAL" {
		msgBufMaxLen = net.DATAGRAM_MTU
	}
	msgBuf.WriteByte(server.Time)
	msgBuf.WriteFloat(s.time)

	s.WriteClientdataToMessage(EntVars(c.edictId), EntityAlpha(c.edictId))

	C.SV_WriteEntitiesToClient(C.int(c.edictId))

	return s.SendDatagram(c)
}

//export SV_UpdateToReliableMessages
func SV_UpdateToReliableMessages() {
	sv.UpdateToReliableMessages()
}

func (s *Server) UpdateToReliableMessages() {
	b := s.reliableDatagram.Bytes()
	for _, cl := range sv_clients {
		newFrags := EntVars(cl.edictId).Frags
		if cl.active {
			// Does it actually matter to compare as float32?
			// These subtle C things...
			if float32(cl.oldFrags) != newFrags {
				cl.msg.WriteByte(server.UpdateFrags)
				cl.msg.WriteByte(cl.id)
				cl.msg.WriteShort(int(newFrags))
			}
			cl.msg.WriteBytes(b)
		}
		cl.oldFrags = int(newFrags)
	}
	s.reliableDatagram.ClearMessage()
}

//export SV_Impact
func SV_Impact(e1, e2 C.int) {
	sv.Impact(int(e1), int(e2))
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
		C.PR_ExecuteProgram(C.int(ent1.Touch))
	}

	if ent2.Touch != 0 && ent2.Solid != SOLID_NOT {
		progsdat.Globals.Self = int32(e2)
		progsdat.Globals.Other = int32(e1)
		C.PR_ExecuteProgram(C.int(ent2.Touch))
	}

	progsdat.Globals.Self = oldSelf
	progsdat.Globals.Other = oldOther
}

//export SV_CheckVelocity
func SV_CheckVelocity(e C.int) {
	CheckVelocity(EntVars(int(e)))
}

func CheckVelocity(ent *progs.EntVars) {
	maxVelocity := cvars.ServerMaxVelocity.Value()
	for i := 0; i < 3; i++ {
		if ent.Velocity[i] != ent.Velocity[i] {
			s := PRGetString(int(ent.ClassName))
			conlog.Printf("Got a NaN velocity on %s\n", *s)
			ent.Velocity[i] = 0
		}
		if ent.Origin[i] != ent.Origin[i] {
			s := PRGetString(int(ent.ClassName))
			conlog.Printf("Got a NaN origin on %s\n", *s)
			ent.Origin[i] = 0
		}
		if ent.Velocity[i] > maxVelocity {
			ent.Velocity[i] = maxVelocity
		} else if ent.Velocity[i] < -maxVelocity {
			ent.Velocity[i] = -maxVelocity
		}
	}
}
