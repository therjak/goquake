package spr

const (
	ST_SYNC = iota
	ST_RAND
)

const (
	SPR_SINGLE = iota
	SPR_GROUP
)

const (
	SPR_VP_PARALLEL_UPRIGHT = iota
	SPR_FACING_UPRIGHT
	SPR_VP_PARALLEL
	SPR_ORIENTED
	SPR_VP_PARALLEL_ORIENTED
)

const (
	spriteVersion = 1
	Magic         = 'P'<<24 | 'S'<<16 | 'D'<<8 | 'I'
)

type header struct { // dsprite_t
	Name           [4]byte // "IDSP"
	Version        int32   // SPRITE_VERSION
	Typ            int32   // SPR_SINGLE or SPR_GROUP
	BoundingRadius float32
	MaxWidth       int32
	MaxHeight      int32
	FrameCount     int32
	BeamLength     float32
	SyncType       int32 // ST_SYNC or ST_RAND
}

type frame struct {
	Origin [2]int32
	Width  int32
	Height int32
}
