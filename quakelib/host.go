// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"fmt"
	"io"
	"log"
	"runtime/debug"

	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/gametime"
	"goquake/math"
	"goquake/rand"
)

var (
	host  gametime.GameTime
	sRand = rand.New(0)
)

func hostInit() {
	// TODO: this is some random stuff and needs cleanup
	svs.maxClients = 1
	if cmdl.Dedicated() {
		svs.maxClients = cmdl.DedicatedNum()
	}
	if cmdl.Listen() {
		if cmdl.Dedicated() {
			debug.PrintStack()
			log.Fatalf("Only one of -dedicated or -listen can be specified")
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
	if !sv.Active() {
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
		v := int(cv.Value())
		c := math.Clamp(MIN_EDICTS, v, MAX_EDICTS)
		if v != c {
			cv.SetValue(float32(c))
		} else {
			conlog.Printf("Changes to max_edicts will not take effect until the next time a map is loaded.\n")
		}
	})
}

func serverFrame() error {
	// run the world state
	progsdat.Globals.FrameTime = float32(host.FrameTime())

	// set the time and clear the general datagram
	sv.datagram.ClearMessage()

	// check for new clients
	if err := CheckForNewClients(); err != nil {
		return err
	}

	// read client messages
	if err := SV_RunClients(); err != nil {
		return err
	}

	// move things around and think
	if !sv.paused {
		// TODO(therjak): is this pause stuff really needed?
		// always pause in single player if in console or menus
		//if svs.maxClients > 1 || keyDestination == keys.Game {
		if err := RunPhysics(); err != nil {
			return err
		}
		//}
	}
	// send all messages to the clients
	if err := sv.SendClientMessages(); err != nil {
		return err
	}
	return nil
}

func writeCvarVariables(w io.Writer) error {
	for _, c := range commandVars.All() {
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
