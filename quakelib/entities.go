package quakelib

//#include "q_stdinc.h"
//#include "render.h"
//extern entity_t *cl_entities;
//extern entity_t cl_viewent;
//typedef entity_t* entityPtr;
//entity_t* getCLEntity(int i) { return &cl_entities[i]; }
//entity_t *CL_EntityNum(int num); // has error checks
import "C"

import (
	"github.com/therjak/goquake/math/vec"
)

type Entity struct {
	ptr C.entityPtr
}

func cl_entities(i int) Entity {
	return Entity{C.getCLEntity(C.int(i))}
}

func (e *Entity) origin() vec.Vec3 {
	return vec.Vec3{
		float32(e.ptr.origin[0]),
		float32(e.ptr.origin[1]),
		float32(e.ptr.origin[2]),
	}
}

func (e *Entity) angles() vec.Vec3 {
	return vec.Vec3{
		float32(e.ptr.angles[0]),
		float32(e.ptr.angles[1]),
		float32(e.ptr.angles[2]),
	}
}

func cl_weapon() Entity {
	return Entity{&C.cl_viewent}
}

func CL_EntityNum(num int) *Entity {
	return &Entity{C.CL_EntityNum(C.int(num))}
}

func (e *Entity) SetBaseline(state *EntityState) {
	e.ptr.baseline.origin[0] = C.float(state.Origin[0])
	e.ptr.baseline.origin[1] = C.float(state.Origin[1])
	e.ptr.baseline.origin[2] = C.float(state.Origin[2])
	e.ptr.baseline.angles[0] = C.float(state.Angles[0])
	e.ptr.baseline.angles[1] = C.float(state.Angles[1])
	e.ptr.baseline.angles[2] = C.float(state.Angles[2])
	e.ptr.baseline.modelindex = C.ushort(state.ModelIndex)
	e.ptr.baseline.frame = C.ushort(state.Frame)
	e.ptr.baseline.skin = C.uchar(state.Skin)
	e.ptr.baseline.alpha = C.uchar(state.Alpha)
}
