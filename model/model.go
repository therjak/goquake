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
	name    string  // alias + sprite + brush
	modType ModType // alias + sprite + brush

	Flags int // alias
	// Cache // alias + sprite
	// vboindexofs // alias
	// vboxyzofs // alias
	// vbostofs // alias
	// meshindexesvbo // alias
	// meshvbo // alias

	mins vec.Vec3 // sprite + alias + brush
	maxs vec.Vec3 // sprite + alias + brush
	// rmins // alias + brush
	// rmaxs // alias + brush
	// ymins // alias + brush
	// ymaxs // alias + brush
	ClipMins vec.Vec3 // brush
	ClipMaxs vec.Vec3 // brush

	Submodels    []*Submodel // brush
	Planes       []*Plane    // brush
	Leafs        []*MLeaf    // brush
	Vertexes     []*MVertex  // brush
	Edges        []*MEdge    // brush
	Nodes        []*MNode    // brush
	TexInfos     []*TexInfo  // only in brush
	Surfaces     []*Surface  // only in brush
	SurfaceEdges []int       // brush
	ClipNodes    []*ClipNode // brush
	MarkSurfaces []*Surface  // brush
	Textures     []*Texture  // only in brush

	FrameCount int // numframes, alias + sprite + brush
	SyncType   int // alias + sprite

	Hulls   [MAX_MAP_HULLS]Hull // brush
	VisData []byte              // brush

	Entities []*Entity // brush

	Node Node // brush
}

func (q *QModel) Mins() vec.Vec3 {
	return q.mins
}
func (q *QModel) Maxs() vec.Vec3 {
	return q.maxs
}
func (q *QModel) Type() ModType {
	return q.modType
}
func (q *QModel) Name() string {
	return q.name
}

func (q *QModel) SetMins(m vec.Vec3) {
	q.mins = m
}
func (q *QModel) SetMaxs(m vec.Vec3) {
	q.maxs = m
}
func (q *QModel) SetType(t ModType) {
	q.modType = t
}
func (q *QModel) SetName(n string) {
	q.name = n
}

type Model interface {
	Name() string
	Type() ModType
	Mins() vec.Vec3
	Maxs() vec.Vec3
}
