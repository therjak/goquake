package mdl

import (
	"quake/math"
	qm "quake/model"
)

type aliashdr struct {
	PoseCount    int
	VerticeCount int
}

func Load(name string, data []byte) ([]*qm.QModel, error) {
	var ret []*qm.QModel
	mod := &qm.QModel{
		Name: name,
		Type: qm.ModAlias,
		Mins: math.Vec3{999999, 999999, 999999},
		// YMins: math.Vec3{999999,999999,999999},
		// RMins: math.Vec3{999999,999999,999999},
		Maxs: math.Vec3{-999999, -999999, -999999},
		// YMaxs: math.Vec3{-999999,-999999,-999999},
		// RMaxs: math.Vec3{-999999,-999999,-999999},
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
			v := math.Vec3{
				float32(poseverts[i][j].PackedPosition[0])*pheader.Scale[0] + pheader.ScaleOrigin[0],
				float32(poseverts[i][j].PackedPosition[1])*pheader.Scale[1] + pheader.ScaleOrigin[1],
				float32(poseverts[i][j].PackedPosition[2])*pheader.Scale[0] + pheader.ScaleOrigin[2],
			}
			mod.Mins.X = min(mod.Mins.X, v.X)
			mod.Mins.Y = min(mod.Mins.Y, v.Y)
			mod.Mins.Z = min(mod.Mins.Z, v.Z)
			mod.Maxs.X = max(mod.Maxs.X, v.X)
			mod.Maxs.Y = max(mod.Maxs.Y, v.Y)
			mod.Maxs.Z = max(mod.Maxs.Z, v.Z)
		}
	}
}
