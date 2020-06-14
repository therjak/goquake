package model

import (
	"github.com/therjak/goquake/math/vec"
	"github.com/therjak/goquake/texture"
)

// Would be great to type these but positive values are node numbers or so....
const (
	CONTENTS_EMPTY        = -1
	CONTENTS_SOLID        = -2
	CONTENTS_WATER        = -3
	CONTENTS_SLIME        = -4
	CONTENTS_LAVA         = -5
	CONTENTS_SKY          = -6
	CONTENTS_ORIGIN       = -7
	CONTENTS_CLIP         = -8
	CONTENTS_CURRENT_0    = -9
	CONTENTS_CURRENT_90   = -10
	CONTENTS_CURRENT_180  = -11
	CONTENTS_CURRENT_270  = -12
	CONTENTS_CURRENT_UP   = -13
	CONTENTS_CURRENT_DOWN = -14
)

type Plane struct {
	Normal   vec.Vec3
	Dist     float32
	Type     byte
	SignBits byte
}

type ClipNode struct {
	Plane    *Plane
	Children [2]int
}

type Hull struct {
	ClipNodes     []*ClipNode
	Planes        []*Plane
	FirstClipNode int
	LastClipNode  int
	ClipMins      vec.Vec3
	ClipMaxs      vec.Vec3
}

type NodeBase struct {
	contents int // 0 to differentiate from leafs
	visFrame int

	minMaxs [6]float32
}

func NewNodeBase(contents, visframe int, minmax [6]float32) NodeBase {
	return NodeBase{
		contents: contents,
		visFrame: visframe,
		minMaxs:  minmax,
	}
}

type Node interface {
	Contents() int
}

func (n *NodeBase) Contents() int {
	return n.contents
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
	Flags        int
	TextureChain *Surface
}

type Texinfo struct{}

type Texture struct {
	Width        int
	Height       int
	TextureChans [2]*Surface
	Texture      *texture.Texture
	Fullbright   *texture.Texture
	Warp         *texture.Texture
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
	MAX_MAP_LEAFS = 70000
)

type Submodel struct {
	Mins         vec.Vec3
	Maxs         vec.Vec3
	Origin       vec.Vec3
	HeadNode     [4]int
	VisLeafCount int
	FirstFace    int
	FaceCount    int
}

// Knows currently only what sv.models needs to know
type QModel struct {
	Name string
	Type ModType

	Flags int

	Mins     vec.Vec3
	Maxs     vec.Vec3
	ClipMins vec.Vec3
	ClipMaxs vec.Vec3

	Submodels []*Submodel
	Planes    []*Plane
	Leafs     []*MLeaf
	// vertexes
	// edges
	Nodes    []*MNode
	Texinfos []*Texinfo
	Surfaces []*Surface
	// surfedge []int ?
	ClipNodes    []*ClipNode
	MarkSurfaces []*Surface
	Textures     []*Texture // only in brush

	FrameCount int
	SyncType   int

	Hulls   [MAX_MAP_HULLS]Hull
	VisData []byte

	Entities []map[string]string

	Node Node
}
