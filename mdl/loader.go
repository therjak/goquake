// SPDX-License-Identifier: GPL-2.0-or-later
package mdl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/therjak/goquake/math/vec"
	qm "github.com/therjak/goquake/model"
)

func init() {
	qm.Register(Magic, loadM)
}

type Model struct {
	name  string
	mins  vec.Vec3
	maxs  vec.Vec3
	flags int

	FrameCount int
	SyncType   int

	Header *AliasHeader
}

type AliasHeader struct {
	Scale         vec.Vec3
	ScaleOrigin   vec.Vec3
	SkinCount     int
	SkinWidth     int
	SkinHeight    int
	VerticeCount  int
	TriangleCount int
	FrameCount    int
	SyncType      int
	Flags         int

	// numverts_vbo
	// meshdesc intptr_t
	// numindexes int
	// indexs intptr_t
	// vertexes intptr_t

	// PoseCount int
	// PoseVerts int
	// PoseData  int
	// Commands  int

	// GlTextures [32][4]uint32
	// FbTextures [32][4]uint32
	// Texels     [32]int
	// Frames [FrameCount-1]Frame
}

const (
	MaxAliasVerts  = 2000
	MaxAliasFrames = 256
	MaxAliasTris   = 2048
)

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
	return q.flags
}

func (q *Model) AddFlag(f int) {
	q.flags |= f
}

func loadM(name string, data []byte) ([]qm.Model, error) {
	mod, err := load(name, data)
	if err != nil {
		return nil, err
	}
	return []qm.Model{mod}, nil
}

func load(name string, data []byte) (*Model, error) {
	mod := &Model{
		name:   name,
		Header: &AliasHeader{},
	}

	buf := bytes.NewReader(data)
	h := header{}
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return nil, err
	}
	if h.Version != aliasVersion {
		return nil, fmt.Errorf("%s has wrong version number (%d should be %d)", name, h.Version, aliasVersion)
	}
	if h.SkinHeight > 480 {
		return nil, fmt.Errorf("model %s has a skin taller than %d", name, 480)
	}
	if h.VerticeCount <= 0 {
		return nil, fmt.Errorf("model %s has no vertices", name)
	}
	if h.VerticeCount > MaxAliasVerts {
		return nil, fmt.Errorf("model %s has too many vertices", name)
	}
	if h.TriangleCount <= 0 {
		return nil, fmt.Errorf("model %s has no triangles", name)
	}
	if h.FrameCount < 1 {
		return nil, fmt.Errorf("model %s has invalid # of frames: %d", name, h.FrameCount)
	}
	header := mod.Header
	header.SkinCount = int(h.SkinCount)
	header.SkinWidth = int(h.SkinWidth)
	header.SkinHeight = int(h.SkinHeight)
	header.VerticeCount = int(h.VerticeCount)
	header.TriangleCount = int(h.TriangleCount)
	header.FrameCount = int(h.FrameCount)
	header.Scale = vec.Vec3{h.Scale[0], h.Scale[1], h.Scale[2]}
	header.ScaleOrigin = vec.Vec3{h.ScaleOrigin[0], h.ScaleOrigin[1], h.ScaleOrigin[2]}
	mod.SyncType = int(h.SyncType)
	mod.flags = (int(h.Flags & 0xff))

	for i := int32(0); i < h.SkinCount; i++ { // See Mod_LoadAllSkins
		skinCount := int32(1)
		skinType := int32(0)
		err := binary.Read(buf, binary.LittleEndian, &skinType)
		if err != nil {
			return nil, err
		}
		if skinType != ALIAS_SKIN_SINGLE {
			err = binary.Read(buf, binary.LittleEndian, &skinCount)
			if err != nil {
				return nil, err
			}
			for j := int32(0); j < skinCount; j++ {
				skinInterval := float32(0) // how long each skin should be shown
				// TODO: shouldn't we do something with this data?
				err = binary.Read(buf, binary.LittleEndian, &skinInterval)
				if err != nil {
					return nil, err
				}
			}
		}
		for j := int32(0); j < skinCount; j++ {
			// TODO: actually read the groupskins instead of just skipping them
			buf.Seek(int64(h.SkinWidth)*int64(h.SkinHeight), io.SeekCurrent)
		}
	}

	verts := make([]skinVertex, h.VerticeCount) // read in gl_mesh.c
	if err := binary.Read(buf, binary.LittleEndian, &verts); err != nil {
		return nil, err
	}

	triangles := make([]triangle, h.TriangleCount) // read in gl_mesh.c
	if err := binary.Read(buf, binary.LittleEndian, &triangles); err != nil {
		return nil, err
	}

	// Now the h.FrameCount frames

	// read int32 to determine the pframetype
	// frameVertex gets filled in Mod_LoadAliasFrame and/or Mod_LoadAliasGroup

	/*
		pframetype = //int32
		for i := 0; i < h.FrameCount; i++ {
			if pframetype.type == ALIAS_SINGLE {
				pframetype, frame = Mod_LoadAliasFrame(pframetype + 1)
				pheader.frames[i] = frame
			} else {
				pframetype, frame = Mod_LoadAliasGroup(pframetype + 1)
				pheader.frames[i] = frame
			}
		}
	*/
	pv := [][]frameVertex{} // 256 per line?
	calcAliasBounds(mod, &h, pv)

	return mod, nil
}

func calcAliasBounds(mod *Model,
	pheader *header, poseverts [][]frameVertex) {
	min := func(a, b float32) float32 {
		if a < b {
			return a
		}
		return b
	}
	max := func(a, b float32) float32 {
		if a < b {
			return b
		}
		return a
	}

	mins := vec.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	maxs := vec.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
	/*
	 radius := float32(0)
	 yawradius := float32(0)
	 mod.YMins = mod.Mins
	 mod.RMins = mod.Mins
	 mod.YMaxs = mod.Maxs
	 mod.RMaxs = mod.Maxs
	*/

	for i := 0; i < len(poseverts); i++ {
		for j := 0; j < len(poseverts[i]); j++ {
			v := vec.Vec3{
				float32(poseverts[i][j].PackedPosition[0])*pheader.Scale[0] + pheader.ScaleOrigin[0],
				float32(poseverts[i][j].PackedPosition[1])*pheader.Scale[1] + pheader.ScaleOrigin[1],
				float32(poseverts[i][j].PackedPosition[2])*pheader.Scale[0] + pheader.ScaleOrigin[2],
			}
			mins[0] = min(mins[0], v[0])
			mins[1] = min(mins[1], v[1])
			mins[2] = min(mins[2], v[2])
			maxs[0] = max(maxs[0], v[0])
			maxs[1] = max(maxs[1], v[1])
			maxs[2] = max(maxs[2], v[2])
			/*
				dist := v[0]*v[0] + v[1]*v[1]
				if yawradius < dist {
					yawradius = dist
				}
				dist += v[2] * v[2]
				if radius < dist {
					radius = dist
				}
			*/
		}
	}
	mod.mins = mins
	mod.maxs = maxs
	/*
		radius = math32.Sqrt(radius)
		yawradius = math32.Sqrt(yawradius)
		mod.YMins = vec.Vec3{-yawradius, -yawradius, mod.Mins[2]}
		mod.YMaxs = vec.Vec3{yawradius, yawradius, mod.Maxs[2]}

		mod.RMins = vec.Vec3{-radius, -radius, -radius}
		mod.RMaxs = vec.Vec3{radius, radius, radius}
	*/
}
