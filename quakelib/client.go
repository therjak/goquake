package quakelib

//void V_StopPitchDrift(void);
//void CL_Disconnect(void);
import "C"

import (
	"bytes"
	"log"
	"quake/cmd"
	"quake/conlog"
	"quake/cvars"
	"quake/input"
	"quake/math"
	"quake/net"
	clc "quake/protocol/client"
	svc "quake/protocol/server"
	"quake/stat"
	"unsafe"
)

func init() {
	cmd.AddCommand("disconnect", clDisconnect)
}

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
	inMessageBackup    *net.QReader
	outMessage         bytes.Buffer
	forceTrack         int // -1 to use normal cd track
	timeDemoLastFrame  int
	timeDemoStartFrame int
	timeDemoStartTime  float32
	/*
		spawnParms string
		demos []string
		demoFile 'filehandle'
	*/
	msgBadRead bool

	// net.PacketCon
}

type Client struct {
	pitch          float32 // 0
	yaw            float32 // 1
	roll           float32 // 2
	movemessages   int
	cmdForwardMove float32
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
	// stats
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
	stats ClientStats
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
	// Are theres used?
	// cs16, cs17, cs18, cs19, cs20, cs21, cs22, cs23, cs24, cs25, cs26, cs27, cs28, cs29, cs30, cs31, cs32 int
}

// cl: there is a memset 0 in CL_ClearState

var (
	cls = ClientStatic{}
	cl  = Client{}
)

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

func cl_stats(s int) int {
	switch s {
	case stat.HEALTH:
		return cl.stats.health
	case stat.FRAGS:
		return cl.stats.frags
	case stat.WEAPON:
		return cl.stats.weapon
	case stat.AMMO:
		return cl.stats.ammo
	case stat.ARMOR:
		return cl.stats.armor
	case stat.WEAPONFRAME:
		return cl.stats.weaponFrame
	case stat.SHELLS:
		return cl.stats.shells
	case stat.NAILS:
		return cl.stats.nails
	case stat.ROCKETS:
		return cl.stats.rockets
	case stat.CELLS:
		return cl.stats.cells
	case stat.ACTIVEWEAPON:
		return cl.stats.activeWeapon
	case stat.TOTALSECRETS:
		return cl.stats.totalSecrets
	case stat.TOTALMONSTERS:
		return cl.stats.totalMonsters
	case stat.SECRETS:
		return cl.stats.secrets
	case stat.MONSTERS:
		return cl.stats.monsters
	default:
		log.Printf("Unknown cl stat %v", s)
		return 0
	}
}

func cl_setStats(s, v int) {
	switch s {
	case stat.HEALTH:
		cl.stats.health = v
	case stat.FRAGS:
		cl.stats.frags = v
	case stat.WEAPON:
		cl.stats.weapon = v
	case stat.AMMO:
		cl.stats.ammo = v
	case stat.ARMOR:
		cl.stats.armor = v
	case stat.WEAPONFRAME:
		cl.stats.weaponFrame = v
	case stat.SHELLS:
		cl.stats.shells = v
	case stat.NAILS:
		cl.stats.nails = v
	case stat.ROCKETS:
		cl.stats.rockets = v
	case stat.CELLS:
		cl.stats.cells = v
	case stat.ACTIVEWEAPON:
		cl.stats.activeWeapon = v
	case stat.TOTALSECRETS:
		cl.stats.totalSecrets = v
	case stat.TOTALMONSTERS:
		cl.stats.totalMonsters = v
	case stat.SECRETS:
		cl.stats.secrets = v
	case stat.MONSTERS:
		cl.stats.monsters = v
	default:
		log.Printf("Unknown cl set stat %v", s)
	}
}

//export CL_Clear
func CL_Clear() {
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

func executeOnServer(args []cmd.QArg) {
	if cls.state != ca_connected {
		conlog.Printf("Can't \"cmd\", not connected\n")
		return
	}
	if cls.demoPlayback {
		return
	}
	if len(args) > 0 {
		cls.outMessage.WriteByte(clc.StringCmd)
		cls.outMessage.WriteString(cmd.CmdArgs())
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
		cls.outMessage.WriteString(cmd.CmdArgs())
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

//export CL_MSG_Backup
func CL_MSG_Backup() {
	cls.inMessageBackup = cls.inMessage
}

//export CL_MSG_Restore
func CL_MSG_Restore() {
	cls.inMessage = cls.inMessageBackup
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

//export CLS_Connect
func CLS_Connect(host *C.char) C.int {
	c, err := net.Connect(C.GoString(host))
	if err != nil {
		cls.connection = nil
		return 0
	} else {
		cls.connection = c
		return 1
	}
}

/*
//Sends a disconnect message to the server
//This is also called on Host_Error, so it shouldn't cause any errors
// export CL_Disconnect
void CL_Disconnect(void) {
  cls.Disconnect()
}
*/

func (c *ClientStatic) Disconnect() {
	C.CL_Disconnect()
}

/*
func (c *ClientStatic) Disconnect() {
  if (GetKeyDest() == key_message)
    Key_EndChat();  // don't get stuck in chat mode

  // stop sounds (especially looping!)
  S_StopAllSounds(true);

  // if running a local server, shut it down
  if (CLS_IsDemoPlayback())
    CL_StopPlayback();
  else if (CLS_GetState() == ca_connected) {
    if (CLS_IsDemoRecording()) CL_Stop_f();

    Con_DPrintf("Sending clc_disconnect\n");
    CLSMessageClear();
    CLSMessageWriteByte(clc_disconnect);
    CLSMessageSendUnreliable();
    CLSMessageClear();
    CLS_NET_Close();

    CLS_SetState(ca_disconnected);
    if (SV_Active()) Host_ShutdownServer(false);
  }

  CLS_SetDemoPlayback(false);
  CLS_SetTimeDemo(false);
  CLS_SetDemoPaused(false);
  CLS_SetSignon(0);
  CL_SetIntermission(0);
}
*/

//export CL_Disconnect_f
func CL_Disconnect_f() {
	clDisconnect([]cmd.QArg{})
}

func clDisconnect(args []cmd.QArg) {
	cls.Disconnect()
	if sv.active {
		hostShutdownServer(false)
	}
}
