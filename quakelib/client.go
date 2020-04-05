package quakelib

//#define SFX_WIZHIT  0
//#define SFX_KNIGHTHIT  1
//#define SFX_TINK1  2
//#define SFX_RIC1  3
//#define SFX_RIC2  4
//#define SFX_RIC3  5
//#define SFX_R_EXP3  6
//#include "cgo_help.h"
//void SetCLWeaponModel(int v);
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/chewxy/math32"
	"log"
	"quake/cbuf"
	"quake/cmd"
	cmdl "quake/commandline"
	"quake/conlog"
	"quake/cvars"
	"quake/execute"
	"quake/filesystem"
	"quake/input"
	"quake/keys"
	"quake/math"
	"quake/math/vec"
	"quake/model"
	"quake/net"
	"quake/progs"
	clc "quake/protocol/client"
	svc "quake/protocol/server"
	"quake/snd"
	"quake/stat"
	"strings"
	"time"
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

const (
	ColorShiftContents = iota
	ColorShiftDamage
	ColorShiftBonus
	ColorShiftPowerup
)

func init() {
	cmd.AddCommand("disconnect", func(args []cmd.QArg, _ int) { clientDisconnect() })
	cmd.AddCommand("reconnect", func(args []cmd.QArg, _ int) { clientReconnect() })

	cmd.AddCommand("startdemos", clientStartDemos)
	cmd.AddCommand("record", clientRecordDemo)
	cmd.AddCommand("stop", clientStopDemoRecording)
	cmd.AddCommand("playdemo", clientPlayDemo)
	cmd.AddCommand("timedemo", clientTimeDemo)

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

func (c *Client) adjustAngles() {
	speed := func() float32 {
		if (cvars.ClientForwardSpeed.Value() > 200) != input.Speed.Down() {
			return float32(FrameTime()) * cvars.ClientAngleSpeedKey.Value()
		}
		return float32(FrameTime())
	}()
	if !input.Strafe.Down() {
		c.yaw -= speed * cvars.ClientYawSpeed.Value() * input.Right.ConsumeImpulse()
		c.yaw += speed * cvars.ClientYawSpeed.Value() * input.Left.ConsumeImpulse()
		c.yaw = math.AngleMod32(cl.yaw)
	}
	if input.KLook.Down() {
		c.stopPitchDrift()
		c.pitch -= speed * cvars.ClientPitchSpeed.Value() * input.Forward.ConsumeImpulse()
		c.pitch += speed * cvars.ClientPitchSpeed.Value() * input.Back.ConsumeImpulse()
	}

	up := input.LookUp.ConsumeImpulse()
	down := input.LookDown.ConsumeImpulse()

	c.pitch -= speed * cvars.ClientPitchSpeed.Value() * up
	c.pitch += speed * cvars.ClientPitchSpeed.Value() * down

	if up != 0 || down != 0 {
		c.stopPitchDrift()
	}

	c.pitch = math.Clamp32(cvars.ClientMinPitch.Value(), c.pitch, cvars.ClientMaxPitch.Value())
	c.roll = math.Clamp32(-50, c.roll, 50)
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
	timeDemoLastFrame  int
	timeDemoStartFrame int
	timeDemoStartTime  float64
	// personalization data sent to server
	// to restart a level
	spawnParms string
	demos      []string
	/*
		demoFile 'filehandle'
	*/
	demoData   []byte
	msgBadRead bool

	// net.PacketCon
}

type score struct {
	name        string // (len == 0) => do not draw
	frags       int
	topColor    int
	bottomColor int
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
	// model_precache
	// viewent.model

	//

	mViewAngles     [2]vec.Vec3
	mVelocity       [2]vec.Vec3 // update by server
	velocity        vec.Vec3    // lerped from mvelocity
	punchAngle      [2]vec.Vec3 // v_punchangle
	idealPitch      float32
	pitchVel        float32
	drift           bool
	driftMove       float32
	lastStop        float64
	viewHeight      float32
	colorShifts     [4]Color
	colorShiftsPrev [4]Color
	dmgTime         float32
	dmgRoll         float32
	dmgPitch        float32

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

	stats        ClientStats
	items        uint32      // 32bit bit field
	itemGetTime  [32]float64 // for blinking
	faceAnimTime float64
	maxEdicts    int

	scores []score // len() == maxClients

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

//export CL_MViewAngles
func CL_MViewAngles(i, j int) float32 {
	return cl.mViewAngles[i][j]
}

//export SetCL_MViewAngles
func SetCL_MViewAngles(i, j int, v float32) {
	cl.mViewAngles[i][j] = v
}

//export CL_SetMVelocity
func CL_SetMVelocity(i, j int, v float32) {
	cl.mVelocity[i][j] = v
}

//export CL_MVelocity
func CL_MVelocity(i, j int) float32 {
	return cl.mVelocity[i][j]
}

//export CL_SetVelocity
func CL_SetVelocity(i int, v float32) {
	cl.velocity[i] = v
}

//export CL_SetLastStop
func CL_SetLastStop(s float64) {
	cl.lastStop = s
}

//export CL_MaxEdicts
func CL_MaxEdicts() int {
	return cl.maxEdicts
}

//export CL_SetMaxEdicts
func CL_SetMaxEdicts(num int) {
	cl.maxEdicts = num
}

//export CL_SetIntermission
func CL_SetIntermission(i C.int) {
	cl.intermission = int(i)
}

func (c *Client) UpdateFaceAnimTime() {
	c.faceAnimTime = c.time + 0.2
}

func (c *Client) CheckFaceAnimTime() bool {
	return cl.time <= cl.faceAnimTime
}

//export CL_MaxClients
func CL_MaxClients() int {
	return cl.maxClients
}

//export CL_SetMaxClients
func CL_SetMaxClients(m int) {
	cl.maxClients = m
	cl.scores = make([]score, m)
}

//export CL_ScoresSetName
func CL_ScoresSetName(i int, c *C.char) {
	cl.scores[i].name = C.GoString(c)
}

//export CL_SetLevelName
func CL_SetLevelName(c *C.char) {
	cl.levelName = C.GoString(c)
}

//export CL_SetMapName
func CL_SetMapName(c *C.char) {
	cl.mapName = C.GoString(c)
}

//export CL_ScoresSetFrags
func CL_ScoresSetFrags(i int, f int) {
	cl.scores[i].frags = f
}

//export CL_ScoresFrags
func CL_ScoresFrags(i int) int {
	return cl.scores[i].frags
}

//export CL_ScoresSetColors
func CL_ScoresSetColors(i int, c int) {
	cl.scores[i].topColor = (c & 0xf0) >> 4
	cl.scores[i].bottomColor = c & 0x0f
}

//export CL_ScoresColors
func CL_ScoresColors(i int) int {
	return (cl.scores[i].topColor << 4) + cl.scores[i].bottomColor
}

//export CL_Stats
func CL_Stats(s C.int) C.int {
	return C.int(cl_stats(int(s)))
}

//export CL_SetStats
func CL_SetStats(s, v C.int) {
	cl_setStats(int(s), int(v))
}

//export CL_SetGameType
func CL_SetGameType(t int) {
	cl.gameType = t
}

func (c *Client) DeathMatch() bool {
	return cl.gameType == svc.GameDeathmatch
}

//export CL_GameTypeDeathMatch
func CL_GameTypeDeathMatch() bool {
	return cl.DeathMatch()
}

//export CL_HasItem
func CL_HasItem(item uint32) bool {
	return cl.items&item != 0
}

//export CL_ItemGetTime
func CL_ItemGetTime(item int) float64 {
	return cl.itemGetTime[item]
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
	clearLightStyles()
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

//export CL_Paused
func CL_Paused() C.int {
	return b2i(cl.paused)
}

//export CL_SetPaused
func CL_SetPaused(t C.int) {
	cl.paused = (t != 0)
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

//export CL_SetOldTime
func CL_SetOldTime(t C.double) {
	cl.oldTime = float64(t)
}

//export CL_SetLastReceivedMessage
func CL_SetLastReceivedMessage(t C.double) {
	cl.lastReceivedMessageTime = float64(t)
}

//export CL_UpdateCompletedTime
func CL_UpdateCompletedTime() {
	cl.intermissionTime = int(cl.time)
}

//export CLS_GetState
func CLS_GetState() C.int {
	return C.int(cls.state)
}

//export CLS_NextDemoInCycle
func CLS_NextDemoInCycle() {
	cls.demoNum++
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

//export CLS_GetSignon
func CLS_GetSignon() C.int {
	return C.int(cls.signon)
}

//export CLS_SetSignon
func CLS_SetSignon(s C.int) {
	cls.signon = int(s)
}

//export CLSMessageClear
func CLSMessageClear() {
	cls.outMessage.Reset()
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
	return C.int(cls.getMessage())
}

func (c *ClientStatic) getMessage() int {
	// for cl_main: return -1 on error, return 0 for message end, everything else is continue
	// for cl_parse: return 0 for end message, 2 && ReadByte == Nop continue, everything else is Host_Error
	if c.demoPlayback {
		c.msgBadRead = false
		return cls.getDemoMessage()
	}

	r := 0
	for {
		c.msgBadRead = false

		m, err := c.connection.GetMessage()
		if err != nil {
			return -1
		}
		if m == nil || m.Len() == 0 {
			return 0
		}
		c.inMessage = m
		b, err := c.inMessage.ReadByte()
		if err != nil {
			return -1
		}
		r = int(b)

		// discard nop keepalive message
		if c.inMessage.Len() == 1 {
			m, err := c.inMessage.ReadByte()
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
				c.inMessage.UnreadByte()
				break
			}
		} else {
			break
		}
	}

	if c.demoRecording {
		cl.writeDemoMessage()
	}

	if c.signon < 2 {
		// record messages before full connection, so that a
		// demo record can happen after connection is done
		cacheStartConnectionForDemo()
	}

	return r
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

//Sends a disconnect message to the server
//This is also called on Host_Error, so it shouldn't cause any errors
func (c *ClientStatic) Disconnect() {
	if keyDestination == keys.Message {
		// don't get stuck in chat mode
		chatEnd()
	}

	// stop sounds (especially looping!)
	snd.StopAll()

	// if running a local server, shut it down
	if c.demoPlayback {
		c.stopPlayback()
	} else if c.state == ca_connected {
		if c.demoRecording {
			cl.stopDemoRecording()
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
	screen.BeginLoadingPlaque()
	// need new connection messages
	cls.signon = 0
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
		screen.EndLoadingPlaque() // allow normal screen updates
	}
}

func CL_SendCmd() {
	if cls.state != ca_connected {
		return
	}

	if cls.signon == numSignonMessagesBeforeConn {
		cl.adjustAngles()
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
		switch ret := cls.getMessage(); ret {
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

// Determines the fraction between the last two messages that the objects
// should be put at.
//export CL_LerpPoint
func CL_LerpPoint() float64 {
	return cl.LerpPoint()
}

func (c *Client) LerpPoint() float64 {
	f := c.messageTime - c.messageTimeOld

	if f == 0 || cls.timeDemo || sv.active {
		c.time = c.messageTime
		return 1
	}

	// dropped packet, or start of demo
	if f > 0.1 {
		c.messageTimeOld = c.messageTime - 0.1
		f = 0.1
	}

	frac := (c.time - c.messageTimeOld) / f
	if frac < 0 {
		if frac < -0.01 {
			c.time = c.messageTimeOld
		}
		frac = 0
	} else if frac > 1 {
		if frac > 1.01 {
			c.time = c.messageTime
		}
		frac = 1
	}

	if cvars.ClientNoLerp.Bool() {
		return 1
	}

	return frac
}

func (c *Client) calcBob() float32 {
	square := func(v float32) float32 {
		return v * v
	}
	bobCycle := cvars.ClientBobCycle.Value()
	bobUp := cvars.ClientBobUp.Value()
	t := float32(c.time)

	cycle := (t - math32.Trunc(t/bobCycle)*bobCycle) / bobCycle
	if cycle < bobUp {
		cycle = math32.Pi * cycle / bobUp
	} else {
		cycle = math32.Pi + math32.Pi*(cycle-bobUp)/(1-bobUp)
	}
	// bob is proportional to velocity in the xy plane
	// (don't count Z, or jumping messes it up)
	v := math32.Sqrt(square(cl.velocity[0]) + square(cl.velocity[1]))
	bob := v * cvars.ClientBob.Value()
	bob = 0.3*bob + 0.7*bob*math32.Sin(cycle)
	if bob > 4 {
		return 4
	} else if bob < -7 {
		return -7
	}
	return bob
}

func (c *Client) startPitchDrift() {
	if c.lastStop == c.time {
		return // something else is keeping it from drifting
	}
	if !c.drift || cl.pitchVel == 0 {
		c.pitchVel = cvars.ViewCenterSpeed.Value()
		c.drift = true
		c.driftMove = 0
	}
}

func (c *Client) stopPitchDrift() {
	c.lastStop = c.time
	c.drift = false
	c.pitchVel = 0
}

/*
Moves the client pitch angle towards cl.idealPitch sent by the server.

If the user is adjusting pitch manually, either with lookup/lookdown,
mlook and mouse, or klook and keyboard, pitch drifting is constantly stopped.
*/
func (c *Client) driftPitch() {
	//float delta, move;

	if /* noclip_anglehack ||*/ !c.onGround || cls.demoPlayback {
		// FIXME: noclip_anglehack is set on the server, so in a nonlocal game this
		// won't work.
		c.driftMove = 0
		c.pitchVel = 0
		return
	}

	// don't count small mouse motion
	if !cl.drift {
		if math32.Abs(cl.cmdForwardMove) < cvars.ClientForwardSpeed.Value() {
			cl.driftMove = 0
		} else {
			cl.driftMove += float32(host.frameTime)
		}

		if cl.driftMove > cvars.ViewCenterMove.Value() {
			if cvars.LookSpring.Bool() {
				c.startPitchDrift()
			}
		}
		return
	}

	delta := cl.idealPitch - cl.pitch
	if delta == 0 {
		cl.pitchVel = 0
		return
	}

	move := float32(host.frameTime) * cl.pitchVel
	cl.pitchVel += float32(host.frameTime) * cvars.ViewCenterSpeed.Value()

	if delta > 0 {
		if move > delta {
			cl.pitchVel = 0
			move = delta
		}
		cl.pitch += move
	} else {
		if move > -delta {
			cl.pitchVel = 0
			move = -delta
		}
		cl.pitch -= move
	}
}

//export V_CalcBlend
func V_CalcBlend() {
	view.blendColor = cl.calcBlend()
}

func (c *Client) calcBlend() Color {
	color := Color{}
	p := cvars.GlColorShiftPercent.Value() / 100
	if p != 0 {
		for j, cs := range c.colorShifts {
			if c.intermission != 0 && j != ColorShiftContents {
				continue
			}
			a := cs.A * p
			if a == 0 {
				continue
			}
			color.A += (1 - color.A) * a
			a /= color.A
			color.R = math.Lerp(color.R, cs.R, a)
			color.G = math.Lerp(color.G, cs.G, a)
			color.B = math.Lerp(color.B, cs.B, a)
		}
	}
	color.A = math.Clamp32(0, color.A, 1)
	return color
}

func (c *Client) updateBlend() {
	c.calcPowerupColorShift()
	changed := false
	for i := 0; i < len(c.colorShifts); i++ {
		if c.colorShifts[i] != c.colorShiftsPrev[i] {
			changed = true
			c.colorShiftsPrev[i] = c.colorShifts[i]
		}
	}
	ft := float32(host.frameTime)
	c.colorShifts[ColorShiftDamage].A -= ft * 150 / 255.0
	if c.colorShifts[ColorShiftDamage].A < 0 {
		c.colorShifts[ColorShiftDamage].A = 0
	}
	c.colorShifts[ColorShiftBonus].A -= ft * 150 / 255.0
	if c.colorShifts[ColorShiftBonus].A < 0 {
		c.colorShifts[ColorShiftBonus].A = 0
	}
	if changed {
		view.blendColor = c.calcBlend()
	}
}

func (c *Client) calcPowerupColorShift() {
	switch {
	case c.items&progs.ItemQuad != 0:
		c.colorShifts[ColorShiftPowerup] = intColor(0, 0, 255, 30)
	case c.items&progs.ItemSuit != 0:
		c.colorShifts[ColorShiftPowerup] = intColor(0, 255, 0, 20)
	case c.items&progs.ItemInvisibility != 0:
		c.colorShifts[ColorShiftPowerup] = intColor(100, 100, 100, 100)
	case c.items&progs.ItemInvulnerability != 0:
		c.colorShifts[ColorShiftPowerup] = intColor(255, 255, 0, 30)
	default:
		c.colorShifts[ColorShiftPowerup].A = 0
	}
}

//export V_ParseDamage
func V_ParseDamage(armor, blood int, x, y, z float32) {
	cl.parseDamage(armor, blood, vec.Vec3{x, y, z})
}

func (c *Client) parseDamage(armor, blood int, from vec.Vec3) {
	count := float32(armor)*0.5 + float32(blood)*0.5
	if count < 10 {
		count = 10
	}
	count /= 255

	//put statusbar face into pain frame
	c.UpdateFaceAnimTime()

	cs := &c.colorShifts[ColorShiftDamage]

	cs.A += 3 * count
	cs.A = math.Clamp32(0, cs.A, 150/255.0)
	switch {
	case armor > blood:
		cs.R = 200 / 255.0
		cs.G = 100 / 255.0
		cs.B = 100 / 255.0
	case armor != 0:
		cs.R = 220 / 255.0
		cs.G = 50 / 255.0
		cs.B = 50 / 255.0
	default:
		cs.R = 1
		cs.G = 0
		cs.B = 0
	}

	ent := cl_entities(c.viewentity)
	origin := ent.origin()
	from = from.Sub(origin).Normalize()
	angles := ent.angles()
	forward, right, _ := vec.AngleVectors(angles)
	cl.dmgRoll = count * vec.Dot(from, right) * cvars.ViewKickRoll.Value()
	cl.dmgPitch = count * vec.Dot(from, forward) * cvars.ViewKickPitch.Value()
	cl.dmgTime = cvars.ViewKickTime.Value()
}

func (c *Client) calcViewRoll() {
	ent := cl_entities(c.viewentity)
	angles := ent.angles()
	side := CalcRoll(angles, c.velocity)
	qRefreshRect.viewAngles[ROLL] += side

	if c.dmgTime > 0 {
		kt := cvars.ViewKickTime.Value()
		qRefreshRect.viewAngles[ROLL] += c.dmgTime / kt * c.dmgRoll
		qRefreshRect.viewAngles[PITCH] += c.dmgTime / kt * c.dmgPitch
		c.dmgTime -= float32(host.frameTime)
	}
	if c.stats.health <= 0 {
		// dead view angle
		qRefreshRect.viewAngles[ROLL] = 80
	}
}

func (c *Client) boundOffsets() {
	ent := cl_entities(c.viewentity)

	// absolutely bound refresh relative to entity clipping hull
	// so the view can never be inside a solid wall
	o := ent.origin()
	qRefreshRect.viewOrg[0] = math.Clamp32(o[0]-14, qRefreshRect.viewOrg[0], o[0]+14)
	qRefreshRect.viewOrg[1] = math.Clamp32(o[1]-14, qRefreshRect.viewOrg[1], o[1]+14)
	qRefreshRect.viewOrg[2] = math.Clamp32(o[2]-22, qRefreshRect.viewOrg[2], o[2]+30)
}

func (c *Client) calcWeaponAngle() {
	idlescale := cvars.ViewIdleScale.Value()
	sway := func(cycle, level float32) float32 {
		return idlescale * math32.Sin(float32(c.time)*cycle) * level
	}
	yaw := qRefreshRect.viewAngles[YAW]
	pitch := qRefreshRect.viewAngles[PITCH]
	w := cl_weapon()
	w.ptr.angles[YAW] = C.float(yaw)
	w.ptr.angles[PITCH] = C.float(-pitch)
	w.ptr.angles[ROLL] -= C.float(sway(cvars.ViewIRollCycle.Value(), cvars.ViewIRollLevel.Value()))
	w.ptr.angles[PITCH] -= C.float(sway(cvars.ViewIPitchCycle.Value(), cvars.ViewIPitchLevel.Value()))
	w.ptr.angles[YAW] -= C.float(sway(cvars.ViewIYawCycle.Value(), cvars.ViewIYawLevel.Value()))
}

func (c *Client) addIdle(idlescale float32) {
	sway := func(cycle, level float32) float32 {
		return idlescale * math32.Sin(float32(c.time)*cycle) * level
	}
	qRefreshRect.viewAngles[ROLL] += sway(cvars.ViewIRollCycle.Value(), cvars.ViewIRollLevel.Value())
	qRefreshRect.viewAngles[PITCH] += sway(cvars.ViewIPitchCycle.Value(), cvars.ViewIPitchLevel.Value())
	qRefreshRect.viewAngles[YAW] += sway(cvars.ViewIYawCycle.Value(), cvars.ViewIYawLevel.Value())
}

func (c *Client) calcIntermissionRefreshRect() {
	ent := cl_entities(c.viewentity)
	// body
	qRefreshRect.viewOrg = ent.origin()
	qRefreshRect.viewAngles = ent.angles()
	// weaponmodel
	w := cl_weapon()
	w.ptr.model = nil

	c.addIdle(1)
}

//export V_SetContentsColor
func V_SetContentsColor(c int) {
	cl.setContentsColor(c)
}

func intColor(r, g, b, a float32) Color {
	f := float32(255.0)
	return Color{r / f, g / f, b / f, a / f}
}

var (
	cshiftEmpty = intColor(130, 80, 50, 0)
	cshiftLava  = intColor(255, 80, 0, 150)
	cshiftSlime = intColor(0, 25, 5, 150)
	cshiftWater = intColor(130, 80, 50, 128)
)

func (c *Client) setContentsColor(con int) {
	switch con {
	case model.CONTENTS_EMPTY, model.CONTENTS_SOLID, model.CONTENTS_SKY:
		c.colorShifts[ColorShiftContents] = cshiftEmpty
	case model.CONTENTS_LAVA:
		c.colorShifts[ColorShiftContents] = cshiftLava
	case model.CONTENTS_SLIME:
		c.colorShifts[ColorShiftContents] = cshiftSlime
	default:
		c.colorShifts[ColorShiftContents] = cshiftWater
	}
}

func (c *Client) bonusFlash() {
	c.colorShifts[ColorShiftBonus] = intColor(215, 186, 69, 50)
}

func init() {
	cmd.AddCommand("v_cshift", func(a []cmd.QArg, _ int) {
		cshiftEmpty = Color{0, 0, 0, 0}
		switch l := len(a); {
		case l >= 4:
			cshiftEmpty.A = a[3].Float32() / 255
			fallthrough
		case l == 3:
			cshiftEmpty.B = a[2].Float32() / 255
			fallthrough
		case l == 2:
			cshiftEmpty.G = a[1].Float32() / 255
			fallthrough
		case l == 1:
			cshiftEmpty.R = a[0].Float32() / 255
		}
	})
	cmd.AddCommand("bf", func(_ []cmd.QArg, _ int) { cl.bonusFlash() })
	cmd.AddCommand("centerview", func(_ []cmd.QArg, _ int) { cl.startPitchDrift() })
}

var (
	calcRefreshRectOldZ  = float32(0)
	calcRefreshRectPunch vec.Vec3
)

func (c *Client) calcRefreshRect() {
	c.driftPitch()

	// ent is the player model (visible when out of body)
	ent := cl_entities(c.viewentity)
	// view is the weapon model (only visible from inside body)
	w := cl_weapon() // view

	// transform the view offset by the model's matrix to get the offset from
	// model origin for the view
	ent.ptr.angles[YAW] = C.float(c.yaw) // the model should face the view dir
	// the model should face the view dir
	ent.ptr.angles[PITCH] = -C.float(c.pitch)

	bob := c.calcBob()

	// refresh position
	qRefreshRect.viewOrg = ent.origin()
	qRefreshRect.viewOrg[2] += c.viewHeight + bob

	// never let it sit exactly on a node line, because a water plane can
	// dissapear when viewed with the eye exactly on it.
	// the server protocol only specifies to 1/16 pixel, so add 1/32 in each axis
	qRefreshRect.viewOrg[0] += 1.0 / 32
	qRefreshRect.viewOrg[1] += 1.0 / 32
	qRefreshRect.viewOrg[2] += 1.0 / 32

	qRefreshRect.viewAngles[ROLL] = c.roll
	qRefreshRect.viewAngles[PITCH] = c.pitch
	qRefreshRect.viewAngles[YAW] = c.yaw

	c.calcViewRoll()
	c.addIdle(cvars.ViewIdleScale.Value())

	// offsets
	angles := ent.angles()
	// because entity pitches are actually backward
	angles[PITCH] *= -1
	forward, right, up := vec.AngleVectors(angles)

	if c.maxClients <= 1 {
		sx := cvars.ScreenOffsetX.Value()
		sy := cvars.ScreenOffsetY.Value()
		sz := cvars.ScreenOffsetZ.Value()
		qRefreshRect.viewOrg[0] += sx*forward[0] + sy*right[0] + sz*up[0]
		qRefreshRect.viewOrg[1] += sx*forward[1] + sy*right[1] + sz*up[1]
		qRefreshRect.viewOrg[2] += sx*forward[2] + sy*right[2] + sz*up[2]
	}

	c.boundOffsets()

	w.ptr.angles[ROLL] = C.float(c.roll)
	w.ptr.angles[PITCH] = C.float(c.pitch)
	w.ptr.angles[YAW] = C.float(c.yaw)

	c.calcWeaponAngle()
	w.ptr.origin[0] = ent.ptr.origin[0]
	w.ptr.origin[1] = ent.ptr.origin[1]
	w.ptr.origin[2] = ent.ptr.origin[2] + C.float(c.viewHeight)

	w.ptr.origin[0] += C.float(forward[0] * bob * 0.4)
	w.ptr.origin[1] += C.float(forward[1] * bob * 0.4)
	w.ptr.origin[2] += C.float(forward[2] * bob * 0.4)

	w.ptr.origin[2] += C.float(bob)

	C.SetCLWeaponModel(C.int(c.stats.weapon))
	w.ptr.frame = C.int(cl.stats.weaponFrame)

	switch cvars.ViewGunKick.Value() {
	case 1:
		// original quake kick
		qRefreshRect.viewAngles.Add(c.punchAngle[0])
	case 2:
		// lerped kick
		for i := 0; i < 3; i++ {
			if calcRefreshRectPunch[i] != c.punchAngle[0][i] {
				// speed determined by how far we need to lerp in 1/10th of a second
				delta := (c.punchAngle[0][i] - c.punchAngle[1][i]) * float32(host.frameTime) * 10
				if delta > 0 {
					calcRefreshRectPunch[i] = math32.Min(
						calcRefreshRectPunch[i]+delta,
						c.punchAngle[0][i])
				} else if delta < 0 {
					calcRefreshRectPunch[i] = math32.Max(
						calcRefreshRectPunch[i]+delta,
						c.punchAngle[0][i])
				}
			}
		}

		qRefreshRect.viewAngles.Add(calcRefreshRectPunch)
	}

	// smooth out stair step ups
	origin := ent.origin()
	if /*!noclip_anglehack &&*/ c.onGround && origin[2]-calcRefreshRectOldZ > 0 {
		// FIXME: noclip_anglehack is set on the server, so in a nonlocal game this
		// won't work.

		steptime := float32(c.time - c.oldTime)
		if steptime < 0 {
			steptime = 0
		}

		calcRefreshRectOldZ += steptime * 80
		if calcRefreshRectOldZ > origin[2] {
			calcRefreshRectOldZ = origin[2]
		}
		if origin[2]-calcRefreshRectOldZ > 12 {
			calcRefreshRectOldZ = origin[2] - 12
		}
		qRefreshRect.viewOrg[2] += calcRefreshRectOldZ - origin[2]
		w.ptr.origin[2] += C.float(calcRefreshRectOldZ - origin[2])
	} else {
		calcRefreshRectOldZ = origin[2]
	}

	if cvars.ChaseActive.Bool() {
		Chase_UpdateForDrawing()
	}
}

// Server information pertaining to this client only
//export CL_ParseClientdata
func CL_ParseClientdata() {
	err := cl.parseClientData()
	if err != nil {
		cls.msgBadRead = true
	}
}

func (c *Client) parseClientData() error {
	m, err := cls.inMessage.ReadUint16()
	if err != nil {
		return err
	}
	bits := int(m)

	if bits&svc.SU_EXTEND1 != 0 {
		m, err := cls.inMessage.ReadByte()
		if err != nil {
			return err
		}
		bits |= int(m) << 16
	}
	if bits&svc.SU_EXTEND2 != 0 {
		m, err := cls.inMessage.ReadByte()
		if err != nil {
			return err
		}
		bits |= int(m) << 24
	}

	if bits&svc.SU_VIEWHEIGHT != 0 {
		m, err := cls.inMessage.ReadInt8()
		if err != nil {
			return err
		}
		c.viewHeight = float32(m)
	} else {
		c.viewHeight = svc.DEFAULT_VIEWHEIGHT
	}

	if bits&svc.SU_IDEALPITCH != 0 {
		m, err := cls.inMessage.ReadInt8()
		if err != nil {
			return err
		}
		c.idealPitch = float32(m)
	} else {
		c.idealPitch = 0
	}

	var punchAngle vec.Vec3
	c.mVelocity[1] = c.mVelocity[0]
	for i := 0; i < 3; i++ {
		if bits&(svc.SU_PUNCH1<<i) != 0 {
			m, err := cls.inMessage.ReadInt8()
			if err != nil {
				return err
			}
			punchAngle[i] = float32(m)
		} else {
			punchAngle[i] = 0
		}

		if bits&(svc.SU_VELOCITY1<<i) != 0 {
			m, err := cls.inMessage.ReadInt8()
			if err != nil {
				return err
			}
			c.mVelocity[0][i] = float32(m) * 16
		} else {
			c.mVelocity[0][i] = 0
		}
	}

	if c.punchAngle[0] != punchAngle {
		c.punchAngle[1] = c.punchAngle[0]
		c.punchAngle[0] = punchAngle
	}

	// [always sent]	if (bits & svc.SU_ITEMS) != 0
	items, err := cls.inMessage.ReadUint32()
	if err != nil {
		return err
	}

	if c.items != items {
		// set flash times
		statusbar.MarkChanged()
		d := c.items ^ items
		for i := 0; i < 32; i++ {
			if d&(1<<i) != 0 {
				//if (i & (1 << j)) && !(CL_HasItem(1 << j)) {
				cl.itemGetTime[i] = cl.time
			}
		}
		c.items = items
	}

	c.onGround = bits&svc.SU_ONGROUND != 0

	if bits&svc.SU_WEAPONFRAME != 0 {
		m, err := cls.inMessage.ReadByte()
		if err != nil {
			return err
		}
		c.stats.weaponFrame = int(m)
	} else {
		c.stats.weaponFrame = 0
	}

	armor := 0
	if bits&svc.SU_ARMOR != 0 {
		m, err := cls.inMessage.ReadByte()
		if err != nil {
			return err
		}
		armor = int(m)
	}
	if c.stats.armor != armor {
		c.stats.armor = armor
		statusbar.MarkChanged()
	}

	weapon := 0
	if bits&svc.SU_WEAPON != 0 {
		m, err := cls.inMessage.ReadByte()
		if err != nil {
			return err
		}
		weapon = int(m)
	}

	if c.stats.weapon != weapon {
		c.stats.weapon = weapon
		statusbar.MarkChanged()
	}

	health, err := cls.inMessage.ReadInt16()
	if err != nil {
		return err
	}
	if c.stats.health != int(health) {
		c.stats.health = int(health)
		statusbar.MarkChanged()
	}

	readByte := func(v *int) error {
		nv, err := cls.inMessage.ReadByte()
		if err != nil {
			return err
		}
		if *v != int(nv) {
			*v = int(nv)
			statusbar.MarkChanged()
		}
		return nil
	}

	if err := readByte(&c.stats.ammo); err != nil {
		return err
	}
	if err := readByte(&c.stats.shells); err != nil {
		return err
	}
	if err := readByte(&c.stats.nails); err != nil {
		return err
	}
	if err := readByte(&c.stats.rockets); err != nil {
		return err
	}
	if err := readByte(&c.stats.cells); err != nil {
		return err
	}

	activeWeapon, err := cls.inMessage.ReadByte()
	if err != nil {
		return err
	}
	if cmdl.Quoth() || cmdl.Rogue() || cmdl.Hipnotic() {
		//TODO(therjak): why is the command line setting responsible for how the server
		// message is interpreted?
		if c.stats.activeWeapon != (1 << activeWeapon) {
			c.stats.activeWeapon = (1 << activeWeapon)
			statusbar.MarkChanged()
		}
	} else {
		// StandardQuake
		if c.stats.activeWeapon != int(activeWeapon) {
			c.stats.activeWeapon = int(activeWeapon)
			statusbar.MarkChanged()
		}
	}

	statOr := func(v *int) error {
		s, err := cls.inMessage.ReadByte()
		if err != nil {
			return err
		}
		*v |= int(s) << 8
		return nil
	}
	if bits&svc.SU_WEAPON2 != 0 {
		if err := statOr(&c.stats.weapon); err != nil {
			return err
		}
	}
	if bits&svc.SU_ARMOR2 != 0 {
		if err := statOr(&c.stats.armor); err != nil {
			return err
		}
	}
	if bits&svc.SU_AMMO2 != 0 {
		if err := statOr(&c.stats.ammo); err != nil {
			return err
		}
	}
	if bits&svc.SU_SHELLS2 != 0 {
		if err := statOr(&c.stats.shells); err != nil {
			return err
		}
	}
	if bits&svc.SU_NAILS2 != 0 {
		if err := statOr(&c.stats.nails); err != nil {
			return err
		}
	}
	if bits&svc.SU_ROCKETS2 != 0 {
		if err := statOr(&c.stats.rockets); err != nil {
			return err
		}
	}
	if bits&svc.SU_CELLS2 != 0 {
		if err := statOr(&c.stats.cells); err != nil {
			return err
		}
	}
	if bits&svc.SU_WEAPONFRAME2 != 0 {
		if err := statOr(&c.stats.weaponFrame); err != nil {
			return err
		}
	}
	cl_weapon().ptr.alpha = 0 // ENTALPHA_DEFAULT
	if bits&svc.SU_WEAPONALPHA != 0 {
		a, err := cls.inMessage.ReadByte()
		if err != nil {
			return err
		}
		cl_weapon().ptr.alpha = C.uchar(a)
	}
	//TODO(THERJAK)
	/*
		// this was done before the upper 8 bits of cl.stats[STAT_WEAPON]
		// were filled in, breaking on large maps like zendar.bsp
		if cl_viewent.model != cl.model_precache[CL_Stats(STAT_WEAPON)] {
			// don't lerp animation across model changes
			cl_viewent.lerpflags |= LERP_RESETANIM
		}
	*/
	return nil
}

func (c *ClientStatic) stopPlayback() {
	if !c.demoPlayback {
		return
	}

	// TODO: close file and null file handle
	c.demoPlayback = false
	c.demoPaused = false
	c.state = ca_disconnected

	if c.timeDemo {
		c.finishTimeDemo()
	}
}

func (c *ClientStatic) finishTimeDemo() {
	c.timeDemo = false
	// the first frame didn't count
	frames := host.frameCount - c.timeDemoStartFrame - 1
	time := host.time - float64(c.timeDemoStartTime)
	if time == 0 {
		time = 1
	}
	conlog.Printf("%d frames %5.1f seconds %5.1f fps\n", frames, time, float64(frames)/time)
}

func (c *Client) writeDemoMessage() {
	// write 4 bytes: length of net_message
	// write 4 bytes: float of cl.pitch
	// write 4 bytes: float of cl.yaw
	// write 4 bytes: float of cl.roll
	// write net_message
	// flush
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

func clientRecordDemo(args []cmd.QArg, player int) {
	if !execute.IsSrcCommand() {
		return
	}
	if cls.demoPlayback {
		conlog.Printf("Can''t record during demo playback\n")
		return
	}
	// len(args) must be 2,3,4...
	// TODO
	cl.recordDemo()
}

func clientStopDemoRecording(args []cmd.QArg, player int) {
	if !execute.IsSrcCommand() {
		return
	}
	if !cls.demoRecording {
		conlog.Printf("Not recording a demo.\n")
		return
	}
	cl.stopDemoRecording()
}

func clientPlayDemo(args []cmd.QArg, player int) {
	if !execute.IsSrcCommand() {
		return
	}

	if len(args) != 1 {
		conlog.Printf("playdemo <demoname> : plays a demo\n")
		return
	}

	if err := cls.playDemo(args[0].String()); err != nil {
		conlog.Printf("Error: %v", err)
	}
}

func clientTimeDemo(args []cmd.QArg, player int) {
	if !execute.IsSrcCommand() {
		return
	}

	if len(args) != 1 {
		conlog.Printf("timedemo <demoname> : gets demo speeds\n")
		return
	}

	cls.startTimeDemo(args[0].String())
}

// Called to play the next demo in the demo loop
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

	screen.BeginLoadingPlaque()

	cbuf.InsertText(fmt.Sprintf("playdemo %s\n", cls.demos[cls.demoNum]))
	cls.demoNum++
}

func (c *Client) stopDemoRecording() {
	// TODO:
	// svc_disconnect
	// demomessage
	// close file
	// cls.demofile = null

	cls.demoRecording = false
	conlog.Printf("Completed demo\n")
}

func (c *Client) recordDemo() {
	if cls.demoRecording {
		c.stopDemoRecording()
	}
	// TODO
}

func (c *ClientStatic) getDemoMessage() int {
	if c.demoPaused {
		return 0
	}

	// decide if it is time to grab the next message
	if c.signon == 4 /*SIGNONS*/ {
		// always grab until fully connected
		if c.timeDemo {
			if host.frameCount == c.timeDemoLastFrame {
				// already read this frame's message
				return 0
			}
			c.timeDemoLastFrame = host.frameCount
			// if this is the second frame, grab the real timeDemoStartTime
			// so the bogus time on the first frame doesn't count
			if host.frameCount == c.timeDemoStartFrame+1 {
				c.timeDemoStartTime = host.time
			}
		} else if cl.time <= cl.messageTime {
			// don't need another message yet
			return 0
		}
	}
	// 32bit integer message size
	// 3x 32bit float mViewAngle
	// message
	type demoHeader struct {
		Size       int32
		ViewAngleX float32
		ViewAngleY float32
		ViewAngleZ float32
	}
	var h demoHeader
	r := bytes.NewReader(c.demoData)
	if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
		c.stopPlayback()
		return 0
	}

	c.demoData = c.demoData[16:]
	cl.mViewAngles[1][0] = cl.mViewAngles[0][0]
	cl.mViewAngles[1][1] = cl.mViewAngles[0][1]
	cl.mViewAngles[1][2] = cl.mViewAngles[0][2]
	cl.mViewAngles[0][0] = h.ViewAngleX
	cl.mViewAngles[0][1] = h.ViewAngleY
	cl.mViewAngles[0][2] = h.ViewAngleZ

	if len(c.demoData) < int(h.Size) {
		c.stopPlayback()
		return 0
	}
	c.inMessage = net.NewQReader(c.demoData[:h.Size])
	c.demoData = c.demoData[h.Size:]
	return 1
}

func (c *ClientStatic) playDemo(name string) error {
	c.Disconnect()
	if !strings.HasSuffix(name, ".dem") {
		name += ".dem"
	}
	b, err := filesystem.GetFileContents(name)
	if err != nil {
		c.demoNum = -1 // stop demo loop
		return err
	}
	i := bytes.IndexByte(b, '\n')
	if i > 13 {
		c.demoData = []byte{}
		c.demoNum = -1 // stop demo loop
		return fmt.Errorf("demo \"%s\" is invalid\n", name)
	}
	c.demoData = b[i+1:] // cut the cd track + line break
	c.demoPlayback = true
	c.demoPaused = false
	c.state = ca_connected
	// get rid of the menu and/or console
	keyDestination = keys.Game
	return nil
}

func (c *ClientStatic) startTimeDemo(name string) {
	if err := c.playDemo(name); err != nil {
		return
	}
	// cls.timeDemoStartTime will be grabbed at the second frame of the demo,
	// so all the loading time doesn't get counted
	c.timeDemo = true
	c.timeDemoStartFrame = host.frameCount
	c.timeDemoLastFrame = -1 // get a new message this frame
}
