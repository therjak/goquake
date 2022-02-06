// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import "C"

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"

	"goquake/cmd"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/input"
	"goquake/keys"
	"goquake/math"
	"goquake/net"
	"goquake/qtime"
	"goquake/rand"
	"goquake/snd"
)

var (
	host  = Host{}
	sRand = rand.New(0)
)

type Host struct {
	time        float64
	oldTime     float64
	frameTime   float64
	initialized bool
	isDown      bool
	frameCount  int
}

func Time() float64 {
	return host.time
}

func (h *Host) Reset() {
	h.frameTime = 0.1
}

// UpdateTime updates the host time.
// Returns false if it would exceed max fps
func (h *Host) UpdateTime() bool {
	h.time = qtime.QTime().Seconds()
	maxFPS := math.Clamp(10.0, float64(cvars.HostMaxFps.Value()), 1000.0)
	if !cls.timeDemo && (h.time-h.oldTime < 1/maxFPS) {
		return false
	}
	h.frameTime = h.time - h.oldTime
	h.oldTime = h.time

	if cvars.HostTimeScale.Value() > 0 {
		h.frameTime *= float64(cvars.HostTimeScale.Value())
	} else if cvars.HostFrameRate.Value() > 0 {
		h.frameTime = float64(cvars.HostFrameRate.Value())
	} else {
		h.frameTime = math.Clamp(0.001, h.frameTime, 0.1)
	}
	return true
}

func hostInit() {
	// TODO: this is some random stuff and needs cleanup
	// Like why is cls.state here? Do we need cls.state at all?
	svs.maxClients = 1
	if cmdl.Dedicated() {
		svs.maxClients = cmdl.DedicatedNum()
	}
	cls.state = ca_disconnected
	if cmdl.Listen() {
		if cmdl.Dedicated() {
			Error("Only one of -dedicated or -listen can be specified")
		}
		svs.maxClients = cmdl.ListenNum()
	}
	if svs.maxClients < 1 {
		svs.maxClients = 8
	} else if svs.maxClients > 16 {
		svs.maxClients = 16
	}

	svs.maxClientsLimit = svs.maxClients
	if svs.maxClientsLimit < 4 {
		svs.maxClientsLimit = 4
	}
	CreateSVClients()
	if svs.maxClients > 1 {
		cvars.DeathMatch.SetByString("1")
	} else {
		cvars.DeathMatch.SetByString("0")
	}
}

func hostCallbackNotify(cv *cvar.Cvar) {
	if !sv.active {
		return
	}
	SV_BroadcastPrintf("\"%s\" changed to \"%s\"\n", cv.Name(), cv.String())
}

func init() {
	cvars.ServerGravity.SetCallback(hostCallbackNotify)
	cvars.ServerFriction.SetCallback(hostCallbackNotify)
	cvars.ServerMaxSpeed.SetCallback(hostCallbackNotify)
	cvars.TimeLimit.SetCallback(hostCallbackNotify)
	cvars.FragLimit.SetCallback(hostCallbackNotify)
	cvars.TeamPlay.SetCallback(hostCallbackNotify)
	cvars.NoExit.SetCallback(hostCallbackNotify)

	cvars.Coop.SetCallback(func(cv *cvar.Cvar) {
		conlog.Printf("Changed coop: %v\n", cv.Bool())
		if cv.Bool() {
			cvars.DeathMatch.SetByString("0")
		}
	})
	cvars.DeathMatch.SetCallback(func(cv *cvar.Cvar) {
		conlog.Printf("Changed deathmatch: %v\n", cv.Bool())
		if cv.Bool() {
			cvars.Coop.SetByString("0")
		}
	})

	cvars.MaxEdicts.SetCallback(func(cv *cvar.Cvar) {
		// TODO: clamp it here?
		if cls.state == ca_connected || sv.active {
			conlog.Printf("Changes to max_edicts will not take effect until the next time a map is loaded.\n")
		}
	})
}

func serverFrame() error {
	// run the world state
	progsdat.Globals.FrameTime = float32(host.frameTime)

	// set the time and clear the general datagram
	sv.datagram.ClearMessage()

	// check for new clients
	CheckForNewClients()

	// read client messages
	if err := SV_RunClients(); err != nil {
		return err
	}

	// move things around and think
	// always pause in single player if in console or menus
	if !sv.paused && (svs.maxClients > 1 || keyDestination == keys.Game) {
		if err := RunPhysics(); err != nil {
			return err
		}
	}
	// send all messages to the clients
	if err := sv.SendClientMessages(); err != nil {
		return err
	}
	return nil
}

// Return to looping demos
func hostStopDemo(_ []cmd.QArg, _ int) error {
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
func hostDemos(_ []cmd.QArg, _ int) error {
	if cmdl.Dedicated() {
		return nil
	}
	if cls.demoNum == -1 {
		cls.demoNum = 1
	}
	if err := clientDisconnect(); err != nil {
		return err
	}
	CL_NextDemo()
	return nil
}

func init() {
	addCommand("stopdemo", hostStopDemo)
	addCommand("demos", hostDemos)
}

func writeCvarVariables(w io.Writer) error {
	for _, c := range cvar.All() {
		if c.Archive() {
			if c.UserDefined() || c.SetA() {
				if _, err := w.Write([]byte("seta ")); err != nil {
					return err
				}
			}
			if _, err := w.Write([]byte(fmt.Sprintf("%s \"%s\"\n", c.Name(), c.String()))); err != nil {
				return err
			}
		}
	}
	return nil
}

// Writes key bindings and archived cvars to config.cfg
func HostWriteConfiguration() error {
	// dedicated servers initialize the host but don't parse and set the
	// config.cfg cvars
	if cmdl.Dedicated() {
		return nil
	}

	var b bytes.Buffer
	if err := writeKeyBindings(&b); err != nil {
		return fmt.Errorf("Couldn't write config.cfg: %w\n", err)
	}
	if err := writeCvarVariables(&b); err != nil {
		return fmt.Errorf("Couldn't write config.cfg: %w\n", err)
	}

	b.WriteString("vid_restart\n")
	if input.MLook.Down() {
		b.WriteString("+mlook\n")
	}

	filename := filepath.Join(gameDirectory, "config.cfg")
	err := ioutil.WriteFile(filename, b.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Couldn't write config.cfg: %w\n", err)
	}
	return nil
}

//export Host_Shutdown
func Host_Shutdown() {
	host.Shutdown()
}

func (h *Host) Shutdown() {
	if h.isDown {
		log.Printf("recursive shutdown\n")
		return
	}
	h.isDown = true
	screen.disabled = true
	if host.initialized {
		if err := HostWriteConfiguration(); err != nil {
			log.Printf(err.Error())
		}
	}
	net.Shutdown()
	if !cmdl.Dedicated() {
		if console.initialized {
			history.Save()
		}
		snd.Shutdown()
		videoShutdown()
	}
}
