// SPDX-License-Identifier: GPL-2.0-or-later
package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"github.com/therjak/goquake/quakelib"

	// register the model loaders
	_ "github.com/therjak/goquake/bsp"
	_ "github.com/therjak/goquake/mdl"
	_ "github.com/therjak/goquake/spr"
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
	if err := quakelib.CallCMain(); err != nil {
		log.Fatal(err)
	}
}
