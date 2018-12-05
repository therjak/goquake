package main

import (
	"flag"
	"quake/quakelib"
)

func main() {
	flag.Parse()
	quakelib.CallCMain()
}
