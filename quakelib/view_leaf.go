// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

//void R_MarkSurfaces(void);
import "C"

import (
	"log"

	"goquake/bsp"
	"goquake/cvar"
	"goquake/cvars"
)

type qViewLeaf struct {
	current *bsp.MLeaf
	old     *bsp.MLeaf
}

var viewLeaf qViewLeaf

const (
	chainWorld = 0
	chainModel = 1
)

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

var markSurfacesVis []byte
var markSurfacesVisChanged bool

func init() {
	f := func(*cvar.Cvar) {
		markSurfacesVisChanged = true
	}
	cvars.RNoVis.SetCallback(f)
	cvars.ROldSkyLeaf.SetCallback(f)
}

//export MarkSurfaces
func MarkSurfaces() {
	C.R_MarkSurfaces()
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
			if markSurfacesVis[i>>3]&(1<<(i&7)) != 0 {
				MakeEntitiesVisible(leaf)
			}
		}
		return
	}
	markSurfacesVisChanged = false
	// TODO: I am quite sure this is the one that is not needed.
	viewLeaf.old = viewLeaf.current

	// iterate through leaves, marking surfaces
	for i, leaf := range cl.worldModel.Leafs[1:] {
		if markSurfacesVis[i>>3]&(1<<(i&7)) != 0 {
			if cvars.ROldSkyLeaf.Bool() || leaf.Contents() != bsp.CONTENTS_SKY {
				for _, ms := range leaf.MarkSurfaces {
					// TODO: why is this needed? any option to not have the bsp know about this?
					ms.VisFrame = renderer.visFrameCount
				}
			}
			MakeEntitiesVisible(leaf)
		}
	}

	// clear and rebuild texture chains
	for _, t := range cl.worldModel.Textures {
		if t != nil {
			t.TextureChains[chainWorld] = nil
		}
	}
	for _, n := range cl.worldModel.Nodes {
		for _, s := range n.Surfaces {
			if s.VisFrame == renderer.visFrameCount {
				s.TextureChain = s.TexInfo.Texture.TextureChains[chainWorld]
				s.TexInfo.Texture.TextureChains[chainWorld] = s
			}
		}
	}
}
