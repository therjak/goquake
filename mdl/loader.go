package mdl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"quake/math/vec"
	qm "quake/model"
	// "github.com/chewxy/math32"
)

func init() {
	qm.Register(Magic, Load)
}

func Load(name string, data []byte) ([]*qm.QModel, error) {
	var ret []*qm.QModel
	mod := &qm.QModel{
		Name: name,
		Type: qm.ModAlias,
	}

	buf := bytes.NewReader(data)
	h := header{}
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return nil, err
	}
	if h.Version != aliasVersion {
		return nil, fmt.Errorf("%s has wrong version number (%d should be %d)", name, h.Version, aliasVersion)
	}
	mod.SyncType = int(h.SyncType)
	mod.Flags = int(h.Flags)

	for i := int32(0); i < h.SkinCount; i++ {
		skinCount := int32(1)
		skinType := int32(0)
		err := binary.Read(buf, binary.LittleEndian, &skinType)
		if err != nil {
			return nil, err
		}
		if skinType != ALIAS_SKIN_SINGLE {
			err = binary.Read(buf, binary.LittleEndian, &skinCount)
			if err != nil {
				return nil, err
			}
			for j := int32(0); j < skinCount; j++ {
				skinInterval := float32(0) // how long each skin should be shown
				// TODO: shouldn't we do something with this data?
				err = binary.Read(buf, binary.LittleEndian, &skinInterval)
				if err != nil {
					return nil, err
				}
			}
		}
		for j := int32(0); j < skinCount; j++ {
			// TODO: actually read the groupskins instead of just skipping them
			buf.Seek(int64(h.SkinWidth)*int64(h.SkinHeight), io.SeekCurrent)
		}
	}

	vert := skinVertex{}
	for i := int32(0); i < h.VerticeCount; i++ {
		err := binary.Read(buf, binary.LittleEndian, &vert)
		if err != nil {
			return nil, err
		}
	}

	triangle := triangle{}
	for i := int32(0); i < h.TriangleCount; i++ {
		err := binary.Read(buf, binary.LittleEndian, &triangle)
		if err != nil {
			return nil, err
		}
	}

	// Now the h.FrameCount frames

	// read int32 to determine the pframetype
	// frameVertex gets filled in Mod_LoadAliasFrame and/or Mod_LoadAliasGroup

	/*
		pframetype = //int32
		for i := 0; i < h.FrameCount; i++ {
			if pframetype.type == ALIAS_SINGLE {
				pframetype, frame = Mod_LoadAliasFrame(pframetype + 1)
				pheader.frames[i] = frame
			} else {
				pframetype, frame = Mod_LoadAliasGroup(pframetype + 1)
				pheader.frames[i] = frame
			}
		}
	*/
	pv := [][]frameVertex{} // 256 per line?
	calcAliasBounds(mod, &h, pv)

	ret = append(ret, mod)
	return ret, nil
}

func calcAliasBounds(mod *qm.QModel,
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

	mod.Mins = vec.Vec3{999999, 999999, 999999}
	mod.Maxs = vec.Vec3{-999999, -999999, -999999}
	/*
	 radius := float32(0)
	 yawradius := float32(0)
	 mod.YMins = mod.Mins
	 mod.RMins = mod.Mins
	 mod.YMaxs = mod.Maxs
	 mod.RMaxs = mod.Maxs
	*/

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
			/*
				dist := v[0]*v[0] + v[1]*v[1]
				if yawradius < dist {
					yawradius = dist
				}
				dist += v[2] * v[2]
				if radius < dist {
					radius = dist
				}
			*/
		}
	}
	/*
		radius = math32.Sqrt(radius)
		yawradius = math32.Sqrt(yawradius)
		mod.YMins = vec.Vec3{-yawradius, -yawradius, mod.Mins[2]}
		mod.YMaxs = vec.Vec3{yawradius, yawradius, mod.Maxs[2]}

		mod.RMins = vec.Vec3{-radius, -radius, -radius}
		mod.RMaxs = vec.Vec3{radius, radius, radius}
	*/
}
