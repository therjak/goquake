package model

import (
	"quake/math"
)

type Plane struct {
	Normal   math.Vec3
	Dist     float32
	Type     byte
	SignBits byte
	Pad      [2]byte
}

type ClipNode struct {
	PlaneNum int
	Children [2]int
}

type Hull struct {
	ClipNodes     []ClipNode
	Planes        []Plane
	FirstClipNode int
	LastClipNode  int
	ClipMins      math.Vec3
	ClipMaxs      math.Vec3
}
