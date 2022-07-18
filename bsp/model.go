// SPDX-License-Identifier: GPL-2.0-or-later

package bsp

import (
	"goquake/math/vec"
	"goquake/texture"
)

// Would be great to type these but positive values are node numbers or so....
const (
	_ = -iota
	CONTENTS_EMPTY
	CONTENTS_SOLID
	CONTENTS_WATER
	CONTENTS_SLIME
	CONTENTS_LAVA
	CONTENTS_SKY
	CONTENTS_ORIGIN
	CONTENTS_CLIP
	CONTENTS_CURRENT_0
	CONTENTS_CURRENT_90
	CONTENTS_CURRENT_180
	CONTENTS_CURRENT_270
	CONTENTS_CURRENT_UP
	CONTENTS_CURRENT_DOWN
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

const BackFaceEpsilon = 0.01

type ST byte

const (
	S ST = iota
	T
)

type Color struct {
	R float32
	G float32
	B float32
	A float32
}

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
	Surfaces []*Surface
}

type MLeaf struct {
	NodeBase
	CompressedVis []byte
	MarkSurfaces  []*Surface // FirstMarkSurface
	// NumMarkSurfaces   int == len(MarkSurfaces)
	Key               int
	AmbientSoundLevel [4]byte
	Temporary         interface{}
}

// TODO: rename Vertex
type TexCoord struct {
	// verts[0-2]
	Pos vec.Vec3
	// verts[3]
	S float32
	// verts[4]
	T float32
	// verts[5]+verts[6]
	LightMapS float32
	LightMapT float32
}

type Poly struct {
	Next  *Poly
	Chain *Poly
	Verts []TexCoord
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

	textureMins [2]int
	extents     [2]int
	// lightS and lightT are 0 as we now use separate textures
	lightS int // gl lightmap coordinates
	lightT int

	Polys        *Poly
	TextureChain *Surface

	TexInfo *TexInfo

	// This is not actually static model data but metadata for the renderer
	VboFirstVert int // index of this surface's first vert in the VBO

	// old MAX_DLIGHTS == 64
	DLightFrame int
	DLightBits  []bool
	// LightmapTextureNum int
	LightmapTexture *texture.Texture // from r_brush lightmap_textures
	LightmapData    []byte           // from r_brush lightmaps
	lightmapName    string
	// MAXLIGHTMAPS == 4
	Styles      [4]byte
	CachedLight [4]int
	// CachedDLight bool
	LightSamples []byte
	lightMapOfs  int32
}

type TexInfoPos struct {
	Pos    vec.Vec3
	Offset float32
}

type TexInfo struct {
	Vecs    [2]TexInfoPos
	Texture *Texture
	Flags   uint32
}

type Texture struct {
	Width         int
	Height        int
	name          string
	Texture       *texture.Texture
	Fullbright    *texture.Texture
	SolidSky      *texture.Texture
	AlphaSky      *texture.Texture
	FlatSky       Color
	TextureChains [2]*Surface
	// AnimTotal int
	// AnimMin int
	// AnimMax int
	// AnimNext *Texture
	// AlternateAnims *Texture
	// Offsets [4]uint32
	Data []byte // raw texture data from the bsp
}

func (t *Texture) Name() string {
	return t.name
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

	mins   vec.Vec3
	maxs   vec.Vec3
	Radius float32
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

	Hulls     [MaxMapHulls]Hull
	VisData   []byte
	lightData []byte

	Entities []*Entity

	Node Node
}

func (q *Model) Mins() vec.Vec3 {
	return q.mins
}
func (q *Model) Maxs() vec.Vec3 {
	return q.maxs
}
func (q *Model) Name() string {
	return q.name
}
func (q *Model) Flags() int {
	return 0
}
