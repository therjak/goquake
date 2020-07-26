package quakelib

//#include "dlight.h"
//#include "gl_model.h"
//typedef dlight_t* dlightPtr;
//void R_MarkLight(dlight_t* light, int num, mnode_t *node);
//#ifndef HAS_GETDLIGHT
//#define HAS_GETDLIGHT
//static inline dlight_t* getDlight(int i) { return &cl_dlights[i]; }
//#endif
import "C"

type dynamicLight C.dlight_t

//export CL_AllocDlight
func CL_AllocDlight(key int) *C.dlight_t {
	clean := func(i int) {
		dl := &C.cl_dlights[i]
		dl.origin[0] = 0
		dl.origin[1] = 0
		dl.origin[2] = 0
		dl.radius = 0
		dl.die = 0
		dl.decay = 0
		dl.minlight = 0
		dl.key = C.int(key)
		dl.color[0] = 1
		dl.color[1] = 1
		dl.color[2] = 1
	}
	if key != 0 {
		for i := 0; i < C.MAX_DLIGHTS; i++ {
			d := &C.cl_dlights[i]
			if d.key == C.int(key) {
				clean(i)
				return C.getDlight(C.int(i))
			}
		}
	}
	for i := 0; i < C.MAX_DLIGHTS; i++ {
		d := &C.cl_dlights[i]
		if d.die < C.float(cl.time) {
			clean(i)
			return C.getDlight(C.int(i))
		}
	}
	clean(0)
	return C.getDlight(0)
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
