// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"goquake/cbuf"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/filesystem"
	"goquake/input"
	"goquake/math/vec"
	"goquake/net"
	"goquake/protocol"
	"goquake/wad"
	"goquake/window"
)

var (
	sv_protocol     int
	hostInitialized bool
)

func svProtocol(a cbuf.Arguments) error {
	args := a.Args()[1:]
	switch len(args) {
	default:
		conlog.SafePrintf("usage: sv_protocol <protocol>\n")
	case 0:
		conlog.Printf(`"sv_protocol" is "%v"`+"\n", sv_protocol)
	case 1:
		i := args[0].Int()
		switch i {
		case protocol.NetQuake, protocol.FitzQuake, protocol.RMQ:
			sv_protocol = i
			if sv.Active() {
				conlog.Printf("changes will not take effect until the next level load.\n")
			}
		default:
			conlog.Printf("sv_protocol must be %v or %v or %v\n",
				protocol.NetQuake, protocol.FitzQuake, protocol.RMQ)
		}
	}
	return nil
}

func init() {
	addCommand("sv_protocol", svProtocol)
}

func serverInit() error {
	sv_protocol = cmdl.Protocol()
	switch sv_protocol {
	case protocol.NetQuake:
		log.Printf("Server using protocol %v (NetQuake)\n", sv_protocol)
	case protocol.FitzQuake:
		log.Printf("Server using protocol %v (FitzQuake)\n", sv_protocol)
	case protocol.RMQ:
		log.Printf("Server using protocol %v (RMQ)\n", sv_protocol)
	case protocol.GoQuake:
		log.Printf("Server using protocol %v (GoQuake)\n", sv_protocol)
	default:
		return fmt.Errorf("Bad protocol version request %v. Accepted values: %v, %v, %v.",
			sv_protocol, protocol.NetQuake, protocol.FitzQuake, protocol.RMQ)
	}
	return nil
}

func ServerActive() bool {
	return sv.Active()
}

func CallCMain() error {
	stop := make(chan struct{})
	defer close(stop)

	vm = NewVirtualMachine()

	filesystemInit()
	hostInit()
	if err := wad.Load(); err != nil {
		return err
	}
	if !cmdl.Dedicated() {
		if err := history.Load(); err != nil {
			log.Printf("%v\n", err)
		}
		consoleInit()
	}

	net.SetPort(cmdl.Port())
	clients := svs.maxClientsLimit
	if !cmdl.Dedicated() {
		clients++
	}
	if cmdl.Listen() || cmdl.Dedicated() {
		log.Printf("Listening to network")
		net.Listen(clients)
		defer net.StopListen()
	}
	net.SetTime()

	if err := serverInit(); err != nil {
		return err
	}

	if !cmdl.Dedicated() {
		// ExtraMaps_Init();
		// Modlist_Init();
		// DemoList_Init();
		if err := videoInit(); err != nil {
			return err
		}
		if err := shaderInit(); err != nil {
			return err
		}
		textureManagerInit()
		if err := drawInit(); err != nil {
			return err
		}
		screen.initialized = true
		particlesInit()
		setClearColor(cvars.RClearColor)
		soundInit(stop)
		statusbar.LoadPictures()
		clientInit()
	}

	hostInitialized = true
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
		cbuf.Execute()
		if !sv.Active() {
			cbuf.AddText("startmap_dm\n")
		}
	}

	r := newRunner(cmdl.Dedicated())
	r.run()
	return nil
}

func shaderInit() error {
	// All our shaders:
	if err := CreateAliasDrawer(); err != nil {
		return err
	}
	if err := CreateBrushDrawer(); err != nil {
		return err
	}
	if err := CreateSpriteDrawer(); err != nil {
		return err
	}
	CreateSkyDrawer()
	if err := CreateParticleDrawer(); err != nil {
		return err
	}
	if err := CreatePostProcess(); err != nil {
		return err
	}
	if err := CreateConeDrawer(); err != nil {
		return err
	}
	CreateUiDrawer()
	return nil
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
	if !host.UpdateTime(cls.timeDemo) {
		return // don't run too fast, or packets will flood out
	}

	// get new key events
	updateKeyDest()
	updateInputMode()
	sendKeyEvents()

	// process console commands
	cbuf.Execute()

	net.SetTime()

	// if running the server locally, make intentions now
	if sv.Active() {
		if err := CL_SendCmd(); err != nil {
			HostError(err)
		}
	}

	// server operations

	// check for commands typed to the host
	hostGetConsoleCommands()

	if sv.Active() {
		if err := serverFrame(); err != nil {
			HostError(err)
		}
	}

	// client operations

	// if running the server remotely, send intentions now after
	// the incoming messages have been read
	if !sv.Active() {
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

	// THERJAK: screenUpdate
	if err := screen.Update(); err != nil {
		QError(err.Error())
	}

	particlesRun(float32(cl.time), float32(cl.oldTime)) // separated from rendering

	if cvars.HostSpeeds.Bool() {
		time2 = time.Now()
	}

	// update audio
	listenerID := cl.viewentity
	var listenerOrigin vec.Vec3
	var listenerRight vec.Vec3
	if cls.signon == 4 {
		listenerOrigin = qRefreshRect.viewOrg
		listenerRight = qRefreshRect.viewRight
		cl.DecayLights()
	}
	snd.Update(listenerID, listenerOrigin, listenerRight)

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
	host.FrameIncrease()
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

	filename := filepath.Join(filesystem.GameDir(), "config.cfg")
	err := os.WriteFile(filename, b.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Couldn't write config.cfg: %w\n", err)
	}
	return nil
}

func shutdown() {
	screen.disabled = true
	if hostInitialized {
		if err := HostWriteConfiguration(); err != nil {
			log.Print(err.Error())
		}
	}
	net.Shutdown()
	if !cmdl.Dedicated() {
		if console.initialized {
			if err := history.Save(); err != nil {
				conlog.Printf("%v\n", err)
			}
		}
		snd.Shutdown()
		videoShutdown()
	}
}
