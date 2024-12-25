// SPDX-License-Identifier: GPL-2.0-or-later

package mdl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"unsafe"

	"goquake/conlog"
	"goquake/filesystem"
	"goquake/glh"
	"goquake/math/vec"
	qm "goquake/model"
	"goquake/texture"

	"github.com/chewxy/math32"
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
	name  string
	mins  vec.Vec3
	maxs  vec.Vec3
	flags int

	Scale     vec.Vec3
	Translate vec.Vec3
	SkinCount int
	// VerticeCount is the number of vertices in VertexArrayBuffer
	VerticeCount int
	verticeCount int32 // internal from header
	// IndiceCount is the number of indices in VertexElementArrayBuffer
	IndiceCount int
	// STOffset is the offset inside the VertexArrayBuffer to the st values
	STOffset      int
	triangleCount int
	frameCount    int
	SyncType      int

	poseCount int32

	textureCoords []TextureCoord

	Textures   [][]*texture.Texture
	FBTextures [][]*texture.Texture
	Frames     []Frame
	triangles  []Triangle
	Radius     float32
	YawRadius  float32

	VertexElementArrayBuffer     *glh.Buffer
	vertexElementArrayBufferData []uint16

	// VertexArrayBuffer is a gl.Buffer with following data layout:
	// ([ 4 int8 xyzw, 4 uint8 normal(xyzw) ] * numverts ) * posecount
	// (2 float32 s,t texcoord)
	// use an offset to jump to correct pose, afterwards you have numverts
	// vertices with normals as the actual 'model'
	// the texcoords are consecutive at the end of the vbo (use STOffset) and
	// match the order of the verts inside a pose
	VertexArrayBuffer     *glh.Buffer
	vertexArrayBufferData []byte
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

func loadM(name string, file filesystem.File) ([]qm.Model, error) {
	mod, err := load(name, file)
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

func load(name string, buf io.ReadSeeker) (*Model, error) {
	mod := &Model{
		name: name,
	}

	h := header{}
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return nil, err
	}
	if h.Version != aliasVersion {
		return nil, fmt.Errorf("%s has wrong version number (%d should be %d)", name, h.Version, aliasVersion)
	}
	if h.SkinHeight > 480 {
		conlog.DWarning("model %s has a skin taller than %d", name, 480)
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
	mod.verticeCount = h.VerticeCount
	mod.triangleCount = int(h.TriangleCount)
	mod.frameCount = int(h.FrameCount)
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

	textureCoords := make([]skinVertex, h.VerticeCount)
	if err := binary.Read(buf, binary.LittleEndian, textureCoords); err != nil {
		return nil, err
	}
	mod.textureCoords = make([]TextureCoord, h.VerticeCount)
	for i := int32(0); i < h.VerticeCount; i++ {
		mod.textureCoords[i] = TextureCoord{
			OnSeam: textureCoords[i].OnSeam != 0,
			S:      (float32(textureCoords[i].S) + 0.5) / float32(h.SkinWidth),
			T:      (float32(textureCoords[i].T) + 0.5) / float32(h.SkinHeight),
		}
	}

	triangles := make([]triangle, h.TriangleCount)
	if err := binary.Read(buf, binary.LittleEndian, triangles); err != nil {
		return nil, err
	}
	mod.triangles = make([]Triangle, len(triangles))
	for i := range triangles {
		t := &triangles[i]
		mod.triangles[i] = Triangle{
			FacesFront: t.FacesFront != 0,
			Indices:    [3]int{int(t.Vertices[0]), int(t.Vertices[1]), int(t.Vertices[2])},
		}
	}

	mod.poseCount = 0
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
			// This should be able to support variable frame rates. It does not look
			// like any engine supports it but all just read the first.
			fs[i].interval = intervals[0]
		}
		mod.poseCount += groupFrames
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
	mod.setupBuffers(fs)

	return mod, nil
}

type aliasmesh struct {
	s         float32
	t         float32
	vertindex uint16
}

func aNormal(f float32) byte {
	var r byte
	*(*int8)(unsafe.Pointer(&r)) = int8(127 * f)
	return byte(r)
}

func (m *Model) setupBuffers(frames []frame) {
	maxVerts := len(m.triangles) * 3
	indices := make([]uint16, 0, maxVerts)
	desc := make([]aliasmesh, 0, maxVerts)

	for _, t := range m.triangles {
		for j := 0; j < 3; j++ {
			idx := uint16(t.Indices[j])
			tcoord := m.textureCoords[idx]
			// Check for back side
			if !t.FacesFront && tcoord.OnSeam {
				tcoord.S += 0.5
			}

			var v int
			for v = 0; v < len(desc); v++ {
				d := &desc[v]
				if d.vertindex == idx && d.s == tcoord.S && d.t == tcoord.T {
					indices = append(indices, uint16(v))
					break
				}
			}
			if v == len(desc) {
				indices = append(indices, uint16(v))
				desc = append(desc, aliasmesh{
					vertindex: idx,
					s:         tcoord.S,
					t:         tcoord.T,
				})
			}
		}
	}

	m.IndiceCount = len(indices)
	m.vertexElementArrayBufferData = indices

	sizeofMeshPos := m.poseCount * m.verticeCount * 8 // 4 uint8 + 4 int8
	sizeofTS := m.verticeCount * 8                    // 2 float32
	vboBuf := bytes.NewBuffer(make([]byte, 0, sizeofMeshPos+sizeofTS))

	m.VerticeCount = len(desc)
	for fi := range frames {
		f := &frames[fi]
		for gi := range f.fg {
			fg := &f.fg[gi]
			for _, d := range desc {
				fv := &fg.fv[d.vertindex]
				av := avertexNormals[fv.LightNormalIndex]
				buf := [8]byte{
					fv.PackedPosition[0], fv.PackedPosition[1], fv.PackedPosition[2], 1,
					aNormal(av[0]), aNormal(av[1]), aNormal(av[2]), 0}
				vboBuf.Write(buf[:])
			}
		}
	}

	m.STOffset = vboBuf.Len()
	for _, d := range desc {
		buf := [8]byte{}
		*(*float32)(unsafe.Pointer(&buf[0])) = d.s
		*(*float32)(unsafe.Pointer(&buf[4])) = d.t
		vboBuf.Write(buf[:])
	}

	m.vertexArrayBufferData = vboBuf.Bytes()
}

func (m *Model) UploadBuffer() {
	m.VertexElementArrayBuffer = glh.NewBuffer(glh.ElementArrayBuffer)
	m.VertexElementArrayBuffer.Bind()
	m.VertexElementArrayBuffer.SetData(2*len(m.vertexElementArrayBufferData), glh.Ptr(m.vertexElementArrayBufferData))

	m.VertexArrayBuffer = glh.NewBuffer(glh.ArrayBuffer)
	m.VertexArrayBuffer.Bind()
	m.VertexArrayBuffer.SetData(len(m.vertexArrayBufferData), glh.Ptr(m.vertexArrayBufferData))
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
