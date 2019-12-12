package quakelib

//#include "q_stdinc.h"
//#include "render.h"
//extern entity_t *cl_entities;
//extern entity_t cl_viewent;
//typedef entity_t* entityPtr;
//entity_t* getCLEntity(int i) { return &cl_entities[i]; }
import "C"

import (
	"quake/math/vec"
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
