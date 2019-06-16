package mdl

import (
	"quake/math/vec"
	qm "quake/model"
)

type aliashdr struct {
	PoseCount    int
	VerticeCount int
}

func init() {
	qm.Register(Magic, Load)
}

func Load(name string, data []byte) ([]*qm.QModel, error) {
	var ret []*qm.QModel
	mod := &qm.QModel{
		Name: name,
		Type: qm.ModAlias,
		Mins: vec.Vec3{999999, 999999, 999999},
		// YMins: vec.Vec3{999999,999999,999999},
		// RMins: vec.Vec3{999999,999999,999999},
		Maxs: vec.Vec3{-999999, -999999, -999999},
		// YMaxs: vec.Vec3{-999999,-999999,-999999},
		// RMaxs: vec.Vec3{-999999,-999999,-999999},
	}

	// TODO: load the actual model

	pv := [][]frameVertex{} // 256 per line?
	ah := aliashdr{}
	ph := header{}
	calcAliasBounds(mod, &ah, &ph, pv)

	ret = append(ret, mod)
	return ret, nil
}

func calcAliasBounds(mod *qm.QModel, ah *aliashdr,
	pheader *header, poseverts [][]frameVertex) {
	min := func(a, b float32) float32 {
		if a < b {
			return a
		}
		return b
	}
	max := func(a, b float32) float32 {
		if a < b {
			return b
		}
		return a
	}

	for i := 0; i < len(poseverts); i++ {
		for j := 0; j < len(poseverts[i]); j++ {
			v := vec.Vec3{
				float32(poseverts[i][j].PackedPosition[0])*pheader.Scale[0] + pheader.ScaleOrigin[0],
				float32(poseverts[i][j].PackedPosition[1])*pheader.Scale[1] + pheader.ScaleOrigin[1],
				float32(poseverts[i][j].PackedPosition[2])*pheader.Scale[0] + pheader.ScaleOrigin[2],
			}
			mod.Mins[0] = min(mod.Mins[0], v[0])
			mod.Mins[1] = min(mod.Mins[1], v[1])
			mod.Mins[2] = min(mod.Mins[2], v[2])
			mod.Maxs[0] = max(mod.Maxs[0], v[0])
			mod.Maxs[1] = max(mod.Maxs[1], v[1])
			mod.Maxs[2] = max(mod.Maxs[2], v[2])
		}
	}
}
