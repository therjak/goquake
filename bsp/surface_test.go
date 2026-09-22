// SPDX-License-Identifier: GPL-2.0-or-later

package bsp

import (
	"testing"

	"goquake/math/vec"
)

type testDLight struct {
	origin   vec.Vec3
	radius   float32
	minLight float32
	color    vec.Vec3
}

func (d *testDLight) Origin() vec.Vec3  { return d.origin }
func (d *testDLight) Radius() float32   { return d.radius }
func (d *testDLight) MinLight() float32 { return d.minLight }
func (d *testDLight) Color() vec.Vec3   { return d.color }

func TestBuildLightMap_DynamicLights(t *testing.T) {
	surf := &Surface{
		extents:     [2]int{16, 16},
		textureMins: [2]int{0, 0},
		Plane: &Plane{
			Normal: vec.Vec3{0, 0, 1},
			Dist:   0,
		},
		TexInfo: &TexInfo{
			Vecs: [2]TexInfoPos{
				{Pos: vec.Vec3{1, 0, 0}, Offset: 0},
				{Pos: vec.Vec3{0, 1, 0}, Offset: 0},
			},
		},
		Styles:       [4]byte{0xff, 0xff, 0xff, 0xff},
		DLightFrame:  1,
		DLightBits:   []bool{true},
		LightmapData: make([]byte, 2*2*4),
	}

	dl := &testDLight{
		origin:   vec.Vec3{8, 8, 10},
		radius:   100,
		minLight: 0,
		color:    vec.Vec3{1, 1, 1},
	}

	lights := []DynamicLight{dl}
	dynamicStyles := LightStyles{}

	// 1. Frame 1: dynamic light is active
	surf.BuildLightMap(dynamicStyles, 1, lights, false)
	if !surf.CachedDLight {
		t.Errorf("expected CachedDLight to be true on active dynamic light frame")
	}
	if surf.LightmapData[0] == 0 && surf.LightmapData[1] == 0 && surf.LightmapData[2] == 0 {
		t.Errorf("expected surface to be lit by dynamic light, got black")
	}

	// 2. Frame 2: dynamic light moved away (DLightFrame is still 1, current frame is 2)
	if !surf.NeedsLightmapUpdate(dynamicStyles, 2) {
		t.Errorf("expected NeedsLightmapUpdate to be true on frame following dynamic light")
	}

	surf.BuildLightMap(dynamicStyles, 2, lights, false)
	if surf.CachedDLight {
		t.Errorf("expected CachedDLight to be false after dynamic light ended")
	}
	if surf.LightmapData[0] != 0 || surf.LightmapData[1] != 0 || surf.LightmapData[2] != 0 {
		t.Errorf("expected surface to return to unlit (black) after dynamic light ended, got %v", surf.LightmapData[:3])
	}

	// 3. Frame 3: no dynamic lights, static lightstyles unchanged -> NeedsLightmapUpdate should be false
	if surf.NeedsLightmapUpdate(dynamicStyles, 3) {
		t.Errorf("expected NeedsLightmapUpdate to be false when no lights changed")
	}
}

func TestBuildLightMap_MultiLightStyles(t *testing.T) {
	// Surface with 2x2 luxels (4 luxels total, 12 bytes per lightstyle layer).
	// Style 0: base light (all 10)
	// Style 1: torch light (first luxel 50, rest 0)
	size := 2 * 2 * 3
	lightSamples := make([]byte, size*2)
	for i := range size {
		lightSamples[i] = 10
	}
	// Second layer: first pixel R=50, G=50, B=50
	lightSamples[size+0] = 50
	lightSamples[size+1] = 50
	lightSamples[size+2] = 50

	surf := &Surface{
		extents:      [2]int{16, 16},
		textureMins:  [2]int{0, 0},
		Styles:       [4]byte{0, 1, 0xff, 0xff},
		LightSamples: lightSamples,
		LightmapData: make([]byte, 2*2*4),
	}

	dynamicStyles := LightStyles{
		0: 256, // style 0 scale = 1.0 (256/256)
		1: 256, // style 1 scale = 1.0 (256/256)
	}

	// Build with overbright=true (>>8)
	surf.BuildLightMap(dynamicStyles, 1, nil, true)

	// Pixel 0 should combine style 0 (10) + style 1 (50) = 60
	if surf.LightmapData[0] != 60 {
		t.Errorf("expected pixel 0 to have combined value 60, got %d", surf.LightmapData[0])
	}
	// Pixel 1 should have only style 0 (10) + style 1 (0) = 10
	if surf.LightmapData[4] != 10 {
		t.Errorf("expected pixel 1 to have value 10, got %d", surf.LightmapData[4])
	}
}

func TestLightAt_BoundaryConditions(t *testing.T) {
	// Surface 16x16 extents -> 2x2 luxels = 12 bytes
	size := 2 * 2 * 3
	lightSamples := make([]byte, size)
	for i := range size {
		lightSamples[i] = 128
	}

	surf := &Surface{
		extents:     [2]int{16, 16},
		textureMins: [2]int{0, 0},
		Plane: &Plane{
			Normal: vec.Vec3{0, 0, 1},
			Dist:   0,
		},
		TexInfo: &TexInfo{
			Vecs: [2]TexInfoPos{
				{Pos: vec.Vec3{1, 0, 0}, Offset: 0},
				{Pos: vec.Vec3{0, 1, 0}, Offset: 0},
			},
		},
		Styles:       [4]byte{0, 0xff, 0xff, 0xff},
		LightSamples: lightSamples,
	}

	leaf0 := &MLeaf{NodeBase: NewNodeBase(-1, 0, [6]float32{})}
	leaf1 := &MLeaf{NodeBase: NewNodeBase(-1, 0, [6]float32{})}
	node := &MNode{
		Plane: &Plane{
			Normal: vec.Vec3{0, 0, 1},
			Dist:   0,
			Type:   2,
		},
		Children: [2]Node{leaf0, leaf1},
		Surfaces: []*Surface{surf},
	}

	mod := &Model{
		Node:      node,
		lightData: lightSamples,
	}

	styles := &LightStyles{
		0: 256,
	}

	// Test boundary points (ds=0, ds=16, dt=0, dt=16, ds=8, dt=8)
	testPoints := []vec.Vec3{
		{0, 0, 10},
		{16, 0, 10},
		{0, 16, 10},
		{16, 16, 10},
		{8, 8, 10},
		{15.9, 15.9, 10},
	}

	for _, p := range testPoints {
		c := mod.LightAt(p, styles)
		if c[0] == 0 && c[1] == 0 && c[2] == 0 {
			t.Errorf("LightAt(%v) returned (0,0,0), expected light", p)
		}
	}
}
