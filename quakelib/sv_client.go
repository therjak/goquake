// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"goquake/conlog"
	"goquake/cvars"
	"goquake/execute"
	"goquake/keys"
	"goquake/net"
	"goquake/protocol"
	clc "goquake/protocol/client"
	svc "goquake/protocol/server"
	"goquake/protos"
)

type movecmd struct {
	forwardmove float32
	sidemove    float32
	upmove      float32
}

type SVClient struct {
	active     bool // false = client is free
	spawned    bool // false = don't send datagrams
	sendSignon bool // only valid before spawned

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

func CheckForNewClients() {
	for {
		con := net.CheckNewConnections()
		if con == nil {
			return
		}
		foundFree := false
		for _, c := range sv_clients {
			if c.active {
				continue
			}
			foundFree = true
			c.netConnection = con
			ConnectClient(c.id)
			break
		}
		if !foundFree {
			Error("Host_CheckForNewClients: no free clients")
		}
	}
}

func (cl *SVClient) CanSendMessage() bool {
	return cl.netConnection.CanSendMessage()
}

func (cl *SVClient) Close() {
	cl.netConnection.Close()
	cl.netConnection = nil
	cl.active = false
	cl.name = ""
	cl.oldFrags = -999999
}

func (cl *SVClient) ConnectTime() time.Duration {
	return cl.netConnection.ConnectTime()
}

func (cl *SVClient) Address() string {
	return cl.netConnection.Address()
}

func (cl *SVClient) SendMessage() int {
	return cl.netConnection.SendMessage(cl.msg.Bytes())
}

func (cl *SVClient) SendNop() {
	if cl.netConnection.SendUnreliableMessage([]byte{svc.Nop}) == -1 {
		cl.Drop(true)
	}
	cl.lastMessage = host.time
}

func (cl *SVClient) Drop(crash bool) {
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
			vm.ExecuteProgram(progsdat.Globals.ClientDisconnect)
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
		if time.Now().Sub(start) > 5*time.Second {
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

func (cl *SVClient) GetMessage() {
	cl.netConnection.GetMessage()
}

func (c *SVClient) SendServerinfo() {
	m := &c.msg
	m.WriteByte(svc.Print)
	m.WriteString(
		fmt.Sprintf("%s\nFITZQUAKE %1.2f SERVER (%d CRC)\n",
			[]byte{2}, FITZQUAKE_VERSION, progsdat.CRC))

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

	s, err := progsdat.String(EntVars(0).Message)
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
	m.WriteByte(int(EntVars(0).Sounds))
	m.WriteByte(int(EntVars(0).Sounds))

	m.WriteByte(svc.SetView)
	m.WriteShort(c.edictId)

	m.WriteByte(svc.SignonNum)
	m.WriteByte(1)

	c.sendSignon = true
	c.spawned = false
}

func (s *Server) ModelIndex(n string) int {
	if len(n) == 0 {
		return 0
	}
	for i, m := range s.modelPrecache {
		if m == n {
			return i
		}
	}
	Error("SV_ModelIndex: model %v not precached", n)
	return 0
}

// Returns false if the client should be killed
func (c *SVClient) ReadClientMessage() bool {
	hasPrefix := func(s, prefix string) bool {
		return len(s) >= len(prefix) && strings.ToLower(s[0:len(prefix)]) == prefix
	}
	for {
		data, err := c.netConnection.GetMessage()
		if err != nil {
			log.Printf("SV_ReadClientMessage: ClientGetMessage failed\n")
			return false
		}
		if len(data) == 0 {
			return true // this is the default exit
		}
		// we do not care about the first byte as it only indicates if it was
		// send reliably (1) or not (2)
		pb, err := clc.FromBytes(data[1:], sv.protocol, sv.protocolFlags)
		if err != nil {
			log.Printf("SV_ReadClientMessage: %v", err)
			return false
		}
		for _, cmd := range pb.GetCmds() {
			if !c.active {
				// a command caused an error
				return false
			}
			switch cmd.Union.(type) {
			default:
				// nop
			case *protos.Cmd_Disconnect:
				return false
			case *protos.Cmd_StringCmd:
				s := cmd.GetStringCmd()
				switch {
				default:
					conlog.Printf("%s tried to %s\n", c.name, s)
				case
					hasPrefix(s, "status"),
					hasPrefix(s, "god"),
					hasPrefix(s, "notarget"),
					hasPrefix(s, "fly"),
					hasPrefix(s, "name"),
					hasPrefix(s, "noclip"),
					hasPrefix(s, "setpos"),
					hasPrefix(s, "say"),
					hasPrefix(s, "say_team"),
					hasPrefix(s, "tell"),
					hasPrefix(s, "color"),
					hasPrefix(s, "kill"),
					hasPrefix(s, "pause"),
					hasPrefix(s, "spawn"),
					hasPrefix(s, "begin"),
					hasPrefix(s, "prespawn"),
					hasPrefix(s, "kick"),
					hasPrefix(s, "ping"),
					hasPrefix(s, "give"),
					hasPrefix(s, "ban"):
					execute.Execute(s, execute.Client, c.edictId)
				}
			case *protos.Cmd_MoveCmd:
				mc := cmd.GetMoveCmd()
				c.pingTimes[c.numPings%len(c.pingTimes)] = sv.time - mc.GetMessageTime()
				c.numPings++
				c.numPings %= len(c.pingTimes)
				ev := EntVars(c.edictId)
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

func SV_RunClients() {
	for i := 0; i < svs.maxClients; i++ {
		host_client = i

		hc := sv_clients[i]
		if !hc.active {
			continue
		}

		if !hc.ReadClientMessage() {
			hc.Drop(false)
			continue
		}

		if !hc.spawned {
			// clear client movement until a new packet is received
			hc.cmd = movecmd{0, 0, 0}
			continue
		}

		// always pause in single player if in console or menus
		if !sv.paused && svs.maxClients > 1 || keyDestination == keys.Game {
			hc.Think()
		}
	}
}
