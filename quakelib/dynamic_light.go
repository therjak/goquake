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

//export CL_AllocDlight
func CL_AllocDlight(key int) *C.dlight_t {
	clean := func(i int) {
		cl.dynamicLights[i] = DynamicLight{
			Key:   key, // == index in cl_entities
			Color: vec.Vec3{1, 1, 1},
		}
		cl.dynamicLights[i].Sync()
	}
	if key != 0 {
		// key 0 is worldEntity or 'unowned'. world can have more than one
		for i := 0; i < C.MAX_DLIGHTS; i++ {
			d := &C.cl_dlights[i]
			if d.key == C.int(key) {
				clean(i)
				return &C.cl_dlights[i]
			}
		}
	}
	for i := 0; i < C.MAX_DLIGHTS; i++ {
		d := &C.cl_dlights[i]
		if d.die < C.float(cl.time) {
			clean(i)
			return &C.cl_dlights[i]
		}
	}
	clean(0)
	return &C.cl_dlights[0]
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

//export CL_ClearDLights
func CL_ClearDLights() {
	for i, _ := range cl.dynamicLights {
		cl.dynamicLights[i] = DynamicLight{}
		cl.dynamicLights[i].ptr = &C.cl_dlights[i]
		cl.dynamicLights[i].Sync()
	}
}

func (d *DynamicLight) Sync() {
	d.ptr.origin[0] = C.float(d.Origin[0])
	d.ptr.origin[1] = C.float(d.Origin[1])
	d.ptr.origin[2] = C.float(d.Origin[2])
	d.ptr.radius = C.float(d.Radius)
	d.ptr.die = C.float(d.DieTime)
	d.ptr.decay = C.float(d.Decay)
	d.ptr.minlight = C.float(d.MinLight)
	d.ptr.key = C.int(d.Key)
	d.ptr.color[0] = C.float(d.Color[0])
	d.ptr.color[1] = C.float(d.Color[1])
	d.ptr.color[2] = C.float(d.Color[2])
}

//export CL_Dlight
func CL_Dlight(idx int) *C.dlight_t {
	return cl.dynamicLights[idx].ptr
}

func CL_DecayLights() {
	t := C.float(cl.time - cl.oldTime)
	for i := 0; i < C.MAX_DLIGHTS; i++ {
		dl := &C.cl_dlights[i]
		if dl.die < t || dl.radius == 0 {
			continue
		}
		dl.radius -= t * dl.decay
		if dl.radius < 0 {
			dl.radius = 0
		}
	}
}

//export R_MarkLights
func R_MarkLights(node *C.mnode_t) {
	for i := 0; i < C.MAX_DLIGHTS; i++ {
		dl := &C.cl_dlights[i]
		if float64(dl.die) < cl.time || dl.radius == 0 {
			continue
		}
		C.R_MarkLight(dl, C.int(i), node)
	}
}
