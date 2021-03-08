// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"

import (
	"log"

	"github.com/therjak/goquake/bsp"
	"github.com/therjak/goquake/cvar"
	"github.com/therjak/goquake/cvars"
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

var markSurfacesVis []byte
var markSurfacesVisChanged bool

func init() {
	f := func(*cvar.Cvar) {
		markSurfacesVisChanged = true
	}
	cvars.RNoVis.SetCallback(f)
	cvars.ROldSkyLeaf.SetCallback(f)
}

//export MarkSurfacesAddStaticEntities
func MarkSurfacesAddStaticEntities() {
	// check for water portals
	nearWaterPortal := false
	for _, mark := range viewLeaf.current.MarkSurfaces {
		if mark.Flags&bsp.SurfaceDrawTurb != 0 {
			nearWaterPortal = true
			break
		}
	}

	// choose vis data
	if cvars.RNoVis.Bool() ||
		viewLeaf.current.Contents() == bsp.CONTENTS_SOLID ||
		viewLeaf.current.Contents() == bsp.CONTENTS_SKY {
		markSurfacesVis = bsp.NoVis
	} else if nearWaterPortal {
		markSurfacesVis = cl.worldModel.FatPVS(qRefreshRect.viewOrg)
	} else {
		markSurfacesVis = cl.worldModel.LeafPVS(viewLeaf.current)
	}

	if viewLeaf.old == viewLeaf.current && !markSurfacesVisChanged && !nearWaterPortal {
		for i, leaf := range cl.worldModel.Leafs[1:] {
			if markSurfacesVis[i>>3]&markSurfacesVis[i&7] != 0 {
				MakeEntitiesVisible(leaf)
			}
		}
	}
	markSurfacesVisChanged = false
}

//export MarkSurfacesAddStaticEntitiesAndMark
func MarkSurfacesAddStaticEntitiesAndMark() {
	for i, leaf := range cl.worldModel.Leafs[1:] {
		if markSurfacesVis[i>>3]&markSurfacesVis[i&7] != 0 {
			if cvars.ROldSkyLeaf.Bool() || leaf.Contents() != bsp.CONTENTS_SKY {
				for _, ms := range leaf.MarkSurfaces {
					// TODO: why is this needed? any option to not have the bsp know about this?
					ms.VisFrame = renderer.visFrameCount
				}
			}
			MakeEntitiesVisible(leaf)
		}
	}
}
