package quakelib

//void V_StopPitchDrift(void);
//void CL_StopPlayback(void);
//void CL_Stop_f(void);
//#define SFX_WIZHIT  0
//#define SFX_KNIGHTHIT  1
//#define SFX_TINK1  2
//#define SFX_RIC1  3
//#define SFX_RIC2  4
//#define SFX_RIC3  5
//#define SFX_R_EXP3  6
import "C"

import (
	"bytes"
	"fmt"
	"log"
	"quake/cbuf"
	"quake/cmd"
	cmdl "quake/commandline"
	"quake/conlog"
	"quake/cvars"
	"quake/input"
	"quake/keys"
	"quake/math"
	"quake/math/vec"
	"quake/model"
	"quake/net"
	clc "quake/protocol/client"
	svc "quake/protocol/server"
	"quake/snd"
	"quake/stat"
	"time"
	"unsafe"
)

type sfx int

const (
	WizHit    sfx = C.SFX_WIZHIT
	KnightHit sfx = C.SFX_KNIGHTHIT
	Tink1     sfx = C.SFX_TINK1
	Ric1      sfx = C.SFX_RIC1
	Ric2      sfx = C.SFX_RIC2
	Ric3      sfx = C.SFX_RIC3
	RExp3     sfx = C.SFX_R_EXP3
)

func init() {
	cmd.AddCommand("disconnect", func(args []cmd.QArg, _ int) { clientDisconnect() })
	cmd.AddCommand("reconnect", func(args []cmd.QArg, _ int) { clientReconnect() })

	cmd.AddCommand("startdemos", clientStartDemos)

	//Cmd_AddCommand("mcache", Mod_Print);
}

const (
	numSignonMessagesBeforeConn = 4 // signon messages to receive before connected
)

const (
	ca_dedicated    = 0
	ca_disconnected = 1
	ca_connected    = 2
)

//export CLPitch
func CLPitch() C.float {
	return C.float(cl.pitch)
}

//export CLYaw
func CLYaw() C.float {
	return C.float(cl.yaw)
}

//export CLRoll
func CLRoll() C.float {
	return C.float(cl.roll)
}

//export SetCLPitch
func SetCLPitch(v C.float) {
	cl.pitch = float32(v)
}

//export SetCLYaw
func SetCLYaw(v C.float) {
	cl.yaw = float32(v)
}

//export SetCLRoll
func SetCLRoll(v C.float) {
	cl.roll = float32(v)
}

//export IncCLPitch
func IncCLPitch(v C.float) {
	cl.pitch += float32(v)
}

//export DecCLPitch
func DecCLPitch(v C.float) {
	cl.pitch -= float32(v)
}

//export CL_AdjustAngles
func CL_AdjustAngles() {
	speed := func() float32 {
		if (cvars.ClientForwardSpeed.Value() > 200) != input.Speed.Down() {
			return float32(FrameTime()) * cvars.ClientAngleSpeedKey.Value()
		}
		return float32(FrameTime())
	}()
	if !input.Strafe.Down() {
		cl.yaw -= speed * cvars.ClientYawSpeed.Value() * input.Right.ConsumeImpulse()
		cl.yaw += speed * cvars.ClientYawSpeed.Value() * input.Left.ConsumeImpulse()
		cl.yaw = math.AngleMod32(cl.yaw)
	}
	if input.KLook.Down() {
		C.V_StopPitchDrift()
		cl.pitch -= speed * cvars.ClientPitchSpeed.Value() * input.Forward.ConsumeImpulse()
		cl.pitch += speed * cvars.ClientPitchSpeed.Value() * input.Back.ConsumeImpulse()
	}

	up := input.LookUp.ConsumeImpulse()
	down := input.LookDown.ConsumeImpulse()

	cl.pitch -= speed * cvars.ClientPitchSpeed.Value() * up
	cl.pitch += speed * cvars.ClientPitchSpeed.Value() * down

	if up != 0 || down != 0 {
		C.V_StopPitchDrift()
	}

	cl.pitch = math.Clamp32(cvars.ClientMinPitch.Value(), cl.pitch, cvars.ClientMaxPitch.Value())
	cl.roll = math.Clamp32(-50, cl.roll, 50)
}

// this is persistent through an arbitrary number of server connections
type ClientStatic struct {
	state              int // enum dedicated = 0, disconnected, connected
	demoNum            int
	demoRecording      bool
	demoPlayback       bool
	demoPaused         bool
	timeDemo           bool
	signon             int
	connection         *net.Connection
	inMessage          *net.QReader
	outMessage         bytes.Buffer
	forceTrack         int // -1 to use normal cd track
	timeDemoLastFrame  int
	timeDemoStartFrame int
	timeDemoStartTime  float32
	// personalization data sent to server
	// to restart a level
	spawnParms string
	demos      []string
	/*
		demoFile 'filehandle'
	*/
	msgBadRead bool

	// net.PacketCon
}

type Client struct {
	pitch          float32 // 0
	yaw            float32 // 1
	roll           float32 // 2
	movemessages   int     // number of messages since connecting to skip the first couple
	cmdForwardMove float32 // last command sent to the server
	protocolFlags  uint16
	protocol       uint16
	viewentity     int //cl_entities[cl.viewentity] = player
	onGround       bool
	paused         bool // music paused

	messageTime      float64
	messageTimeOld   float64
	time             float64
	oldTime          float64
	intermissionTime int // was called completed_time

	lastReceivedMessageTime float64

	// don't change view angle, full screen, etc
	intermission int
	// num_statics
	// maxclients
	// scores
	// sound_precache
	// model_precache
	// scores
	// viewent.model
	// items
	// item_gettime
	// mvelocity
	// punchangle
	// idealpitch
	// viewheight
	// mapname
	// gametype
	// levelname
	//

	mapName    string
	levelName  string
	worldModel *model.QModel

	maxClients int
	gameType   int

	// the server sends first a name and afterwards just the index of the name
	// the int represents the sfx num within snd. Result of Precache and sfx input of Start
	// TODO: change the int to a sfx type
	soundPrecache []int

	stats     ClientStats
	maxEdicts int
}

type ClientStats struct {
	health        int
	frags         int
	weapon        int
	ammo          int
	armor         int
	weaponFrame   int
	shells        int
	nails         int
	rockets       int
	cells         int
	activeWeapon  int
	totalSecrets  int
	totalMonsters int
	secrets       int
	monsters      int
	// Are these used?
	// cs16, cs17, cs18, cs19, cs20, cs21, cs22, cs23, cs24, cs25, cs26, cs27, cs28, cs29, cs30, cs31, cs32 int
}

var (
	cls = ClientStatic{}
	cl  = Client{}
)

//export CL_MaxEdicts
func CL_MaxEdicts() int {
	return cl.maxEdicts
}

//export CL_SetMaxEdicts
func CL_SetMaxEdicts(num int) {
	cl.maxEdicts = num
}

//export CL_Intermission
func CL_Intermission() C.int {
	return C.int(cl.intermission)
}

//export CL_SetIntermission
func CL_SetIntermission(i C.int) {
	cl.intermission = int(i)
}

//export CL_Stats
func CL_Stats(s C.int) C.int {
	return C.int(cl_stats(int(s)))
}

//export CL_SetStats
func CL_SetStats(s, v C.int) {
	cl_setStats(int(s), int(v))
}

//export CL_SoundPrecache
func CL_SoundPrecache(idx int) int {
	return cl.soundPrecache[idx-1]
}

//export CL_SoundPrecacheClear
func CL_SoundPrecacheClear() {
	cl.soundPrecache = cl.soundPrecache[:0]
}

//export CL_SoundPrecacheAdd
func CL_SoundPrecacheAdd(snd int) {
	cl.soundPrecache = append(cl.soundPrecache, snd)
}

func cl_stats(s int) int {
	switch s {
	case stat.Health:
		return cl.stats.health
	case stat.Frags:
		return cl.stats.frags
	case stat.Weapon:
		return cl.stats.weapon
	case stat.Ammo:
		return cl.stats.ammo
	case stat.Armor:
		return cl.stats.armor
	case stat.WeaponFrame:
		return cl.stats.weaponFrame
	case stat.Shells:
		return cl.stats.shells
	case stat.Nails:
		return cl.stats.nails
	case stat.Rockets:
		return cl.stats.rockets
	case stat.Cells:
		return cl.stats.cells
	case stat.ActiveWeapon:
		return cl.stats.activeWeapon
	case stat.TotalSecrets:
		return cl.stats.totalSecrets
	case stat.TotalMonsters:
		return cl.stats.totalMonsters
	case stat.Secrets:
		return cl.stats.secrets
	case stat.Monsters:
		return cl.stats.monsters
	default:
		log.Printf("Unknown cl stat %v", s)
		return 0
	}
}

func cl_setStats(s, v int) {
	switch s {
	case stat.Health:
		cl.stats.health = v
	case stat.Frags:
		cl.stats.frags = v
	case stat.Weapon:
		cl.stats.weapon = v
	case stat.Ammo:
		cl.stats.ammo = v
	case stat.Armor:
		cl.stats.armor = v
	case stat.WeaponFrame:
		cl.stats.weaponFrame = v
	case stat.Shells:
		cl.stats.shells = v
	case stat.Nails:
		cl.stats.nails = v
	case stat.Rockets:
		cl.stats.rockets = v
	case stat.Cells:
		cl.stats.cells = v
	case stat.ActiveWeapon:
		cl.stats.activeWeapon = v
	case stat.TotalSecrets:
		cl.stats.totalSecrets = v
	case stat.TotalMonsters:
		cl.stats.totalMonsters = v
	case stat.Secrets:
		cl.stats.secrets = v
	case stat.Monsters:
		cl.stats.monsters = v
	default:
		log.Printf("Unknown cl set stat %v", s)
	}
}

//export CL_Clear
func CL_Clear() {
	// cl: there is a memset 0 in CL_ClearState
	cl = Client{}
}

//export CL_SetViewentity
func CL_SetViewentity(v C.int) {
	cl.viewentity = int(v)
}

//export CL_Viewentity
func CL_Viewentity() C.int {
	return C.int(cl.viewentity)
}

//export CL_Protocol
func CL_Protocol() C.uint {
	return C.uint(cl.protocol)
}

//export CL_SetProtocol
func CL_SetProtocol(v C.uint) {
	cl.protocol = uint16(v)
}

//export CL_SetProtocolFlags
func CL_SetProtocolFlags(v C.uint) {
	cl.protocolFlags = uint16(v)
}

//export CL_ProtocolFlags
func CL_ProtocolFlags() C.uint {
	return C.uint(cl.protocolFlags)
}

//export CL_CmdForwardMove
func CL_CmdForwardMove() C.float {
	return C.float(cl.cmdForwardMove)
}

//export CL_Paused
func CL_Paused() C.int {
	return b2i(cl.paused)
}

//export CL_SetPaused
func CL_SetPaused(t C.int) {
	cl.paused = (t != 0)
}

//export CL_OnGround
func CL_OnGround() C.int {
	return b2i(cl.onGround)
}

//export CL_SetOnGround
func CL_SetOnGround(t C.int) {
	cl.onGround = (t != 0)
}

//export CL_Time
func CL_Time() C.double {
	return C.double(cl.time)
}

//export CL_SetTime
func CL_SetTime(t C.double) {
	cl.time = float64(t)
}

//export CL_MTime
func CL_MTime() C.double {
	return C.double(cl.messageTime)
}

//export CL_SetMTime
func CL_SetMTime(t C.double) {
	cl.messageTime = float64(t)
}

//export CL_MTimeOld
func CL_MTimeOld() C.double {
	return C.double(cl.messageTimeOld)
}

//export CL_SetMTimeOld
func CL_SetMTimeOld(t C.double) {
	cl.messageTimeOld = float64(t)
}

//export CL_OldTime
func CL_OldTime() C.double {
	return C.double(cl.oldTime)
}

//export CL_SetOldTime
func CL_SetOldTime(t C.double) {
	cl.oldTime = float64(t)
}

//export CL_LastReceivedMessage
func CL_LastReceivedMessage() C.double {
	return C.double(cl.lastReceivedMessageTime)
}

//export CL_SetLastReceivedMessage
func CL_SetLastReceivedMessage(t C.double) {
	cl.lastReceivedMessageTime = float64(t)
}

//export CL_UpdateCompletedTime
func CL_UpdateCompletedTime() {
	cl.intermissionTime = int(cl.time)
}

//export CL_CompletedTime
func CL_CompletedTime() C.int {
	return C.int(cl.intermissionTime)
}

//export CLS_GetForceTrack
func CLS_GetForceTrack() C.int {
	return C.int(cls.forceTrack)
}

//export CLS_SetForceTrack
func CLS_SetForceTrack(t C.int) {
	cls.forceTrack = int(t)
}

//export CLS_GetTimeDemoStartFrame
func CLS_GetTimeDemoStartFrame() C.int {
	return C.int(cls.timeDemoStartFrame)
}

//export CLS_SetTimeDemoStartFrame
func CLS_SetTimeDemoStartFrame(f C.int) {
	cls.timeDemoStartFrame = int(f)
}

//export CLS_GetTimeDemoStartTime
func CLS_GetTimeDemoStartTime() C.float {
	return C.float(cls.timeDemoStartTime)
}

//export CLS_SetTimeDemoStartTime
func CLS_SetTimeDemoStartTime(f C.float) {
	cls.timeDemoStartTime = float32(f)
}

//export CLS_GetTimeDemoLastFrame
func CLS_GetTimeDemoLastFrame() C.int {
	return C.int(cls.timeDemoLastFrame)
}

//export CLS_SetTimeDemoLastFrame
func CLS_SetTimeDemoLastFrame(f C.int) {
	cls.timeDemoLastFrame = int(f)
}

//export CLS_GetState
func CLS_GetState() C.int {
	return C.int(cls.state)
}

//export CLS_SetState
func CLS_SetState(s C.int) {
	cls.state = int(s)
}

//export CLS_IsDemoCycleStopped
func CLS_IsDemoCycleStopped() C.int {
	return b2i(clsIsDemoCycleStopped())
}

func clsIsDemoCycleStopped() bool {
	return cls.demoNum == -1
}

//export CLS_StopDemoCycle
func CLS_StopDemoCycle() {
	cls.demoNum = -1
}

//export CLS_NextDemoInCycle
func CLS_NextDemoInCycle() {
	cls.demoNum++
}

//export CLS_StartDemoCycle
func CLS_StartDemoCycle() {
	cls.demoNum = 0
}

//export CLS_GetDemoNum
func CLS_GetDemoNum() C.int {
	return C.int(cls.demoNum)
}

//export CLS_SetDemoNum
func CLS_SetDemoNum(num C.int) {
	cls.demoNum = int(num)
}

//export CLS_IsDemoRecording
func CLS_IsDemoRecording() C.int {
	return b2i(cls.demoRecording)
}

//export CLS_SetDemoRecording
func CLS_SetDemoRecording(state C.int) {
	cls.demoRecording = (state != 0)
}

//export CLS_IsDemoPlayback
func CLS_IsDemoPlayback() C.int {
	return b2i(cls.demoPlayback)
}

//export CLS_SetDemoPlayback
func CLS_SetDemoPlayback(state C.int) {
	cls.demoPlayback = (state != 0)
}

//export CLS_IsDemoPaused
func CLS_IsDemoPaused() C.int {
	return b2i(cls.demoPaused)
}

//export CLS_SetDemoPaused
func CLS_SetDemoPaused(state C.int) {
	cls.demoPaused = (state != 0)
}

//export CLS_IsTimeDemo
func CLS_IsTimeDemo() C.int {
	return b2i(cls.timeDemo)
}

//export CLS_SetTimeDemo
func CLS_SetTimeDemo(state C.int) {
	cls.timeDemo = (state != 0)
}

//export CLS_GetSignon
func CLS_GetSignon() C.int {
	return C.int(cls.signon)
}

//export CLS_SetSignon
func CLS_SetSignon(s C.int) {
	cls.signon = int(s)
}

//export CLSMessageWriteByte
func CLSMessageWriteByte(c C.int) {
	CLSMessageWriteByte2(byte(c))
}

func CLSMessageWriteByte2(c byte) {
	cls.outMessage.WriteByte(c)
}

//export CLSMessageWriteString
func CLSMessageWriteString(data *C.char) {
	s := C.GoString(data)
	CLSMessageWriteString2(s)
}

func CLSMessageWriteString2(s string) {
	cls.outMessage.WriteString(s)
	cls.outMessage.WriteByte(0)
}

//export CLSMessagePrint
func CLSMessagePrint(data *C.char) {
	s := C.GoString(data)
	CLSMessagePrint2(s)
}

func CLSMessagePrint2(s string) {
	// the original would override a trailing 0
	// changed to not write a trailing 0 so it must be explicitly added by
	// WriteByte if needed
	cls.outMessage.WriteString(s)
}

//export CLSMessageClear
func CLSMessageClear() {
	cls.outMessage.Reset()
}

//export CLSHasMessage
func CLSHasMessage() C.int {
	return b2i(cls.outMessage.Len() > 0)
}

func executeOnServer(args []cmd.QArg, _ int) {
	if cls.state != ca_connected {
		conlog.Printf("Can't \"cmd\", not connected\n")
		return
	}
	if cls.demoPlayback {
		return
	}
	if len(args) > 0 {
		cls.outMessage.WriteByte(clc.StringCmd)
		cls.outMessage.WriteString(cmd.Full())
		cls.outMessage.WriteByte(0)
	}
}

func forwardToServer(c string, args []cmd.QArg) {
	if cls.state != ca_connected {
		conlog.Printf("Can't \"%s\", not connected\n", c)
		return
	}
	if cls.demoPlayback {
		return
	}
	cls.outMessage.WriteByte(clc.StringCmd)
	cls.outMessage.WriteString(c)
	if len(args) > 0 {
		cls.outMessage.WriteString(" ")
		cls.outMessage.WriteString(cmd.Full())
	} else {
		cls.outMessage.WriteString("\n")
	}
	cls.outMessage.WriteByte(0)
}

func init() {
	cmd.AddCommand("cmd", executeOnServer)
}

//export CL_GetMessage
func CL_GetMessage() C.int {
	return C.int(getMessage())
}

func getMessage() int {
	// for cl_main: return -1 on error, return 0 for message end, everything else is continue
	// for cl_parse: return 0 for end message, 2 && ReadByte == Nop continue, everything else is Host_Error
	if cls.demoPlayback {
		return getDemoMessage()
	}

	r := 0
	for {
		cls.msgBadRead = false

		m, err := cls.connection.GetMessage()
		if err != nil {
			return -1
		}
		if m == nil || m.Len() == 0 {
			return 0
		}
		cls.inMessage = m
		b, err := cls.inMessage.ReadByte()
		if err != nil {
			return -1
		}
		r = int(b)

		// discard nop keepalive message
		if cls.inMessage.Len() == 1 {
			m, err := cls.inMessage.ReadByte()
			if err != nil {
				// Should never happen as we already know there is exactly one byte
				log.Fatalf("Error in GetMessage: %v", err)
			}
			if m == svc.Nop {
				// Con_Printf("<-- server to client keepalive\n")
			} else {
				// The original was doing a BeginReading which was setting the read cursor to
				// the begining so this was not needed. As the BeginReading was removed step
				// back.
				cls.inMessage.UnreadByte()
				break
			}
		} else {
			break
		}
	}

	if cls.demoRecording {
		writeDemoMessage()
	}

	if cls.signon < 2 {
		// record messages before full connection, so that a
		// demo record can happen after connection is done
		cacheStartConnectionForDemo()
	}

	return r
}

func getDemoMessage() int {
	//TODO
	// CL_GetDemoMessage
	return 0
}
func writeDemoMessage() {
	//TODO
	//CL_WriteDemoMessage
}
func cacheStartConnectionForDemo() {
	// TODO
	//  memcpy(demo_head[CLS_GetSignon()], net_message.daTa, SB_GetCurSize(&net_message));
	//  demo_head_size[CLS_GetSigon()] = SB_GetCurSize(&net_message);
	//
}

//export CL_MSG_BadRead
func CL_MSG_BadRead() C.int {
	// poor mans error handling :(
	return b2i(cls.msgBadRead)
}

//export CL_MSG_ReadChar
func CL_MSG_ReadChar() C.int {
	i, err := cls.inMessage.ReadInt8()
	if err != nil {
		cls.msgBadRead = true
		return -1
	}
	return C.int(i)
}

//export CL_MSG_ReadByte
func CL_MSG_ReadByte() C.int {
	i, err := cls.inMessage.ReadByte()
	if err != nil {
		cls.msgBadRead = true
		return -1
	}
	return C.int(i)
}

//export CL_MSG_ReadShort
func CL_MSG_ReadShort() C.int {
	i, err := cls.inMessage.ReadInt16()
	if err != nil {
		cls.msgBadRead = true
		return -1
	}
	return C.int(i)
}

//export CL_MSG_ReadLong
func CL_MSG_ReadLong() C.int {
	i, err := cls.inMessage.ReadInt32()
	if err != nil {
		cls.msgBadRead = true
		return -1
	}
	return C.int(i)
}

//export CL_MSG_ReadFloat
func CL_MSG_ReadFloat() C.float {
	f, err := cls.inMessage.ReadFloat32()
	if err != nil {
		cls.msgBadRead = true
		return -1
	}
	return C.float(f)
}

//export CL_MSG_ReadCoord16
func CL_MSG_ReadCoord16() C.float {
	f, err := cls.inMessage.ReadCoord16()
	if err != nil {
		cls.msgBadRead = true
		return -1
	}
	return C.float(f)
}

//export CL_MSG_ReadCoord24
func CL_MSG_ReadCoord24() C.float {
	f, err := cls.inMessage.ReadCoord24()
	if err != nil {
		cls.msgBadRead = true
		return -1
	}
	return C.float(f)
}

//export CL_MSG_ReadCoord32f
func CL_MSG_ReadCoord32f() C.float {
	f, err := cls.inMessage.ReadCoord32f()
	if err != nil {
		cls.msgBadRead = true
		return -1
	}
	return C.float(f)
}

//export CL_MSG_ReadCoord
func CL_MSG_ReadCoord() C.float {
	f, err := cls.inMessage.ReadCoord(cl.protocolFlags)
	if err != nil {
		cls.msgBadRead = true
		return -1
	}
	return C.float(f)
}

//export CL_MSG_ReadAngle
func CL_MSG_ReadAngle(flags C.uint) C.float {
	f, err := cls.inMessage.ReadAngle(uint32(flags))
	if err != nil {
		cls.msgBadRead = true
		return -1
	}
	return C.float(f)
}

//export CL_MSG_ReadAngle16
func CL_MSG_ReadAngle16(flags C.uint) C.float {
	f, err := cls.inMessage.ReadAngle16(uint32(flags))
	if err != nil {
		cls.msgBadRead = true
		return -1
	}
	return C.float(f)
}

//export CL_MSG_Replace
func CL_MSG_Replace(data unsafe.Pointer, size C.size_t) {
	m := C.GoBytes(data, C.int(size))
	cls.inMessage = net.NewQReader(m)
}

//export CLSMessageSend
func CLSMessageSend() C.int {
	b := cls.outMessage.Bytes()
	i := cls.connection.SendMessage(b)
	return C.int(i)
}

//export CLSMessageSendUnreliable
func CLSMessageSendUnreliable() C.int {
	b := cls.outMessage.Bytes()
	i := cls.connection.SendUnreliableMessage(b)
	return C.int(i)
}

//export CLS_NET_Close
func CLS_NET_Close() {
	cls.connection.Close()
}

//export CLS_NET_CanSendMessage
func CLS_NET_CanSendMessage() C.int {
	return b2i(cls.connection.CanSendMessage())
}

//Sends a disconnect message to the server
//This is also called on Host_Error, so it shouldn't cause any errors
//export CL_Disconnect
func CL_Disconnect() {
	cls.Disconnect()
}

func (c *ClientStatic) Disconnect() {
	if keyDestination == keys.Message {
		// don't get stuck in chat mode
		keyEndChat()
	}

	// stop sounds (especially looping!)
	snd.StopAll()

	// if running a local server, shut it down
	if c.demoPlayback {
		C.CL_StopPlayback()
	} else if c.state == ca_connected {
		if c.demoRecording {
			C.CL_Stop_f()
		}

		conlog.DPrintf("Sending clc_disconnect\n")
		c.outMessage.Reset()
		c.outMessage.WriteByte(clc.Disconnect)
		b := c.outMessage.Bytes()
		c.connection.SendUnreliableMessage(b)
		c.outMessage.Reset()
		cls.connection.Close()

		c.state = ca_disconnected
		if sv.active {
			hostShutdownServer(false)
		}
	}

	c.demoPlayback = false
	c.timeDemo = false
	c.demoPaused = false
	c.signon = 0
	cl.intermission = 0
}

//export CL_Disconnect_f
func CL_Disconnect_f() {
	clientDisconnect()
}

func clientDisconnect() {
	cls.Disconnect()
	if sv.active {
		hostShutdownServer(false)
	}
}

// This command causes the client to wait for the signon messages again.
// This is sent just before a server changes levels
func clientReconnect() {
	if cls.demoPlayback {
		return
	}
	SCR_BeginLoadingPlaque()
	// need new connection messages
	cls.signon = 0
}

//export Host_Reconnect_f
func Host_Reconnect_f() {
	clientReconnect()
}

//export CL_EstablishConnection
func CL_EstablishConnection(host *C.char) {
	clEstablishConnection(C.GoString(host))
}

// Host should be either "local" or a net address to be passed on
func clEstablishConnection(host string) {
	if cls.state == ca_dedicated {
		return
	}

	if cls.demoPlayback {
		return
	}

	cls.Disconnect()

	c, err := net.Connect(host)
	if err != nil {
		// TODO: this is bad, looks like orig just quits this call without returning
		// and waits for the next sdl input.
		cls.connection = nil
		HostError("CLS_Connect: connect failed\n")
	}
	cls.connection = c
	conlog.DPrintf("CL_EstablishConnection: connected to %s\n", host)

	// not in the demo loop now
	cls.demoNum = -1
	cls.state = ca_connected
	// need all the signon messages before playing
	cls.signon = 0
	cls.outMessage.WriteByte(clc.Nop)
}

// An svc_signonnum has been received, perform a client side setup
//export CL_SignonReply
func CL_SignonReply() {
	conlog.DPrintf("CL_SignonReply: %d\n", cls.signon)

	switch cls.signon {
	case 1:
		cls.outMessage.WriteByte(clc.StringCmd)
		cls.outMessage.WriteString("prespawn")
		cls.outMessage.WriteByte(0)

	case 2:
		cls.outMessage.WriteByte(clc.StringCmd)
		cls.outMessage.WriteString(fmt.Sprintf("name \"%s\"", cvars.ClientName.String()))
		cls.outMessage.WriteByte(0)

		color := int(cvars.ClientColor.Value())
		cls.outMessage.WriteByte(clc.StringCmd)
		cls.outMessage.WriteString(fmt.Sprintf("color %d %d\n", color>>4, color&15))
		cls.outMessage.WriteByte(0)

		cls.outMessage.WriteByte(clc.StringCmd)
		cls.outMessage.WriteString(fmt.Sprintf("spawn %s", cls.spawnParms))
		cls.outMessage.WriteByte(0)

	case 3:
		cls.outMessage.WriteByte(clc.StringCmd)
		cls.outMessage.WriteString("begin")
		cls.outMessage.WriteByte(0)

	case 4:
		SCR_EndLoadingPlaque() // allow normal screen updates
	}
}

func clientStartDemos(args []cmd.QArg, player int) {
	if cls.state == ca_dedicated {
		return
	}

	cls.demos = cls.demos[:0]
	for _, a := range args {
		cls.demos = append(cls.demos, a.String())
	}
	conlog.Printf("%d demo(s) in loop\n", len(cls.demos))

	if !sv.active && cls.demoNum != -1 && !cls.demoPlayback {
		cls.demoNum = 0
		if !cmdl.Fitz() { // QuakeSpasm customization:
			// go straight to menu, no CL_NextDemo
			cls.demoNum = -1
			cbuf.InsertText("menu_main\n")
			return
		}
		CL_NextDemo()
	} else {
		cls.demoNum = -1
	}
}

// Called to play the next demo in the demo loop
//export CL_NextDemo
func CL_NextDemo() {
	if cls.demoNum == -1 {
		// don't play demos
		return
	}

	if len(cls.demos) == 0 {
		conlog.Printf("No demos listed with startdemos\n")
		cls.demoNum = -1
		cls.Disconnect()
		return
	}

	// TODO(therjak): Can this be integrated into CLS_NextDemoInCycle?
	if cls.demoNum == len(cls.demos) {
		cls.demoNum = 0
	}

	SCR_BeginLoadingPlaque()

	cbuf.InsertText(fmt.Sprintf("playdemo %s\n", cls.demos[cls.demoNum]))
	cls.demoNum++
}

func CL_SendCmd() {
	if cls.state != ca_connected {
		return
	}

	if cls.signon == numSignonMessagesBeforeConn {
		CL_AdjustAngles()
		HandleMove()
	}

	if cls.demoPlayback {
		cls.outMessage.Reset()
		return
	}

	// send the reliable message
	if cls.outMessage.Len() == 0 {
		return // no message at all
	}

	if !cls.connection.CanSendMessage() {
		conlog.DPrintf("CL_SendCmd: can't send\n")
		return
	}

	b := cls.outMessage.Bytes()
	i := cls.connection.SendMessage(b)
	if i == -1 {
		HostError("CL_SendCmd: lost server connection")
	}
	cls.outMessage.Reset()
}

//export CL_ParseStartSoundPacket
func CL_ParseStartSoundPacket() {
	err := parseStartSoundPacket(cls.inMessage)
	if err != nil {
		HostError("%v", err)
	}
}

func parseStartSoundPacket(msg *net.QReader) error {
	const (
		maxSounds = 2048
	)

	fieldMask, err := msg.ReadByte()
	if err != nil {
		return fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
	}
	volume := byte(255)
	attenuation := float32(1.0)

	if fieldMask&svc.SoundVolume != 0 {
		volume, err = msg.ReadByte()
		if err != nil {
			return fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
	}

	if fieldMask&svc.SoundAttenuation != 0 {
		a, err := msg.ReadByte()
		if err != nil {
			return fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		attenuation = float32(a) / 64.0
	}

	ent := uint16(0)
	channel := byte(0)
	if fieldMask&svc.SoundLargeEntity != 0 {
		e, err := msg.ReadInt16()
		if err != nil {
			return fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		c, err := msg.ReadByte()
		if err != nil {
			return fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		ent = uint16(e)
		channel = c
	} else {
		s, err := msg.ReadInt16()
		if err != nil {
			return fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		ent = uint16(s >> 3)
		channel = byte(s & 7)
	}

	soundNum := uint16(0)
	if fieldMask&svc.SoundLargeSound != 0 {
		n, err := msg.ReadInt16()
		if err != nil {
			return fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		soundNum = uint16(n - 1)
	} else {
		n, err := msg.ReadByte()
		if err != nil {
			return fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		soundNum = uint16(n - 1)
	}
	if soundNum > maxSounds {
		return fmt.Errorf("CL_ParseStartSoundPacket: %d > MAX_SOUNDS", soundNum)
	}
	if int(ent) > cl.maxEdicts {
		return fmt.Errorf("CL_ParseStartSoundPacket: ent = %d", ent)
	}
	var origin vec.Vec3
	for i := 0; i < 3; i++ {
		f, err := msg.ReadCoord(cl.protocolFlags)
		if err != nil {
			return fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		origin[i] = f
	}

	snd.Start(int(ent), int(channel), cl.soundPrecache[soundNum], origin, float32(volume)/255, attenuation, !loopingSound)
	return nil
}

var (
	clientKeepAliveTime time.Time
)

// When the client is taking a long time to load stuff, send keepalive messages
// so the server doesn't disconnect.
//export CL_KeepaliveMessage
func CL_KeepaliveMessage() {
	//float time;
	//static float lastmsg;
	//int ret;

	if sv.active {
		// no need if server is local
		return
	}
	if cls.demoPlayback {
		return
	}

	// read messages from server, should just be nops
	msgBackup := cls.inMessage

Outer:
	for {
		switch ret := getMessage(); ret {
		default:
			HostError("CL_KeepaliveMessage: CL_GetMessage failed")
		case 0:
			break Outer
		case 1:
			HostError("CL_KeepaliveMessage: received a message")
		case 2:
			i, err := cls.inMessage.ReadByte()
			if err != nil || i != svc.Nop {
				HostError("CL_KeepaliveMessage: datagram wasn't a nop")
			}
		}
	}

	cls.inMessage = msgBackup

	// check time
	curTime := time.Now()
	if curTime.Sub(clientKeepAliveTime) < time.Second*5 {
		return
	}
	if !cls.connection.CanSendMessage() {
		return
	}
	clientKeepAliveTime = curTime

	// write out a nop
	conlog.Printf("--> client to server keepalive\n")

	cls.outMessage.WriteByte(clc.Nop)
	b := cls.outMessage.Bytes()
	cls.connection.SendMessage(b)
	cls.outMessage.Reset()
}

var (
	clSounds map[sfx]int
)

//export CL_Sound
func CL_Sound(s sfx, origin *C.float) {
	S_StartSound(-1, 0, C.int(clSounds[s]), origin, 1, 1)
}

//export CL_InitTEnts
func CL_InitTEnts() {
	clSounds = make(map[sfx]int)
	clSounds[WizHit] = snd.PrecacheSound("wizard/hit.wav")
	clSounds[KnightHit] = snd.PrecacheSound("hknight/hit.wav")
	clSounds[Tink1] = snd.PrecacheSound("weapons/tink1.wav")
	clSounds[Ric1] = snd.PrecacheSound("weapons/ric1.wav")
	clSounds[Ric2] = snd.PrecacheSound("weapons/ric2.wav")
	clSounds[Ric3] = snd.PrecacheSound("weapons/ric3.wav")
	clSounds[RExp3] = snd.PrecacheSound("weapons/r_exp3.wav")
}
