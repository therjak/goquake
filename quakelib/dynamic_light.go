// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"goquake/bsp"
	"goquake/math/vec"
)

type DynamicLight struct {
	origin   vec.Vec3
	radius   float32
	dieTime  float64 // stop after this time
	decay    float32 // drop this each second
	minLight float32 // don't add when contributing less
	key      int
	color    vec.Vec3
}

func (d *DynamicLight) Color() vec.Vec3 {
	return d.color
}
func (d *DynamicLight) MinLight() float32 {
	return d.minLight
}
func (d *DynamicLight) Origin() vec.Vec3 {
	return d.origin
}
func (d *DynamicLight) Radius() float32 {
	return d.radius
}

//GetDynamicLightByKey return the light with the same key or if none exists a free light
func (c *Client) GetDynamicLightByKey(key int) *DynamicLight {
	// key 0 is worldEntity or 'unowned'. world can have more than one
	for i := range c.dynamicLights {
		d := &c.dynamicLights[i]
		if d.key == key {
			return d
		}
	}
	return c.GetFreeDynamicLight()
}

func (c *Client) GetFreeDynamicLight() *DynamicLight {
	for i := range c.dynamicLights {
		d := &c.dynamicLights[i]
		if d.dieTime < c.time {
			return d
		}
	}
	c.dynamicLights = append(c.dynamicLights, DynamicLight{})
	return &c.dynamicLights[len(c.dynamicLights)-1]
}

func (c *Client) clearDLights() {
	for i := range c.dynamicLights {
		c.dynamicLights[i] = DynamicLight{}
	}
}

func (c *Client) DecayLights() {
	t := cl.time - cl.oldTime
	for i := range c.dynamicLights {
		dl := &c.dynamicLights[i]
		if dl.dieTime < t || dl.radius == 0 {
			continue
		}
		dl.radius -= float32(t) * dl.decay
		if dl.radius < 0 {
			dl.radius = 0
		}
	}
}

func markLights(node bsp.Node) {
	for i := range cl.dynamicLights {
		dl := &cl.dynamicLights[i]
		if dl.dieTime < cl.time || dl.radius == 0 {
			continue
		}
		markLight(dl, i, node)
	}
}

func markLight(dl *DynamicLight, num int, node bsp.Node) {
	switch n := node.(type) {
	case *bsp.MNode:
		markLight2(dl, num, n)
	}
}

func markLight2(dl *DynamicLight, num int, node *bsp.MNode) {
	sp := node.Plane
	var dist float32
	if sp.Type < 3 {
		dist = dl.origin[sp.Type] - sp.Dist
	} else {
		dist = vec.Dot(dl.origin, sp.Normal) - sp.Dist
	}
	if dist > dl.radius {
		markLight(dl, num, node.Children[0])
		return
	}
	if dist < -dl.radius {
		markLight(dl, num, node.Children[1])
		return
	}
	markLight3(dl, num, node, dist)
}

func markLight3(dl *DynamicLight, num int, node *bsp.MNode, dist float32) {
	maxDist := dl.radius * dl.radius
	for i := range node.Surfaces {
		surf := node.Surfaces[i]
		impact := vec.Sub(dl.origin, (vec.Scale(dist, surf.Plane.Normal)))
		s := surf.LightImpactCenter(impact, bsp.S)
		t := surf.LightImpactCenter(impact, bsp.T)
		if s*s+t*t+dist*dist < maxDist {
			for num >= len(surf.DLightBits) {
				surf.DLightBits = append(surf.DLightBits, make([]bool, 8)...)
			}
			surf.DLightBits[num] = true
			surf.DLightFrame = renderer.lightFrameCount
		}
	}
	markLight(dl, num, node.Children[0])
	markLight(dl, num, node.Children[1])
}
