package quakelib

import (
	"github.com/therjak/goquake/progs"
	"log"
)

var (
	progsdat *progs.LoadedProg
)

func LoadProgs() {
	log.Printf("LOADING PROGS")
	p, err := progs.LoadProgs()
	if err != nil {
		log.Fatalf("Failed to load progs.dat: %v", err)
	}
	progsdat = p
	vm.prog = p
}
