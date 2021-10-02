// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

/*
void Sys_Init();
void Host_Init();
void HostInitAllocEnd();
void R_Init();
void GL_SetupState();
void Mod_Init();
*/
import "C"

import (
	"log"
	"time"

	"goquake/cbuf"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvars"
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

func CallCMain() error {
	v := sdl.Version{}
	sdl.GetVersion(&v)
	log.Printf("Found SDL version %d.%d.%d\n", v.Major, v.Minor, v.Patch)
	if err := sdl.Init(0); err != nil {
		return err
	}
	defer sdl.Quit()

	vm = NewVirtualMachine()

	C.Sys_Init()

	log.Printf("Quake %1.2f (c) id Software\n", VERSION)
	log.Printf("GLQuake %1.2f (c) id Software\n", GLQUAKE_VERSION)
	log.Printf("FitzQuake %1.2f (c) John Fitzgibbons\n", FITZQUAKE_VERSION)
	log.Printf("FitzQuake SDL port (c) SleepwalkR, Baker\n")
	log.Printf("QuakeSpasm %1.2f.%d (c) Ozkan Sezer, Eric Wasylishen & others\n", QUAKESPASM_VERSION, QUAKESPASM_VER_PATCH)
	log.Printf("Host_Init\n")
	C.Host_Init()

	filesystemInit()
	hostInit()
	if err := wad.Load(); err != nil {
		return err
	}
	if cls.state != ca_dedicated {
		history.Load()
		consoleInit()
	}
	C.Mod_Init()
	networkInit()
	serverInit()

	if cls.state != ca_dedicated {
		// ExtraMaps_Init();
		// Modlist_Init();
		// DemoList_Init();
		if err := videoInit(); err != nil {
			return err
		}
		shaderInit()
		C.GL_SetupState()
		textureManagerInit()
		drawInit()
		screen.initialized = true
		C.R_Init()
		soundInit()
		statusbar.LoadPictures()
		clientInit()
	}

	C.HostInitAllocEnd()

	host.initialized = true
	conlog.Printf("\n========= Quake Initialized =========\n\n")
	cbuf.AddText("alias startmap_sp \"map start\"\n")
	cbuf.AddText("alias startmap_dm \"map start\"\n")

	if cls.state != ca_dedicated {
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

	if cmdl.Dedicated() {
		runDedicated()
	} else {
		runNormal()
	}
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

func runDedicated() {
	oldtime := time.Now()
	for {
		select {
		case <-quitChan:
			return
		default:
			timediff := time.Since(oldtime)
			oldtime = time.Now()
			w := time.Duration(cvars.TicRate.Value()*float32(time.Second)) - timediff
			time.Sleep(w)

			hostFrame()
		}
	}
}

func runNormal() {
	oldtime := time.Now()
	for {
		select {
		case <-quitChan:
			return
		default:
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

			timediff := time.Since(oldtime)
			oldtime = time.Now()
			if !cls.timeDemo {
				w := time.Duration(cvars.Throttle.Value()*float32(time.Second)) - timediff
				time.Sleep(w)
			}

			hostFrame()
		}
	}
}
