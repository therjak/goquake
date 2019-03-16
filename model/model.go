package model

import (
	"quake/math"
)

type Plane struct {
	Normal   math.Vec3
	Dist     float32
	Type     byte
	SignBits byte
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

type Node struct {
	Contents int // 0 to differentiate from leafs
	VisFrame int

	MinMaxs  [6]float32
	Parent   *Node
	Children [2]*Node
	Plane    *Plane

	FirstSurface uint32
	NumSurfaces  uint32
}

// GLuint == uint32

type ModType int

const (
	ModBrush  = ModType(iota)
	ModSprite = ModType(iota)
	ModAlias  = ModType(iota)
)

const (
	MAX_MAP_HULLS = 4
	MAX_MODELS    = 2048
)

// Knows currently only what sv.models needs to know
type QModel struct {
	Name string
	Type ModType

	Mins     math.Vec3
	Maxs     math.Vec3
	ClipMins math.Vec3
	ClipMaxs math.Vec3

	Node *Node

	Hulls [MAX_MAP_HULLS]Hull
}
