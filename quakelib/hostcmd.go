// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"fmt"
	"strings"

	"goquake/cmd"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvars"
	"goquake/filesystem"
	"goquake/keys"
	"goquake/net"
	"goquake/version"
)

func hostFwd(a cmd.Arguments) error {
	forwardToServer(a)
	return nil
}

func init() {
	addCommand("color", hostColor)
	addCommand("fly", hostFly)
	addCommand("god", hostGod)
	addCommand("kick", hostKick)
	addCommand("name", hostName)
	addCommand("noclip", hostNoClip)
	addCommand("notarget", hostNoTarget)
	addCommand("pause", hostPause)
	addCommand("say", hostSay)
	addCommand("say_team", hostSay)
	addCommand("tell", hostTell)
	addCommand("changelevel", hostChangelevel)
	addCommand("connect", hostConnect)
	addCommand("map", hostMap)
	addCommand("mapname", hostMapName)
	addCommand("quit", func(a cmd.Arguments) error { return hostQuit() })
	addCommand("restart", hostRestart)
	addCommand("version", hostVersion)
	addCommand("stopdemo", hostStopDemo)
	addCommand("demos", hostDemos)

	addCommand("setpos", hostFwd)
	addCommand("give", hostFwd)
	addCommand("edict", hostFwd)
	addCommand("edicts", hostFwd)
	addCommand("edictcount", hostFwd)
	addCommand("status", hostFwd)
	addCommand("ping", hostFwd)
	addCommand("kill", hostFwd)
}

// Return to looping demos
func hostStopDemo(a cmd.Arguments) error {
	if cmdl.Dedicated() {
		return nil
	}
	if !cls.demoPlayback {
		return nil
	}
	cls.stopPlayback()
	if err := cls.Disconnect(); err != nil {
		return err
	}
	return nil
}

// Return to looping demos
func hostDemos(a cmd.Arguments) error {
	if cmdl.Dedicated() {
		return nil
	}
	if cls.demoNum == -1 {
		cls.demoNum = 1
	}
	if err := clientDisconnect(); err != nil {
		return err
	}
	if err := CL_NextDemo(); err != nil {
		return err
	}
	return nil
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

func hostVersion(a cmd.Arguments) error {
	conlog.Printf("GoQuake Version %1.2f.%d\n", version.Base, version.Patch)
	return nil
}

func hostGod(a cmd.Arguments) error {
	args := a.Args()
	if len(args) > 2 {
		conlog.Printf("god [value] : toggle god mode. values: 0 = off, 1 = on\n")
		return nil
	}
	forwardToServer(a)
	return nil
}

func hostNoTarget(a cmd.Arguments) error {
	args := a.Args()
	if len(args) > 2 {
		conlog.Printf("notarget [value] : toggle notarget mode. values: 0 = off, 1 = on\n")
		return nil
	}
	forwardToServer(a)
	return nil
}

func hostFly(a cmd.Arguments) error {
	args := a.Args()
	if len(args) > 2 {
		conlog.Printf("fly [value] : toggle fly mode. values: 0 = off, 1 = on\n")
		return nil
	}
	forwardToServer(a)
	return nil
}

func hostColor(a cmd.Arguments) error {
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

func hostPause(a cmd.Arguments) error {
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

func hostTell(a cmd.Arguments) error {
	if len(a.Args()) < 3 {
		// need at least destination and message
		return nil
	}
	forwardToServer(a)
	return nil
}

func hostSay(a cmd.Arguments) error {
	if len(a.Args()) < 2 {
		return nil
	}
	forwardToServer(a)
	return nil
}

func hostNoClip(a cmd.Arguments) error {
	args := a.Args()
	if len(args) > 2 {
		conlog.Printf("noclip [value] : toggle noclip mode. values: 0 = off, 1 = on\n")
		return nil
	}
	forwardToServer(a)
	return nil
}

func hostName(a cmd.Arguments) error {
	if len(a.Args()) < 2 {
		conlog.Printf("\"name\" is %q\n", cvars.ClientName.String())
		return nil
	}
	newName := a.ArgumentString()
	if len(newName) > 15 {
		newName = newName[:15]
	}
	if cvars.ClientName.String() == newName {
		return nil
	}
	cvars.ClientName.SetByString(newName)
	if cls.state == ca_connected {
		forwardToServer(a)
	}
	return nil
}

func hostMapName(a cmd.Arguments) error {
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
	// stop all client sounds immediately
	if cls.state == ca_connected {
		if err := cls.Disconnect(); err != nil {
			return err
		}
	}
	return ShutdownServer(crash)
}

// Kicks a user off of the server
func hostKick(a cmd.Arguments) error {
	args := a.Args()
	if len(args) < 2 {
		return nil
	}
	forwardToServer(a)
	return nil
}

// User command to connect to server
func hostConnect(a cmd.Arguments) error {
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
func hostMap(a cmd.Arguments) error {
	args := a.Args()[1:]
	if len(args) == 0 {
		// no map name given
		if cmdl.Dedicated() {
			if sv.Active() {
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

	if err := sv.SpawnServer(mapName, sv_protocol); err != nil {
		return err
	}

	if !cmdl.Dedicated() {
		if err := clEstablishConnection(net.LocalAddress); err != nil {
			return err
		}
		clientReconnect()
	}
	return nil
}

// Goes to a new map, taking all clients along
func hostChangelevel(a cmd.Arguments) error {
	args := a.Args()[1:]
	if len(args) != 1 {
		conlog.Printf("changelevel <levelname> : continue game on a new level\n")
		return nil
	}

	if cls.demoPlayback || !sv.Active() {
		conlog.Printf("Only the server may changelevel\n")
		return nil
	}
	level := args[0].String()
	if _, err := filesystem.GetFile(fmt.Sprintf("maps/%s.bsp", level)); err != nil {
		return fmt.Errorf("cannot find map %s", level)
	}
	if err := sv.ChangeLevel(level, sv_protocol); err != nil {
		return err
	}
	if !cmdl.Dedicated() {
		inputActivate()
	}
	// remove console or menu
	keyDestination = keys.Game

	return nil
}

// Restarts the current server for a dead player
func hostRestart(a cmd.Arguments) error {
	if cls.demoPlayback {
		return nil
	}
	if err := sv.ResetServer(); err != nil {
		return err
	}
	return nil
}
