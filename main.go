package main

import (
	"flag"
	"quake/quakelib"
)

import (
	// register the model loaders
	_ "quake/bsp"
	_ "quake/mdl"
	_ "quake/spr"
)

func main() {
	flag.Parse()
	quakelib.CallCMain()
}
