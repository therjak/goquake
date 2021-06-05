// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

//#include <stdio.h>
//#include "dlight.h"
//#include "gl_model.h"
//void R_MarkLight(dlight_t* light, int num, mnode_t *node);
import "C"
import (
	"github.com/therjak/goquake/math/vec"
)

type DynamicLight struct {
	ptr      *C.dlight_t
	Origin   vec.Vec3
	Radius   float32
	DieTime  float64 // stop after this time
	Decay    float32 // drop this each second
	MinLight float32 // don't add when contributing less
	Key      int
	Color    vec.Vec3
}

//GetDynamicLightByKey return the light with the same key or if none exists a free light
func (c *Client) GetDynamicLightByKey(key int) *DynamicLight {
	// key 0 is worldEntity or 'unowned'. world can have more than one
	for i := range c.dynamicLights {
		d := &c.dynamicLights[i]
		if d.Key == key {
			return d
		}
	}
	return c.GetFreeDynamicLight()
}

func (c *Client) GetFreeDynamicLight() *DynamicLight {
	for i := range c.dynamicLights {
		d := &c.dynamicLights[i]
		if d.DieTime < c.time {
			return d
		}
	}
	return &c.dynamicLights[0]
}

func (c *Client) clearDLights() {
	for i := range c.dynamicLights {
		c.dynamicLights[i] = DynamicLight{}
		c.dynamicLights[i].ptr = &C.cl_dlights[i]
		c.dynamicLights[i].Sync()
	}
}

func (d *DynamicLight) Sync() {
	d.ptr.origin[0] = C.float(d.Origin[0])
	d.ptr.origin[1] = C.float(d.Origin[1])
	d.ptr.origin[2] = C.float(d.Origin[2])
	d.ptr.radius = C.float(d.Radius)
	d.ptr.die = C.float(d.DieTime)
	d.ptr.minlight = C.float(d.MinLight)
	d.ptr.color[0] = C.float(d.Color[0])
	d.ptr.color[1] = C.float(d.Color[1])
	d.ptr.color[2] = C.float(d.Color[2])
}

//export CL_Dlight
func CL_Dlight(idx int) *C.dlight_t {
	return cl.dynamicLights[idx].ptr
}

func (c *Client) DecayLights() {
	t := cl.time - cl.oldTime
	for i := range c.dynamicLights {
		dl := &c.dynamicLights[i]
		if dl.DieTime < t || dl.Radius == 0 {
			continue
		}
		dl.Radius -= float32(t) * dl.Decay
		if dl.Radius < 0 {
			dl.Radius = 0
		}
		dl.Sync()
	}
}

//export R_MarkLights
func R_MarkLights(node *C.mnode_t) {
	for i := range cl.dynamicLights {
		dl := &cl.dynamicLights[i]
		if dl.DieTime < cl.time || dl.Radius == 0 {
			continue
		}
		C.R_MarkLight(dl.ptr, C.int(i), node)
	}
}
