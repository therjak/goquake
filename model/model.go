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

const (
	EntityEffectBrightField = 1 << iota
	EntityEffectMuzzleFlash // 2
	EntityEffectBrightLight // 4
	EntityEffectDimLight    // 8
)

const (
	EntityEffectRocket  = 1 << iota
	EntityEffectGrenade // 2
	EntityEffectGib     // 4
	EntityEffectRotate  // 8
	EntityEffectTracer  // 16
	EntityEffectZomGib  // 32
	EntityEffectTracer2 // 64
	EntityEffectTracer3 // 128
)

const (
	SurfaceNone           = 1 << iota
	SurfacePlaneBack      // 0x0002
	SurfaceDrawSky        // 0x0004
	SurfaceDrawSprite     // 0x0008
	SurfaceDrawTurb       // 0x0010
	SurfaceDrawTiled      // 0x0020
	SurfaceDrawBackground // 0x0040
	SurfaceUnderWater     // 0x0080
	SurfaceNoTexture      // 0x0100
	SurfaceDrawFence      // 0x0200
	SurfaceDrawLava       // 0x0400
	SurfaceDrawSlime      // 0x0800
	SurfaceDrawTele       // 0x1000
	SurfaceDrawWater      // 0x2000
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

type TexInfo struct {
	Vecs    [2][4]float32
	Texture *Texture
	Flags   uint32
}

type Texture struct {
	Width         int
	Height        int
	Name          string
	TextureChains [2]*Surface
	Texture       *texture.Texture
	Fullbright    *texture.Texture
	Warp          *texture.Texture
}

// GLuint == uint32

type ModType int

const (
	ModBrush ModType = iota
	ModSprite
	ModAlias
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

type MVertex struct {
	Position vec.Vec3
}

type MEdge struct {
	V                [2]int // unsigned int
	CachedEdgeOffset int    // unsigned int
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

	Submodels    []*Submodel
	Planes       []*Plane
	Leafs        []*MLeaf
	Vertexes     []*MVertex
	Edges        []*MEdge
	Nodes        []*MNode
	TexInfos     []*TexInfo // only in brush
	Surfaces     []*Surface
	SurfaceEdges []int
	ClipNodes    []*ClipNode
	MarkSurfaces []*Surface
	Textures     []*Texture // only in brush

	FrameCount int // numframes
	SyncType   int

	Hulls   [MAX_MAP_HULLS]Hull
	VisData []byte

	Entities []*Entity

	Node Node
}
