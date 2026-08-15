// SPDX-License-Identifier: GPL-2.0-or-later

package bsp

import (
	"goquake/math"
	"goquake/math/vec"
)

const MaxLightStyles = 64

// LightStyles contain MaxLightStyles values to scale light inside a map.
type LightStyles [MaxLightStyles]int

type color struct {
	R, G, B int
}

func (m *Model) recursiveLight(s *LightStyles, node Node, start, end vec.Vec3, c *vec.Vec3) bool {
	nextChild := func(f float32) int {
		if f < 0 {
			return 1
		}
		return 0
	}
	var n *MNode
	var front, back float32
	for (back < 0) == (front < 0) {
		if node.Contents() < 0 {
			return false
		}
		n = node.(*MNode)
		plane := n.Plane
		if plane.Type < 3 {
			front = start[plane.Type] - plane.Dist
			back = end[plane.Type] - plane.Dist
		} else {
			front = vec.Dot(start, plane.Normal) - plane.Dist
			back = vec.Dot(end, plane.Normal) - plane.Dist
		}
		node = n.Children[nextChild(front)]
	}
	frac := front / (front - back)
	mid := vec.Lerp(start, end, frac)

	// front side
	if m.recursiveLight(s, n.Children[nextChild(front)], start, mid, c) {
		return true
	}

	for _, surface := range n.Surfaces {
		if surface.Flags&SurfaceDrawTiled != 0 {
			continue
		}
		ti := surface.TexInfo
		ds := int(vec.DoublePrecDot(mid, ti.Vecs[S].Pos) + float64(ti.Vecs[S].Offset))
		dt := int(vec.DoublePrecDot(mid, ti.Vecs[T].Pos) + float64(ti.Vecs[T].Offset))
		if ds < surface.textureMins[0] || dt < surface.textureMins[1] {
			continue
		}
		ds -= surface.textureMins[0]
		dt -= surface.textureMins[1]
		if ds > surface.extents[S] || dt > surface.extents[T] {
			continue
		}
		if len(surface.LightSamples) > 0 {
			var c00, c01, c10, c11 color
			dsfrac := ds & 15
			dtfrac := dt & 15
			s0 := ds >> 4
			t0 := dt >> 4
			smax := (surface.extents[S] >> 4) + 1
			tmax := (surface.extents[T] >> 4) + 1

			s1 := s0 + 1
			if s1 >= smax {
				s1 = s0
			}
			t1 := t0 + 1
			if t1 >= tmax {
				t1 = t0
			}

			p00 := (t0*smax + s0) * 3
			p01 := (t0*smax + s1) * 3
			p10 := (t1*smax + s0) * 3
			p11 := (t1*smax + s1) * 3
			mapSize := smax * tmax * 3

			for maps := 0; maps < 4 && surface.Styles[maps] != 255; maps++ {
				mapBase := maps * mapSize
				if mapBase+mapSize <= len(surface.LightSamples) {
					scale := float32(s[surface.Styles[maps]]) / 256.0
					c00.R += int(float32(surface.LightSamples[mapBase+p00+0]) * scale)
					c00.G += int(float32(surface.LightSamples[mapBase+p00+1]) * scale)
					c00.B += int(float32(surface.LightSamples[mapBase+p00+2]) * scale)
					c01.R += int(float32(surface.LightSamples[mapBase+p01+0]) * scale)
					c01.G += int(float32(surface.LightSamples[mapBase+p01+1]) * scale)
					c01.B += int(float32(surface.LightSamples[mapBase+p01+2]) * scale)
					c10.R += int(float32(surface.LightSamples[mapBase+p10+0]) * scale)
					c10.G += int(float32(surface.LightSamples[mapBase+p10+1]) * scale)
					c10.B += int(float32(surface.LightSamples[mapBase+p10+2]) * scale)
					c11.R += int(float32(surface.LightSamples[mapBase+p11+0]) * scale)
					c11.G += int(float32(surface.LightSamples[mapBase+p11+1]) * scale)
					c11.B += int(float32(surface.LightSamples[mapBase+p11+2]) * scale)
				}
			}
			(*c)[0] += float32((((((((c11.R - c10.R) * dsfrac) >> 4) + c10.R) -
				((((c01.R - c00.R) * dsfrac) >> 4) + c00.R)) * dtfrac) >> 4) +
				((((c01.R - c00.R) * dsfrac) >> 4) + c00.R))
			(*c)[1] += float32((((((((c11.G - c10.G) * dsfrac) >> 4) + c10.G) -
				((((c01.G - c00.G) * dsfrac) >> 4) + c00.G)) * dtfrac) >> 4) +
				((((c01.G - c00.G) * dsfrac) >> 4) + c00.G))
			(*c)[2] += float32((((((((c11.B - c10.B) * dsfrac) >> 4) + c10.B) -
				((((c01.B - c00.B) * dsfrac) >> 4) + c00.B)) * dtfrac) >> 4) +
				((((c01.B - c00.B) * dsfrac) >> 4) + c00.B))
		}
		return true
	}
	// back side
	return m.recursiveLight(s, n.Children[nextChild(-front)], mid, end, c)
}

// LightAt return the light color at point p scaled by light style values in s
func (m *Model) LightAt(p vec.Vec3, s *LightStyles) vec.Vec3 {
	if len(m.lightData) == 0 {
		return vec.Vec3{255, 255, 255}
	}

	end := p
	end[2] -= 8192

	color := vec.Vec3{0, 0, 0}
	m.recursiveLight(s, m.Node, p, end, &color)
	return color
}

func (s *Surface) LightImpactCenter(impact vec.Vec3, st ST) float32 {
	v := s.TexInfo.Vecs[st]
	// clamp center of light to corner and check brightness
	l := vec.Dot(impact, v.Pos) + v.Offset - float32(s.textureMins[st])
	return l - math.Clamp(0, l+0.5, float32(s.extents[st]))
}
