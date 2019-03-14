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

	Hulls [MAX_MAP_HULLS]Hull
}
