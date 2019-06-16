package spr

import (
	"quake/math/vec"
	qm "quake/model"
)

func init() {
	qm.Register(Magic, Load)
}

func Load(name string, data []byte) ([]*qm.QModel, error) {
	var ret []*qm.QModel
	mod := &qm.QModel{
		Name: name,
		Type: qm.ModSprite,
		Mins: vec.Vec3{999999, 999999, 999999},
		// YMins: vec.Vec3{999999,999999,999999},
		// RMins: vec.Vec3{999999,999999,999999},
		Maxs: vec.Vec3{-999999, -999999, -999999},
		// YMaxs: vec.Vec3{-999999,-999999,-999999},
		// RMaxs: vec.Vec3{-999999,-999999,-999999},
	}

	// TODO: load the actual model
	ret = append(ret, mod)
	return ret, nil
}
