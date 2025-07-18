// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"goquake/bsp"
	"goquake/cbuf"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvar"
	"goquake/cvars"
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
	qsnd "goquake/snd"
	"goquake/stat"

	"github.com/chewxy/math32"
	"google.golang.org/protobuf/proto"
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
	addCommand("disconnect", func(args cbuf.Arguments) error {
		return clientDisconnect()
	})
	addCommand("reconnect", func(args cbuf.Arguments) error {
		clientReconnect()
		return nil
	})

	addCommand("startdemos", clientStartDemos)
	addCommand("record", clientRecordDemo)
	addCommand("stop", clientStopDemoRecording)
	addCommand("playdemo", clientPlayDemo)
	addCommand("timedemo", clientTimeDemo)

	addCommand("tracepos", tracePosition)
	//cmd.AddCommand("mcache", Mod_Print);
	updateExtraFlags := func() {
		for _, m := range cl.modelPrecache {
			setExtraFlags(m)
		}
	}
	cvars.RNoLerpList.SetCallback(func(cv *cvar.Cvar) {
		updateExtraFlags()
	})
	cvars.RNoShadowList.SetCallback(func(cv *cvar.Cvar) {
		updateExtraFlags()
	})
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
			return float32(host.FrameTime()) * cvars.ClientAngleSpeedKey.Value()
		}
		return float32(host.FrameTime())
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

	c.pitch = math.Clamp(cvars.ClientMinPitch.Value(), c.pitch, cvars.ClientMaxPitch.Value())
	c.roll = math.Clamp(-50, c.roll, 50)
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
	outProto           *protos.ClientMessage
	timeDemoLastFrame  int
	timeDemoStartFrame int
	timeDemoStartTime  float64
	demos              []string
	demoWriter         io.WriteCloser
	demoData           []byte
	msgBadRead         bool
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
	sound *qsnd.SoundPrecache

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

func (c *Client) setMaxEdicts(num int) error {
	cl.entities = make([]*Entity, 0, num)
	// ensure at least a world entity at the start
	_, err := cl.GetOrCreateEntity(0)
	return err
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

func viewPositionCommand(args cbuf.Arguments) error {
	if cls.state != ca_connected {
		return nil
	}
	printPosition()
	return nil
}

func printPosition() {
	player := cl.Entities(cl.viewentity)
	pos := player.Origin
	conlog.Printf("Viewpos: (%.f %.f %.f) %.f %.f %.f\n",
		pos[0], pos[1], pos[2],
		cl.pitch, cl.yaw, cl.roll)
}

func executeOnServer(a cbuf.Arguments) error {
	if cls.state != ca_connected {
		conlog.Printf("Can't \"cmd\", not connected\n")
		return nil
	}
	if cls.demoPlayback {
		return nil
	}
	args := a.Args()
	if len(args) > 1 {
		cls.outProto.SetCmds(append(cls.outProto.GetCmds(), protos.Cmd_builder{
			StringCmd: proto.String(a.ArgumentString()),
		}.Build()))
	}
	return nil
}

func forwardToServer(a cbuf.Arguments) {
	args := a.Args()
	if cls.state != ca_connected {
		conlog.Printf("Can't \"%s\", not connected\n", args[0])
		return
	}
	if cls.demoPlayback {
		return
	}
	cls.outProto.SetCmds(append(cls.outProto.GetCmds(), protos.Cmd_builder{
		StringCmd: proto.String(a.Full()),
	}.Build()))
}

func init() {
	addCommand("cmd", executeOnServer)
	addCommand("viewpos", viewPositionCommand)
}

// Read all incoming data from the server
func (c *Client) ReadFromServer() (serverState, error) {
	c.oldTime = cl.time
	c.time += host.FrameTime()
	for {
		// TODO: code needs major cleanup (getMessage + CL_ParseServerMessage)
		ret := cls.getMessage()
		if ret == -1 {
			return serverDisconnected, fmt.Errorf("CL_ReadFromServer: lost server connection")
		}
		if ret == 0 {
			break
		}
		c.lastReceivedMessageTime = host.Time()
		pb, err := svc.ParseServerMessage(cls.inMessage, c.protocol, c.protocolFlags)
		if err != nil {
			log.Printf("Bad server message\n %v", err)
			return serverDisconnected, fmt.Errorf("CL_ParseServerMessage: Bad server message")
		}
		if serverState, err := CL_ParseServerMessage(pb); err != nil {
			return serverDisconnected, err
		} else if serverState == serverDisconnected {
			return serverDisconnected, nil
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
	return serverRunning, nil
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

// Sends a disconnect message to the server
// This is also called on Host_Error, so it shouldn't cause any errors
func (c *ClientStatic) Disconnect() error {
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

		slog.Debug("Sending clc_disconnect")

		cls.outProto.SetCmds(append(cls.outProto.GetCmds()[:0], protos.Cmd_builder{
			Disconnect: proto.Bool(true),
		}.Build()))
		b, err := clc.ToBytes(cls.outProto, cl.protocol, cl.protocolFlags)
		if err != nil {
			return err
		}
		c.connection.SendUnreliableMessage(b)
		cls.outProto.Reset()
		cls.connection.Close()

		c.state = ca_disconnected
		if ServerActive() {
			if err := hostShutdownServer(false); err != nil {
				return err
			}
		}
	}

	c.demoPlayback = false
	c.timeDemo = false
	c.demoPaused = false
	c.signon = 0
	cl.intermission = 0
	return nil
}

func clientDisconnect() error {
	if err := cls.Disconnect(); err != nil {
		return err
	}
	if ServerActive() {
		if err := hostShutdownServer(false); err != nil {
			return err
		}
	}
	return nil
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
func clEstablishConnection(host string) error {
	if cmdl.Dedicated() {
		return nil
	}

	if cls.demoPlayback {
		return nil
	}

	if err := cls.Disconnect(); err != nil {
		return err
	}

	c, err := net.Connect(host)
	if err != nil {
		// TODO: this is bad, looks like orig just quits this call without returning
		// and waits for the next sdl input.
		cls.connection = nil
		return fmt.Errorf("CLS_Connect: connect failed\n")
	}
	cls.connection = c
	slog.Debug("CL_EstablishConnectionconnected", slog.String("Host", host))

	// not in the demo loop now
	cls.demoNum = -1
	cls.state = ca_connected
	// need all the signon messages before playing
	cls.signon = 0
	cls.outProto.SetCmds(append(cls.outProto.GetCmds(), &protos.Cmd{}))
	return nil
}

// An svc_signonnum has been received, perform a client side setup
func CL_SignonReply() {
	slog.Debug("CL_SignonReply", slog.Int("signon", cls.signon))

	switch cls.signon {
	case 1:
		cls.outProto.SetCmds(append(cls.outProto.GetCmds(), protos.Cmd_builder{
			StringCmd: proto.String("prespawn"),
		}.Build()))

	case 2:
		color := int(cvars.ClientColor.Value())
		cls.outProto.SetCmds(append(cls.outProto.GetCmds(),
			protos.Cmd_builder{
				StringCmd: proto.String(fmt.Sprintf("name \"%s\"", cvars.ClientName.String())),
			}.Build(),
			protos.Cmd_builder{
				StringCmd: proto.String(fmt.Sprintf("color %d %d", color>>4, color&15)),
			}.Build(),
			protos.Cmd_builder{
				StringCmd: proto.String("spawn"),
			}.Build(),
		))

	case 3:
		cls.outProto.SetCmds(append(cls.outProto.GetCmds(), protos.Cmd_builder{
			StringCmd: proto.String("begin"),
		}.Build()))

	case 4:
		screen.EndLoadingPlaque() // allow normal screen updates
	}
}

func CL_SendCmd() error {
	if cls.state != ca_connected {
		return nil
	}

	if cls.signon == numSignonMessagesBeforeConn {
		cl.adjustAngles()
		if err := HandleMove(); err != nil {
			return err
		}
	}

	if cls.demoPlayback {
		cls.outProto.Reset()
		return nil
	}

	if len(cls.outProto.GetCmds()) == 0 {
		return nil // no message at all
	}

	if !cls.connection.CanSendMessage() {
		slog.Debug("CL_SendCmd: can't send")
		return nil
	}

	b, err := clc.ToBytes(cls.outProto, cl.protocol, cl.protocolFlags)
	if err != nil {
		return err
	}
	i := cls.connection.SendMessage(b)
	if i == -1 {
		return fmt.Errorf("CL_SendCmd: lost server connection")
	}
	cls.outProto.Reset()
	return nil
}

func CL_ParseStartSoundPacket(m *protos.Sound) error {
	const (
		maxSounds = 2048
	)
	if m.GetSoundNum() > maxSounds {
		return fmt.Errorf("CL_ParseStartSoundPacket: %d > MAX_SOUNDS", m.GetSoundNum())
	}
	if m.GetEntity() > int32(cap(cl.entities)) {
		return fmt.Errorf("CL_ParseStartSoundPacket: ent = %d", m.GetEntity())
	}
	volume := float32(1.0)
	if m.HasVolume() {
		volume = float32(m.GetVolume()) / 255
	}
	attenuation := float32(1.0)
	if m.HasAttenuation() {
		attenuation = float32(m.GetAttenuation()) / 64.0
	}
	origin := vec.Vec3{m.GetOrigin().GetX(), m.GetOrigin().GetY(), m.GetOrigin().GetZ()}
	cl.sound.Start(int(m.GetEntity()), int(m.GetChannel()), int(m.GetSoundNum()), origin, volume, attenuation)
	return nil
}

var (
	clientKeepAliveTime time.Time
)

// When the client is taking a long time to load stuff, send keepalive messages
// so the server doesn't disconnect.
func CL_KeepaliveMessage() error {
	if ServerActive() {
		// no need if server is local
		return nil
	}
	if cls.demoPlayback {
		return nil
	}

	msgBackup := cls.inMessage

	// read messages from server, should just be noops
Outer:
	for {
		switch ret := cls.getMessage(); ret {
		default:
			return fmt.Errorf("CL_KeepaliveMessage: CL_GetMessage failed")
		case 0:
			break Outer
		case 1:
			return fmt.Errorf("CL_KeepaliveMessage: received a message")
		case 2:
			conlog.Printf("WTF? This should never happen")
			i, err := cls.inMessage.ReadByte()
			if err != nil || i != svc.Nop {
				return fmt.Errorf("CL_KeepaliveMessage: datagram wasn't a nop")
			}
		}
	}

	cls.inMessage = msgBackup

	// check time
	curTime := time.Now()
	if curTime.Sub(clientKeepAliveTime) < time.Second*5 {
		return nil
	}
	if !cls.connection.CanSendMessage() {
		return nil
	}
	clientKeepAliveTime = curTime

	// write out a nop
	conlog.Printf("--> client to server keepalive\n")

	cls.outProto.SetCmds(append(cls.outProto.GetCmds(), &protos.Cmd{}))
	b, err := clc.ToBytes(cls.outProto, cl.protocol, cl.protocolFlags)
	if err != nil {
		return err
	}
	cls.connection.SendMessage(b)
	cls.outProto.Reset()
	return nil
}

func clientInit() {
	cls.outProto = &protos.ClientMessage{}
}

// Determines the fraction between the last two messages that the objects
// should be put at.
func (c *Client) LerpPoint() float64 {
	f := c.messageTime - c.messageTimeOld

	if f == 0 || cls.timeDemo || ServerActive() {
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
			cl.driftMove += float32(host.FrameTime())
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

	move := float32(host.FrameTime()) * cl.pitchVel
	cl.pitchVel += float32(host.FrameTime()) * cvars.ViewCenterSpeed.Value()

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
	color.A = math.Clamp(0, color.A, 1)
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
	ft := float32(host.FrameTime())
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
	cs.A = math.Clamp(0, cs.A, 150/255.0)
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
	side := cvars.CalcRoll(angles, c.velocity)
	qRefreshRect.viewAngles[ROLL] += side

	if c.dmgTime > 0 {
		kt := cvars.ViewKickTime.Value()
		qRefreshRect.viewAngles[ROLL] += c.dmgTime / kt * c.dmgRoll
		qRefreshRect.viewAngles[PITCH] += c.dmgTime / kt * c.dmgPitch
		c.dmgTime -= float32(host.FrameTime())
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
	qRefreshRect.viewOrg[0] = math.Clamp(o[0]-14, qRefreshRect.viewOrg[0], o[0]+14)
	qRefreshRect.viewOrg[1] = math.Clamp(o[1]-14, qRefreshRect.viewOrg[1], o[1]+14)
	qRefreshRect.viewOrg[2] = math.Clamp(o[2]-22, qRefreshRect.viewOrg[2], o[2]+30)
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
	addCommand("v_cshift", func(arg cbuf.Arguments) error {
		a := arg.Args()[1:]
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
		return nil
	})
	addCommand("bf", func(a cbuf.Arguments) error {
		cl.bonusFlash()
		return nil
	})
	addCommand("centerview", func(a cbuf.Arguments) error {
		cl.startPitchDrift()
		return nil
	})
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
				delta := (c.punchAngle[0][i] - c.punchAngle[1][i]) * float32(host.FrameTime()) * 10
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
func tracePosition(args cbuf.Arguments) error {
	if cls.state != ca_connected {
		return nil
	}
	org := qRefreshRect.viewOrg
	vpn := qRefreshRect.viewForward
	v := vec.Add(org, vec.Scale(8192, vpn))
	w := bsp.Trace{}
	cl.worldModel.Hulls[0].RecursiveCheck(0, 0, 1, org, v, &w)

	if w.EndPos.Length() == 0 {
		conlog.Printf("Tracepos: trace didn't hit anything\n")
	} else {
		conlog.Printf("Tracepos: (%d %d %d)\n", w.EndPos[0], w.EndPos[1], w.EndPos[2])
	}
	return nil
}

// Server information pertaining to this client only
func (c *Client) parseClientData(cdp *protos.ClientData) {
	if cdp.HasViewHeight() {
		c.viewHeight = float32(cdp.GetViewHeight())
	} else {
		c.viewHeight = svc.DEFAULT_VIEWHEIGHT
	}
	c.idealPitch = float32(cdp.GetIdealPitch())

	punchAngle := vec.Vec3{
		float32(cdp.GetPunchAngle().GetX()),
		float32(cdp.GetPunchAngle().GetY()),
		float32(cdp.GetPunchAngle().GetZ()),
	}
	if c.punchAngle[0] != punchAngle {
		c.punchAngle[1] = c.punchAngle[0]
		c.punchAngle[0] = punchAngle
	}
	c.mVelocity[1] = c.mVelocity[0]
	c.mVelocity[0] = vec.Vec3{
		float32(cdp.GetVelocity().GetX()) * 16,
		float32(cdp.GetVelocity().GetY()) * 16,
		float32(cdp.GetVelocity().GetZ()) * 16,
	}

	items := cdp.GetItems()
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

	c.onGround = cdp.GetOnGround()

	c.stats.weaponFrame = int(cdp.GetWeaponFrame())
	statusBarFlash := int(0)
	set := func(v *int, nv int32) {
		// skip some branching
		statusBarFlash |= *v ^ int(nv)
		*v = int(nv)
	}
	set(&c.stats.armor, cdp.GetArmor())
	set(&c.stats.weapon, cdp.GetWeapon())
	set(&c.stats.health, cdp.GetHealth())
	set(&c.stats.ammo, cdp.GetAmmo())
	set(&c.stats.shells, cdp.GetShells())
	set(&c.stats.nails, cdp.GetNails())
	set(&c.stats.rockets, cdp.GetRockets())
	set(&c.stats.cells, cdp.GetCells())
	if statusBarFlash != 0 {
		statusbar.MarkChanged()
	}

	activeWeapon := cdp.GetActiveWeapon()
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
	weaponE.Alpha = byte(cdp.GetWeaponAlpha())
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
	frames := host.FrameCount() - c.timeDemoStartFrame - 1
	time := host.Time() - float64(c.timeDemoStartTime)
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

func clientStartDemos(a cbuf.Arguments) error {
	if cmdl.Dedicated() {
		return nil
	}
	args := a.Args()[1:]

	cls.demos = cls.demos[:0]
	for _, a := range args {
		cls.demos = append(cls.demos, a.String())
	}
	conlog.Printf("%d demo(s) in loop\n", len(cls.demos))

	if !ServerActive() && cls.demoNum != -1 && !cls.demoPlayback {
		cls.demoNum = 0
		if !cmdl.Fitz() { // QuakeSpasm customization:
			// go straight to menu, no CL_NextDemo
			cls.demoNum = -1
			cbuf.InsertText("menu_main\n")
			return nil
		}
		if err := CL_NextDemo(); err != nil {
			fmt.Printf("CL_NextDemo: %v", err)
		}

	} else {
		cls.demoNum = -1
	}
	return nil
}

func clientRecordDemo(a cbuf.Arguments) error {
	if cls.demoPlayback {
		conlog.Printf("Can''t record during demo playback\n")
		return nil
	}

	cls.stopDemoRecording()
	args := a.Args()[1:]

	switch len(args) {
	case 1, 2, 3:
		break
	default:
		conlog.Printf("record <demoname> [<map> [cd track]]\n")
		return nil
	}
	if strings.Contains(args[0].String(), "..") {
		conlog.Printf("Relavite pathnames are not allowed.\n")
		return nil
	}
	if len(args) == 1 && cls.state == ca_connected && cls.signon < 2 {
		conlog.Printf("Can't record - try again when connected\n")
		return nil
	}
	track := -1
	if len(args) == 3 {
		track = args[2].Int()
		conlog.Printf("Forcing CD track to %i\n", track)
	}
	if len(args) > 1 {
		// THERJAK: this should be the same as
		// if err := hostMap(cmd.Parse(fmt.Sprintf(....)); ...
		if err := cbuf.ExecuteCommand(
			fmt.Sprintf("map %s", args[1].String())); err != nil {
			return err
		}
		if cls.state != ca_connected {
			return nil
		}
	}
	err := cls.createDemoFile(args[0].String(), track)
	if err != nil {
		conlog.Printf(err.Error())
		return nil
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
			if err := binary.Write(&buf, binary.LittleEndian, uint16(s.frags)); err != nil {
				return fmt.Errorf("Could not write demo: %w", err)
			}

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
		if err := binary.Write(&buf, binary.LittleEndian, uint32(cl.stats.totalSecrets)); err != nil {
			return fmt.Errorf("Could not write demo: %w", err)
		}

		buf.WriteByte(svc.UpdateStat)
		buf.WriteByte(stat.TotalMonsters)
		if err := binary.Write(&buf, binary.LittleEndian, uint32(cl.stats.totalMonsters)); err != nil {
			return fmt.Errorf("Could not write demo: %w", err)
		}

		buf.WriteByte(svc.UpdateStat)
		buf.WriteByte(stat.Secrets)
		if err := binary.Write(&buf, binary.LittleEndian, uint32(cl.stats.secrets)); err != nil {
			return fmt.Errorf("Could not write demo: %w", err)
		}

		buf.WriteByte(svc.UpdateStat)
		buf.WriteByte(stat.Monsters)
		if err := binary.Write(&buf, binary.LittleEndian, uint32(cl.stats.monsters)); err != nil {
			return fmt.Errorf("Could not write demo: %w", err)
		}

		buf.WriteByte(svc.SetView)
		if err := binary.Write(&buf, binary.LittleEndian, uint16(cl.viewentity)); err != nil {
			return fmt.Errorf("Could not write demo: %w", err)
		}

		buf.WriteByte(svc.SignonNum)
		buf.WriteByte(3)

		if err := cls.writeDemoMessage(cls.demoSignon[0].Bytes()); err != nil {
			return fmt.Errorf("Could not write demo: %w", err)
		}
		if err := cls.writeDemoMessage(cls.demoSignon[1].Bytes()); err != nil {
			return fmt.Errorf("Could not write demo: %w", err)
		}
		if err := cls.writeDemoMessage(buf.Bytes()); err != nil {
			return fmt.Errorf("Could not write demo: %w", err)
		}
	}
	return nil
}

func clientStopDemoRecording(a cbuf.Arguments) error {
	if cls.demoWriter == nil {
		conlog.Printf("Not recording a demo.\n")
		return nil
	}
	cls.stopDemoRecording()
	return nil
}

func clientPlayDemo(a cbuf.Arguments) error {
	args := a.Args()[1:]
	if len(args) != 1 {
		conlog.Printf("playdemo <demoname> : plays a demo\n")
		return nil
	}

	if err := cls.playDemo(args[0].String()); err != nil {
		conlog.Printf("Error: %v", err)
	}
	return nil
}

func clientTimeDemo(a cbuf.Arguments) error {
	args := a.Args()[1:]
	if len(args) != 1 {
		conlog.Printf("timedemo <demoname> : gets demo speeds\n")
		return nil
	}

	cls.startTimeDemo(args[0].String())
	return nil
}

// Called to play the next demo in the demo loop
func CL_NextDemo() error {
	if cls.demoNum == -1 {
		// don't play demos
		return nil
	}

	if len(cls.demos) == 0 {
		conlog.Printf("No demos listed with startdemos\n")
		cls.demoNum = -1
		if err := cls.Disconnect(); err != nil {
			return err
		}
		return nil
	}

	// TODO(therjak): Can this be integrated into CLS_NextDemoInCycle?
	if cls.demoNum == len(cls.demos) {
		cls.demoNum = 0
	}

	screen.BeginLoadingPlaque()

	cbuf.InsertText(fmt.Sprintf("playdemo %s\n", cls.demos[cls.demoNum]))
	cls.demoNum++
	return nil
}

func (c *ClientStatic) stopDemoRecording() {
	if c.demoWriter == nil {
		return
	}

	if err := c.writeDemoMessage([]byte{svc.Disconnect}); err != nil {
		conlog.Printf("Failed to finish demo: %v", err)
	}
	if err := c.demoWriter.Close(); err != nil {
		conlog.Printf("Failed to finish demo: %v", err)
	}
	c.demoWriter = nil

	conlog.Printf("Completed demo\n")
}

func (c *ClientStatic) createDemoFile(filename string, cdtrack int) error {
	path := filepath.Join(filesystem.GameDir(), filename)
	if !strings.HasSuffix(filename, ".dem") {
		path += ".dem"
	}
	conlog.Printf("recording to %s\n", path)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("ERROR: couldn't create %s\n", path)
	}
	c.demoWriter = f
	_, err = fmt.Fprintf(c.demoWriter, "%d\n", cdtrack)
	return err
}

func (c *ClientStatic) getDemoMessage() int {
	if c.demoPaused {
		return 0
	}

	// decide if it is time to grab the next message
	if c.signon == 4 /*SIGNONS*/ {
		// always grab until fully connected
		if c.timeDemo {
			if host.FrameCount() == c.timeDemoLastFrame {
				// already read this frame's message
				return 0
			}
			c.timeDemoLastFrame = host.FrameCount()
			// if this is the second frame, grab the real timeDemoStartTime
			// so the bogus time on the first frame doesn't count
			if host.FrameCount() == c.timeDemoStartFrame+1 {
				c.timeDemoStartTime = host.Time()
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
	if err := c.Disconnect(); err != nil {
		return err
	}
	if !strings.HasSuffix(name, ".dem") {
		name += ".dem"
	}
	b, err := filesystem.ReadFile(name)
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
	c.timeDemoStartFrame = host.FrameCount()
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

func (c *ClientStatic) parseTempEntity(tep *protos.TempEntity) error {
	switch tep.WhichUnion() {
	case protos.TempEntity_Spike_case:
		// spike hitting wall
		pos := v3FC(tep.GetSpike())
		particlesRunEffect(pos, vec.Vec3{}, 0, 10, float32(cl.time))
		s := func() lSound {
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
		clientSound(s, pos)
	case protos.TempEntity_SuperSpike_case:
		// spike hitting wall
		pos := v3FC(tep.GetSuperSpike())
		particlesRunEffect(pos, vec.Vec3{}, 0, 20, float32(cl.time))
		s := func() lSound {
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
		clientSound(s, pos)
	case protos.TempEntity_Gunshot_case:
		// bullet hitting wall
		pos := v3FC(tep.GetGunshot())
		particlesRunEffect(pos, vec.Vec3{}, 0, 20, float32(cl.time))
	case protos.TempEntity_Explosion_case:
		// rocket explosion
		pos := v3FC(tep.GetExplosion())
		particlesAddExplosion(pos, float32(cl.time))
		l := cl.GetFreeDynamicLight()
		*l = DynamicLight{
			origin:  pos,
			radius:  350,
			dieTime: cl.time + 0.5,
			decay:   300,
			color:   vec.Vec3{1, 1, 1},
		}
		clientSound(RExp3, pos)
	case protos.TempEntity_TarExplosion_case:
		// tarbaby explosion
		pos := v3FC(tep.GetTarExplosion())
		particlesAddBlobExplosion(pos, float32(cl.time))
		clientSound(RExp3, pos)
	case protos.TempEntity_Lightning1_case:
		// lightning bolts
		l := tep.GetLightning1()
		s := v3FC(l.GetStart())
		e := v3FC(l.GetEnd())
		parseBeam(mdlBolt1, l.GetEntity(), s, e)
	case protos.TempEntity_Lightning2_case:
		// lightning bolts
		l := tep.GetLightning2()
		s := v3FC(l.GetStart())
		e := v3FC(l.GetEnd())
		parseBeam(mdlBolt2, l.GetEntity(), s, e)
	case protos.TempEntity_WizSpike_case:
		// spike hitting wall
		pos := v3FC(tep.GetWizSpike())
		particlesRunEffect(pos, vec.Vec3{}, 20, 30, float32(cl.time))
		clientSound(WizHit, pos)
	case protos.TempEntity_KnightSpike_case:
		// spike hitting wall
		pos := v3FC(tep.GetKnightSpike())
		particlesRunEffect(pos, vec.Vec3{}, 226, 20, float32(cl.time))
		clientSound(KnightHit, pos)
	case protos.TempEntity_Lightning3_case:
		// lightning bolts
		l := tep.GetLightning3()
		s := v3FC(l.GetStart())
		e := v3FC(l.GetEnd())
		parseBeam(mdlBolt3, l.GetEntity(), s, e)
	case protos.TempEntity_LavaSplash_case:
		pos := v3FC(tep.GetLavaSplash())
		particlesAddLavaSplash(pos, float32(cl.time))
	case protos.TempEntity_Teleport_case:
		pos := v3FC(tep.GetTeleport())
		particlesAddTeleportSplash(pos, float32(cl.time))
	case protos.TempEntity_Explosion2_case:
		// color mapped explosion
		e := tep.GetExplosion2()
		pos := v3FC(e.GetPosition())
		particlesAddExplosion2(pos, int(e.GetStartColor()), int(e.GetStopColor()), float32(cl.time))
		l := cl.GetFreeDynamicLight()
		*l = DynamicLight{
			origin:  pos,
			radius:  350,
			dieTime: cl.time + 0.5,
			decay:   300,
			color:   vec.Vec3{1, 1, 1},
		}
		clientSound(RExp3, pos)
	case protos.TempEntity_Beam_case:
		// grappling hook beam
		l := tep.GetBeam()
		s := v3FC(l.GetStart())
		e := v3FC(l.GetEnd())
		parseBeam(mdlBeam, l.GetEntity(), s, e)
	default:
		return fmt.Errorf("CL_ParseTEnt: bad type")
	}
	return nil
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

func (c *Client) ClearState() error {
	cls.signon = 0
	clientClear()
	cls.outProto.Reset()
	c.clearDLights()

	maxEdicts := int(cvars.MaxEdicts.Value())
	return c.setMaxEdicts(maxEdicts)
}
