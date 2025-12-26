// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"
	"unicode"

	"goquake/cbuf"
	cmdl "goquake/commandline"
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

func (s *Server) BroadcastPrintf(format string, v ...interface{}) {
	s.BroadcastPrint(fmt.Sprintf(format, v...))
}

func (s *Server) BroadcastPrint(m string) {
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

func (sc *SVClient) Printf(format string, v ...interface{}) {
	sc.print(fmt.Sprintf(format, v...))
}

func (sc *SVClient) print(msg string) {
	sc.msg.WriteByte(svc.Print)
	sc.msg.WriteString(msg)
}

func (sc *SVClient) ClientCommands(msg string) {
	sc.msg.WriteByte(svc.StuffText)
	sc.msg.WriteString(msg)
}

func (sc *SVClient) PingTime() float32 {
	r := float32(0)
	for _, p := range sc.pingTimes {
		r += p
	}
	return r / float32(len(sc.pingTimes))
}

func (s *Server) checkForNewClients() error {
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
			if err := s.connectClient(c.id); err != nil {
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

func (sc *SVClient) CanSendMessage() bool {
	return sc.netConnection.CanSendMessage()
}

func (sc *SVClient) Close() {
	sc.netConnection.Close()
	sc.netConnection = nil
	sc.admin = false
	sc.active = false
	sc.name = ""
	sc.oldFrags = -999999
}

func (sc *SVClient) ConnectTime() time.Time {
	return sc.netConnection.ConnectTime()
}

func (sc *SVClient) Address() string {
	return sc.netConnection.Address()
}

func (sc *SVClient) SendMessage() int {
	return sc.netConnection.SendMessage(sc.msg.Bytes())
}

func (sc *SVClient) SendNop() error {
	if sc.netConnection.SendUnreliableMessage([]byte{svc.Nop}) == -1 {
		if err := sc.Drop(true); err != nil {
			return err
		}
	}
	sc.lastMessage = host.Time()
	return nil
}

func (sc *SVClient) Drop(crash bool) error {
	if !crash {
		// send any final messages (don't check for errors)
		if sc.CanSendMessage() {
			sc.msg.WriteByte(svc.Disconnect)
			sc.SendMessage()
		}

		if sc.spawned {
			// call the prog function for removing a client
			// this will set the body to a dead frame, among other things
			saveSelf := progsdat.Globals.Self
			progsdat.Globals.Self = int32(sc.edictId)
			if err := vm.ExecuteProgram(progsdat.Globals.ClientDisconnect, &sv); err != nil {
				return err
			}
			progsdat.Globals.Self = saveSelf
		}
		log.Printf("Client %s removed", sc.name)
	}

	// break the net connection
	sc.Close()

	// send notification to all clients
	for _, c := range sv_clients {
		if !c.active {
			continue
		}
		c.msg.WriteByte(svc.UpdateName)
		c.msg.WriteByte(sc.id)
		c.msg.WriteString("")
		c.msg.WriteByte(svc.UpdateFrags)
		c.msg.WriteByte(sc.id)
		c.msg.WriteShort(0)
		c.msg.WriteByte(svc.UpdateColors)
		c.msg.WriteByte(sc.id)
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

func (sc *SVClient) GetMessage() error {
	_, err := sc.netConnection.GetMessage()
	return err
}

func (sc *SVClient) SendServerinfo(s *Server) {
	m := &sc.msg
	m.WriteByte(svc.Print)
	m.WriteString(
		fmt.Sprintf("%s\nGOQUAKE %1.2f SERVER (%d CRC)\n",
			[]byte{2}, version.Base, progsdat.CRC))

	m.WriteByte(int(svc.ServerInfo))
	m.WriteLong(int(s.protocol))

	if s.protocol == protocol.RMQ {
		m.WriteLong(int(s.protocolFlags))
	}

	sc.msg.WriteByte(svs.maxClients)

	if !cvars.Coop.Bool() && cvars.DeathMatch.Bool() {
		m.WriteByte(svc.GameDeathmatch)
	} else {
		m.WriteByte(svc.GameCoop)
	}

	sm, err := progsdat.String(entvars.Get(0).Message)
	if err != nil {
		sm = ""
	}
	m.WriteString(sm)

	for i, mn := range s.modelPrecache[1:] {
		if s.protocol == protocol.NetQuake && i >= 256 {
			break
		}
		m.WriteString(mn)
	}
	m.WriteByte(0)

	for i, sn := range s.soundPrecache[1:] {
		if s.protocol == protocol.NetQuake && i >= 256 {
			break
		}
		m.WriteString(sn)
	}
	m.WriteByte(0)

	m.WriteByte(svc.CDTrack)
	m.WriteByte(int(entvars.Get(0).Sounds))
	m.WriteByte(int(entvars.Get(0).Sounds))

	m.WriteByte(svc.SetView)
	m.WriteShort(sc.edictId)

	m.WriteByte(svc.SignonNum)
	m.WriteByte(1)

	sc.sendSignon = true
	sc.spawned = false
}

// Returns false if the client should be killed
func (sc *SVClient) ReadClientMessage(s *Server) (bool, error) {
	for {
		data, err := sc.netConnection.GetMessage()
		if err != nil {
			log.Printf("SV_ReadClientMessage: ClientGetMessage failed\n")
			return false, nil
		}
		if len(data) == 0 {
			return true, nil // this is the default exit
		}
		// we do not care about the first byte as it only indicates if it was
		// send reliably (1) or not (2)
		pb, err := clc.FromBytes(data[1:], s.protocol, s.protocolFlags)
		if err != nil {
			log.Printf("SV_ReadClientMessage: %v", err)
			return false, nil
		}
		for _, cmd := range pb.GetCmds() {
			if !sc.active {
				// a command caused an error
				return false, nil
			}
			switch cmd.WhichUnion() {
			default:
				// nop
			case protos.Cmd_Disconnect_case:
				return false, nil
			case protos.Cmd_StringCmd_case:
				if sc != HostClient() {
					log.Fatalf("HostClient differs")
				}
				scmd := cmd.GetStringCmd()
				a := cbuf.Parse(scmd)
				if len(a.Args()) == 0 {
					continue
				}
				switch strings.ToLower(a.Args()[0].String()) {
				default:
					slog.Warn("player tried something", slog.String("player", sc.name), slog.String("action", scmd))
				case "begin":
					sc.spawned = true
				case "color":
					color := sc.colorCmd(a)
					svc.WriteUpdateColors(color, s.protocol, s.protocolFlags, &s.reliableDatagram)
				case "fly":
					sc.flyCmd(a)
				case "kill":
					if err := sc.killCmd(s.time, a); err != nil {
						return false, err
					}
				case "noclip":
					sc.noClipCmd(a)
				case "notarget":
					sc.noTargetCmd(a)
				case "god":
					sc.godCmd(a)

				case "pause":
					if cvars.Pausable.String() != "1" {
						sc.Printf("Pause not allowed.\n")
						continue
					}
					s.paused = !s.paused
					s.BroadcastPrintf("%s %s the game\n", sc.playerName(), func() string {
						if s.paused {
							return "paused"
						}
						return "unpaused"
					}())
					svc.WriteSetPause(s.paused, s.protocol, s.protocolFlags, &s.reliableDatagram)
				case "ping":
					sc.pingCmd(a)
				case "prespawn":
					sc.preSpawnCmd(s.signon.Bytes())
				case "setpos":
					if err := sc.setPosCmd(a); err != nil {
						return false, err
					}
				case "spawn":
					if err := sc.spawnCmd(s); err != nil {
						return false, err
					}
				case "give":
					sc.giveCmd(a)
				case "mapname":
					// this is for a dedicated server
					if s.Active() {
						fmt.Printf("\"mapname\" is %q", s.name)
					} else {
						fmt.Printf("no map loaded")
					}
				//case "map":
				// TODO(therjak):
				// see Host_Map_f in orig
				// in case of hostFwd
				case "edicts":
					s.edictPrintEdicts()
				case "edictcount":
					s.edictCount()
				case "edict":
					s.edictPrintEdictFunc(a)
				case "tell":
					sc.tellCmd(a)
				case "kick":
					if err := sc.kickCmd(a); err != nil {
						fmt.Printf("Drop error: %v", err)
					}
				case "name":
					args := a.Args()
					if len(args) < 2 {
						continue
					}
					nn := sc.nameCmd(a)
					svc.WriteUpdateName(nn, s.protocol, s.protocolFlags, &s.reliableDatagram)
				case "save":
					sc.saveCmd(a)
				case "status":
					sc.statusCmd(s.name)
				case "say_team":
					sc.sayCmd(true && cvars.TeamPlay.Bool(), a)
				case "say":
					sc.sayCmd(false, a)
				}
			case protos.Cmd_MoveCmd_case:
				mc := cmd.GetMoveCmd()
				sc.pingTimes[sc.numPings%len(sc.pingTimes)] = s.time - mc.GetMessageTime()
				sc.numPings++
				sc.numPings %= len(sc.pingTimes)
				ev := entvars.Get(sc.edictId)
				ev.VAngle[0] = mc.GetPitch()
				ev.VAngle[1] = mc.GetYaw()
				ev.VAngle[2] = mc.GetRoll()
				sc.cmd.forwardmove = mc.GetForward()
				sc.cmd.sidemove = mc.GetSide()
				sc.cmd.upmove = mc.GetUp()
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

func (sc *SVClient) colorCmd(a cbuf.Arguments) *protos.UpdateColors {
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
	sc.colors = color
	entvars.Get(sc.edictId).Team = float32(b + 1)
	return protos.UpdateColors_builder{
		Player:   int32(sc.id),
		NewColor: int32(color),
	}.Build()
}

func (sc *SVClient) flyCmd(a cbuf.Arguments) {
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := entvars.Get(sc.edictId)
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
		sc.Printf("flymode %v\n", qFormatI(1))
	} else {
		sc.Printf("flymode %v\n", qFormatI(0))
	}
}

func (sc *SVClient) godCmd(a cbuf.Arguments) {
	args := a.Args()[1:]
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := entvars.Get(sc.edictId)
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
	sc.Printf("godmode %v\n", qFormatI(f&flag))
}

func (sc *SVClient) killCmd(time float32, a cbuf.Arguments) error {
	ev := entvars.Get(sc.edictId)

	if ev.Health <= 0 {
		sc.Printf("Can't suicide -- already dead!\n")
		return nil
	}

	progsdat.Globals.Time = time
	progsdat.Globals.Self = int32(sc.edictId)
	if err := vm.ExecuteProgram(progsdat.Globals.ClientKill, &sv); err != nil {
		return err
	}
	return nil
}

func (sc *SVClient) noClipCmd(a cbuf.Arguments) {
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := entvars.Get(sc.edictId)
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
	sc.Printf("noclip %v\n", qFormatI(m&progs.MoveTypeNoClip))
}

func (sc *SVClient) noTargetCmd(a cbuf.Arguments) {
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := entvars.Get(sc.edictId)
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
	sc.Printf("notarget %v\n", qFormatI(f&flag))
}

func (sc *SVClient) playerName() string {
	ev := entvars.Get(sc.edictId)
	name, _ := progsdat.String(ev.NetName)
	return name
}

func (sc *SVClient) pingCmd(a cbuf.Arguments) {
	sc.Printf("Client ping times:\n")
	for _, ac := range sv_clients {
		if !ac.active {
			continue
		}
		sc.Printf("%4d %s\n", int(ac.PingTime()*1000), ac.name)
	}
}

func (sc *SVClient) preSpawnCmd(signon []byte) {
	if sc.spawned {
		slog.Warn("prespawn not valid -- already spawned")
		return
	}
	sc.msg.WriteBytes(signon)
	sc.msg.WriteByte(svc.SignonNum)
	sc.msg.WriteByte(2)
	sc.sendSignon = true
}

func (sc *SVClient) setPosCmd(a cbuf.Arguments) error {
	if progsdat.Globals.DeathMatch != 0 {
		return nil
	}
	ev := entvars.Get(sc.edictId)
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
			sc.Printf("noclip ON\n")
		}
		// make sure they're not going to whizz away from it
		ev.Velocity = [3]float32{0, 0, 0}
		ev.Origin = [3]float32{
			args[1].Float32(),
			args[2].Float32(),
			args[3].Float32(),
		}
		if err := vm.LinkEdict(sc.edictId, false, &sv); err != nil {
			return err
		}
		return nil
	default:
		sc.Printf("usage:\n")
		sc.Printf("   setpos <x> <y> <z>\n")
		sc.Printf("   setpos <x> <y> <z> <pitch> <yaw> <roll>\n")
		sc.Printf("current values:\n")
		sc.Printf("   %d %d %d %d %d %d\n",
			int(ev.Origin[0]), int(ev.Origin[1]), int(ev.Origin[2]),
			int(ev.VAngle[0]), int(ev.VAngle[1]), int(ev.VAngle[2]))
		return nil
	}
}

func (sc *SVClient) spawnCmd(s *Server) error {
	if sc.spawned {
		slog.Warn("Spawn not valid -- already spawned")
		return nil
	}
	// run the entrance script
	if s.loadGame {
		// loaded games are fully inited already
		// if this is the last client to be connected, unpause
		s.paused = false
	} else {
		entvars.Clear(sc.edictId)
		ev := entvars.Get(sc.edictId)
		ev.ColorMap = float32(sc.edictId)
		ev.Team = float32((sc.colors & 15) + 1)
		ev.NetName = progsdat.AddString(sc.name)
		progsdat.Globals.Parm = sc.spawnParams
		progsdat.Globals.Time = s.time
		progsdat.Globals.Self = int32(sc.edictId)
		if err := vm.ExecuteProgram(progsdat.Globals.ClientConnect, &sv); err != nil {
			return err
		}
		if time.Since(sc.ConnectTime()).Seconds() <= float64(s.time) {
			log.Printf("%v entered the game\n", sc.name)
		}
		if err := vm.ExecuteProgram(progsdat.Globals.PutClientInServer, &sv); err != nil {
			return err
		}
	}

	// send all current names, colors, and frag counts
	sc.msg.Reset()

	// send time of update
	svc.WriteTime(s.time, s.protocol, s.protocolFlags, &sc.msg)

	for i, scs := range sv_clients {
		if i >= svs.maxClients {
			// TODO: figure out why it ever makes sense to have len(sv_clients) svs.maxClients
			break
		}
		un := protos.UpdateName_builder{
			Player:  int32(i),
			NewName: scs.name,
		}.Build()
		svc.WriteUpdateName(un, s.protocol, s.protocolFlags, &sc.msg)
		uf := protos.UpdateFrags_builder{
			Player:   int32(i),
			NewFrags: int32(scs.oldFrags),
		}.Build()
		svc.WriteUpdateFrags(uf, s.protocol, s.protocolFlags, &sc.msg)
		uc := protos.UpdateColors_builder{
			Player:   int32(i),
			NewColor: int32(scs.colors),
		}.Build()
		svc.WriteUpdateColors(uc, s.protocol, s.protocolFlags, &sc.msg)
	}

	// send all current light styles
	for i, ls := range s.lightStyles {
		sc.msg.WriteByte(svc.LightStyle)
		sc.msg.WriteByte(i)
		sc.msg.WriteString(ls)
	}

	sc.msg.WriteByte(svc.UpdateStat)
	sc.msg.WriteByte(svc.StatTotalSecrets)
	sc.msg.WriteLong(int(progsdat.Globals.TotalSecrets))

	sc.msg.WriteByte(svc.UpdateStat)
	sc.msg.WriteByte(svc.StatTotalMonsters)
	sc.msg.WriteLong(int(progsdat.Globals.TotalMonsters))

	sc.msg.WriteByte(svc.UpdateStat)
	sc.msg.WriteByte(svc.StatSecrets)
	sc.msg.WriteLong(int(progsdat.Globals.FoundSecrets))

	sc.msg.WriteByte(svc.UpdateStat)
	sc.msg.WriteByte(svc.StatMonsters)
	sc.msg.WriteLong(int(progsdat.Globals.KilledMonsters))

	// send a fixangle
	// Never send a roll angle, because savegames can catch the server
	// in a state where it is expecting the client to correct the angle
	// and it won't happen if the game was just loaded, so you wind up
	// with a permanent head tilt
	sa := protos.Coord_builder{
		X: entvars.Get(sc.edictId).Angles[0],
		Y: entvars.Get(sc.edictId).Angles[1],
		Z: 0,
	}.Build()
	svc.WriteSetAngle(sa, s.protocol, s.protocolFlags, &sc.msg)

	msgBuf.Reset()
	msgBufMaxLen = protocol.MaxDatagram
	s.WriteClientdataToMessage(sc.edictId)
	sc.msg.WriteBytes(msgBuf.Bytes())

	sc.msg.WriteByte(svc.SignonNum)
	sc.msg.WriteByte(3)
	sc.sendSignon = true
	return nil
}

func (sc *SVClient) giveCmd(a cbuf.Arguments) {
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := entvars.Get(sc.edictId)
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

func (sc *SVClient) tellCmd(a cbuf.Arguments) {
	args := a.Args()
	if len(args) < 3 {
		// need at least destination and message
		return
	}
	text := fmt.Sprintf("%s: %s\n", sc.name, a.Message())
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
}

// Kicks a user off of the server
func (sc *SVClient) kickCmd(a cbuf.Arguments) error {
	args := a.Args()[1:]
	if len(args) == 0 {
		return nil
	}
	// TODO(therjak): admin mode
	if !sc.admin && progsdat.Globals.DeathMatch != 0 {
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
	if sc.edictId == toKick.edictId {
		// can't kick yourself!
		return nil
	}
	who := sc.name
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

func (sc *SVClient) nameCmd(a cbuf.Arguments) *protos.UpdateName {
	newName := a.ArgumentString()
	if len(newName) > 15 {
		newName = newName[:15]
	}

	if len(sc.name) != 0 && sc.name != "unconnected" && sc.name != newName {
		log.Printf("%s renamed to %s\n", sc.name, newName)
	}
	sc.name = newName
	entvars.Get(sc.edictId).NetName = progsdat.AddString(newName)

	// send notification to all clients
	return protos.UpdateName_builder{
		Player:  int32(sc.id),
		NewName: newName,
	}.Build()
}

func (sc *SVClient) statusCmd(mapname string) {
	const baseVersion = 1.09
	sc.Printf("host:    %s\n", cvars.HostName.String())
	sc.Printf("version: %4.2f\n", baseVersion)
	sc.Printf("tcp/ip:  %s\n", net.Address())
	sc.Printf("map:     %s\n", mapname)
	active := 0
	for _, ac := range sv_clients {
		if ac.active {
			active++
		}
	}
	sc.Printf("players: %d active (%d max)\n\n", active, svs.maxClients)
	ntime := net.Time()
	for i, ac := range sv_clients {
		if !ac.active {
			continue
		}
		d := ntime.Sub(ac.ConnectTime())
		d = d.Truncate(time.Second)
		ev := entvars.Get(sc.edictId)
		sc.Printf("#%-2d %-16.16s  %3d  %9s\n", i+1, ac.name, int(ev.Frags), d.String())
		sc.Printf("   %s\n", ac.Address())
	}
}

func (sc *SVClient) sayCmd(team bool, a cbuf.Arguments) {
	if len(a.Args()) < 2 {
		return
	}
	text := fmt.Sprintf("\001%s: %s\n", sc.name, a.ArgumentString())
	for _, ac := range sv_clients {
		if !ac.active || !ac.spawned {
			continue
		}
		if team &&
			entvars.Get(ac.edictId).Team != entvars.Get(sc.edictId).Team {
			continue
		}
		ac.Printf(text)
	}
	if cmdl.Dedicated() {
		log.Print(text)
	}
}

func (s *Server) runClients() error {
	for i := 0; i < svs.maxClients; i++ {
		host_client = i

		hc := sv_clients[i]
		if !hc.active {
			continue
		}

		ok, err := hc.ReadClientMessage(s)
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

		if !s.paused {
			// TODO(therjak): is this pause stuff really needed?
			// always pause in single player if in console or menus
			//if svs.maxClients > 1 || keyDestination == keys.Game {
			hc.Think(s.time)
			//}
		}
	}
	return nil
}

// For debugging
func (s *Server) edictPrint(ed int) {
	if s.edicts[ed].Free {
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
func (s *Server) edictPrintEdicts() {
	if !s.Active() {
		return
	}

	fmt.Printf("%d entities\n", s.numEdicts)
	for i := 0; i < s.numEdicts; i++ {
		s.edictPrint(i)
	}
}

// For debugging, prints a single edict
func (s *Server) edictPrintEdictFunc(a cbuf.Arguments) {
	args := a.Args()
	if !s.Active() || len(args) < 2 {
		return
	}

	i := args[1].Int()
	if i < 0 || i >= s.numEdicts {
		fmt.Printf("Bad edict number\n")
		return
	}
	s.edictPrint(i)
}

// For debugging
func (s *Server) edictCount() {
	if !s.Active() {
		return
	}

	active := 0
	models := 0
	solid := 0
	step := 0
	for i := 0; i < s.numEdicts; i++ {
		if s.edicts[i].Free {
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

	fmt.Printf("num_edicts:%3d\n", s.numEdicts)
	fmt.Printf("active    :%3d\n", active)
	fmt.Printf("view      :%3d\n", models)
	fmt.Printf("touch     :%3d\n", solid)
	fmt.Printf("step      :%3d\n", step)
}
