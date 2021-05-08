// SPDX-License-Identifier: GPL-2.0-or-later

package mdl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"

	"github.com/chewxy/math32"
	"github.com/therjak/goquake/math/vec"
	qm "github.com/therjak/goquake/model"
	"github.com/therjak/goquake/texture"
)

const (
	NoLerp         = 256
	NoShadow       = 512
	FullBrightHack = 1024
)

func init() {
	qm.Register(Magic, loadM)
}

type Model struct {
	AliasHeader
	name  string
	mins  vec.Vec3
	maxs  vec.Vec3
	flags int
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
	Frames    []Frame
	Triangles []Triangle
	Radius    float32
	YawRadius float32
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
	Vertices []Vertex
}

type Vertex struct {
	Point  vec.Vec3
	Normal vec.Vec3
}

type frameGroup struct {
	af aliasFrame
	fv []frameVertex
}

type Triangle struct {
	FacesFront bool
	Indices    [3]int
}

func load(name string, data []byte) (*Model, error) {
	mod := &Model{
		name: name,
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
	mod.SkinCount = int(h.SkinCount)
	mod.SkinWidth = int(h.SkinWidth)
	mod.SkinHeight = int(h.SkinHeight)
	mod.VerticeCount = int(h.VerticeCount)
	mod.TriangleCount = int(h.TriangleCount)
	mod.FrameCount = int(h.FrameCount)
	mod.Scale = vec.Vec3{h.Scale[0], h.Scale[1], h.Scale[2]}
	mod.Translate = vec.Vec3{h.Translate[0], h.Translate[1], h.Translate[2]}
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
				mod.Textures = append(mod.Textures, []*texture.Texture{t})
				mod.FBTextures = append(mod.FBTextures, []*texture.Texture{fbt})
			} else {
				tf := texture.TexPrefPad
				t := texture.NewTexture(h.SkinWidth, h.SkinHeight, tf, tn, texture.ColorTypeIndexed, data)
				mod.Textures = append(mod.Textures, []*texture.Texture{t})
				mod.FBTextures = append(mod.FBTextures, []*texture.Texture{})
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

	textureCoords := make([]skinVertex, h.VerticeCount) // read in gl_mesh.c
	if err := binary.Read(buf, binary.LittleEndian, textureCoords); err != nil {
		return nil, err
	}
	mod.TextureCoords = make([]TextureCoord, h.VerticeCount)
	for i := int32(0); i < h.VerticeCount; i++ {
		mod.TextureCoords[i] = TextureCoord{
			OnSeam: textureCoords[i].OnSeam != 0,
			S:      (float32(textureCoords[i].S) + 0.5) / float32(h.SkinWidth),
			T:      (float32(textureCoords[i].T) + 0.5) / float32(h.SkinHeight),
		}
	}

	triangles := make([]triangle, h.TriangleCount) // read in gl_mesh.c
	if err := binary.Read(buf, binary.LittleEndian, triangles); err != nil {
		return nil, err
	}
	mod.Triangles = make([]Triangle, len(triangles))
	for i := range triangles {
		t := &triangles[i]
		mod.Triangles[i] = Triangle{
			FacesFront: t.FacesFront != 0,
			Indices:    [3]int{int(t.Vertices[0]), int(t.Vertices[1]), int(t.Vertices[2])},
		}
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
		for fgi := range fs[i].fg {
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

	mod.Frames = make([]Frame, len(frames))

	var radius float32
	var yawRadius float32
	for i := 0; i < len(frames); i++ {
		f := &frames[i]
		F := &mod.Frames[i]
		F.Interval = f.interval
		F.Group = make([]FrameGroup, len(f.fg))
		for j := 0; j < len(f.fg); j++ {
			fg := &f.fg[j]
			FG := &F.Group[j]
			FG.Vertices = make([]Vertex, len(fg.fv))
			for k := 0; k < len(fg.fv); k++ {
				fv := &fg.fv[k]
				V := &FG.Vertices[k]
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

				dist := v[0]*v[0] + v[1]*v[1]
				if yawRadius < dist {
					yawRadius = dist
				}
				dist += v[2] * v[2]
				if radius < dist {
					radius = dist
				}
			}
		}
	}
	mod.mins = mins
	mod.maxs = maxs
	mod.Radius = math32.Sqrt(radius)
	mod.YawRadius = math32.Sqrt(yawRadius)
}
