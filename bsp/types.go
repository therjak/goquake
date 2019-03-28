package bsp

// called lump_t in c
type directory struct {
	Offset int32
	Size   int32
}

type header struct {
	Version      int32
	Entities     directory
	Planes       directory
	Textures     directory
	Vertexes     directory
	Visibility   directory
	Nodes        directory
	Texinfo      directory
	Faces        directory
	Lighting     directory
	ClipNodes    directory
	Leafs        directory
	MarkSurfaces directory
	Edges        directory
	SurfaceEdges directory // SURFEDGES
	Models       directory
}

type scalar float32

type vec3 struct {
	X scalar
	Y scalar
	Z scalar
}

// Bounding box, float32 values
type boundBox struct {
	Min vec3
	Max vec3
}

// Bounding box, int16 values
type bBoxShort struct {
	Min [3]int16 // minimum values of X,Y,Z
	Max [3]int16 // maximum values of X,Y,Z
}

// Model, either a big zone, the level or parts inside that zone
type model struct {
	BoundingBox  [6]float32
	Origin       [3]float32
	HeadNode     [4]int32
	VisLeafCount int32 // not including the solid leaf 0
	FirstFace    int32
	FaceCount    int32
}

type vertex struct {
	X float32
	Y float32
	Z float32
}

// the first edge of the list is never used
type edgeV0 struct {
	Vertex0 uint16 // id of start vertex, must be in [0,numvertices[
	Vertex1 uint16 // id of end vertex, must be in [0,numvertices[
}

type edgeV1 struct {
	Vertex0 uint32 // id of start vertex, must be in [0,numvertices[
	Vertex1 uint32 // id of end vertex, must be in [0,numvertices[
}

type surface struct {
	VectorS   [3]float32 // S vector, horizontal in texture space
	DistS     float32    // horizontal offset in texture space
	VectorT   [3]float32 // T vector, vertical in texture space
	DistT     float32    // vertical offset in texture space
	TextureID uint32     // Index of mip texture, must be in [0,numtex[
	Animated  uint32     // 0 for ordinary textures, 1 for water
}

type faceV0 struct {
	PlaneID        int16 // The plane in which the face lies, must be in [0,numplanes[
	Side           int16
	ListEdgeID     int32
	ListEdgeNumber int16
	TexInfoID      int16
	LightStyle     [4]uint8
	LightMap       int32 // Pointer inside the general light map, or -1. this defines the start of the face light map
}

type faceV1 struct {
	PlaneID        int32 // The plane in which the face lies, must be in [0,numplanes[
	Side           int32
	ListEdgeID     int32
	ListEdgeNumber int32
	TexInfoID      int32
	LightStyle     [4]uint8
	LightMap       int32 // Pointer inside the general light map, or -1. this defines the start of the face light map
}

type mipHeader struct {
	NumberOfTextures int32
	Offsets          []int32 // Variable length, has length of NumberOfTextures
}

type mipTexture struct {
	Name   [16]byte
	Width  uint32
	Height uint32
	// Offes[0] to Pix[width * height]
	// 1: to Pix[width/2 * height/2]
	// 2: to Pix[width/4 * height/4]
	// 3: to Pix[width/8 * height/8]
	Offset [4]uint32
}

type nodeV0 struct {
	PlaneID      int32
	Children     [2]uint16
	Box          [6]int16
	FirstSurface uint16
	SurfaceCount uint16
}
type nodeV1 struct {
	PlaneID      int32
	Children     [2]int32
	Box          [6]int16
	FirstSurface uint32
	SurfaceCount uint32
}
type nodeV2 struct {
	PlaneID      int32
	Children     [2]int32
	Box          [6]float32
	FirstSurface uint32
	SurfaceCount uint32
}

type leafV0 struct {
	Type             int32 // Contents
	VisOfs           int32
	Box              [6]int16 // mins & maxs
	FirstMarkSurface uint16   // firstmarksurface
	MarkSurfaceCount uint16   // nummarksurfaces
	Ambients         [4]byte  // ambient_level
}

type leafV1 struct {
	Type             int32
	VisibilityList   int32
	Box              [6]int16
	FirstMarkSurface uint32
	MarkSurfaceCount uint32
	Ambients         [4]byte
}

type leafV2 struct {
	Type             int32
	VisibilityList   int32
	Box              [6]float32
	FirstMarkSurface uint32
	MarkSurfaceCount uint32
	Ambients         [4]byte
}

const (
	_                   = iota
	LeafTypeEmpty       = -iota // was CONTENTS_EMPTY...
	LeafTypeSolid       = -iota
	LeafTypeWater       = -iota
	LeafTypeSlime       = -iota
	LeafTypeLava        = -iota
	LeafTypeSky         = -iota
	LeafTypeOrigin      = -iota
	LeafTypeClip        = -iota
	LeafTypeCurrent0    = -iota
	LeafTypeCurrent90   = -iota
	LeafTypeCurrent180  = -iota
	LeafTypeCurrent270  = -iota
	LeafTypeCurrentUp   = -iota
	LeafTypeCurrentDown = -iota
)

type plane struct {
	Normal   [3]float32
	Distance float32
	Type     int32 // 0: axial plane in X, 1: axial plane in Y, 2 axial in Z, 3,4,5 similar but non axial
}

type clipNodeV0 struct {
	PlaneNumber int32     // the plane which splits the node
	Children    [2]uint16 // if positive id of the child node, -2 if front part inside the model, -1 if outside the model
}

type clipNodeV1 struct {
	PlaneNumber int32    // the plane which splits the node
	Children    [2]int32 // if positive id of the child node, -2 if front part inside the model, -1 if outside the model
}
