// SPDX-License-Identifier: GPL-2.0-or-later
package mdl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"

	"github.com/therjak/goquake/math/vec"
	qm "github.com/therjak/goquake/model"
	"github.com/therjak/goquake/texture"
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
	Translate     vec.Vec3
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

	TextureCoords []TextureCoord

	Textures   [][]*texture.Texture
	FBTextures [][]*texture.Texture
	// Texels     [32]int
	Frames []Frame
}

type TextureCoord struct {
	OnSeam bool
	S      float32
	T      float32
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

func fullBright(data []byte) bool {
	for _, d := range data {
		if d > 223 {
			return true
		}
	}
	return false
}

type Frame struct {
	Group    []FrameGroup
	Interval float32
}

type frame struct {
	fg       []frameGroup
	interval float32
}

type FrameGroup struct {
	Verticies []Vertex
}

type Vertex struct {
	Point  vec.Vec3
	Normal vec.Vec3
}

type frameGroup struct {
	af aliasFrame
	fv []frameVertex
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
	if h.SkinCount < 1 || h.SkinCount > 32 {
		return nil, fmt.Errorf("model %s has invalid # of skins: %d", name, h.SkinCount)
	}
	header := mod.Header
	header.SkinCount = int(h.SkinCount)
	header.SkinWidth = int(h.SkinWidth)
	header.SkinHeight = int(h.SkinHeight)
	header.VerticeCount = int(h.VerticeCount)
	header.TriangleCount = int(h.TriangleCount)
	header.FrameCount = int(h.FrameCount)
	header.Scale = vec.Vec3{h.Scale[0], h.Scale[1], h.Scale[2]}
	header.Translate = vec.Vec3{h.Translate[0], h.Translate[1], h.Translate[2]}
	mod.SyncType = int(h.SyncType)
	mod.flags = (int(h.Flags & 0xff))

	skinSize := int64(h.SkinWidth) * int64(h.SkinHeight)
	for i := int32(0); i < h.SkinCount; i++ { // See Mod_LoadAllSkins
		skinType := int32(0)
		err := binary.Read(buf, binary.LittleEndian, &skinType)
		if err != nil {
			return nil, err
		}
		if skinType == ALIAS_SKIN_SINGLE {
			// TODO: FloodFillSkin
			tn := fmt.Sprintf("%s:frame%d", name, i)
			data := make([]byte, skinSize)
			buf.Read(data)
			if fullBright(data) {
				fbtn := fmt.Sprintf("%s:frame%d_glow", name, i)
				fbtf := texture.TexPrefPad | texture.TexPrefFullBright
				tf := texture.TexPrefPad | texture.TexPrefNoBright
				t := texture.NewTexture(h.SkinWidth, h.SkinHeight, tf, tn, texture.ColorTypeIndexed, data)
				fbt := texture.NewTexture(h.SkinWidth, h.SkinHeight, fbtf, fbtn, texture.ColorTypeIndexed, data)
				header.Textures = append(header.Textures, []*texture.Texture{t})
				header.FBTextures = append(header.Textures, []*texture.Texture{fbt})
			} else {
				tf := texture.TexPrefPad
				t := texture.NewTexture(h.SkinWidth, h.SkinHeight, tf, tn, texture.ColorTypeIndexed, data)
				header.Textures = append(header.Textures, []*texture.Texture{t})
				header.FBTextures = append(header.Textures, []*texture.Texture{})
			}
		} else {
			log.Printf("TODO: ALIAS_SKIN_GROUP")
			skinCount := int32(1)
			err = binary.Read(buf, binary.LittleEndian, &skinCount)
			if err != nil {
				return nil, err
			}
			// TODO: shouldn't we do something with this data?
			skinInterval := make([]float32, skinCount)
			if err := binary.Read(buf, binary.LittleEndian, &skinInterval); err != nil {
				return nil, err
			}
			// TODO: actually read the groupskins instead of just skipping them
			buf.Seek(skinSize*int64(skinCount), io.SeekCurrent)
		}
	}

	// texture coordinates
	// move to (0.0, 1.0) by adding 0.5 and divide by skinWidth for s and skinHeight for t
	textureCoords := make([]skinVertex, h.VerticeCount) // read in gl_mesh.c
	if err := binary.Read(buf, binary.LittleEndian, textureCoords); err != nil {
		return nil, err
	}
	header.TextureCoords = make([]TextureCoord, h.VerticeCount)
	for i := int32(0); i < h.VerticeCount; i++ {
		header.TextureCoords[i] = TextureCoord{
			OnSeam: textureCoords[i].OnSeam != 0,
			S:      (float32(textureCoords[i].S) + 0.5) / float32(h.SkinWidth),
			T:      (float32(textureCoords[i].T) + 0.5) / float32(h.SkinHeight),
		}
	}

	triangles := make([]triangle, h.TriangleCount) // read in gl_mesh.c
	if err := binary.Read(buf, binary.LittleEndian, triangles); err != nil {
		return nil, err
	}

	fs := make([]frame, h.FrameCount)
	for i := int32(0); i < h.FrameCount; i++ {
		frameType := int32(0)
		if err := binary.Read(buf, binary.LittleEndian, &frameType); err != nil {
			log.Printf("TODO: ERR")
			return nil, err
		}
		groupFrames := int32(1)
		fs[i].interval = 0.1
		if frameType != ALIAS_SINGLE {
			log.Printf("FrameType: %v, %s", frameType, name)
			fg := aliasFrameGroup{}
			if err := binary.Read(buf, binary.LittleEndian, &fg); err != nil {
				log.Printf("TODO: ERR")
				return nil, err
			}
			groupFrames = fg.FrameCount
			intervals := make([]float32, fg.FrameCount)
			if err := binary.Read(buf, binary.LittleEndian, intervals); err != nil {
				log.Printf("TODO: ERR")
				return nil, err
			}
			// This should be able to support variable frame rates. It does not look like
			// any engine supports it but all just read the first.
			fs[i].interval = intervals[0]
		}
		fs[i].fg = make([]frameGroup, groupFrames)
		for fgi := range fs[i].fg { // int32(0); gf < groupFrames; gf++ {
			fg := &fs[i].fg[fgi]
			if err := binary.Read(buf, binary.LittleEndian, &(fg.af)); err != nil {
				log.Printf("TODO: ERR")
				return nil, err
			}
			fg.fv = make([]frameVertex, h.VerticeCount)
			if err := binary.Read(buf, binary.LittleEndian, fg.fv); err != nil {
				log.Printf("TODO: ERR, %v, %v", fgi, groupFrames)
				return nil, err
			}
			for _, v := range fg.fv {
				if int(v.LightNormalIndex) >= len(avertexNormals) {
					return nil, fmt.Errorf("Normals out of bounds")
				}
			}
		}
	}

	calcFrames(mod, &h, fs)

	return mod, nil
}

func calcFrames(mod *Model, pheader *header, frames []frame) {
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

	mod.Header.Frames = make([]Frame, len(frames))

	for i := 0; i < len(frames); i++ {
		f := &frames[i]
		F := &mod.Header.Frames[i]
		F.Interval = f.interval
		F.Group = make([]FrameGroup, len(f.fg))
		for j := 0; j < len(f.fg); j++ {
			fg := &f.fg[j]
			FG := &F.Group[j]
			FG.Verticies = make([]Vertex, len(fg.fv))
			for k := 0; k < len(fg.fv); k++ {
				fv := &fg.fv[k]
				V := &FG.Verticies[k]
				v := vec.Vec3{
					float32(fv.PackedPosition[0])*pheader.Scale[0] + pheader.Translate[0],
					float32(fv.PackedPosition[1])*pheader.Scale[1] + pheader.Translate[1],
					float32(fv.PackedPosition[2])*pheader.Scale[2] + pheader.Translate[2],
				}
				V.Point = v
				V.Normal = avertexNormals[fv.LightNormalIndex]

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
