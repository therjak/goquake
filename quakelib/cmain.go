package quakelib

/*
#include <stdlib.h>
void Sys_Init();
void Host_Init();
void Host_Frame();
*/
import "C"

import (
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"os"
	cmdl "quake/commandline"
	"quake/cvars"
	"quake/window"
	"time"
	"unsafe"
)

const (
	VERSION              = 1.09
	GLQUAKE_VERSION      = 1.00
	FITZQUAKE_VERSION    = 0.85
	QUAKESPASM_VERSION   = 0.92
	QUAKESPASM_VER_PATCH = 2
)

func CallCMain() {
	args := make([](*C.char), 0)
	for _, a := range os.Args {
		carg := C.CString(a)
		defer C.free(unsafe.Pointer(carg))
		strptr := (*C.char)(unsafe.Pointer(carg))
		args = append(args, strptr)
	}
	v := sdl.Version{}
	sdl.GetVersion(&v)
	log.Printf("Found SDL version %d.%d.%d\n", v.Major, v.Minor, v.Patch)
	if err := sdl.Init(0); err != nil {
		panic(err)
	}
	defer sdl.Quit()

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

func runDedicated() {
	oldtime := time.Now()
	for {
		timediff := time.Since(oldtime)
		oldtime = time.Now()
		w := time.Duration(cvars.TicRate.Value()*float32(time.Second)) - timediff
		time.Sleep(w)

		C.Host_Frame()
	}
}

func runNormal() {
	oldtime := time.Now()
	for {
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

		C.Host_Frame()
	}
}
