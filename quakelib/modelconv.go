package quakelib

//#include "gl_model.h"
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
	return [4]model.Hull{}
}
