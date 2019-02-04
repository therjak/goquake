package quakelib

//extern float *pr_globals;
import "C"

import (
	"log"
	pr "quake/progs"
	"unsafe"
)

var (
	progsdat *pr.LoadedProg
)

//export PR_LoadProgsGo
func PR_LoadProgsGo() {
	p, err := pr.LoadProgs()
	if err != nil {
		log.Fatalf("Failed to load progs.dat: %v", err)
	}
	progsdat = p
	log.Printf("go: %v\n", progsdat.Globals)
	log.Printf("c: %v\n", *(*[91]int32)(unsafe.Pointer(C.pr_globals)))
}
