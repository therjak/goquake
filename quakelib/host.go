package quakelib

//void _Host_Frame();
import "C"

import (
	"bytes"
	"io/ioutil"
	"log"
	"math/rand"
	"path/filepath"
	"quake/cbuf"
	"quake/cmd"
	cmdl "quake/commandline"
	"quake/conlog"
	"quake/cvar"
	"quake/cvars"
	"quake/input"
	"quake/keys"
	"quake/math"
	"quake/net"
	"quake/qtime"
	"quake/snd"
	"time"
)

var (
	host = Host{}
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

func serverFrame() {
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
		    active := 0
			  for (i = 0; i < SV_NumEdicts(); i++) {
			    if (!sv.edicts[i].Free){
			   	  active++;
			   	}
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
	cl.stopPlayback()
	cls.Disconnect()
}

// Return to looping demos
func hostDemos(_ []cmd.QArg, _ int) {
	if cls.state == ca_dedicated {
		return
	}
	if cls.demoNum == -1 {
		cls.demoNum = 0
	}
	clientDisconnect()
	CL_NextDemo()
}

func init() {
	cmd.AddCommand("stopdemo", hostStopDemo)
	cmd.AddCommand("demos", hostDemos)
}

var (
	frameCount          = 0
	frameCountStartTime time.Time
)

// Runs all active servers
func executeFrame() {
	// keep the random time dependent
	rand.Seed(time.Now().UnixNano())

	// decide the simulation time
	if !host.UpdateTime() {
		return // don't run too fast, or packets will flood out
	}

	// get new key events
	updateKeyDest()
	updateInputMode()
	sendKeyEvents()

	// process console commands
	cbuf.Execute(sv_player)

	net.SetTime()

	// if running the server locally, make intentions now
	if sv.active {
		CL_SendCmd()
	}

	// server operations

	// check for commands typed to the host
	hostGetConsoleCommands()

	if sv.active {
		serverFrame()
	}

	// client operations

	// if running the server remotely, send intentions now after
	// the incoming messages have been read
	if !sv.active {
		CL_SendCmd()
	}

	C._Host_Frame()
	/*
		  // fetch results from server
		  if (CLS_GetState() == ca_connected) {
				CL_ReadFromServer();
			}

		  // update video
		  if (Cvar_GetValue(&host_speeds)) time1 = Sys_DoubleTime();

		  screen.Update()

		  ParticlesRun();  // johnfitz -- seperated from rendering

		  if (Cvar_GetValue(&host_speeds)) time2 = Sys_DoubleTime();

		  // update audio
		  // adds music raw samples and/or advances midi driver
		  if (CLS_GetSignon() == SIGNONS) {
		    S_Update(r_origin, vpn, vright, vup);
		    CL_DecayLights();
		  } else
		    S_Update(vec3_origin, vec3_origin, vec3_origin, vec3_origin);

		  if (Cvar_GetValue(&host_speeds)) {
		    pass1 = (time1 - time3) * 1000;
		    time3 = Sys_DoubleTime();
		    pass2 = (time2 - time1) * 1000;
		    pass3 = (time3 - time2) * 1000;
		    Con_Printf("%3i tot %3i server %3i gfx %3i snd\n", pass1 + pass2 + pass3,
		               pass1, pass2, pass3);
		  }

			// this is for demo syncing
		  host_framecount++;
	*/
	host.frameCount++
}

func hostFrame() {
	defer func() {
		if r := recover(); r != nil {
			frameCount = 0
			// something bad happened, or the server disconnected
			conlog.Printf("%v\n", r)
			return
		}
	}()
	if !cvars.ServerProfile.Bool() {
		executeFrame()
		return
	}
	if frameCount == 0 {
		frameCountStartTime = time.Now()
	}

	executeFrame()

	frameCount++
	if frameCount < 1000 {
		return
	}

	end := time.Now()
	div := end.Sub(frameCountStartTime)
	frameCount = 0

	clientNum := 0
	for i := 0; i < svs.maxClients; i++ {
		if sv_clients[i].active {
			clientNum++
		}
	}
	conlog.Printf("serverprofile: %2d clients %v\n", clientNum, div.String())
}

// Writes key bindings and archived cvars to config.cfg
func HostWriteConfiguration() {
	// dedicated servers initialize the host but don't parse and set the
	// config.cfg cvars
	if cmdl.Dedicated() {
		return
	}

	// write actual current mode to config file, in case cvars were messed with
	syncVideoCvars()

	var b bytes.Buffer
	writeKeyBindings(&b)
	writeCvarVariables(&b)

	b.WriteString("vid_restart\n")
	if input.MLook.Down() {
		b.WriteString("+mlook\n")
	}

	filename := filepath.Join(gameDirectory, "config.cfg")
	err := ioutil.WriteFile(filename, b.Bytes(), 0644)
	if err != nil {
		conlog.Printf("Couldn't write config.cfg.\n")
	}
}

//export HostSetInitialized
func HostSetInitialized() {
	host.initialized = true
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
		HostWriteConfiguration()
	}
	net.Shutdown()
	if cls.state != ca_dedicated {
		if console.initialized {
			history.Save()
		}
		snd.Shutdown()
		videoShutdown()
	}
}
