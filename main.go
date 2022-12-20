// SPDX-License-Identifier: GPL-2.0-or-later
package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"goquake/quakelib"
	"goquake/version"

	"github.com/veandco/go-sdl2/sdl"

	// register the model loaders
	_ "goquake/bsp"
	_ "goquake/mdl"
	_ "goquake/spr"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	v := sdl.Version{}
	sdl.GetVersion(&v)
	log.Printf("Found SDL version %d.%d.%d\n", v.Major, v.Minor, v.Patch)
	if err := sdl.Init(0); err != nil {
		log.Fatal(err)
	}
	defer sdl.Quit()

	log.Printf("GoQuake %1.2f.%d\n", version.Base, version.Patch)

	if err := quakelib.CallCMain(); err != nil {
		log.Fatal(err)
	}
}
