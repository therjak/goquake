package quakelib

import "C"

import (
	"log"

	"github.com/therjak/goquake/bsp"
)

type qViewLeaf struct {
	current *bsp.MLeaf
	old     *bsp.MLeaf
}

var viewLeaf qViewLeaf

//export UpdateViewLeafGo
func UpdateViewLeafGo() {
	// TODO: it feels like there is a 'bug' if two places update viewLeaf.old
	viewLeaf.old = viewLeaf.current
	c, err := cl.worldModel.PointInLeaf(qRefreshRect.viewOrg)
	if err != nil {
		log.Printf("UpdateViewLeaf: %v", err)
	}
	viewLeaf.current = c
}

//export UpdateOldViewLeafGo
func UpdateOldViewLeafGo() {
	// TODO: I am quite sure this is the one that is not needed.
	viewLeaf.old = viewLeaf.current
}
