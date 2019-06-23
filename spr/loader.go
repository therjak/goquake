package spr

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	}
	buf := bytes.NewReader(data)
	h := header{}
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return nil, err
	}
	if h.Version != spriteVersion {
		return nil, fmt.Errorf("%s has wrong version number (%d should be %d)", name, h.Version, spriteVersion)
	}
	mod.Mins = vec.Vec3{
		float32(-h.MaxWidth / 2),
		float32(-h.MaxWidth / 2),
		float32(-h.MaxHeight / 2),
	}
	mod.Maxs = vec.Vec3{
		float32(h.MaxWidth / 2),
		float32(h.MaxWidth / 2),
		float32(h.MaxHeight / 2),
	}
	mod.FrameCount = int(h.FrameCount)
	if mod.FrameCount < 1 {
		return nil, fmt.Errorf("Mod_LoadSpriteModel: Invalid # of frames: %v", mod.FrameCount)
	}
	mod.SyncType = int(h.SyncType)

	// TODO: load the 'extra data' filled in cache.data
	//       it gets accessed in r_sprite.c
	// Do something better than this hacky cache.data + random cast
	ret = append(ret, mod)
	return ret, nil
}