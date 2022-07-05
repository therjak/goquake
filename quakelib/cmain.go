// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

/*
void R_Init();
*/
import "C"

import (
	"log"
	"time"

	"goquake/cbuf"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/net"
	"goquake/snd"
	"goquake/wad"
	"goquake/window"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	VERSION              = 1.09
	GLQUAKE_VERSION      = 1.00
	FITZQUAKE_VERSION    = 0.85
	QUAKESPASM_VERSION   = 0.92
	QUAKESPASM_VER_PATCH = 2
)

type serverState int

const (
	serverRunning      serverState = 0
	serverDisconnected serverState = 1
)

func CallCMain() error {
	v := sdl.Version{}
	sdl.GetVersion(&v)
	log.Printf("Found SDL version %d.%d.%d\n", v.Major, v.Minor, v.Patch)
	if err := sdl.Init(0); err != nil {
		return err
	}
	defer sdl.Quit()

	vm = NewVirtualMachine()

	log.Printf("Quake %1.2f (c) id Software\n", VERSION)
	log.Printf("GLQuake %1.2f (c) id Software\n", GLQUAKE_VERSION)
	log.Printf("FitzQuake %1.2f (c) John Fitzgibbons\n", FITZQUAKE_VERSION)
	log.Printf("FitzQuake SDL port (c) SleepwalkR, Baker\n")
	log.Printf("QuakeSpasm %1.2f.%d (c) Ozkan Sezer, Eric Wasylishen & others\n", QUAKESPASM_VERSION, QUAKESPASM_VER_PATCH)
	log.Printf("Host_Init\n")

	filesystemInit()
	hostInit()
	if err := wad.Load(); err != nil {
		return err
	}
	if !cmdl.Dedicated() {
		history.Load()
		consoleInit()
	}
	networkInit()
	serverInit()

	if !cmdl.Dedicated() {
		// ExtraMaps_Init();
		// Modlist_Init();
		// DemoList_Init();
		if err := videoInit(); err != nil {
			return err
		}
		shaderInit()
		textureManagerInit()
		drawInit()
		screen.initialized = true
		C.R_Init()
		SkyInit()
		particlesInit()
		setClearColor(cvars.RClearColor)
		soundInit()
		statusbar.LoadPictures()
		clientInit()
	}

	host.initialized = true
	conlog.Printf("\n========= Quake Initialized =========\n\n")
	cbuf.AddText("alias startmap_sp \"map start\"\n")
	cbuf.AddText("alias startmap_dm \"map start\"\n")

	if !cmdl.Dedicated() {
		cbuf.InsertText("exec quake.rc\n")
		// two leading newlines because the command buffer swallows one of them.
		cbuf.AddText("\n\nvid_unlock\n")
	} else {
		cbuf.AddText("exec autoexec.cfg\n")
		cbuf.AddText("stuffcmds")
		cbuf.Execute(0)
		if !sv.active {
			cbuf.AddText("startmap_dm\n")
		}
	}

	r := newRunner(cmdl.Dedicated())
	r.run()
	return nil
}

func shaderInit() {
	// All our shaders:
	CreateAliasDrawer()
	CreateBrushDrawer()
	CreateSpriteDrawer()
	CreateSkyDrawer()
	CreateParticleDrawer()
	CreatePostProcess()
	CreateConeDrawer()
	CreateUiDrawer()
}

var (
	quitChan chan bool
)

func init() {
	quitChan = make(chan bool, 2)
}

func runWindow() {
	// If we have no input focus at all, sleep a bit
	if !window.InputFocus() || cl.paused {
		time.Sleep(16 * time.Millisecond)
	}
	// If we're minimised, sleep a bit more
	if window.Minimized() {
		window.SetSkipUpdates(true)
		time.Sleep(32 * time.Millisecond)
	} else {
		window.SetSkipUpdates(false)
	}
}

type runner struct {
	handleWindow func()
	m            *measure
}

func newRunner(dedicated bool) *runner {
	switch dedicated {
	case true:
		return &runner{
			handleWindow: func() {},
			m:            newMeasure(),
		}
	default:
		return &runner{
			handleWindow: runWindow,
			m:            newMeasure(),
		}
	}
}

func (r *runner) run() {
	oldtime := time.Now()
	for {
		select {
		case <-quitChan:
			return
		default:
			r.handleWindow()
			timediff := time.Since(oldtime)
			oldtime = time.Now()
			if !cls.timeDemo {
				w := time.Duration(cvars.Throttle.Value()*float32(time.Second)) - timediff
				time.Sleep(w)
			}
			r.frame()
		}
	}
}

func (r *runner) frame() {
	r.m.startMeasure()
	executeFrame()
	r.m.endMeasure()
}

var (
	executeFrameTime time.Time
)

// Runs all active servers
func executeFrame() {
	// keep the random time dependent
	sRand.NewSeed(uint32(time.Now().UnixNano()))

	// decide the simulation time
	if !host.UpdateTime() {
		return // don't run too fast, or packets will flood out
	}

	// get new key events
	updateKeyDest()
	updateInputMode()
	sendKeyEvents()

	// process console commands
	cbuf.Execute(0) // TODO: this needs to be the local player, not 0

	net.SetTime()

	// if running the server locally, make intentions now
	if sv.active {
		if err := CL_SendCmd(); err != nil {
			HostError(err)
		}
	}

	// server operations

	// check for commands typed to the host
	hostGetConsoleCommands()

	if sv.active {
		if err := serverFrame(); err != nil {
			HostError(err)
		}
	}

	// client operations

	// if running the server remotely, send intentions now after
	// the incoming messages have been read
	if !sv.active {
		if err := CL_SendCmd(); err != nil {
			HostError(err)
		}
	}

	// fetch results from server
	if cls.state == ca_connected {
		if s, err := cl.ReadFromServer(); err != nil {
			HostError(err)
		} else if s == serverDisconnected {
			return
		}
	}

	var time1, time2, time3 time.Time

	// update video
	if cvars.HostSpeeds.Bool() {
		time1 = time.Now()
	}

	screen.Update()

	particlesRun(float32(cl.time), float32(cl.oldTime)) // separated from rendering

	if cvars.HostSpeeds.Bool() {
		time2 = time.Now()
	}

	// update audio
	listener := snd.Listener{
		ID: cl.viewentity,
	}
	if cls.signon == 4 {
		listener.Origin = qRefreshRect.viewOrg
		listener.Right = qRefreshRect.viewRight
		cl.DecayLights()
	}
	snd.Update(listener)

	if cvars.HostSpeeds.Bool() {
		pass1 := time1.Sub(executeFrameTime)
		executeFrameTime = time.Now()
		pass2 := time2.Sub(time1)
		pass3 := time3.Sub(time2)
		conlog.Printf("%3d tot %3d server %3d gfx %3d snd\n",
			(pass1 + pass2 + pass3).Milliseconds(),
			pass1.Milliseconds(), pass2.Milliseconds(), pass3.Milliseconds())
	}

	// this is for demo syncing
	host.frameCount++
}

type measure struct {
	startMeasure        func()
	endMeasure          func()
	frameCount          int
	frameCountStartTime time.Time
}

func newMeasure() *measure {
	m := &measure{
		frameCount: 0,
	}
	f := func(profile bool) {
		if profile {
			m.startMeasure = m.startMeasureFunc
			m.endMeasure = m.endMeasureFunc
		} else {
			m.startMeasure = func() {}
			m.endMeasure = func() {}
		}
	}
	f(cvars.ServerProfile.Bool())
	cvars.ServerProfile.SetCallback(func(cv *cvar.Cvar) {
		f(cv.Bool())
	})
	return m
}

func (m *measure) startMeasureFunc() {
	if m.frameCount == 0 {
		m.frameCountStartTime = time.Now()
	}
}

func (m *measure) endMeasureFunc() {
	m.frameCount++
	if m.frameCount < 1000 {
		return
	}

	end := time.Now()
	div := end.Sub(m.frameCountStartTime)
	m.frameCount = 0

	clientNum := 0
	for i := 0; i < svs.maxClients; i++ {
		if sv_clients[i].active {
			clientNum++
		}
	}
	conlog.Printf("serverprofile: %2d clients %v\n", clientNum, div.String())
}
