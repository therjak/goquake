// SPDX-License-Identifier: GPL-2.0-or-later

package mdl

const (
	ST_SYNC = iota
	ST_RAND
)

const (
	ALIAS_SINGLE = iota
	ALIAS_GROUP
)

const (
	ALIAS_SKIN_SINGLE = iota
	ALIAS_SKIN_GROUP
)

const (
	DT_FACE_FRONT = 0x0010
)

const (
	aliasVersion = 6
	Magic        = 'O'<<24 | 'P'<<16 | 'D'<<8 | 'I'
)

type header struct { // mdl_t
	ID             int32
	Version        int32
	Scale          [3]float32
	Translate      [3]float32
	BoundingRadius float32
	EyePosition    [3]float32
	SkinCount      int32
	SkinWidth      int32
	SkinHeight     int32
	VerticeCount   int32
	TriangleCount  int32
	FrameCount     int32
	SyncType       int32
	Flags          int32
	Size           float32
}

type skinVertex struct { // texture coordinates
	OnSeam int32 // 0 or 0x20
	S      int32 // position horizontally, [0,SkinWidth[
	T      int32 // position vertically, [0,SkinHeight[
}

type triangle struct {
	FacesFront int32
	Vertices   [3]int32
}

type frameVertex struct {
	PackedPosition   [3]byte
	LightNormalIndex byte
}

type aliasFrame struct {
	BBoxMin frameVertex
	BBoxMax frameVertex
	Name    [16]byte
}

type aliasFrameGroup struct {
	FrameCount int32
	BBoxMin    frameVertex
	BBoxMax    frameVertex
}
