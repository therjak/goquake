package quakelib

//void CL_StopPlayback(void);
//void CL_Stop_f(void);
//#define SFX_WIZHIT  0
//#define SFX_KNIGHTHIT  1
//#define SFX_TINK1  2
//#define SFX_RIC1  3
//#define SFX_RIC2  4
//#define SFX_RIC3  5
//#define SFX_R_EXP3  6
//#include "cgo_help.h"
//extern float v_blend[4];
//void SetCLWeaponModel(int v);
import "C"

import (
	"bytes"
	"fmt"
	"github.com/chewxy/math32"
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
	"quake/progs"
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
		cl.stopPitchDrift()
		cl.pitch -= speed * cvars.ClientPitchSpeed.Value() * input.Forward.ConsumeImpulse()
		cl.pitch += speed * cvars.ClientPitchSpeed.Value() * input.Back.ConsumeImpulse()
	}

	up := input.LookUp.ConsumeImpulse()
	down := input.LookDown.ConsumeImpulse()

	cl.pitch -= speed * cvars.ClientPitchSpeed.Value() * up
	cl.pitch += speed * cvars.ClientPitchSpeed.Value() * down

	if up != 0 || down != 0 {
		cl.stopPitchDrift()
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

//export SetCL_PunchAngle
func SetCL_PunchAngle(i, j int, v float32) {
	cl.punchAngle[i][j] = v
}

//export CL_PunchAngle
func CL_PunchAngle(i, j int) float32 {
	return cl.punchAngle[i][j]
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

//export CL_Velocity
func CL_Velocity(i int) float32 {
	return cl.velocity[i]
}

//export CL_IdealPitch
func CL_IdealPitch() float32 {
	return cl.idealPitch
}

//export CL_SetIdealPitch
func CL_SetIdealPitch(p float32) {
	cl.idealPitch = p
}

//export CL_PitchVel
func CL_PitchVel() float32 {
	return cl.pitchVel
}

//export CL_SetPitchVel
func CL_SetPitchVel(v float32) {
	cl.pitchVel = v
}

//export CL_NoDrift
func CL_NoDrift() bool {
	return !cl.drift
}

//export CL_SetNoDrift
func CL_SetNoDrift(b bool) {
	cl.drift = !b
}

//export CL_DriftMove
func CL_DriftMove() float32 {
	return cl.driftMove
}

//export CL_SetDriftMove
func CL_SetDriftMove(m float32) {
	cl.driftMove = m
}

//export CL_LastStop
func CL_LastStop() float64 {
	return cl.lastStop
}

//export CL_SetLastStop
func CL_SetLastStop(s float64) {
	cl.lastStop = s
}

//export CL_ViewHeight
func CL_ViewHeight() float32 {
	return cl.viewHeight
}

//export CL_SetViewHeight
func CL_SetViewHeight(h float32) {
	cl.viewHeight = h
}

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

//export CL_UpdateFaceAnimTime
func CL_UpdateFaceAnimTime() {
	cl.UpdateFaceAnimTime()
}

func (c *Client) UpdateFaceAnimTime() {
	c.faceAnimTime = c.time + 0.2
}

//export CL_CheckFaceAnimTime
func CL_CheckFaceAnimTime() bool {
	return cl.CheckFaceAnimTime()
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

//export CL_Items
func CL_Items() uint32 {
	return cl.items
}

//export CL_SetItems
func CL_SetItems(items uint32) {
	cl.items = items
}

//export CL_ItemGetTime
func CL_ItemGetTime(item int) float64 {
	return cl.itemGetTime[item]
}

//export CL_SetItemGetTime
func CL_SetItemGetTime(item int) {
	cl.itemGetTime[item] = cl.time
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
	screen.BeginLoadingPlaque()
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
		screen.EndLoadingPlaque() // allow normal screen updates
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

	screen.BeginLoadingPlaque()

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

//export V_CalcBob
func V_CalcBob() float32 {
	return cl.calcBob()
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

//export V_StartPitchDrift
func V_StartPitchDrift() {
	cl.startPitchDrift()
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

//export V_DriftPitch
func V_DriftPitch() {
	cl.driftPitch()
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
	cl.calcBlend()
}

func (c *Client) calcBlend() {
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

	C.v_blend[0] = C.float(color.R)
	C.v_blend[1] = C.float(color.G)
	C.v_blend[2] = C.float(color.B)
	C.v_blend[3] = C.float(color.A)
}

//export V_UpdateBlend
func V_UpdateBlend() {
	cl.updateBlend()
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
		c.calcBlend()
	}
}

//export V_PolyBlend
func V_PolyBlend(vb *C.float) {
	c := Color{
		float32(C.cf(0, vb)),
		float32(C.cf(1, vb)),
		float32(C.cf(2, vb)),
		float32(C.cf(3, vb)),
	}
	if !cvars.GlPolyBlend.Bool() || c.A == 0 {
		return
	}

	textureManager.DisableMultiTexture()
	qRecDrawer.Draw(0, 0, float32(viewport.width), float32(viewport.height), c)
}

//export V_CalcPowerupCshift
func V_CalcPowerupCshift() {
	cl.calcPowerupColorShift()
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

//export V_CalcViewRoll
func V_CalcViewRoll() {
	cl.calcViewRoll()
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

//export V_BoundOffsets
func V_BoundOffsets() {
	cl.boundOffsets()
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

//export CalcGunAngle
func CalcGunAngle() {
	cl.calcWeaponAngle()
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

//export V_AddIdle
func V_AddIdle(idlescale float32) {
	cl.addIdle(idlescale)
}

func (c *Client) addIdle(idlescale float32) {
	sway := func(cycle, level float32) float32 {
		return idlescale * math32.Sin(float32(c.time)*cycle) * level
	}
	qRefreshRect.viewAngles[ROLL] += sway(cvars.ViewIRollCycle.Value(), cvars.ViewIRollLevel.Value())
	qRefreshRect.viewAngles[PITCH] += sway(cvars.ViewIPitchCycle.Value(), cvars.ViewIPitchLevel.Value())
	qRefreshRect.viewAngles[YAW] += sway(cvars.ViewIYawCycle.Value(), cvars.ViewIYawLevel.Value())
}

//export V_CalcIntermissionRefdef
func V_CalcIntermissionRefdef() {
	cl.calcIntermissionRefreshRect()
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

//export V_CalcRefdef
func V_CalcRefdef() {
	cl.calcRefreshRect()
}

var (
	calcRefreshRectOldZ = float32(0)
)

func (c *Client) calcRefreshRect() {
	/*
	  entity_t *ent, *view;
	  int i;
	  vec3_t forward, right, up;
	  vec3_t angles;
	  float bob;
	  static float oldz = 0;
	  static vec3_t punch = {0, 0, 0};  // johnfitz -- v_gunkick
	  float delta;                      // johnfitz -- v_gunkick
	*/
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

	/*
	  // offsets
	  // because entity pitches are actually backward
	  angles[PITCH] = -ent->angles[PITCH];
	  angles[YAW] = ent->angles[YAW];
	  angles[ROLL] = ent->angles[ROLL];

	  AngleVectors(angles, forward, right, up);

	  if (CL_MaxClients() <= 1)
	    for (i = 0; i < 3; i++)
	      r_refdef.vieworg[i] += Cvar_GetValue(&scr_ofsx) * forward[i] +
	                             Cvar_GetValue(&scr_ofsy) * right[i] +
	                             Cvar_GetValue(&scr_ofsz) * up[i];
	*/
	c.boundOffsets()

	w.ptr.angles[ROLL] = C.float(c.roll)
	w.ptr.angles[PITCH] = C.float(c.pitch)
	w.ptr.angles[YAW] = C.float(c.yaw)

	c.calcWeaponAngle()
	w.ptr.origin[0] = ent.ptr.origin[0]
	w.ptr.origin[1] = ent.ptr.origin[1]
	w.ptr.origin[2] = ent.ptr.origin[2] + C.float(c.viewHeight)
	/*
	  for (i = 0; i < 3; i++) view->origin[i] += forward[i] * bob * 0.4;
	*/
	w.ptr.origin[2] += C.float(bob)

	C.SetCLWeaponModel(C.int(c.stats.weapon))
	w.ptr.frame = C.int(cl.stats.weaponFrame)

	/*
	  if (Cvar_GetValue(&v_gunkick) == 1) { // original quake kick
	    r_refdef.viewangles[0] += CL_PunchAngle(0,0);
	    r_refdef.viewangles[1] += CL_PunchAngle(0,1);
	    r_refdef.viewangles[2] += CL_PunchAngle(0,2);
	  }
	  if (Cvar_GetValue(&v_gunkick) == 2) { // lerped kick
	    for (i = 0; i < 3; i++)
	      if (punch[i] != CL_PunchAngle(0,i)) {
	        // speed determined by how far we need to lerp in 1/10th of a second
	        delta =
	            (CL_PunchAngle(0,i) - CL_PunchAngle(1,i)) * HostFrameTime() * 10;

	        if (delta > 0)
	          punch[i] = q_min(punch[i] + delta, CL_PunchAngle(0,i));
	        else if (delta < 0)
	          punch[i] = q_max(punch[i] + delta, CL_PunchAngle(0,i));
	      }

	    VectorAdd(r_refdef.viewangles, punch, r_refdef.viewangles);
	  }
	*/

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
