package quakelib

//#include "gl_model.h"
//#ifndef MODELCONF_H
//#define MODELCONF_H
//inline mclipnode_t* getClipNode(mclipnode_t* n, int idx) { return &n[idx]; }
//inline mplane_t* getPlane(mplane_t* n, int idx) { return &n[idx]; }
//inline mleaf_t* AsLeaf(mnode_t* n) { return (mleaf_t*)(n); }
//#endif
//int SVWorldModelLeafIndex(mleaf_t* l);
import "C"

import (
	"log"
	"quake/math"
	"quake/model"
)

//export SVSetModel
func SVSetModel(m *C.qmodel_t, idx C.int) {
	nm := convCModel(m)
	if int(idx) == len(sv.models) {
		sv.models = append(sv.models, nm)
	} else {
		sv.models[int(idx)] = nm
	}
}

//export SVSetWorldModel
func SVSetWorldModel(m *C.qmodel_t) {
	sv.worldModel = convCModel(m)
}

func convCModel(m *C.qmodel_t) *model.QModel {
	log.Printf("ModelName: %s", C.GoString(&m.name[0]))
	return &model.QModel{
		Name:     C.GoString(&m.name[0]),
		Type:     model.ModType(m.Type),
		Mins:     math.Vec3{float32(m.mins[0]), float32(m.mins[1]), float32(m.mins[2])},
		Maxs:     math.Vec3{float32(m.maxs[0]), float32(m.maxs[1]), float32(m.maxs[2])},
		ClipMins: math.Vec3{float32(m.clipmins[0]), float32(m.clipmins[1]), float32(m.clipmins[2])},
		ClipMaxs: math.Vec3{float32(m.clipmaxs[0]), float32(m.clipmaxs[1]), float32(m.clipmaxs[2])},
		Hulls:    convHulls(&m.hulls),
		Node:     convNode(m.nodes),
	}
}

func convNode(n *C.mnode_t) model.Node {
	if n == nil {
		return nil
	}
	if n.contents == 0 {
		plane := convPlane(n.plane)
		r := &model.MNode{
			NodeBase: model.NewNodeBase(
				0, int(n.visframe),
				[6]float32{
					float32(n.minmaxs[0]), float32(n.minmaxs[1]), float32(n.minmaxs[2]),
					float32(n.minmaxs[3]), float32(n.minmaxs[4]), float32(n.minmaxs[5]),
				}),
			Plane: &plane,
			Children: [2]model.Node{
				convNode(n.children[0]),
				convNode(n.children[1]),
			},
			FirstSurface: uint32(n.firstsurface),
			NumSurfaces:  uint32(n.numsurfaces),
		}
		if r.Children[0] != nil {
			r.Children[0].SetParent(r)
		}
		if r.Children[1] != nil {
			r.Children[1].SetParent(r)
		}
		return r
	}
	// we actually got a C.mleaf_t
	// TODO: we need a pointer to the same leaf in QModel.Leafs
	idx := C.SVWorldModelLeafIndex(C.AsLeaf(n))
	log.Printf("Leaf nr: %d, cont: %d", idx, n.contents)
	return nil
}

func convHulls(h *[4]C.hull_t) [4]model.Hull {
	var r [4]model.Hull
	for i := 0; i < 4; i++ {
		r[i].FirstClipNode = int(h[i].firstclipnode)
		r[i].LastClipNode = int(h[i].lastclipnode)
		r[i].ClipMins = v3v3(h[i].clip_mins)
		r[i].ClipMaxs = v3v3(h[i].clip_maxs)
		r[i].Planes = convPlanes(h[i].planes, int(h[i].numPlanes))
		if h[i].clipnodes != nil {
			r[i].ClipNodes = convClipNodes(h[i].clipnodes, int(h[i].lastclipnode)+1)
		}
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

func convPlane(p *C.mplane_t) model.Plane {
	return model.Plane{
		Normal:   v3v3(p.normal),
		Dist:     float32(p.dist),
		Type:     byte(p.Type),
		SignBits: byte(p.signbits),
	}
}

func convPlanes(ps *C.mplane_t, num int) []model.Plane {
	var r []model.Plane
	for i := 0; i < num; i++ {
		p := C.getPlane(ps, C.int(i))
		r = append(r, convPlane(p))
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
