package main

import (
	"flag"
	"github.com/therjak/goquake/quakelib"
)

import (
	// register the model loaders
	_ "github.com/therjak/goquake/bsp"
	_ "github.com/therjak/goquake/mdl"
	_ "github.com/therjak/goquake/spr"
)

func main() {
	flag.Parse()
	quakelib.CallCMain()
}
