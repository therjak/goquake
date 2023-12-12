// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"bytes"
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"time"
	"unicode"

	"goquake/cmd"
	qcmd "goquake/cmd"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvars"
	"goquake/net"
	"goquake/progs"
	"goquake/protocol"
	clc "goquake/protocol/client"
	svc "goquake/protocol/server"
	"goquake/protos"
	"goquake/version"
)

func qFormatI(b int32) string {
	if b == 0 {
		return "OFF"
	}
	return "ON"
}

type movecmd struct {
	forwardmove float32
	sidemove    float32
	upmove      float32
}

type SVClient struct {
	active     bool // false = client is free
	spawned    bool // false = don't send datagrams
	sendSignon bool // only valid before spawned
	admin      bool

	// reliable messages must be sent periodically
	lastMessage float64

	netConnection *net.Connection // communications handle

	cmd movecmd // movement

	// can be added to at any time, copied and clear once per frame
	//  had max length of 64000
	msg     net.Message
	edictId int    // == clientnum + 1
	name    string //[32];  // for printing to other people
	colors  int

	pingTimes [16]float32
	numPings  int // ping_times[num_pings%NUM_PING_TIMES]

	// spawn params are carried from level to level
	spawnParams [16]float32

	// client known data for deltas
	oldFrags int
	id       int // Needed to communicate with the 'client' side

	badRead bool
}

var (
	sv_clients []*SVClient
)

func CreateSVClients() {
	sv_clients = make([]*SVClient, svs.maxClientsLimit)
	for i := range sv_clients {
		sv_clients[i] = &SVClient{
			id: i,
		}
	}
}

func SV_BroadcastPrintf(format string, v ...interface{}) {
	SV_BroadcastPrint(fmt.Sprintf(format, v...))
}

func SV_BroadcastPrint(m string) {
	for _, c := range sv_clients {
		if c.active && c.spawned {
			c.Printf(m)
		}
	}
}

// TODO: HostClient/host_client should get removed. It is only used in hostcmd.go
// and the playerEdictId should be sufficient.
func HostClient() *SVClient {
	return sv_clients[host_client]
}

func (c *SVClient) Printf(format string, v ...interface{}) {
	c.print(fmt.Sprintf(format, v...))
}

func (c *SVClient) print(msg string) {
	c.msg.WriteByte(svc.Print)
	c.msg.WriteString(msg)
}

func (c *SVClient) ClientCommands(msg string) {
	c.msg.WriteByte(svc.StuffText)
	c.msg.WriteString(msg)
}

func (c *SVClient) PingTime() float32 {
	r := float32(0)
	for _, p := range c.pingTimes {
		r += p
	}
	return r / float32(len(c.pingTimes))
}

func CheckForNewClients() error {
	for {
		con := net.CheckNewConnections()
		if con == nil {
			return nil
		}
		foundFree := false
		for _, c := range sv_clients {
			if c.active {
				continue
			}
			foundFree = true
			c.netConnection = con
			c.admin = con.Address() == net.LocalAddress
			if err := ConnectClient(c.id); err != nil {
				return err
			}
			break
		}
		if !foundFree {
			debug.PrintStack()
			log.Fatalf("Host_CheckForNewClients: no free clients")
		}
	}
}

func (cl *SVClient) CanSendMessage() bool {
	return cl.netConnection.CanSendMessage()
}

func (cl *SVClient) Close() {
	cl.netConnection.Close()
	cl.netConnection = nil
	cl.admin = false
	cl.active = false
	cl.name = ""
	cl.oldFrags = -999999
}

func (cl *SVClient) ConnectTime() time.Time {
	return cl.netConnection.ConnectTime()
}

func (cl *SVClient) Address() string {
	return cl.netConnection.Address()
}

func (cl *SVClient) SendMessage() int {
	return cl.netConnection.SendMessage(cl.msg.Bytes())
}

func (cl *SVClient) SendNop() error {
	if cl.netConnection.SendUnreliableMessage([]byte{svc.Nop}) == -1 {
		if err := cl.Drop(true); err != nil {
			return err
		}
	}
	cl.lastMessage = host.Time()
	return nil
}

func (cl *SVClient) Drop(crash bool) error {
	if !crash {
		// send any final messages (don't check for errors)
		if cl.CanSendMessage() {
			cl.msg.WriteByte(svc.Disconnect)
			cl.SendMessage()
		}

		if cl.spawned {
			// call the prog function for removing a client
			// this will set the body to a dead frame, among other things
			saveSelf := progsdat.Globals.Self
			progsdat.Globals.Self = int32(cl.edictId)
			if err := vm.ExecuteProgram(progsdat.Globals.ClientDisconnect); err != nil {
				return err
			}
			progsdat.Globals.Self = saveSelf
		}
		log.Printf("Client %s removed", cl.name)
	}

	// break the net connection
	cl.Close()

	// send notification to all clients
	for _, c := range sv_clients {
		if !c.active {
			continue
		}
		c.msg.WriteByte(svc.UpdateName)
		c.msg.WriteByte(cl.id)
		c.msg.WriteString("")
		c.msg.WriteByte(svc.UpdateFrags)
		c.msg.WriteByte(cl.id)
		c.msg.WriteShort(0)
		c.msg.WriteByte(svc.UpdateColors)
		c.msg.WriteByte(cl.id)
		c.msg.WriteByte(0)
	}
	return nil
}

func SendReconnectToAll() {
	s := "reconnect\n\x00"
	m := make([]byte, 0, len(s)+1)
	buf := bytes.NewBuffer(m)
	buf.WriteByte(svc.StuffText)
	buf.WriteString(s)
	SendToAll(buf.Bytes())
}

func SendToAll(data []byte) {
	// We try for 5 seconds to send the message to everyone
	s := make([]bool, len(sv_clients))
	start := time.Now()
TimeoutLoop:
	for {
		if time.Since(start) > 5*time.Second {
			return
		}
		for i, c := range sv_clients {
			if s[i] {
				continue
			}
			if !c.active {
				s[i] = true
				continue
			}
			if c.CanSendMessage() {
				c.netConnection.SendMessage(data)
				s[i] = true
			}
		}
		for _, c := range s {
			if !c {
				// There is no need to spin too fast, we are waiting for
				// the last ACK of one of the clients.
				time.Sleep(time.Millisecond)
				continue TimeoutLoop
			}
		}
		return
	}
}

func (cl *SVClient) GetMessage() error {
	_, err := cl.netConnection.GetMessage()
	return err
}

func (c *SVClient) SendServerinfo() {
	m := &c.msg
	m.WriteByte(svc.Print)
	m.WriteString(
		fmt.Sprintf("%s\nGOQUAKE %1.2f SERVER (%d CRC)\n",
			[]byte{2}, version.Base, progsdat.CRC))

	m.WriteByte(int(svc.ServerInfo))
	m.WriteLong(int(sv.protocol))

	if sv.protocol == protocol.RMQ {
		m.WriteLong(int(sv.protocolFlags))
	}

	c.msg.WriteByte(svs.maxClients)

	if !cvars.Coop.Bool() && cvars.DeathMatch.Bool() {
		m.WriteByte(svc.GameDeathmatch)
	} else {
		m.WriteByte(svc.GameCoop)
	}

	s, err := progsdat.String(entvars.Get(0).Message)
	if err != nil {
		s = ""
	}
	m.WriteString(s)

	for i, mn := range sv.modelPrecache[1:] {
		if sv.protocol == protocol.NetQuake && i >= 256 {
			break
		}
		m.WriteString(mn)
	}
	m.WriteByte(0)

	for i, sn := range sv.soundPrecache[1:] {
		if sv.protocol == protocol.NetQuake && i >= 256 {
			break
		}
		m.WriteString(sn)
	}
	m.WriteByte(0)

	m.WriteByte(svc.CDTrack)
	m.WriteByte(int(entvars.Get(0).Sounds))
	m.WriteByte(int(entvars.Get(0).Sounds))

	m.WriteByte(svc.SetView)
	m.WriteShort(c.edictId)

	m.WriteByte(svc.SignonNum)
	m.WriteByte(1)

	c.sendSignon = true
	c.spawned = false
}

// Returns false if the client should be killed
func (c *SVClient) ReadClientMessage() (bool, error) {
	for {
		data, err := c.netConnection.GetMessage()
		if err != nil {
			log.Printf("SV_ReadClientMessage: ClientGetMessage failed\n")
			return false, nil
		}
		if len(data) == 0 {
			return true, nil // this is the default exit
		}
		// we do not care about the first byte as it only indicates if it was
		// send reliably (1) or not (2)
		pb, err := clc.FromBytes(data[1:], sv.protocol, sv.protocolFlags)
		if err != nil {
			log.Printf("SV_ReadClientMessage: %v", err)
			return false, nil
		}
		for _, cmd := range pb.GetCmds() {
			if !c.active {
				// a command caused an error
				return false, nil
			}
			switch cmd.Union.(type) {
			default:
				// nop
			case *protos.Cmd_Disconnect:
				return false, nil
			case *protos.Cmd_StringCmd:
				if c != HostClient() {
					log.Fatalf("HostClient differs")
				}
				s := cmd.GetStringCmd()
				a := qcmd.Parse(s)
				if len(a.Args()) == 0 {
					continue
				}
				switch strings.ToLower(a.Args()[0].String()) {
				default:
					conlog.Printf("%s tried to %s\n", c.name, s)
				case "begin":
					c.spawned = true
				case "color":
					c.colorCmd(a)
				case "fly":
					c.flyCmd(a)
				case "kill":
					if err := c.killCmd(a); err != nil {
						return false, err
					}
				case "noclip":
					c.noClipCmd(a)
				case "notarget":
					c.noTargetCmd(a)
				case "god":
					c.godCmd(a)
				case "pause":
					c.pauseCmd(a)
				case "ping":
					c.pingCmd(a)
				case "prespawn":
					c.preSpawnCmd()
				case "setpos":
					if err := c.setPosCmd(a); err != nil {
						return false, err
					}
				case "spawn":
					if err := c.spawnCmd(); err != nil {
						return false, err
					}
				case "give":
					c.giveCmd(a)
				case "mapname":
					// this is for a dedicated server
					if sv.Active() {
						fmt.Printf("\"mapname\" is %q", sv.name)
					} else {
						fmt.Printf("no map loaded")
					}
					// TODO(therjak):
					// case hasPrefix(s, "map"):
				case "edicts":
					edictPrintEdicts()
				case "edictcount":
					edictCount()
				case "edict":
					edictPrintEdictFunc(a)
				case "tell":
					c.tellCmd(a)
				case "kick":
					c.kickCmd(a)
				case "name":
					c.nameCmd(a)
				case "save":
					c.saveCmd(a)
				case "status":
					c.statusCmd()
				case "say_team":
					c.sayCmd(true && cvars.TeamPlay.Bool(), a)
				case "say":
					c.sayCmd(false, a)
				}
			case *protos.Cmd_MoveCmd:
				mc := cmd.GetMoveCmd()
				c.pingTimes[c.numPings%len(c.pingTimes)] = sv.time - mc.GetMessageTime()
				c.numPings++
				c.numPings %= len(c.pingTimes)
				ev := entvars.Get(c.edictId)
				ev.VAngle[0] = mc.GetPitch()
				ev.VAngle[1] = mc.GetYaw()
				ev.VAngle[2] = mc.GetRoll()
				c.cmd.forwardmove = mc.GetForward()
				c.cmd.sidemove = mc.GetSide()
				c.cmd.upmove = mc.GetUp()
				ev.Button0 = 0
				ev.Button2 = 0
				if mc.GetAttack() {
					ev.Button0 = 1
				}
				if mc.GetJump() {
					ev.Button2 = 1
				}
				if impulse := mc.GetImpulse(); impulse != 0 {
					ev.Impulse = float32(impulse)
				}
			}
		}
	}
}

func (c *SVClient) colorCmd(a cmd.Arguments) {
	args := a.Args()[1:]
	t := args[0].Int()
	b := t
	if len(args) > 1 {
		b = args[1].Int()
	}
	t &= 0x0f
	if t > 13 {
		t = 13
	}
	b &= 0x0f
	if b > 13 {
		b = 13
	}
	color := t*16 + b
	c.colors = color
	entvars.Get(c.edictId).Team = float32(b + 1)
	uc := &protos.UpdateColors{
		Player:   int32(c.id),
		NewColor: int32(color),
	}
	svc.WriteUpdateColors(uc, sv.protocol, sv.protocolFlags, &sv.reliableDatagram)
}

func (c *SVClient) flyCmd(a cmd.Arguments) {
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := entvars.Get(c.edictId)
	m := int32(ev.MoveType)
	args := a.Args()
	switch len(args) {
	default:
		return
	case 1:
		if m != progs.MoveTypeFly {
			m = progs.MoveTypeFly
		} else {
			m = progs.MoveTypeWalk
		}
	case 2:
		if args[1].Bool() {
			m = progs.MoveTypeFly
		} else {
			m = progs.MoveTypeWalk
		}
	}
	ev.MoveType = float32(m)
	if m == progs.MoveTypeFly {
		c.Printf("flymode %v\n", qFormatI(1))
	} else {
		c.Printf("flymode %v\n", qFormatI(0))
	}
}

func (c *SVClient) godCmd(a cmd.Arguments) {
	args := a.Args()[1:]
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := entvars.Get(c.edictId)
	f := int32(ev.Flags)
	const flag = progs.FlagGodMode
	switch len(args) {
	default:
		return
	case 0:
		f = f ^ flag
	case 1:
		if args[0].Bool() {
			f = f | flag
		} else {
			f = f &^ flag
		}
	}
	ev.Flags = float32(f)
	c.Printf("godmode %v\n", qFormatI(f&flag))
}

func (c *SVClient) killCmd(a cmd.Arguments) error {
	ev := entvars.Get(c.edictId)

	if ev.Health <= 0 {
		c.Printf("Can't suicide -- already dead!\n")
		return nil
	}

	progsdat.Globals.Time = sv.time
	progsdat.Globals.Self = int32(c.edictId)
	if err := vm.ExecuteProgram(progsdat.Globals.ClientKill); err != nil {
		return err
	}
	return nil
}

func (c *SVClient) noClipCmd(a cmd.Arguments) {
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := entvars.Get(c.edictId)
	m := int32(ev.MoveType)
	args := a.Args()[1:]
	switch len(args) {
	default:
		return
	case 0:
		if m != progs.MoveTypeNoClip {
			m = progs.MoveTypeNoClip
		} else {
			m = progs.MoveTypeWalk
		}
	case 1:
		if args[0].Bool() {
			m = progs.MoveTypeNoClip
		} else {
			m = progs.MoveTypeWalk
		}
	}
	ev.MoveType = float32(m)
	c.Printf("noclip %v\n", qFormatI(m&progs.MoveTypeNoClip))
}

func (c *SVClient) noTargetCmd(a cmd.Arguments) {
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := entvars.Get(c.edictId)
	f := int32(ev.Flags)
	const flag = progs.FlagNoTarget
	args := a.Args()[1:]
	switch len(args) {
	default:
		return
	case 0:
		f = f ^ flag
	case 1:
		if args[0].Bool() {
			f = f | flag
		} else {
			f = f &^ flag
		}
	}
	ev.Flags = float32(f)
	c.Printf("notarget %v\n", qFormatI(f&flag))
}

func (c *SVClient) pauseCmd(a cmd.Arguments) {
	if cvars.Pausable.String() != "1" {
		c.Printf("Pause not allowed.\n")
		return
	}
	sv.paused = !sv.paused

	ev := entvars.Get(c.edictId)
	playerName, _ := progsdat.String(ev.NetName)
	SV_BroadcastPrintf("%s %s the game\n", playerName, func() string {
		if sv.paused {
			return "paused"
		}
		return "unpaused"
	}())

	svc.WriteSetPause(sv.paused, sv.protocol, sv.protocolFlags, &sv.reliableDatagram)
}

func (c *SVClient) pingCmd(a cmd.Arguments) {
	c.Printf("Client ping times:\n")
	for _, ac := range sv_clients {
		if !ac.active {
			continue
		}
		c.Printf("%4d %s\n", int(ac.PingTime()*1000), ac.name)
	}
}

func (c *SVClient) preSpawnCmd() {
	if c.spawned {
		conlog.Printf("prespawn not valid -- already spawned\n")
		return
	}
	c.msg.WriteBytes(sv.signon.Bytes())
	c.msg.WriteByte(svc.SignonNum)
	c.msg.WriteByte(2)
	c.sendSignon = true
}

func (c *SVClient) setPosCmd(a cmd.Arguments) error {
	if progsdat.Globals.DeathMatch != 0 {
		return nil
	}
	ev := entvars.Get(c.edictId)
	args := a.Args()
	switch len(args) {
	case 7:
		ev.Angles = [3]float32{
			args[4].Float32(),
			args[5].Float32(),
			args[6].Float32(),
		}
		ev.FixAngle = 1
		fallthrough
	case 4:
		if ev.MoveType != progs.MoveTypeNoClip {
			ev.MoveType = progs.MoveTypeNoClip
			c.Printf("noclip ON\n")
		}
		// make sure they're not going to whizz away from it
		ev.Velocity = [3]float32{0, 0, 0}
		ev.Origin = [3]float32{
			args[1].Float32(),
			args[2].Float32(),
			args[3].Float32(),
		}
		if err := vm.LinkEdict(c.edictId, false); err != nil {
			return err
		}
		return nil
	default:
		c.Printf("usage:\n")
		c.Printf("   setpos <x> <y> <z>\n")
		c.Printf("   setpos <x> <y> <z> <pitch> <yaw> <roll>\n")
		c.Printf("current values:\n")
		c.Printf("   %d %d %d %d %d %d\n",
			int(ev.Origin[0]), int(ev.Origin[1]), int(ev.Origin[2]),
			int(ev.VAngle[0]), int(ev.VAngle[1]), int(ev.VAngle[2]))
		return nil
	}
}

func (c *SVClient) spawnCmd() error {
	if c.spawned {
		conlog.Printf("Spawn not valid -- already spawned\n")
		return nil
	}
	// run the entrance script
	if sv.loadGame {
		// loaded games are fully inited already
		// if this is the last client to be connected, unpause
		sv.paused = false
	} else {
		entvars.Clear(c.edictId)
		ev := entvars.Get(c.edictId)
		ev.ColorMap = float32(c.edictId)
		ev.Team = float32((c.colors & 15) + 1)
		ev.NetName = progsdat.AddString(c.name)
		progsdat.Globals.Parm = c.spawnParams
		progsdat.Globals.Time = sv.time
		progsdat.Globals.Self = int32(c.edictId)
		if err := vm.ExecuteProgram(progsdat.Globals.ClientConnect); err != nil {
			return err
		}
		if time.Since(c.ConnectTime()).Seconds() <= float64(sv.time) {
			log.Printf("%v entered the game\n", c.name)
		}
		if err := vm.ExecuteProgram(progsdat.Globals.PutClientInServer); err != nil {
			return err
		}
	}

	// send all current names, colors, and frag counts
	c.msg.ClearMessage()

	// send time of update
	svc.WriteTime(sv.time, sv.protocol, sv.protocolFlags, &c.msg)

	for i, sc := range sv_clients {
		if i >= svs.maxClients {
			// TODO: figure out why it ever makes sense to have len(sv_clients) svs.maxClients
			break
		}
		un := &protos.UpdateName{
			Player:  int32(i),
			NewName: sc.name,
		}
		svc.WriteUpdateName(un, sv.protocol, sv.protocolFlags, &c.msg)
		uf := &protos.UpdateFrags{
			Player:   int32(i),
			NewFrags: int32(sc.oldFrags),
		}
		svc.WriteUpdateFrags(uf, sv.protocol, sv.protocolFlags, &c.msg)
		uc := &protos.UpdateColors{
			Player:   int32(i),
			NewColor: int32(sc.colors),
		}
		svc.WriteUpdateColors(uc, sv.protocol, sv.protocolFlags, &c.msg)
	}

	// send all current light styles
	for i, ls := range sv.lightStyles {
		c.msg.WriteByte(svc.LightStyle)
		c.msg.WriteByte(i)
		c.msg.WriteString(ls)
	}

	c.msg.WriteByte(svc.UpdateStat)
	c.msg.WriteByte(svc.StatTotalSecrets)
	c.msg.WriteLong(int(progsdat.Globals.TotalSecrets))

	c.msg.WriteByte(svc.UpdateStat)
	c.msg.WriteByte(svc.StatTotalMonsters)
	c.msg.WriteLong(int(progsdat.Globals.TotalMonsters))

	c.msg.WriteByte(svc.UpdateStat)
	c.msg.WriteByte(svc.StatSecrets)
	c.msg.WriteLong(int(progsdat.Globals.FoundSecrets))

	c.msg.WriteByte(svc.UpdateStat)
	c.msg.WriteByte(svc.StatMonsters)
	c.msg.WriteLong(int(progsdat.Globals.KilledMonsters))

	// send a fixangle
	// Never send a roll angle, because savegames can catch the server
	// in a state where it is expecting the client to correct the angle
	// and it won't happen if the game was just loaded, so you wind up
	// with a permanent head tilt
	sa := &protos.Coord{
		X: entvars.Get(c.edictId).Angles[0],
		Y: entvars.Get(c.edictId).Angles[1],
		Z: 0,
	}
	svc.WriteSetAngle(sa, sv.protocol, sv.protocolFlags, &c.msg)

	msgBuf.ClearMessage()
	msgBufMaxLen = protocol.MaxDatagram
	sv.WriteClientdataToMessage(c.edictId)
	c.msg.WriteBytes(msgBuf.Bytes())

	c.msg.WriteByte(svc.SignonNum)
	c.msg.WriteByte(3)
	c.sendSignon = true
	return nil
}

func (c *SVClient) giveCmd(a cmd.Arguments) {
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := entvars.Get(c.edictId)
	args := a.Args()
	if len(args) == 1 {
		return
	}

	t := strings.ToLower(args[1].String())
	v := float32(0)
	if len(args) > 2 {
		v = float32(args[2].Int())
	}

	switch t[0] {
	case byte('0'):
	case byte('1'):
	case byte('2'):
		ev.Items = float32(int32(ev.Items) | progs.ItemShotgun)
	case byte('3'):
		ev.Items = float32(int32(ev.Items) | progs.ItemSuperShotgun)
	case byte('4'):
		ev.Items = float32(int32(ev.Items) | progs.ItemNailgun)
	case byte('5'):
		ev.Items = float32(int32(ev.Items) | progs.ItemSuperNailgun)
	case byte('6'):
		ev.Items = float32(int32(ev.Items) | progs.ItemGrenadeLauncher)
	case byte('7'):
		ev.Items = float32(int32(ev.Items) | progs.ItemRocketLauncher)
	case byte('8'):
		ev.Items = float32(int32(ev.Items) | progs.ItemLightning)
	case byte('9'):
	case byte('s'):
		ev.AmmoShells = v
	case byte('n'):
		ev.AmmoNails = v
	case byte('r'):
		ev.AmmoRockets = v
	case byte('h'):
		ev.Health = v
	case byte('c'):
		ev.AmmoCells = v
	case byte('a'):
		if v > 150 {
			ev.ArmorType = 0.8
			ev.ArmorValue = v
			ev.Items = float32((int32(ev.Items) &^ (progs.ItemArmor1 | progs.ItemArmor2)) | progs.ItemArmor3)
		} else if v > 100 {
			ev.ArmorType = 0.6
			ev.ArmorValue = v
			ev.Items = float32((int32(ev.Items) &^ (progs.ItemArmor1 | progs.ItemArmor3)) | progs.ItemArmor2)
		} else if v >= 0 {
			ev.ArmorType = 0.3
			ev.ArmorValue = v
			ev.Items = float32((int32(ev.Items) &^ (progs.ItemArmor2 | progs.ItemArmor3)) | progs.ItemArmor1)
		}

	}
	/*
	  switch (t[0]) {
	    case '0':
	    case '1':
	    case '2':
	    case '3':
	    case '4':
	    case '5':
	    case '6':
	    case '7':
	    case '8':
	    case '9':
	      // MED 01/04/97 added hipnotic give stuff
	      if (cmdl.Hipnotic() || cmdl.Quoth()) {
	        if (t[0] == '6') {
	          if (t[1] == 'a')
	            pent->items = (int)pent->items | HIT_PROXIMITY_GUN;
	          else
	            pent->items = (int)pent->items | IT_GRENADE_LAUNCHER;
	        } else if (t[0] == '9')
	          pent->items = (int)pent->items | HIT_LASER_CANNON;
	        else if (t[0] == '0')
	          pent->items = (int)pent->items | HIT_MJOLNIR;
	        else if (t[0] >= '2')
	          pent->items = (int)pent->items | (IT_SHOTGUN << (t[0] - '2'));
	      } else {
	        if (t[0] >= '2')
	          pent->items = (int)pent->items | (IT_SHOTGUN << (t[0] - '2'));
	      }
	      break;

	    case 's':
	      if (cmdl.Rogue()) {
	        val = GetEdictFieldValue(pent, "ammo_shells1");
	        if (val) val->_float = v;
	      }
	      pent->ammo_shells = v;
	      break;

	    case 'n':
	      if (cmdl.Rogue()) {
	        val = GetEdictFieldValue(pent, "ammo_nails1");
	        if (val) {
	          val->_float = v;
	          if (pent->weapon <= IT_LIGHTNING) pent->ammo_nails = v;
	        }
	      } else {
	        pent->ammo_nails = v;
	      }
	      break;

	    case 'l':
	      if (cmdl.Rogue()) {
	        val = GetEdictFieldValue(pent, "ammo_lava_nails");
	        if (val) {
	          val->_float = v;
	          if (pent->weapon > IT_LIGHTNING) pent->ammo_nails = v;
	        }
	      }
	      break;

	    case 'r':
	      if (cmdl.Rogue()) {
	        val = GetEdictFieldValue(pent, "ammo_rockets1");
	        if (val) {
	          val->_float = v;
	          if (pent->weapon <= IT_LIGHTNING) pent->ammo_rockets = v;
	        }
	      } else {
	        pent->ammo_rockets = v;
	      }
	      break;

	    case 'm':
	      if (cmdl.Rogue()) {
	        val = GetEdictFieldValue(pent, "ammo_multi_rockets");
	        if (val) {
	          val->_float = v;
	          if (pent->weapon > IT_LIGHTNING) pent->ammo_rockets = v;
	        }
	      }
	      break;

	    case 'h':
	      pent->health = v;
	      break;

	    case 'c':
	      if (cmdl.Rogue()) {
	        val = GetEdictFieldValue(pent, "ammo_cells1");
	        if (val) {
	          val->_float = v;
	          if (pent->weapon <= IT_LIGHTNING) pent->ammo_cells = v;
	        }
	      } else {
	        pent->ammo_cells = v;
	      }
	      break;

	    case 'p':
	      if (cmdl.Rogue()) {
	        val = GetEdictFieldValue(pent, "ammo_plasma");
	        if (val) {
	          val->_float = v;
	          if (pent->weapon > IT_LIGHTNING) pent->ammo_cells = v;
	        }
	      }
	      break;
	  }
	*/

	// Update currentammo to update statusbar correctly
	switch ev.Weapon {
	case progs.ItemShotgun,
		progs.ItemSuperShotgun:
		ev.CurrentAmmo = ev.AmmoShells
	case progs.ItemNailgun,
		progs.ItemSuperNailgun,
		progs.RogueItemLavaSuperNailgun:
		ev.CurrentAmmo = ev.AmmoNails
	case progs.ItemGrenadeLauncher,
		progs.ItemRocketLauncher,
		progs.RogueItemMultiGrenade,
		progs.RogueItemMultiRocket:
		ev.CurrentAmmo = ev.AmmoRockets
	case progs.ItemLightning,
		progs.HipnoticItemLaserCannon,
		progs.HipnoticItemMjolnir:
		ev.CurrentAmmo = ev.AmmoCells
	case progs.RogueItemLavaNailgun:
		// This is the same as ItemAxe so we need to be more careful
		if cmdl.Rogue() {
			ev.CurrentAmmo = ev.AmmoNails
		}
	case progs.RogueItemPlasmaGun:
		// This is the same as HipnoticItemProximityGun, so be more careful
		if cmdl.Rogue() {
			ev.CurrentAmmo = ev.AmmoCells
		} else if cmdl.Hipnotic() {
			ev.CurrentAmmo = ev.AmmoRockets
		}
	}
}

func (c *SVClient) tellCmd(a cmd.Arguments) error {
	args := a.Args()
	if len(args) < 3 {
		// need at least destination and message
		return nil
	}
	text := fmt.Sprintf("%s: %s\n", c.name, a.Message())
	for _, ac := range sv_clients {
		if !ac.active || !ac.spawned {
			continue
		}
		if !strings.EqualFold(ac.name, args[1].String()) {
			continue
		}
		// TODO: We check without case check. Are names unique ignoring the case?
		ac.Printf(text)
	}
	return nil
}

// Kicks a user off of the server
func (c *SVClient) kickCmd(a cmd.Arguments) error {
	args := a.Args()[1:]
	if len(args) == 0 {
		return nil
	}
	// TODO(therjak): admin mode
	if !c.admin && progsdat.Globals.DeathMatch != 0 {
		return nil
	}

	var toKick *SVClient
	message := a.Message()

	if len(args) > 1 && args[0].String() == "#" {
		i := args[1].Int() - 1
		if i < 0 || i >= svs.maxClients {
			return nil
		}
		toKick = sv_clients[i]
		if !toKick.active {
			return nil
		}
		message = strings.TrimLeft(message, "1234567890")
		message = strings.TrimLeftFunc(message, unicode.IsSpace)
	} else {
		for _, c := range sv_clients {
			if !c.active {
				continue
			}
			if c.name == args[0].String() {
				toKick = c
				break
			}
		}
	}
	if toKick == nil {
		return nil
	}
	if c.edictId == toKick.edictId {
		// can't kick yourself!
		return nil
	}
	who := c.name
	if message != "" {
		toKick.Printf("Kicked by %s: %s\n", who, message)
	} else {
		toKick.Printf("Kicked by %s\n", who)
	}
	if err := toKick.Drop(false); err != nil {
		return err
	}
	return nil
}

func (c *SVClient) nameCmd(a cmd.Arguments) {
	args := a.Args()
	if len(args) < 2 {
		return
	}
	newName := a.ArgumentString()
	if len(newName) > 15 {
		newName = newName[:15]
	}

	if len(c.name) != 0 && c.name != "unconnected" && c.name != newName {
		log.Printf("%s renamed to %s\n", c.name, newName)
	}
	c.name = newName
	entvars.Get(c.edictId).NetName = progsdat.AddString(newName)

	// send notification to all clients
	un := &protos.UpdateName{
		Player:  int32(c.id),
		NewName: newName,
	}
	svc.WriteUpdateName(un, sv.protocol, sv.protocolFlags, &sv.reliableDatagram)
}

func (c *SVClient) statusCmd() {
	const baseVersion = 1.09
	c.Printf("host:    %s\n", cvars.HostName.String())
	c.Printf("version: %4.2f\n", baseVersion)
	c.Printf("tcp/ip:  %s\n", net.Address())
	c.Printf("map:     %s\n", sv.name)
	active := 0
	for _, ac := range sv_clients {
		if ac.active {
			active++
		}
	}
	c.Printf("players: %d active (%d max)\n\n", active, svs.maxClients)
	ntime := net.Time()
	for i, ac := range sv_clients {
		if !ac.active {
			continue
		}
		d := ntime.Sub(ac.ConnectTime())
		d = d.Truncate(time.Second)
		ev := entvars.Get(c.edictId)
		c.Printf("#%-2d %-16.16s  %3d  %9s\n", i+1, ac.name, int(ev.Frags), d.String())
		c.Printf("   %s\n", ac.Address())
	}
}

func (c *SVClient) sayCmd(team bool, a cmd.Arguments) {
	if len(a.Args()) < 2 {
		return
	}
	text := fmt.Sprintf("\001%s: %s\n", c.name, a.ArgumentString())
	for _, ac := range sv_clients {
		if !ac.active || !ac.spawned {
			continue
		}
		if team &&
			entvars.Get(ac.edictId).Team != entvars.Get(c.edictId).Team {
			continue
		}
		ac.Printf(text)
	}
	if cmdl.Dedicated() {
		log.Print(text)
	}
}

func SV_RunClients() error {
	for i := 0; i < svs.maxClients; i++ {
		host_client = i

		hc := sv_clients[i]
		if !hc.active {
			continue
		}

		ok, err := hc.ReadClientMessage()
		if err != nil {
			return err
		}
		if !ok {
			if err := hc.Drop(false); err != nil {
				return err
			}
			continue
		}

		if !hc.spawned {
			// clear client movement until a new packet is received
			hc.cmd = movecmd{0, 0, 0}
			continue
		}

		if !sv.paused {
			// TODO(therjak): is this pause stuff really needed?
			// always pause in single player if in console or menus
			//if svs.maxClients > 1 || keyDestination == keys.Game {
			hc.Think()
			//}
		}
	}
	return nil
}

// For debugging
func edictPrint(ed int) {
	if edictNum(ed).Free {
		fmt.Printf("FREE\n")
		return
	}
	fmt.Printf("\nEDICT %d:\n", ed)
	for i := 1; i < len(progsdat.FieldDefs); i++ {
		d := progsdat.FieldDefs[i]
		name, err := progsdat.String(d.SName)
		if err != nil {
			continue
		}
		l := len(name)
		if l > 1 && (name)[l-2] == '_' {
			// skip _x, _y, _z vars
			continue
		}
		// TODO: skip 0 values
		fmt.Printf("%-15s %s\n", name, entvars.Sprint(ed, d))
	}
}

// For debugging, prints all the entities in the current server
func edictPrintEdicts() {
	if !sv.Active() {
		return
	}

	fmt.Printf("%d entities\n", sv.numEdicts)
	for i := 0; i < sv.numEdicts; i++ {
		edictPrint(i)
	}
}

// For debugging, prints a single edict
func edictPrintEdictFunc(a cmd.Arguments) {
	args := a.Args()
	if !sv.Active() || len(args) < 2 {
		return
	}

	i := args[1].Int()
	if i < 0 || i >= sv.numEdicts {
		fmt.Printf("Bad edict number\n")
		return
	}
	edictPrint(i)
}

// For debugging
func edictCount() {
	if !sv.Active() {
		return
	}

	active := 0
	models := 0
	solid := 0
	step := 0
	for i := 0; i < sv.numEdicts; i++ {
		if edictNum(i).Free {
			continue
		}
		active++
		if entvars.Get(i).Solid != 0 {
			solid++
		}
		if entvars.Get(i).Model != 0 {
			models++
		}
		if entvars.Get(i).MoveType == progs.MoveTypeStep {
			step++
		}
	}

	fmt.Printf("num_edicts:%3d\n", sv.numEdicts)
	fmt.Printf("active    :%3d\n", active)
	fmt.Printf("view      :%3d\n", models)
	fmt.Printf("touch     :%3d\n", solid)
	fmt.Printf("step      :%3d\n", step)
}
