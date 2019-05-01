package quakelib

//int HostClient(void);
import "C"

import (
	cmdl "quake/commandline"
	"quake/conlog"
	"quake/cvars"
	"quake/execute"
	"quake/math"
	svc "quake/protocol/server"
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

func Reset() {
	host.frameTime = 0.1
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

func init() {
}

//export HostRealTime
func HostRealTime() C.double {
	return C.double(Time())
}

//export HostFrameTime
func HostFrameTime() C.double {
	return C.double(FrameTime())
}

//export InitHostFrameTime
func InitHostFrameTime() {
	Reset()
}

//export Host_FilterTime
func Host_FilterTime() int {
	if host.UpdateTime() {
		return 1
	}
	return 0
}

//export Host_PreSpawn_f
func Host_PreSpawn_f() {
	if execute.IsSrcCommand() {
		conlog.Printf("prespawn is not valid from the console\n")
		return
	}
	cid := C.HostClient()
	c := sv_clients[int(cid)]
	if c.spawned {
		conlog.Printf("prespawn not valid -- already spawned\n")
		return
	}
	c.msg.WriteBytes(sv.signon.Bytes())
	c.msg.WriteByte(svc.SignonNum)
	c.msg.WriteByte(2)
	c.sendSignon = true
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
