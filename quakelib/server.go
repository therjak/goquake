package quakelib

// void SV_DropClient(int,int);
// void SV_SetIdealPitch();
import "C"

import (
	"fmt"
	"log"
	"quake/cmd"
	cmdl "quake/commandline"
	"quake/execute"
	"quake/math"
	"quake/net"
	"quake/progs"
	"quake/protocol"
	"quake/protocol/server"
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

	time          float64
	lastCheck     int
	lastCheckTime float64

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

	name      string
	modelName string
}

var (
	svs         = ServerStatic{}
	sv          = Server{}
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
		conSafePrintStr("usage: sv_protocol <protocol>\n")
	case 0:
		conPrintf(`"sv_protocol" is "%v"`+"\n", sv_protocol)
	case 1:
		i := args[0].Int()
		switch i {
		case protocol.NetQuake, protocol.FitzQuake, protocol.RMQ:
			sv_protocol = i
			if sv.active {
				conPrintf("changes will not take effect until the next level load.\n")
			}
		default:
			conPrintf("sv_protocol must be %v or %v or %v\n",
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
func SV_Time() C.double {
	return C.double(sv.time)
}

//export SV_SetTime
func SV_SetTime(t C.double) {
	sv.time = float64(t)
}

//export SV_LastCheckTime
func SV_LastCheckTime() C.double {
	return C.double(sv.lastCheckTime)
}

//export SV_SetLastCheckTime
func SV_SetLastCheckTime(t C.double) {
	sv.lastCheckTime = float64(t)
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
	sv = Server{}
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

func svStartParticle(org, dir math.Vec3, color, count int) {
	if sv.datagram.Len()+16 > net.MAX_DATAGRAM {
		return
	}
	sv.datagram.WriteByte(server.Particle)
	sv.datagram.WriteCoord(org.X, int(sv.protocolFlags))
	sv.datagram.WriteCoord(org.Y, int(sv.protocolFlags))
	sv.datagram.WriteCoord(org.Z, int(sv.protocolFlags))
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
	sv.datagram.WriteChar(df(dir.X))
	sv.datagram.WriteChar(df(dir.Y))
	sv.datagram.WriteChar(df(dir.Z))
	sv.datagram.WriteByte(count)
	sv.datagram.WriteByte(color)
}

//export SV_StartParticle
func SV_StartParticle(org, dir *C.float, color, count C.int) {
	svStartParticle(cfloatToVec3(org), cfloatToVec3(dir), int(color), int(count))
}

//export SV_DG_WriteByte
func SV_DG_WriteByte(v C.int) {
	sv.datagram.WriteByte(int(v))
}

//export SV_DG_WriteChar
func SV_DG_WriteChar(v C.int) {
	sv.datagram.WriteChar(int(v))
}

//export SV_DG_WriteShort
func SV_DG_WriteShort(v C.int) {
	sv.datagram.WriteShort(int(v))
}

//export SV_DG_WriteLong
func SV_DG_WriteLong(v C.int) {
	sv.datagram.WriteLong(int(v))
}

//export SV_DG_WriteAngle
func SV_DG_WriteAngle(v C.float) {
	sv.datagram.WriteAngle(float32(v), int(sv.protocolFlags))
}

//export SV_DG_WriteCoord
func SV_DG_WriteCoord(v C.float) {
	sv.datagram.WriteCoord(float32(v), int(sv.protocolFlags))
}

//export SV_DG_WriteString
func SV_DG_WriteString(s *C.char) {
	sv.datagram.WriteString(C.GoString(s))
}

//export SV_DG_Len
func SV_DG_Len() C.int {
	return C.int(sv.datagram.Len())
}

//export SV_DG_ClearMessage
func SV_DG_ClearMessage() {
	sv.datagram.ClearMessage()
}

//export SV_ClearDatagram
func SV_ClearDatagram() {
	SV_DG_ClearMessage()
}

//export SV_DG_SendOut
func SV_DG_SendOut(client C.int) C.int {
	b := msgBuf.Bytes()
	// If there is space add the server datagram
	if len(b)+sv.datagram.Len() < 32000 {
		b = append(b, sv.datagram.Bytes()...)
	}
	con := sv_clients[int(client)].netConnection
	if con.SendUnreliableMessage(b) == -1 {
		C.SV_DropClient(client, 1)
		return 0
	}
	return 1
}

//export SV_RD_WriteByte
func SV_RD_WriteByte(v C.int) {
	sv.reliableDatagram.WriteByte(int(v))
}

//export SV_RD_WriteChar
func SV_RD_WriteChar(v C.int) {
	sv.reliableDatagram.WriteChar(int(v))
}

//export SV_RD_WriteShort
func SV_RD_WriteShort(v C.int) {
	sv.reliableDatagram.WriteShort(int(v))
}

//export SV_RD_WriteLong
func SV_RD_WriteLong(v C.int) {
	sv.reliableDatagram.WriteLong(int(v))
}

//export SV_RD_WriteAngle
func SV_RD_WriteAngle(v C.float) {
	sv.reliableDatagram.WriteAngle(float32(v), int(sv.protocolFlags))
}

//export SV_RD_WriteCoord
func SV_RD_WriteCoord(v C.float) {
	sv.reliableDatagram.WriteCoord(float32(v), int(sv.protocolFlags))
}

//export SV_RD_WriteString
func SV_RD_WriteString(s *C.char) {
	sv.reliableDatagram.WriteString(C.GoString(s))
}

//export SV_RD_SendOut
func SV_RD_SendOut() {
	b := sv.reliableDatagram.Bytes()
	for _, cl := range sv_clients {
		if cl.active {
			cl.msg.WriteBytes(b)
		}
	}
	sv.reliableDatagram.ClearMessage()
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
	conPrintf("SV_StartSound: %s not precacheed", sample)
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
	s.datagram.WriteCoord(ev.Origin[0]+0.5*ev.Mins[0]+ev.Maxs[0], flags)
	s.datagram.WriteCoord(ev.Origin[1]+0.5*ev.Mins[1]+ev.Maxs[1], flags)
	s.datagram.WriteCoord(ev.Origin[2]+0.5*ev.Mins[2]+ev.Maxs[2], flags)
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
		msgBuf.WriteCoord(other.Origin[0]+0.5*other.Mins[0]+other.Maxs[0], flags)
		msgBuf.WriteCoord(other.Origin[1]+0.5*other.Mins[1]+other.Maxs[1], flags)
		msgBuf.WriteCoord(other.Origin[2]+0.5*other.Mins[2]+other.Maxs[2], flags)
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

	wmi := s.ModelIndex(PR_GetStringWrap(int(e.WeaponModel)))
	if s.protocol != protocol.NetQuake {
		if (bits&server.SU_WEAPON != 0) &&
			((wmi & 0xFF00) != 0) {
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
		/*
		   if (bits & SU_WEAPONFRAME && (int)EVars(ent)->weaponframe & 0xFF00)
		     bits |= SU_WEAPONFRAME2;
		   if (bits & SU_WEAPON && EDICT_NUM(ent)->alpha != ENTALPHA_DEFAULT)
		     bits |= SU_WEAPONALPHA;  // for now, weaponalpha = client entity alpha
		*/
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

	if !(cmdl.Quoth() || cmdl.Rogue() || cmdl.Hipnotic()) {
		msgBuf.WriteByte(int(e.Weapon))
	} else {
		for i := 0; i < 32; i++ {
			if int(e.Weapon)&(1<<uint(i)) != 0 {
				msgBuf.WriteByte(i)
				break
			}
		}
	}
	/*
	   // johnfitz -- PROTOCOL_FITZQUAKE
	   if (bits & SU_WEAPON2)
	     SV_MS_WriteByte(SV_ModelIndex(PR_GetString(EVars(ent)->weaponmodel)) >> 8);
	   if (bits & SU_ARMOR2) SV_MS_WriteByte((int)EVars(ent)->armorvalue >> 8);
	   if (bits & SU_AMMO2) SV_MS_WriteByte((int)EVars(ent)->currentammo >> 8);
	   if (bits & SU_SHELLS2) SV_MS_WriteByte((int)EVars(ent)->ammo_shells >> 8);
	   if (bits & SU_NAILS2) SV_MS_WriteByte((int)EVars(ent)->ammo_nails >> 8);
	   if (bits & SU_ROCKETS2) SV_MS_WriteByte((int)EVars(ent)->ammo_rockets >> 8);
	   if (bits & SU_CELLS2) SV_MS_WriteByte((int)EVars(ent)->ammo_cells >> 8);
	   if (bits & SU_WEAPONFRAME2)
	     SV_MS_WriteByte((int)EVars(ent)->weaponframe >> 8);
	   // for now, weaponalpha = client entity alpha
	   if (bits & SU_WEAPONALPHA) SV_MS_WriteByte(EDICT_NUM(ent)->alpha);
	   // johnfitz
	*/
}
