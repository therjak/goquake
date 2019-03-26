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

type NodeBase struct {
	contents int // 0 to differentiate from leafs
	visFrame int

	minMaxs [6]float32
	parent  Node
}

func NewNodeBase(contents, visframe int, minmax [6]float32) NodeBase {
	return NodeBase{
		contents: contents,
		visFrame: visframe,
		minMaxs:  minmax,
		parent:   nil,
	}
}

type Node interface {
	Contents() int
	Parent() Node
	SetParent(p Node)
}

func (n *NodeBase) Contents() int {
	return n.contents
}
func (n *NodeBase) Parent() Node {
	return n.parent
}
func (n *NodeBase) SetParent(p Node) {
	n.parent = p
}

type MNode struct {
	NodeBase
	Children [2]Node
	Plane    *Plane

	FirstSurface uint32
	SurfaceCount uint32
}

type MLeaf struct {
	NodeBase
	CompressedVis []byte
	Efrags        []Efrag
	MarkSurfaces  []*Surface // FirstMarkSurface
	// NumMarkSurfaces   int == len(MarkSurfaces)
	Key               int
	AmbientSoundLevel [4]byte
}

type Efrag struct{}
type Surface struct {
}
type Texinfo struct{}

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

	NumSubmodels int
	// dmodel_t *submodels

	// submodels
	Planes []*Plane
	Leafs  []*MLeaf
	// vertexes
	// edges
	Nodes    []*MNode
	Texinfos []*Texinfo
	Surfaces []*Surface
	// surfedge
	ClipNodes    []*ClipNode
	MarkSurfaces []*Surface

	Hulls [MAX_MAP_HULLS]Hull
	// textures
	VisData []byte

	Node Node
}
