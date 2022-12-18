// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"fmt"
	"log"
	"strings"
	"time"

	"goquake/cmd"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvars"
	"goquake/execute"
	"goquake/filesystem"
	"goquake/keys"
	"goquake/net"
	svc "goquake/protocol/server"
	"goquake/protos"
)

func hostFwd(a cmd.Arguments, p, s int) error {
	forwardToServer(a)
	return nil
}

func init() {
	addCommand("color", hostColor)
	addCommand("fly", hostFly)
	addCommand("god", hostGod)
	addCommand("kick", hostKick)
	addClientCommand("kick", hostKick)
	addCommand("kill", hostKill)
	addCommand("name", hostName)
	addClientCommand("name", hostName)
	addCommand("noclip", hostNoClip)
	addCommand("notarget", hostNoTarget)
	addCommand("pause", hostPause)
	addCommand("ping", hostPing)
	addCommand("say", hostSayAll)
	addClientCommand("say", hostSayAll)
	addCommand("say_team", hostSayTeam)
	addClientCommand("say_team", hostSayTeam)
	addCommand("status", hostStatus)
	addClientCommand("status", hostStatus)
	addCommand("tell", hostTell)
	addClientCommand("tell", hostTell)
	addCommand("changelevel", hostChangelevel)
	addCommand("connect", hostConnect)
	addCommand("map", hostMap)
	addCommand("mapname", hostMapName)
	addCommand("quit", func(a cmd.Arguments, p, s int) error { return hostQuit() })
	addCommand("restart", hostRestart)
	addCommand("version", hostVersion)

	addCommand("setpos", hostFwd)
	addCommand("give", hostFwd)
	addCommand("edict", hostFwd)
	addCommand("edicts", hostFwd)
	addCommand("edictcount", hostFwd)
}

func hostQuit() error {
	if keyDestination != keys.Console && !cmdl.Dedicated() {
		enterQuitMenu()
		return nil
	}
	if err := cls.Disconnect(); err != nil {
		return err
	}

	if err := hostShutdownServer(false); err != nil {
		return err
	}

	Sys_Quit()
	return nil
}

func qFormatI(b int32) string {
	if b == 0 {
		return "OFF"
	}
	return "ON"
}

func hostVersion(a cmd.Arguments, p, s int) error {
	conlog.Printf("GoQuake Version %1.2f.%d\n", GoQuakeVersion, GoQuakePatch)
	return nil
}

func hostGod(a cmd.Arguments, playerEdictId, s int) error {
	args := a.Args()
	if len(args) > 2 {
		conlog.Printf("god [value] : toggle god mode. values: 0 = off, 1 = on\n")
		return nil
	}
	forwardToServer(a)
	return nil
}

func hostNoTarget(a cmd.Arguments, playerEdictId, s int) error {
	args := a.Args()
	if len(args) > 2 {
		conlog.Printf("notarget [value] : toggle notarget mode. values: 0 = off, 1 = on\n")
		return nil
	}
	forwardToServer(a)
	return nil
}

func hostFly(a cmd.Arguments, playerEdictId, s int) error {
	args := a.Args()
	if len(args) > 2 {
		conlog.Printf("fly [value] : toggle fly mode. values: 0 = off, 1 = on\n")
		return nil
	}
	forwardToServer(a)
	return nil
}

func hostColor(a cmd.Arguments, p, s int) error {
	args := a.Args()[1:]
	c := int(cvars.ClientColor.Value())
	t := c >> 4
	b := c & 0x0f
	if len(args) == 0 {
		conlog.Printf("\"color\" is \"%d %d\"\n", t, b)
		conlog.Printf("color <0-13> [0-13]\n")
		return nil
	}
	t = args[0].Int()
	b = t
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
	c = t*16 + b
	cvars.ClientColor.SetValue(float32(c))
	if cls.state == ca_connected {
		forwardToServer(a)
	}
	return nil
}

func hostPause(a cmd.Arguments, playerEdictId, s int) error {
	if cls.demoPlayback {
		cls.demoPaused = !cls.demoPaused
		cl.paused = cls.demoPaused
		return nil
	}
	forwardToServer(a)
	return nil
}

func concatArgs(args []cmd.QArg) string {
	n := len(args)
	for i := 0; i < len(args); i++ {
		n += len(args[i].String())
	}
	var b strings.Builder
	b.Grow(n)
	b.WriteString(args[0].String())
	for _, s := range args[1:] {
		b.WriteString(" ")
		b.WriteString(s.String())
	}
	b.WriteString("\n")
	return b.String()
}

func hostTell(a cmd.Arguments, p, s int) error {
	args := a.Args()[1:]
	if s == execute.Command {
		forwardToServer(a)
		return nil
	}

	if len(args) < 2 {
		// need at least destination and message
		return nil
	}

	cn := HostClient().name
	ms := a.ArgumentString()
	text := fmt.Sprintf("%s: %s", cn, ms)

	for _, c := range sv_clients {
		if !c.active || !c.spawned {
			continue
		}
		if strings.ToLower(c.name) != strings.ToLower(args[0].String()) {
			continue
		}
		// TODO: We check without case check. Are names unique ignoring the case?
		c.Printf(text)
	}
	return nil
}

func hostSay(team bool, a cmd.Arguments, s int) {
	fromServer := false
	if s == execute.Command {
		team = false
		fromServer = true
	}
	ms := a.ArgumentString()
	text := func() string {
		if fromServer {
			return fmt.Sprintf("\001<%s> %s", cvars.HostName.String(), ms)
		} else {
			return fmt.Sprintf("\001%s: %s", HostClient().name, ms)
		}
	}()
	for _, c := range sv_clients {
		if !c.active || !c.spawned {
			continue
		}
		if team && cvars.TeamPlay.Bool() &&
			entvars.Get(c.edictId).Team != entvars.Get(HostClient().edictId).Team {
			continue
		}
		c.Printf(text)
	}
	if cmdl.Dedicated() {
		log.Printf(text)
	}
}

func hostSayAll(a cmd.Arguments, p, s int) error {
	if len(a.Args()) < 2 {
		return nil
	}
	if s == execute.Command {
		if !cmdl.Dedicated() {
			forwardToServer(a)
			return nil
		}
	}
	hostSay(false, a, s)
	return nil
}

func hostSayTeam(a cmd.Arguments, p, s int) error {
	// say_team
	if len(a.Args()) < 2 {
		return nil
	}
	if s == execute.Command {
		if !cmdl.Dedicated() {
			forwardToServer(a)
			return nil
		}
	}
	hostSay(true, a, s)
	return nil
}

func hostPing(a cmd.Arguments, p, s int) error {
	forwardToServer(a)
	return nil
}

func hostNoClip(a cmd.Arguments, playerEdictId, s int) error {
	args := a.Args()
	if len(args) > 2 {
		conlog.Printf("noclip [value] : toggle noclip mode. values: 0 = off, 1 = on\n")
		return nil
	}
	forwardToServer(a)
	return nil
}

func hostName(a cmd.Arguments, p, s int) error {
	args := a.Args()[1:]
	if len(args) == 0 {
		conlog.Printf("\"name\" is %q\n", cvars.ClientName.String())
		return nil
	}
	newName := func() string {
		if len(args) == 1 {
			return args[0].String()
		}
		b := strings.Builder{}
		b.WriteString(args[0].String())
		for _, a := range args[1:] {
			b.WriteRune(' ')
			b.WriteString(a.String())
		}
		return b.String()
	}()
	// client_t structure says name[32]
	if len(newName) > 15 {
		newName = newName[:15]
	}

	if s == execute.Command {
		if cvars.ClientName.String() == newName {
			return nil
		}
		cvars.ClientName.SetByString(newName)
		if cls.state == ca_connected {
			forwardToServer(a)
		}
		return nil
	}

	c := HostClient()
	if len(c.name) != 0 && c.name != "unconnected" && c.name != newName {
		conlog.Printf("%s renamed to %s\n", c.name, newName)
	}
	c.name = newName
	entvars.Get(c.edictId).NetName = progsdat.AddString(newName)

	// send notification to all clients
	un := &protos.UpdateName{
		Player:  int32(c.id),
		NewName: newName,
	}
	svc.WriteUpdateName(un, sv.protocol, sv.protocolFlags, &sv.reliableDatagram)
	return nil
}

func hostKill(a cmd.Arguments, playerEdictId, s int) error {
	forwardToServer(a)
	return nil
}

func hostStatus(a cmd.Arguments, p, s int) error {
	const baseVersion = 1.09
	if s == execute.Command {
		if !sv.active {
			forwardToServer(a)
			return nil
		}

	}
	printf := func() func(format string, v ...interface{}) {
		if s == execute.Command {
			return conlog.Printf
		}
		return HostClient().Printf
	}()

	printf("host:    %s\n", cvars.HostName.String())
	printf("version: %4.2f\n", baseVersion)
	printf("tcp/ip:  %s\n", net.Address())
	printf("map:     %s\n", sv.name)
	active := 0
	for _, c := range sv_clients {
		if c.active {
			active++
		}
	}
	printf("players: %d active (%d max)\n\n", active, svs.maxClients)
	ntime := net.Time()
	for i, c := range sv_clients {
		if !c.active {
			continue
		}
		d := ntime - c.ConnectTime()
		d = d.Truncate(time.Second)
		ev := entvars.Get(c.edictId)
		printf("#%-2d %-16.16s  %3d  %9s\n", i+1, c.name, int(ev.Frags), d.String())
		printf("   %s\n", c.Address())
	}
	return nil
}

func hostMapName(a cmd.Arguments, p, s int) error {
	switch {
	case cmdl.Dedicated():
		forwardToServer(a)
	case cls.state == ca_connected:
		conlog.Printf("\"mapname\" is %q\n", cl.mapName)
	default:
		conlog.Printf("no map loaded\n")
	}
	return nil
}

// This only happens at the end of a game, not between levels
func hostShutdownServer(crash bool) error {
	if !sv.active {
		return nil
	}

	sv.active = false

	// stop all client sounds immediately
	if cls.state == ca_connected {
		if err := cls.Disconnect(); err != nil {
			return err
		}
	}

	// flush any pending messages - like the score!!!
	end := time.Now().Add(3 * time.Second)
	count := 1
	for count != 0 {
		count = 0
		for _, c := range sv_clients {
			if c.active && c.msg.HasMessage() {
				if c.CanSendMessage() {
					c.SendMessage()
					c.msg.ClearMessage()
				} else {
					if err := c.GetMessage(); err != nil {
						return err
					}
					count++
				}
			}
		}
		if time.Now().After(end) {
			break
		}
	}

	// make sure all the clients know we're disconnecting
	SendToAll([]byte{svc.Disconnect})

	for _, c := range sv_clients {
		if c.active {
			if err := c.Drop(crash); err != nil {
				return nil
			}
		}
	}

	sv.worldModel = nil

	CreateSVClients()
	return nil
}

// Kicks a user off of the server
func hostKick(a cmd.Arguments, playerEdictId, s int) error {
	args := a.Args()[1:]
	if len(args) == 0 {
		return nil
	}
	if s == execute.Command {
		if !sv.active {
			forwardToServer(a)
			return nil
		}
	} else if progsdat.Globals.DeathMatch != 0 {
		return nil
	}

	var toKick *SVClient
	var message string

	if len(args) > 1 && args[0].String() == "#" {
		i := args[1].Int() - 1
		if i < 0 || i >= svs.maxClients {
			return nil
		}
		toKick = sv_clients[i]
		if !toKick.active {
			return nil
		}
		if len(args) > 2 {
			// skip # and number
			message = concatArgs(args[2:])
		}
	} else {
		for _, c := range sv_clients {
			if !c.active {
				continue
			}
			if c.name == args[0].String() {
				toKick = c
				if len(args) > 1 {
					// skip name
					message = concatArgs(args[1:])
				}
				break
			}
		}
	}
	if toKick == nil {
		return nil
	}
	if playerEdictId == toKick.edictId {
		// can't kick yourself!
		return nil
	}
	who := func() string {
		if s == execute.Command {
			if cmdl.Dedicated() {
				return "Console"
			} else {
				return cvars.ClientName.String()
			}
		}
		return HostClient().name
	}()

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

// User command to connect to server
func hostConnect(a cmd.Arguments, p, s int) error {
	args := a.Args()[1:]
	if len(args) == 0 {
		return nil
	}
	// stop demo loop in case this fails
	cls.demoNum = -1
	if cls.demoPlayback {
		cls.demoPlayback = false
		if err := cls.Disconnect(); err != nil {
			return err
		}
	}
	if err := clEstablishConnection(args[0].String()); err != nil {
		return err
	}
	clientReconnect()
	return nil
}

// handle a
// map <servername>
// command from the console.  Active clients are kicked off.
func hostMap(a cmd.Arguments, p, s int) error {
	args := a.Args()[1:]
	if len(args) == 0 {
		// no map name given
		if cmdl.Dedicated() {
			if sv.active {
				conlog.Printf("Current map: %s\n", sv.name)
			} else {
				conlog.Printf("Server not active\n")
			}
		} else if cls.state == ca_connected {
			conlog.Printf("Current map: %s ( %s )\n", cl.levelName, cl.mapName)
		} else {
			conlog.Printf("map <levelname>: start a new server\n")
		}
		return nil
	}

	if s != execute.Command {
		return nil
	}

	// stop demo loop in case this fails
	cls.demoNum = -1

	if err := cls.Disconnect(); err != nil {
		return err
	}
	if err := hostShutdownServer(false); err != nil {
		return err
	}

	if !cmdl.Dedicated() {
		inputActivate()
	}

	keyDestination = keys.Game // remove console or menu
	screen.BeginLoadingPlaque()

	svs.serverFlags = 0 // haven't completed an episode yet

	mapName := args[0].String()
	mapName = strings.TrimSuffix(mapName, ".bsp")

	if err := sv.SpawnServer(mapName); err != nil {
		return err
	}
	if !sv.active {
		return nil
	}

	if !cmdl.Dedicated() {
		var b strings.Builder
		for _, a := range args[1:] {
			b.WriteString(a.String())
			b.WriteRune(' ')
		}
		cls.spawnParms = b.String()

		if err := clEstablishConnection("local"); err != nil {
			return err
		}
		clientReconnect()
	}
	return nil
}

// Goes to a new map, taking all clients along
func hostChangelevel(a cmd.Arguments, p, s int) error {
	args := a.Args()[1:]
	if len(args) != 1 {
		conlog.Printf("changelevel <levelname> : continue game on a new level\n")
		return nil
	}

	if cls.demoPlayback || !sv.active {
		conlog.Printf("Only the server may changelevel\n")
		return nil
	}
	level := args[0].String()
	if _, err := filesystem.GetFile(fmt.Sprintf("maps/%s.bsp", level)); err != nil {
		return fmt.Errorf("cannot find map %s", level)
	}
	if !cmdl.Dedicated() {
		inputActivate()
	}

	// remove console or menu
	keyDestination = keys.Game
	if err := SV_SaveSpawnparms(); err != nil {
		return err
	}
	if err := sv.SpawnServer(level); err != nil {
		return err
	}
	// also issue an error if spawn failed -- O.S.
	if !sv.active {
		return fmt.Errorf("cannot run map %s", level)
	}
	return nil
}

// Restarts the current server for a dead player
func hostRestart(a cmd.Arguments, p, s int) error {
	if cls.demoPlayback || !sv.active {
		return nil
	}
	if s != execute.Command {
		return nil
	}
	mapname := sv.name // sv.name gets cleared in spawnserver
	if err := sv.SpawnServer(mapname); err != nil {
		return err
	}

	if !sv.active {
		return fmt.Errorf("cannot restart map %s", mapname)
	}
	return nil
}
