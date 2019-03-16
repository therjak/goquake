package quakelib

//#include "gl_model.h"
//#ifndef MODELCONF_H
//#define MODELCONF_H
//inline mclipnode_t* getClipNode(mclipnode_t* n, int idx) { return &n[idx]; }
//inline mplane_t* getPlane(mplane_t* n, int idx) { return &n[idx]; }
//#endif
import "C"

import (
	"quake/math"
	"quake/model"
)

//export SVSetModel
func SVSetModel(m *C.qmodel_t, idx C.int) {
	nm := &model.QModel{
		Name:     C.GoString(&m.name[0]),
		Type:     model.ModType(m.Type),
		Mins:     math.Vec3{float32(m.mins[0]), float32(m.mins[1]), float32(m.mins[2])},
		Maxs:     math.Vec3{float32(m.maxs[0]), float32(m.maxs[1]), float32(m.maxs[2])},
		ClipMins: math.Vec3{float32(m.clipmins[0]), float32(m.clipmins[1]), float32(m.clipmins[2])},
		ClipMaxs: math.Vec3{float32(m.clipmaxs[0]), float32(m.clipmaxs[1]), float32(m.clipmaxs[2])},
		Hulls:    convHulls(&m.hulls),
	}
	if int(idx) == len(sv.models) {
		sv.models = append(sv.models, nm)
	} else {
		sv.models[int(idx)] = nm
	}
}

func convHulls(h *[4]C.hull_t) [4]model.Hull {
	var r [4]model.Hull
	for i := 0; i < 4; i++ {
		r[i].FirstClipNode = int(h[i].firstclipnode)
		r[i].LastClipNode = int(h[i].lastclipnode)
		r[i].ClipMins = v3v3(h[i].clip_mins)
		r[i].ClipMaxs = v3v3(h[i].clip_maxs)
		r[i].Planes = convPlanes(h[i].planes, int(h[i].numPlanes))
		r[i].ClipNodes = convClipNodes(h[i].clipnodes, int(h[i].lastclipnode)+1)
	}
	return r
}

func v3v3(v C.vec3_t) math.Vec3 {
	return math.Vec3{
		X: float32(v[0]),
		Y: float32(v[1]),
		Z: float32(v[2]),
	}
}

func convPlanes(ps *C.mplane_t, num int) []model.Plane {
	var r []model.Plane
	for i := 0; i < num; i++ {
		p := C.getPlane(ps, C.int(i))
		r = append(r, model.Plane{
			Normal:   v3v3(p.normal),
			Dist:     float32(p.dist),
			Type:     byte(p.Type),
			SignBits: byte(p.signbits),
		})
	}
	return r
}

func convClipNodes(cn *C.mclipnode_t, num int) []model.ClipNode {
	var r []model.ClipNode
	for i := 0; i < num; i++ {
		n := C.getClipNode(cn, C.int(i))
		r = append(r, model.ClipNode{
			PlaneNum: int(n.planenum),
			Children: [2]int{int(n.children[0]), int(n.children[1])},
		})
	}
	return r
}
