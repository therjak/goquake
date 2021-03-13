// SPDX-License-Identifier: GPL-2.0-or-later
package spr

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/therjak/goquake/math/vec"
	qm "github.com/therjak/goquake/model"
	"github.com/therjak/goquake/texture"
)

func init() {
	qm.Register(Magic, loadM)
}

type Model struct {
	name string
	mins vec.Vec3
	maxs vec.Vec3

	FrameCount int // numframes
	SyncType   int

	Data Sprite
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

func loadM(name string, data []byte) ([]qm.Model, error) {
	mod, err := load(name, data)
	if err != nil {
		return nil, err
	}
	return []qm.Model{mod}, nil
}

//TODO: move into Model
type Sprite struct {
	Type      SpriteType
	MaxWidth  int32
	MaxHeight int32
	// FrameCount int32 -> just look at len(Frames)
	Frames []*RawFrame
}

type RawFrame struct {
	// if len(Frames) == 1 => Frame otherwise FrameGroup
	Frames []*Frame
}

func floor(f float32) float32 {
	if f < 1 {
		return 0
	}
	x := math.Float32bits(f)
	e := uint32(x>>(32-8-1))&0xFF - 127
	// Clear the non integer bits.
	if e < 32-8-1 {
		x &^= 1<<(32-8-1-e) - 1
	}
	return math.Float32frombits(x)
}

func (r *RawFrame) Frame(t float32) *Frame {
	if len(r.Frames) == 1 {
		// SPR_SINGLE
		return r.Frames[0]
	}
	lastFrame := r.Frames[len(r.Frames)-1]
	fullInterval := lastFrame.interval
	target := t - floor(t/fullInterval)*fullInterval
	for _, f := range r.Frames {
		if target < f.interval {
			return f
		}
	}
	return lastFrame
}

type Frame struct {
	interval float32
	Up       float32
	Down     float32
	Left     float32
	Right    float32
	Texture  *texture.Texture
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
	if h.Version != spriteVersion {
		return nil, fmt.Errorf("%s has wrong version number (%d should be %d)", name, h.Version, spriteVersion)
	}
	if h.FrameCount < 1 {
		return nil, fmt.Errorf("Mod_LoadSpriteModel: Invalid # of frames: %v", h.FrameCount)
	}
	mod.mins = vec.Vec3{
		float32(-h.MaxWidth / 2),
		float32(-h.MaxWidth / 2),
		float32(-h.MaxHeight / 2),
	}
	mod.maxs = vec.Vec3{
		float32(h.MaxWidth / 2),
		float32(h.MaxWidth / 2),
		float32(h.MaxHeight / 2),
	}
	mod.FrameCount = int(h.FrameCount)
	mod.SyncType = int(h.SyncType)

	sprite := &mod.Data
	sprite.Type = SpriteType(h.Typ)
	sprite.MaxWidth = h.MaxWidth
	sprite.MaxHeight = h.MaxHeight

	for i := 0; i < mod.FrameCount; i++ {
		var t FrameType
		err := binary.Read(buf, binary.LittleEndian, &t)
		if err != nil {
			return nil, err
		}
		switch t {
		case SPR_SINGLE:
			r, err := readSingleFrame(buf, name, i)
			if err != nil {
				return nil, err
			}
			sprite.Frames = append(sprite.Frames, r)
		default: // SPR_GROUP
			g, err := readFrameGroup(buf, name, i)
			if err != nil {
				return nil, err
			}
			sprite.Frames = append(sprite.Frames, g)
		}
	}

	return mod, nil
}

func readSingleFrame(buf *bytes.Reader, name string, index int) (*RawFrame, error) {
	f, err := readFrame(buf, name, index)
	if err != nil {
		return nil, err
	}
	r := &RawFrame{
		Frames: []*Frame{f},
	}
	return r, nil
}

func readFrameGroup(buf *bytes.Reader, name string, index int) (*RawFrame, error) {
	// read int32 as numframes
	// read [numframes]float32 as intervals
	// check all interval > 0
	// for numframes
	//   read frame
	//   frame.interval = intervals[numframe]
	//   r.Frames = append(r.Frames, frame)
	return nil, fmt.Errorf("readFrameGroup: not implemented")
}

func readFrame(buf *bytes.Reader, modName string, index int) (*Frame, error) {
	var f frame
	err := binary.Read(buf, binary.LittleEndian, &f)
	if err != nil {
		return nil, err
	}
	out := &Frame{
		Up:    float32(f.Origin[1]),
		Down:  float32(f.Origin[1] - f.Height),
		Left:  float32(f.Origin[0]),
		Right: float32(f.Origin[0] + f.Width),
	}
	size := f.Width * f.Height
	data := make([]byte, size)
	l, err := buf.Read(data)
	if err != nil {
		return nil, err
	}
	if l != int(size) {
		return nil, fmt.Errorf("readFrame: not enough pixel data")
	}
	flags := texture.TexPrefPad | texture.TexPrefAlpha | texture.TexPrefNoPicMip
	name := fmt.Sprintf("%s:frame%d", modName, index)
	out.Texture = texture.NewTexture(f.Width, f.Height, flags, name, texture.ColorTypeIndexed, data)
	return out, nil
}
