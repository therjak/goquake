// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

/*
void Sys_Init();
void Host_Init();
*/
import "C"

import (
	"log"
	"time"

	cmdl "github.com/therjak/goquake/commandline"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/window"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	VERSION              = 1.09
	GLQUAKE_VERSION      = 1.00
	FITZQUAKE_VERSION    = 0.85
	QUAKESPASM_VERSION   = 0.92
	QUAKESPASM_VER_PATCH = 2
)

func CallCMain() {
	v := sdl.Version{}
	sdl.GetVersion(&v)
	log.Printf("Found SDL version %d.%d.%d\n", v.Major, v.Minor, v.Patch)
	if err := sdl.Init(0); err != nil {
		panic(err)
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

	if cmdl.Dedicated() {
		runDedicated()
	} else {
		runNormal()
	}
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
			w := time.Duration(cvars.Throttle.Value()*float32(time.Second)) - timediff
			time.Sleep(w)

			hostFrame()
		}
	}
}
