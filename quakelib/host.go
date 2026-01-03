// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"log"
	"runtime/debug"

	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/math"
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

func init() {
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

func (s *Server) ServerFrame() error {
	// run the world state
	progsdat.Globals.FrameTime = float32(s.gametime.FrameTime())

	// set the time and clear the general datagram
	s.datagram.ClearMessage()

	// check for new clients
	if err := s.checkForNewClients(); err != nil {
		return err
	}

	// read client messages
	if err := s.runClients(); err != nil {
		return err
	}

	// move things around and think
	if !s.paused {
		// TODO(therjak): is this pause stuff really needed?
		// always pause in single player if in console or menus
		//if svs.maxClients > 1 || keyDestination == keys.Game {
		if err := s.runPhysics(); err != nil {
			return err
		}
		//}
	}
	// send all messages to the clients
	if err := s.SendClientMessages(); err != nil {
		return err
	}
	return nil
}
