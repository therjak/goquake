package snd

import (
	"goquake/math/vec"
	"runtime"

	"github.com/google/uuid"
)

type SoundPrecache struct {
	sys *SndSys
	id  uuid.UUID
}

func (sys *SndSys) NewPrecache(snds ...Sound) *SoundPrecache {
	s := &SoundPrecache{
		sys: sys,
		id:  uuid.Must(uuid.NewV7()),
	}
	sys.addCache <- cacheRequest{
		id:   s.id,
		snds: snds,
	}
	runtime.AddCleanup(s, func(id uuid.UUID) {
		sys.removeCache <- id
	}, s.id)
	return s
}

func (sp *SoundPrecache) Start(entnum int, entchannel int, sfx int, sndOrigin vec.Vec3, fvol float32, attenuation float32) {
	if sp.sys == nil {
		return
	}
	sp.sys.start <- Start{
		entityNum:   entnum,
		entityChan:  entchannel,
		cache:       sp.id,
		sfx:         sfx,
		origin:      sndOrigin,
		volume:      fvol,
		attenuation: attenuation,
		looping:     false,
	}
}

func (sp *SoundPrecache) StartAmbient(sfx int, sndOrigin vec.Vec3, fvol float32, attenuation float32) {
	if sp.sys == nil {
		return
	}
	sp.sys.start <- Start{
		entityNum:   0,
		entityChan:  0,
		cache:       sp.id,
		sfx:         sfx,
		origin:      sndOrigin,
		volume:      fvol,
		attenuation: attenuation,
		looping:     true,
	}
}
