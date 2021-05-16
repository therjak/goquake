// SPDX-License-Identifier: GPL-2.0-or-later

package bsp

import (
	"github.com/therjak/goquake/math/vec"
)

const MaxLightStyles = 64

// LightStyles contain MaxLightStyles values to scale light inside a map.
type LightStyles [MaxLightStyles]int

type color struct {
	R, G, B int
}

func (m *Model) recursiveLight(s *LightStyles, node Node, start, end vec.Vec3, c *vec.Vec3) bool {
	front := float32(1)
	back := float32(-1)
	nextChild := func(f float32) int {
		if f < 0 {
			return 1
		}
		return 0
	}
	var n *MNode
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

	for _, surface := range m.Surfaces[n.FirstSurface : n.FirstSurface+n.SurfaceCount] {
		if surface.Flags&SurfaceDrawTiled != 0 {
			continue
		}
		ti := surface.TexInfo
		ds := int(vec.DoublePrecDot(mid, ti.Vecs[0].Pos) + float64(ti.Vecs[0].Offset))
		dt := int(vec.DoublePrecDot(mid, ti.Vecs[1].Pos) + float64(ti.Vecs[1].Offset))
		if ds < surface.textureMins[0] || dt < surface.textureMins[1] {
			continue
		}
		ds -= surface.textureMins[0]
		dt -= surface.textureMins[1]
		if ds > surface.extents[0] || dt > surface.extents[1] {
			continue
		}
		if len(surface.LightSamples) > 0 {
			var c00, c01, c10, c11 color
			dsfrac := ds & 15
			dtfrac := dt & 15
			line := ((surface.extents[0] >> 4) + 1) * 3
			lightMap := surface.LightSamples[(dt>>4)*line+(ds>>4)*3:]
			for maps := 0; maps < 4 && surface.Styles[maps] != 255; maps++ {
				scale := float32(s[surface.Styles[maps]]) / 256.0
				c00.R += int(float32(lightMap[0]) * scale)
				c00.G += int(float32(lightMap[1]) * scale)
				c00.B += int(float32(lightMap[2]) * scale)
				c01.R += int(float32(lightMap[3]) * scale)
				c01.G += int(float32(lightMap[4]) * scale)
				c01.B += int(float32(lightMap[5]) * scale)
				c10.R += int(float32(lightMap[line+0]) * scale)
				c10.G += int(float32(lightMap[line+1]) * scale)
				c10.B += int(float32(lightMap[line+2]) * scale)
				c11.R += int(float32(lightMap[line+3]) * scale)
				c11.G += int(float32(lightMap[line+4]) * scale)
				c11.B += int(float32(lightMap[line+5]) * scale)
				lightMap = lightMap[((surface.extents[0]>>4)+1)*((surface.extents[1]>>4)+1)*3:]
			}
			(*c)[0] += float32((((((((c11.R - c10.R) * dsfrac) >> 4) + c10.R) -
				(((c01.R - c00.R) * dsfrac) >> 4) + c00.R) * dtfrac) >> 4) +
				((((c01.R - c00.R) * dsfrac) >> 4) + c00.R))
			(*c)[1] += float32((((((((c11.G - c10.G) * dsfrac) >> 4) + c10.G) -
				(((c01.G - c00.G) * dsfrac) >> 4) + c00.G) * dtfrac) >> 4) +
				((((c01.G - c00.G) * dsfrac) >> 4) + c00.G))
			(*c)[2] += float32((((((((c11.B - c10.B) * dsfrac) >> 4) + c10.B) -
				(((c01.B - c00.B) * dsfrac) >> 4) + c00.B) * dtfrac) >> 4) +
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
