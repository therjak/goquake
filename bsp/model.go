package bsp

import (
	"github.com/therjak/goquake/math/vec"
	qm "github.com/therjak/goquake/model"
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

type Poly struct {
	Next     *Poly
	Chain    *Poly
	NumVerts int           // TODO: why
	Verts    [4][7]float32 // TODO: why 7?
}

type Surface struct {
	VisFrame int // should be drawn when node is crossed
	Culled   bool
	Mins     [3]float32
	Maxs     [3]float32

	Plane *Plane
	Flags int

	FirstEdge int
	NumEdges  int

	TextureMins [2]int16
	Extents     [2]int16
	LightS      int // gl lightmap coordinates
	LightT      int

	Polys        *Poly // multiple if warped
	TextureChain *Surface

	TexInfo *TexInfo

	VboFirstVert int // index of this surface's first vert in the VBO

	DLightFrame int
	// MAX_DLIGHTS == 64
	// DLightBits [(MAX_DLIGHTS + 31)>>5]uint32
	LightmapTextureNum int
	// MAXLIGHTMAPS == 4
	// Styles [MAXLIGHTMAPS]byte
	// CachedLight[MAXLIGHTMAPS] int
	CachedDLight bool
	Samples      *byte
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

const (
	MaxMapHulls = 4
	MaxMapLeafs = 70000
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
type Model struct {
	name string

	mins vec.Vec3
	maxs vec.Vec3
	// rmins // alias + brush
	// rmaxs // alias + brush
	// ymins // alias + brush
	// ymaxs // alias + brush
	ClipMins vec.Vec3
	ClipMaxs vec.Vec3

	Submodels    []*Submodel
	Planes       []*Plane
	Leafs        []*MLeaf
	Vertexes     []*MVertex
	Edges        []*MEdge
	Nodes        []*MNode
	TexInfos     []*TexInfo
	Surfaces     []*Surface
	SurfaceEdges []int32
	ClipNodes    []*ClipNode
	MarkSurfaces []*Surface
	Textures     []*Texture

	FrameCount int // numframes

	Hulls   [MaxMapHulls]Hull
	VisData []byte

	Entities []*Entity

	Node Node
}

func (q *Model) Mins() vec.Vec3 {
	return q.mins
}
func (q *Model) Maxs() vec.Vec3 {
	return q.maxs
}
func (q *Model) Type() qm.ModType {
	return qm.ModBrush
}
func (q *Model) Name() string {
	return q.name
}
func (q *Model) Flags() int {
	return 0
}
