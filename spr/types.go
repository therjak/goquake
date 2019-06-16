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
	name           [4]byte // "IDSP"
	version        int32   // SPRITE_VERSION
	typ            int32   // SPR_SINGLE or SPR_GROUP
	boundingRadius float32
	maxWidth       int32
	maxHeight      int32
	frameCount     int32
	beamLength     float32
	syncType       int32 // ST_SYNC or ST_RAND
}

type frame struct {
	origin [2]int32
	width  int32
	height int32
}
