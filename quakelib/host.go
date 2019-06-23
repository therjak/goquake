package quakelib

//void CL_StopPlayback(void);
//void CL_NextDemo(void);
import "C"

import (
	"quake/cmd"
	cmdl "quake/commandline"
	"quake/conlog"
	"quake/cvar"
	"quake/cvars"
	"quake/keys"
	"quake/math"
	"quake/qtime"
)

var (
	host = Host{}
)

type Host struct {
	time      float64
	oldTime   float64
	frameTime float64
}

func Time() float64 {
	return host.time
}

func FrameTime() float64 {
	return host.frameTime
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

//export HostRealTime
func HostRealTime() C.double {
	return C.double(Time())
}

//export HostFrameTime
func HostFrameTime() C.double {
	return C.double(FrameTime())
}

//export Host_FilterTime
func Host_FilterTime() int {
	if host.UpdateTime() {
		return 1
	}
	return 0
}

//export Host_FindMaxClients
func Host_FindMaxClients() {
	svs.maxClients = 1
	if cmdl.Dedicated() {
		cls.state = ca_dedicated
		svs.maxClients = cmdl.DedicatedNum()
	} else {
		cls.state = ca_disconnected
	}
	if cmdl.Listen() {
		if cls.state == ca_dedicated {
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

	cvars.MaxEdicts.SetCallback(func(cv *cvar.Cvar) {
		// TODO: clamp it here?
		if cls.state == ca_connected || sv.active {
			conlog.Printf("Changes to max_edicts will not take effect until the next time a map is loaded.\n")
		}
	})
}

//export Host_ServerFrame
func Host_ServerFrame() {
	// run the world state
	progsdat.Globals.FrameTime = float32(host.frameTime)

	// set the time and clear the general datagram
	sv.datagram.ClearMessage()

	// check for new clients
	CheckForNewClients()

	// read client messages
	SV_RunClients()

	// move things around and think
	// always pause in single player if in console or menus
	if !sv.paused && (svs.maxClients > 1 || keyDestination == keys.Game) {
		RunPhysics()
	}

	/*
	  int i, active;
	  if (CLS_GetSignon() == SIGNONS) {
	    for (i = 0, active = 0; i < SV_NumEdicts(); i++) {
	      if (!EDICT_FREE(i)) active++;
	    }
	    if (active > 600 && dev_peakstats.edicts <= 600)
	      Con_DWarning("%i edicts exceeds standard limit of 600.\n", active);
	    dev_stats.edicts = active;
	    dev_peakstats.edicts = q_max(active, dev_peakstats.edicts);
	  }
	*/

	// send all messages to the clients
	sv.SendClientMessages()
}

// Return to looping demos
func hostStopDemo(_ []cmd.QArg, _ int) {
	if cls.state == ca_dedicated {
		return
	}
	if !cls.demoPlayback {
		return
	}
	C.CL_StopPlayback()
	cls.Disconnect()
}

// Return to looping demos
func hostDemos(_ []cmd.QArg, player int) {
	if cls.state == ca_dedicated {
		return
	}
	if cls.demoNum == -1 {
		cls.demoNum = 0
	}
	clDisconnect([]cmd.QArg{}, player)
	C.CL_NextDemo()
}

func init() {
	cmd.AddCommand("stopdemo", hostStopDemo)
	cmd.AddCommand("demos", hostDemos)
}
