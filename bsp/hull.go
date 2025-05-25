// SPDX-License-Identifier: GPL-2.0-or-later

package bsp

import (
	"goquake/conlog"
	"goquake/math"
	"goquake/math/vec"
	"log"
	"runtime/debug"
)

type Hull struct {
	ClipNodes     []*ClipNode
	Planes        []*Plane
	FirstClipNode int
	LastClipNode  int
	ClipMins      vec.Vec3
	ClipMaxs      vec.Vec3
}

type tracePlane struct {
	Normal   vec.Vec3
	Distance float32
}

type Trace struct {
	AllSolid   bool
	StartSolid bool
	InOpen     bool
	InWater    bool
	Fraction   float32
	EndPos     vec.Vec3
	Plane      tracePlane
	EntPointer bool
	EntNumber  int
}

func (h *Hull) PointContents(num int, p vec.Vec3) int {
	for num >= 0 {
		if num < h.FirstClipNode || num > h.LastClipNode {
			debug.PrintStack()
			log.Fatalf("SV_HullPointContents: bad node number")
		}
		node := h.ClipNodes[num]
		plane := node.Plane
		d := func() float32 {
			if plane.Type < 3 {
				return p[int(plane.Type)] - plane.Dist
			}
			return float32(vec.DoublePrecDot(plane.Normal, p)) - plane.Dist
		}()
		if d < 0 {
			num = node.Children[1]
		} else {
			num = node.Children[0]
		}
	}

	return num
}

func (h *Hull) RecursiveCheck(num int, p1f, p2f float32, p1, p2 vec.Vec3, trace *Trace) bool {
	const epsilon = 0.03125 // (1/32) to keep floating point happy
	if num < 0 {            // check for empty
		if num != CONTENTS_SOLID {
			trace.AllSolid = false
			if num == CONTENTS_EMPTY {
				trace.InOpen = true
			} else {
				trace.InWater = true
			}
		} else {
			trace.StartSolid = true
		}
		return true
	}
	if num < h.FirstClipNode || num > h.LastClipNode {
		debug.PrintStack()
		log.Fatalf("RecursiveHullCheck: bad node number")
	}
	node := h.ClipNodes[num]
	plane := node.Plane
	t1, t2 := func() (float32, float32) {
		if plane.Type < 3 {
			return (p1[int(plane.Type)] - plane.Dist),
				(p2[int(plane.Type)] - plane.Dist)
		} else {
			return float32(vec.DoublePrecDot(plane.Normal, p1)) - plane.Dist,
				float32(vec.DoublePrecDot(plane.Normal, p2)) - plane.Dist
		}
	}()
	if t1 >= 0 && t2 >= 0 {
		return h.RecursiveCheck(node.Children[0], p1f, p2f, p1, p2, trace)
	}
	if t1 < 0 && t2 < 0 {
		return h.RecursiveCheck(node.Children[1], p1f, p2f, p1, p2, trace)
	}

	// put the crosspoint epsilon pixels on the near side
	frac := func() float32 {
		d := t1 - t2
		// In the C implementation epsilon is a float64..
		if t1 < 0 {
			return (t1 + epsilon) / d
		}
		return (t1 - epsilon) / d
	}()
	frac = math.Clamp(0, frac, 1)
	midf := math.Lerp(p1f, p2f, frac)
	mid := vec.Lerp(p1, p2, frac)
	side := func() int {
		if t1 < 0 {
			return 1
		}
		return 0
	}()
	// move up to the node
	if !h.RecursiveCheck(node.Children[side], p1f, midf, p1, mid, trace) {
		return false
	}
	if h.PointContents(node.Children[side^1], mid) != CONTENTS_SOLID {
		return h.RecursiveCheck(node.Children[side^1], midf, p2f, mid, p2, trace)
	}
	if trace.AllSolid {
		return false // never got out of the solid area
	}
	// the other side of the node is solid, this is the impact point
	if side == 0 {
		trace.Plane.Normal = plane.Normal
		trace.Plane.Distance = plane.Dist
	} else {
		trace.Plane.Normal = vec.Sub(vec.Vec3{}, plane.Normal)
		trace.Plane.Distance = -plane.Dist
	}
	for h.PointContents(h.FirstClipNode, mid) == CONTENTS_SOLID {
		// shouldn't really happen, but does occasionally
		frac -= 0.1
		if frac < 0 {
			trace.Fraction = midf
			trace.EndPos = mid
			conlog.DPrint("backup past 0\n")
			return false
		}
		midf = math.Lerp(p1f, p2f, frac)
		mid = vec.Lerp(p1, p2, frac)
	}
	trace.Fraction = midf
	trace.EndPos = mid

	return false
}
