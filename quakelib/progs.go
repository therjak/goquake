// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"log"

	"goquake/progs"
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
