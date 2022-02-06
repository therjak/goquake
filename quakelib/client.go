// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

//void CL_ClearState(void);
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"goquake/bsp"
	"goquake/cbuf"
	"goquake/cmd"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvars"
	"goquake/execute"
	"goquake/filesystem"
	"goquake/input"
	"goquake/keys"
	"goquake/math"
	"goquake/math/vec"
	"goquake/mdl"
	"goquake/model"
	"goquake/net"
	"goquake/progs"
	clc "goquake/protocol/client"
	svc "goquake/protocol/server"
	"goquake/protos"
	"goquake/rand"
	"goquake/snd"
	"goquake/stat"

	"github.com/chewxy/math32"
)

type sfx int

const (
	WizHit    sfx = 0
	KnightHit sfx = 1
	Tink1     sfx = 2
	Ric1      sfx = 3
	Ric2      sfx = 4
	Ric3      sfx = 5
	RExp3     sfx = 6
)

const (
	ColorShiftContents = iota
	ColorShiftDamage
	ColorShiftBonus
	ColorShiftPowerup
)

var (
	cRand = rand.New(0)
)

func init() {
	Must(cmd.AddCommand("disconnect", func(args []cmd.QArg, _ int) { clientDisconnect() }))
	Must(cmd.AddCommand("reconnect", func(args []cmd.QArg, _ int) { clientReconnect() }))

	Must(cmd.AddCommand("startdemos", clientStartDemos))
	Must(cmd.AddCommand("record", clientRecordDemo))
	Must(cmd.AddCommand("stop", clientStopDemoRecording))
	Must(cmd.AddCommand("playdemo", clientPlayDemo))
	Must(cmd.AddCommand("timedemo", clientTimeDemo))

	Must(cmd.AddCommand("tracepos", tracePosition))
	//cmd.AddCommand("mcache", Mod_Print);
}

const (
	numSignonMessagesBeforeConn = 4 // signon messages to receive before connected
)

type clientConnected bool

const (
	ca_disconnected clientConnected = false
	ca_connected    clientConnected = true
)

func (c *Client) adjustAngles() {
	speed := func() float32 {
		if (cvars.ClientForwardSpeed.Value() > 200) != input.Speed.Down() {
			return float32(host.frameTime) * cvars.ClientAngleSpeedKey.Value()
		}
		return float32(host.frameTime)
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
// TODO(therjak): merge with Client
type ClientStatic struct {
	state              clientConnected
	demoNum            int
	demoPlayback       bool
	demoPaused         bool
	demoSignon         [2]bytes.Buffer
	timeDemo           bool
	signon             int
	connection         *net.Connection
	inMessage          *net.QReader
	outProto           protos.ClientMessage
	timeDemoLastFrame  int
	timeDemoStartFrame int
	timeDemoStartTime  float64
	// personalization data sent to server
	// to restart a level
	spawnParms string
	demos      []string
	demoWriter io.WriteCloser
	demoData   []byte
	msgBadRead bool
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
	protocolFlags  uint32
	protocol       int
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
	intermission   int
	staticEntities []Entity
	entities       []*Entity
	dynamicLights  []DynamicLight
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

	mapName       string
	levelName     string
	worldModel    *bsp.Model
	modelPrecache []model.Model

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
	cl  = Client{
		staticEntities: make([]Entity, 0, 512),
	}
)

func (c *Client) setMaxEdicts(num int) {
	cl.entities = make([]*Entity, 0, num)
	// ensure at least a world entity at the start
	cl.GetOrCreateEntity(0)
}

func (c *Client) UpdateFaceAnimTime() {
	c.faceAnimTime = c.time + 0.2
}

func (c *Client) CheckFaceAnimTime() bool {
	return cl.time <= cl.faceAnimTime
}

func (c *Client) DeathMatch() bool {
	return cl.gameType == svc.GameDeathmatch
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

func clientClear() {
	cl = Client{
		staticEntities: make([]Entity, 0, 512),
	}
	clearLightStyles()
	clearBeams()
	clearEntityFragments()
}

//export CL_Paused
func CL_Paused() bool {
	return cl.paused
}

//export CL_Time
func CL_Time() C.double {
	return C.double(cl.time)
}

func viewPositionCommand(args []cmd.QArg, _ int) {
	if cls.state != ca_connected {
		return
	}
	printPosition()
}

func printPosition() {
	player := cl.Entities(cl.viewentity)
	pos := player.Origin
	conlog.Printf("Viewpos: (%.f %.f %.f) %.f %.f %.f\n",
		pos[0], pos[1], pos[2],
		cl.pitch, cl.yaw, cl.roll)
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
		cls.outProto.Cmds = append(cls.outProto.Cmds, &protos.Cmd{
			Union: &protos.Cmd_StringCmd{cmd.Full()},
		})
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
	if len(args) > 0 {
		s := c + " " + cmd.Full()
		cls.outProto.Cmds = append(cls.outProto.Cmds, &protos.Cmd{
			Union: &protos.Cmd_StringCmd{s},
		})
	} else {
		cls.outProto.Cmds = append(cls.outProto.Cmds, &protos.Cmd{
			Union: &protos.Cmd_StringCmd{c},
		})
	}
}

func init() {
	Must(cmd.AddCommand("cmd", executeOnServer))
	Must(cmd.AddCommand("viewpos", viewPositionCommand))
}

// Read all incoming data from the server
func (c *Client) ReadFromServer() serverState {
	c.oldTime = cl.time
	c.time += host.frameTime
	for {
		// TODO: code needs major cleanup (getMessage + CL_ParseServerMessage)
		ret := cls.getMessage()
		if ret == -1 {
			HostError(fmt.Errorf("CL_ReadFromServer: lost server connection"))
		}
		if ret == 0 {
			break
		}
		c.lastReceivedMessageTime = host.time
		pb, err := svc.ParseServerMessage(cls.inMessage, c.protocol, c.protocolFlags)
		if err != nil {
			fmt.Printf("Bad server message\n %v", err)
			HostError(fmt.Errorf("CL_ParseServerMessage: Bad server message"))
		}
		if serverState, err := CL_ParseServerMessage(pb); err != nil {
			HostError(err)
		} else if serverState == serverDisconnected {
			return serverDisconnected
		}
		if cls.state != ca_connected {
			break
		}
	}
	if cvars.ClientShowNet.Bool() {
		conlog.Printf("\n")
	}

	frac := float32(c.LerpPoint())
	// interpolate player info
	c.velocity = vec.Lerp(c.mVelocity[1], c.mVelocity[0], frac)
	// mViewAngles [2]vec.Vec3
	if cls.demoPlayback {
		// interpolate the angles
		// this has some problems as it could be off by 180 and
		// the current computation could even result in values
		// outside of [-180,180] but is consistend with orig
		d := vec.Sub(c.mViewAngles[0], c.mViewAngles[1])
		for i := 0; i < 3; i++ {
			if d[i] > 180 {
				d[i] -= 360
			} else if d[i] < -180 {
				d[i] += 360
			}
		}
		df := vec.Add(c.mViewAngles[1], vec.Scale(frac, d))
		c.pitch = df[0]
		c.yaw = df[1]
		c.roll = df[2]
	}

	c.RelinkEntities(frac)
	c.updateTempEntities()
	return serverRunning
}

func (c *Client) RelinkEntities(frac float32) {
	ClearVisibleEntities()
	bobjrotate := float32(math.AngleMod(100 * c.time))
	for i, e := range c.entities {
		e.Relink(frac, bobjrotate, i)
	}
}

func (c *ClientStatic) getMessage() int {
	// for cl_main: return -1 on error, return 0 for message end, everything else is continue
	// for cl_parse: return 0 for end message, 2 && ReadByte == Nop continue, everything else is Host_Error
	if c.demoPlayback {
		c.msgBadRead = false
		return cls.getDemoMessage()
	}

	r := 0
	var data []byte
	var err error
	for {
		c.msgBadRead = false

		data, err = c.connection.GetMessage()
		if err != nil {
			return -1
		}
		if len(data) == 0 {
			return 0
		}
		// discard nop keepalive message
		if len(data) == 2 && data[1] == svc.Nop {
			// Con_Printf("<-- server to client keepalive\n")
			continue
		}
		r = int(data[0])
		// drop the 'r' value as it is not part of the actual data and only
		// indicates if it was a reliable (1) or unreliable (2) send
		data = data[1:]
		break
	}
	c.inMessage = net.NewQReader(data)

	if c.demoWriter != nil {
		c.writeDemoMessage(data)
	}

	if c.signon < 2 {
		// record messages before full connection, so that a
		// demo record can happen after connection is done
		c.demoSignon[c.signon].Reset()
		c.demoSignon[c.signon].Write(data)
	}

	return r
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
		c.stopDemoRecording()

		conlog.DPrintf("Sending clc_disconnect\n")

		cls.outProto.Cmds = append(cls.outProto.Cmds[:0], &protos.Cmd{
			Union: &protos.Cmd_Disconnect{true},
		})
		b := clc.ToBytes(&cls.outProto, cl.protocol, cl.protocolFlags)
		c.connection.SendUnreliableMessage(b)
		cls.outProto.Reset()
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
	if cmdl.Dedicated() {
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
		HostError(fmt.Errorf("CLS_Connect: connect failed\n"))
	}
	cls.connection = c
	conlog.DPrintf("CL_EstablishConnection: connected to %s\n", host)

	// not in the demo loop now
	cls.demoNum = -1
	cls.state = ca_connected
	// need all the signon messages before playing
	cls.signon = 0
	cls.outProto.Cmds = append(cls.outProto.Cmds, &protos.Cmd{})
}

// An svc_signonnum has been received, perform a client side setup
func CL_SignonReply() {
	conlog.DPrintf("CL_SignonReply: %d\n", cls.signon)

	switch cls.signon {
	case 1:
		cls.outProto.Cmds = append(cls.outProto.Cmds, &protos.Cmd{
			Union: &protos.Cmd_StringCmd{"prespawn"},
		})

	case 2:
		color := int(cvars.ClientColor.Value())
		cls.outProto.Cmds = append(cls.outProto.Cmds,
			&protos.Cmd{
				Union: &protos.Cmd_StringCmd{fmt.Sprintf("name \"%s\"", cvars.ClientName.String())},
			},
			&protos.Cmd{
				Union: &protos.Cmd_StringCmd{fmt.Sprintf("color %d %d", color>>4, color&15)},
			},
			&protos.Cmd{
				Union: &protos.Cmd_StringCmd{fmt.Sprintf("spawn %s", cls.spawnParms)},
			},
		)

	case 3:
		cls.outProto.Cmds = append(cls.outProto.Cmds, &protos.Cmd{
			Union: &protos.Cmd_StringCmd{"begin"},
		})

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
		cls.outProto.Reset()
		return
	}

	if len(cls.outProto.Cmds) == 0 {
		return // no message at all
	}

	if !cls.connection.CanSendMessage() {
		conlog.DPrintf("CL_SendCmd: can't send\n")
		return
	}

	b := clc.ToBytes(&cls.outProto, cl.protocol, cl.protocolFlags)
	i := cls.connection.SendMessage(b)
	if i == -1 {
		HostError(fmt.Errorf("CL_SendCmd: lost server connection"))
	}
	cls.outProto.Reset()
}

func CL_ParseStartSoundPacket(m *protos.Sound) error {
	const (
		maxSounds = 2048
	)
	if m.SoundNum > maxSounds {
		return fmt.Errorf("CL_ParseStartSoundPacket: %d > MAX_SOUNDS", m.SoundNum)
	}
	if m.Entity > int32(cap(cl.entities)) {
		return fmt.Errorf("CL_ParseStartSoundPacket: ent = %d", m.Entity)
	}
	volume := float32(1.0)
	if v := m.Volume; v != nil {
		volume = float32(v.Value) / 255
	}
	attenuation := float32(1.0)
	if a := m.Attenuation; a != nil {
		attenuation = float32(a.Value) / 64.0
	}
	origin := vec.Vec3{m.Origin.X, m.Origin.Y, m.Origin.Z}
	snd.Start(int(m.Entity), int(m.Channel), cl.soundPrecache[m.SoundNum], origin, volume, attenuation, !loopingSound)
	return nil
}

var (
	clientKeepAliveTime time.Time
)

// When the client is taking a long time to load stuff, send keepalive messages
// so the server doesn't disconnect.
func CL_KeepaliveMessage() {
	if sv.active {
		// no need if server is local
		return
	}
	if cls.demoPlayback {
		return
	}

	msgBackup := cls.inMessage

	// read messages from server, should just be nops
Outer:
	for {
		switch ret := cls.getMessage(); ret {
		default:
			HostError(fmt.Errorf("CL_KeepaliveMessage: CL_GetMessage failed"))
		case 0:
			break Outer
		case 1:
			HostError(fmt.Errorf("CL_KeepaliveMessage: received a message"))
		case 2:
			conlog.Printf("WTF? This should never happen")
			i, err := cls.inMessage.ReadByte()
			if err != nil || i != svc.Nop {
				HostError(fmt.Errorf("CL_KeepaliveMessage: datagram wasn't a nop"))
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

	cls.outProto.Cmds = append(cls.outProto.Cmds, &protos.Cmd{})
	b := clc.ToBytes(&cls.outProto, cl.protocol, cl.protocolFlags)
	cls.connection.SendMessage(b)
	cls.outProto.Reset()
}

func clientInit() {
	cls.outProto.Reset()
	CL_InitSounds()
}

var (
	clSounds map[sfx]int
)

func CL_InitSounds() {
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
	if bobCycle == 0 {
		return 0
	}
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

	ent := c.Entities(c.viewentity)
	origin := ent.Origin
	from = from.Sub(origin).Normalize()
	angles := ent.Angles
	forward, right, _ := vec.AngleVectors(angles)
	c.dmgRoll = count * vec.Dot(from, right) * cvars.ViewKickRoll.Value()
	c.dmgPitch = count * vec.Dot(from, forward) * cvars.ViewKickPitch.Value()
	c.dmgTime = cvars.ViewKickTime.Value()
}

func (c *Client) calcViewRoll() {
	ent := c.Entities(c.viewentity)
	angles := ent.Angles
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
	ent := c.Entities(c.viewentity)

	// absolutely bound refresh relative to entity clipping hull
	// so the view can never be inside a solid wall
	o := ent.Origin
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
	w := c.WeaponEntity()
	w.Angles[YAW] = yaw
	w.Angles[PITCH] = -pitch
	w.Angles[ROLL] -= sway(cvars.ViewIRollCycle.Value(), cvars.ViewIRollLevel.Value())
	w.Angles[PITCH] -= sway(cvars.ViewIPitchCycle.Value(), cvars.ViewIPitchLevel.Value())
	w.Angles[YAW] -= sway(cvars.ViewIYawCycle.Value(), cvars.ViewIYawLevel.Value())
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
	ent := c.Entities(c.viewentity)
	// body
	qRefreshRect.viewOrg = ent.Origin
	qRefreshRect.viewAngles = ent.Angles
	// weaponmodel
	w := c.WeaponEntity()
	w.Model = nil

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
	case bsp.CONTENTS_EMPTY, bsp.CONTENTS_SOLID, bsp.CONTENTS_SKY:
		c.colorShifts[ColorShiftContents] = cshiftEmpty
	case bsp.CONTENTS_LAVA:
		c.colorShifts[ColorShiftContents] = cshiftLava
	case bsp.CONTENTS_SLIME:
		c.colorShifts[ColorShiftContents] = cshiftSlime
	default:
		c.colorShifts[ColorShiftContents] = cshiftWater
	}
}

func (c *Client) bonusFlash() {
	c.colorShifts[ColorShiftBonus] = intColor(215, 186, 69, 50)
}

func init() {
	Must(cmd.AddCommand("v_cshift", func(a []cmd.QArg, _ int) {
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
	}))
	Must(cmd.AddCommand("bf", func(_ []cmd.QArg, _ int) { cl.bonusFlash() }))
	Must(cmd.AddCommand("centerview", func(_ []cmd.QArg, _ int) { cl.startPitchDrift() }))
}

var (
	calcRefreshRectOldZ  = float32(0)
	calcRefreshRectPunch vec.Vec3
)

func (c *Client) calcRefreshRect() {
	c.driftPitch()

	// ent is the player model (visible when out of body)
	ent := c.Entity()
	// view is the weapon model (only visible from inside body)
	w := c.WeaponEntity() // view

	// transform the view offset by the model's matrix to get the offset from
	// model origin for the view
	ent.Angles[YAW] = c.yaw // the model should face the view dir
	// the model should face the view dir
	ent.Angles[PITCH] = -c.pitch
	ent.ptr.angles[YAW] = C.float(ent.Angles[YAW])
	ent.ptr.angles[PITCH] = C.float(ent.Angles[PITCH])

	bob := c.calcBob()

	// refresh position
	qRefreshRect.viewOrg = ent.Origin
	qRefreshRect.viewOrg[2] += c.viewHeight + bob

	// never let it sit exactly on a node line, because a water plane can
	// disappear when viewed with the eye exactly on it.
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
	angles := ent.Angles
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

	// set up gun position
	w.Angles[ROLL] = c.roll
	w.Angles[PITCH] = c.pitch
	w.Angles[YAW] = c.yaw

	c.calcWeaponAngle()
	w.Origin = ent.Origin
	w.Origin[2] += c.viewHeight

	w.Origin.Add(vec.Scale(bob*0.4, forward))
	w.Origin[2] += bob

	if c.stats.weapon != 0 {
		w.Model = c.modelPrecache[c.stats.weapon-1]
	} else {
		w.Model = nil
	}
	w.Frame = cl.stats.weaponFrame

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
	origin := ent.Origin
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
		w.Origin[2] += calcRefreshRectOldZ - origin[2]
	} else {
		calcRefreshRectOldZ = origin[2]
	}

	if cvars.ChaseActive.Bool() {
		Chase_UpdateForDrawing()
	}
}

// display impact point of trace along VPN
func tracePosition(args []cmd.QArg, _ int) {
	if cls.state != ca_connected {
		return
	}
	org := qRefreshRect.viewOrg
	vpn := qRefreshRect.viewForward
	v := vec.Add(org, vec.Scale(8192, vpn))
	w := trace{}
	recursiveHullCheck(&cl.worldModel.Hulls[0], 0, 0, 1, org, v, &w)

	if w.EndPos.Length() == 0 {
		conlog.Printf("Tracepos: trace didn't hit anything\n")
	} else {
		conlog.Printf("Tracepos: (%d %d %d)\n", w.EndPos[0], w.EndPos[1], w.EndPos[2])
	}
}

// Server information pertaining to this client only
func (c *Client) parseClientData(cdp *protos.ClientData) {
	vh := cdp.GetViewHeight()
	if vh != nil {
		c.viewHeight = float32(cdp.ViewHeight.Value)
	} else {
		c.viewHeight = svc.DEFAULT_VIEWHEIGHT
	}
	c.idealPitch = float32(cdp.IdealPitch)

	punchAngle := vec.Vec3{
		float32(cdp.PunchAngle.X),
		float32(cdp.PunchAngle.Y),
		float32(cdp.PunchAngle.Z),
	}
	if c.punchAngle[0] != punchAngle {
		c.punchAngle[1] = c.punchAngle[0]
		c.punchAngle[0] = punchAngle
	}
	c.mVelocity[1] = c.mVelocity[0]
	c.mVelocity[0] = vec.Vec3{
		float32(cdp.Velocity.X) * 16,
		float32(cdp.Velocity.Y) * 16,
		float32(cdp.Velocity.Z) * 16,
	}

	items := cdp.Items
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

	c.onGround = cdp.OnGround

	c.stats.weaponFrame = int(cdp.WeaponFrame)
	statusBarFlash := int(0)
	set := func(v *int, nv int32) {
		// skip some branching
		statusBarFlash |= *v ^ int(nv)
		*v = int(nv)
	}
	set(&c.stats.armor, cdp.Armor)
	set(&c.stats.weapon, cdp.Weapon)
	set(&c.stats.health, cdp.Health)
	set(&c.stats.ammo, cdp.Ammo)
	set(&c.stats.shells, cdp.Shells)
	set(&c.stats.nails, cdp.Nails)
	set(&c.stats.rockets, cdp.Rockets)
	set(&c.stats.cells, cdp.Cells)
	if statusBarFlash != 0 {
		statusbar.MarkChanged()
	}

	activeWeapon := cdp.ActiveWeapon
	if cmdl.Quoth() || cmdl.Rogue() || cmdl.Hipnotic() {
		//TODO(therjak): why is a command line setting responsible for how the server
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
	weaponE := c.WeaponEntity()
	weaponE.Alpha = byte(cdp.WeaponAlpha)
	// this was done before the upper 8 bits of cl.stats[STAT_WEAPON]
	// were filled in, breaking on large maps like zendar.bsp
	if weaponE.Model != c.modelPrecache[c.stats.weapon] {
		// don't lerp animation across model changes
		weaponE.LerpFlags |= lerpResetAnim
	}
}

func (c *ClientStatic) stopPlayback() {
	if !c.demoPlayback {
		return
	}

	c.demoData = []byte{}
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

func (c *ClientStatic) writeDemoMessage(data []byte) error {
	if err := binary.Write(c.demoWriter, binary.LittleEndian, int32(len(data))); err != nil {
		return err
	}
	if err := binary.Write(c.demoWriter, binary.LittleEndian, cl.pitch); err != nil {
		return err
	}
	if err := binary.Write(c.demoWriter, binary.LittleEndian, cl.yaw); err != nil {
		return err
	}
	if err := binary.Write(c.demoWriter, binary.LittleEndian, cl.roll); err != nil {
		return err
	}
	if err := binary.Write(c.demoWriter, binary.LittleEndian, data); err != nil {
		return err
	}
	return nil
}

func clientStartDemos(args []cmd.QArg, _ int) {
	if cmdl.Dedicated() {
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

func clientRecordDemo(args []cmd.QArg, playerEdictId int) {
	if !execute.IsSrcCommand() {
		return
	}
	if cls.demoPlayback {
		conlog.Printf("Can''t record during demo playback\n")
		return
	}

	cls.stopDemoRecording()

	switch len(args) {
	case 1, 2, 3:
		break
	default:
		conlog.Printf("record <demoname> [<map> [cd track]]\n")
		return
	}
	if strings.Contains(args[0].String(), "..") {
		conlog.Printf("Relavite pathnames are not allowed.\n")
		return
	}
	if len(args) == 1 && cls.state == ca_connected && cls.signon < 2 {
		conlog.Printf("Can't record - try again when connected\n")
		return
	}
	track := -1
	if len(args) == 3 {
		track = args[2].Int()
		conlog.Printf("Forcing CD track to %i\n", track)
	}
	if len(args) > 1 {
		execute.Execute(fmt.Sprintf("map %s", args[1].String()), execute.Command, playerEdictId)
		if cls.state != ca_connected {
			return
		}
	}
	err := cls.createDemoFile(args[0].String(), track)
	if err != nil {
		conlog.Printf(err.Error())
		return
	}
	if len(args) == 1 && cls.state == ca_connected {
		// initialize the demo file with a start connection dummy
		var buf bytes.Buffer

		for i := 0; i < cl.maxClients; i++ {
			s := &cl.scores[i]

			buf.WriteByte(svc.UpdateName)
			buf.WriteByte(byte(i))
			buf.WriteString(s.name)
			buf.WriteByte(0) // c-strings

			buf.WriteByte(svc.UpdateFrags)
			buf.WriteByte(byte(i))
			binary.Write(&buf, binary.LittleEndian, uint16(s.frags))

			buf.WriteByte(svc.UpdateColors)
			buf.WriteByte(byte(i))
			c := ((s.topColor & 0xf) << 4) + s.bottomColor&0xf
			buf.WriteByte(byte(c))
		}

		for i, ls := range lightStyles {
			buf.WriteByte(svc.LightStyle)
			buf.WriteByte(byte(i))
			buf.WriteString(ls.unprocessed)
			buf.WriteByte(0) // c-strings
		}

		buf.WriteByte(svc.UpdateStat)
		buf.WriteByte(stat.TotalSecrets)
		binary.Write(&buf, binary.LittleEndian, uint32(cl.stats.totalSecrets))

		buf.WriteByte(svc.UpdateStat)
		buf.WriteByte(stat.TotalMonsters)
		binary.Write(&buf, binary.LittleEndian, uint32(cl.stats.totalMonsters))

		buf.WriteByte(svc.UpdateStat)
		buf.WriteByte(stat.Secrets)
		binary.Write(&buf, binary.LittleEndian, uint32(cl.stats.secrets))

		buf.WriteByte(svc.UpdateStat)
		buf.WriteByte(stat.Monsters)
		binary.Write(&buf, binary.LittleEndian, uint32(cl.stats.monsters))

		buf.WriteByte(svc.SetView)
		binary.Write(&buf, binary.LittleEndian, uint16(cl.viewentity))

		buf.WriteByte(svc.SignonNum)
		buf.WriteByte(3)

		cls.writeDemoMessage(cls.demoSignon[0].Bytes())
		cls.writeDemoMessage(cls.demoSignon[1].Bytes())
		cls.writeDemoMessage(buf.Bytes())
	}
}

func clientStopDemoRecording(_ []cmd.QArg, _ int) {
	if !execute.IsSrcCommand() {
		return
	}
	if cls.demoWriter == nil {
		conlog.Printf("Not recording a demo.\n")
		return
	}
	cls.stopDemoRecording()
}

func clientPlayDemo(args []cmd.QArg, _ int) {
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

func clientTimeDemo(args []cmd.QArg, _ int) {
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

func (c *ClientStatic) stopDemoRecording() {
	if c.demoWriter == nil {
		return
	}

	c.writeDemoMessage([]byte{svc.Disconnect})
	c.demoWriter.Close()
	c.demoWriter = nil

	conlog.Printf("Completed demo\n")
}

func (c *ClientStatic) createDemoFile(filename string, cdtrack int) error {
	path := filepath.Join(gameDirectory, filename)
	if !strings.HasSuffix(filename, ".dem") {
		path += ".dem"
	}
	conlog.Printf("recording to %s\n", path)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("ERROR: couldn't create %s\n", path)
	}
	c.demoWriter = f
	fmt.Fprintf(c.demoWriter, "%d\n", cdtrack)
	return nil
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

const (
	mdlBolt1 = "progs/bolt.mdl"
	mdlBolt2 = "progs/bolt2.mdl"
	mdlBolt3 = "progs/bolt3.mdl"
	mdlBeam  = "progs/beam.mdl"
)

func v3FC(c *protos.Coord) vec.Vec3 {
	return vec.Vec3{c.GetX(), c.GetY(), c.GetZ()}
}

func (c *ClientStatic) parseTempEntity(tep *protos.TempEntity) {
	switch te := tep.Union.(type) {
	case *protos.TempEntity_Spike:
		// spike hitting wall
		pos := v3FC(te.Spike)
		particlesRunEffect(pos, vec.Vec3{}, 0, 10, float32(cl.time))
		s := func() sfx {
			if cRand.Uint32n(5) != 0 {
				return Tink1
			}
			switch cRand.Uint32n(4) {
			case 2:
				return Ric2
			default:
				return Ric1
			}
		}()
		snd.Start(-1, 0, clSounds[s], pos, 1, 1, !loopingSound)
	case *protos.TempEntity_SuperSpike:
		// spike hitting wall
		pos := v3FC(te.SuperSpike)
		particlesRunEffect(pos, vec.Vec3{}, 0, 20, float32(cl.time))
		s := func() sfx {
			if cRand.Uint32n(5) != 0 {
				return Tink1
			}
			switch cRand.Uint32n(4) {
			case 1:
				return Ric1
			case 2:
				return Ric2
			default:
				return Ric3
			}
		}()
		snd.Start(-1, 0, clSounds[s], pos, 1, 1, !loopingSound)
	case *protos.TempEntity_Gunshot:
		// bullet hitting wall
		pos := v3FC(te.Gunshot)
		particlesRunEffect(pos, vec.Vec3{}, 0, 20, float32(cl.time))
	case *protos.TempEntity_Explosion:
		// rocket explosion
		pos := v3FC(te.Explosion)
		particlesAddExplosion(pos, float32(cl.time))
		l := cl.GetFreeDynamicLight()
		*l = DynamicLight{
			ptr:     l.ptr,
			origin:  pos,
			radius:  350,
			dieTime: cl.time + 0.5,
			decay:   300,
			color:   vec.Vec3{1, 1, 1},
		}
		l.Sync()
		snd.Start(-1, 0, clSounds[RExp3], pos, 1, 1, !loopingSound)
	case *protos.TempEntity_TarExplosion:
		// tarbaby explosion
		pos := v3FC(te.TarExplosion)
		particlesAddBlobExplosion(pos, float32(cl.time))
		snd.Start(-1, 0, clSounds[RExp3], pos, 1, 1, !loopingSound)
	case *protos.TempEntity_Lightning1:
		// lightning bolts
		l := te.Lightning1
		s := v3FC(l.GetStart())
		e := v3FC(l.GetEnd())
		parseBeam(mdlBolt1, l.GetEntity(), s, e)
	case *protos.TempEntity_Lightning2:
		// lightning bolts
		l := te.Lightning2
		s := v3FC(l.GetStart())
		e := v3FC(l.GetEnd())
		parseBeam(mdlBolt2, l.GetEntity(), s, e)
	case *protos.TempEntity_WizSpike:
		// spike hitting wall
		pos := v3FC(te.WizSpike)
		particlesRunEffect(pos, vec.Vec3{}, 20, 30, float32(cl.time))
		snd.Start(-1, 0, clSounds[WizHit], pos, 1, 1, !loopingSound)
	case *protos.TempEntity_KnightSpike:
		// spike hitting wall
		pos := v3FC(te.KnightSpike)
		particlesRunEffect(pos, vec.Vec3{}, 226, 20, float32(cl.time))
		snd.Start(-1, 0, clSounds[KnightHit], pos, 1, 1, !loopingSound)
	case *protos.TempEntity_Lightning3:
		// lightning bolts
		l := te.Lightning3
		s := v3FC(l.GetStart())
		e := v3FC(l.GetEnd())
		parseBeam(mdlBolt3, l.GetEntity(), s, e)
	case *protos.TempEntity_LavaSplash:
		pos := v3FC(te.LavaSplash)
		particlesAddLavaSplash(pos, float32(cl.time))
	case *protos.TempEntity_Teleport:
		pos := v3FC(te.Teleport)
		particlesAddTeleportSplash(pos, float32(cl.time))
	case *protos.TempEntity_Explosion2:
		// color mapped explosion
		e := te.Explosion2
		pos := v3FC(e.GetPosition())
		particlesAddExplosion2(pos, int(e.GetStartColor()), int(e.GetStopColor()), float32(cl.time))
		l := cl.GetFreeDynamicLight()
		*l = DynamicLight{
			ptr:     l.ptr,
			origin:  pos,
			radius:  350,
			dieTime: cl.time + 0.5,
			decay:   300,
			color:   vec.Vec3{1, 1, 1},
		}
		l.Sync()
		snd.Start(-1, 0, clSounds[RExp3], pos, 1, 1, !loopingSound)
	case *protos.TempEntity_Beam:
		// grappling hook beam
		l := te.Beam
		s := v3FC(l.GetStart())
		e := v3FC(l.GetEnd())
		parseBeam(mdlBeam, l.GetEntity(), s, e)
	default:
		Error("CL_ParseTEnt: bad type")
	}
}

func (c *Client) ColorForEntity(e *Entity) vec.Vec3 {
	lightColor := c.worldModel.LightAt(e.Origin, &lightStyleValues)
	for i := range c.dynamicLights {
		d := &c.dynamicLights[i]
		if d.dieTime < c.time {
			continue
		}
		dist := vec.Sub(e.Origin, d.origin)
		add := d.radius - dist.Length()
		if add > 0 {
			lightColor.Add(vec.Scale(add, d.color))
		}
	}
	if e == c.WeaponEntity() {
		add := 72.0 - (lightColor[0] + lightColor[1] + lightColor[2])
		if add > 0 {
			add /= 3.0
			lightColor.Add(vec.Vec3{add, add, add})
		}
	}
	for i := 0; i < c.maxClients; i++ {
		// EntityIsPlayer
		if c.Entities(i+1) == e {
			add := 24.0 - (lightColor[0] + lightColor[1] + lightColor[2])
			if add > 0 {
				add /= 3.0
				lightColor.Add(vec.Vec3{add, add, add})
			}
			break
		}
	}
	if cvars.GlOverBrightModels.Bool() {
		add := 288.0 / (lightColor[0] + lightColor[1] + lightColor[2])
		if add < 1 {
			lightColor.Scale(add)
		}
	} else if cvars.GlFullBrights.Bool() {
		if e.Model.Flags()&mdl.FullBrightHack != 0 {
			return vec.Vec3{256, 256, 256}
		}
	}
	lightColor.Scale(1.0 / 200.0)
	return lightColor
}

func (c *Client) ClearState() {
	C.CL_ClearState()
	cls.signon = 0
	clientClear()
	cls.outProto.Reset()
	c.clearDLights()

	maxEdicts := math.ClampI(MIN_EDICTS, int(cvars.MaxEdicts.Value()), MAX_EDICTS)
	c.setMaxEdicts(maxEdicts)
}
