package snd

import (
	"goquake/math/vec"
	"log/slog"
)

type SoundPrecache struct {
	sys *SndSys
	c   map[int]int
}

func (sys *SndSys) NewPrecache(snds ...Sound) *SoundPrecache {
	s := &SoundPrecache{
		sys: sys,
		c:   make(map[int]int),
	}
	s.set(snds...)
	return s
}

func (sp *SoundPrecache) Start(entnum int, entchannel int, id int, sndOrigin vec.Vec3, fvol float32, attenuation float32) {
	if sp.sys == nil {
		return
	}
	sfx, ok := sp.c[id]
	if !ok {
		slog.Error("unknown sound started", slog.Int("sfx", id))
		return
	}
	sp.sys.start <- Start{
		entityNum:   entnum,
		entityChan:  entchannel,
		sfx:         sfx,
		origin:      sndOrigin,
		volume:      fvol,
		attenuation: attenuation,
		looping:     false,
	}
}

func (sp *SoundPrecache) StartAmbient(id int, sndOrigin vec.Vec3, fvol float32, attenuation float32) {
	if sp.sys == nil {
		return
	}
	sfx, ok := sp.c[id]
	if !ok {
		slog.Error("unknown sound started", slog.Int("sfx", id))
		return
	}
	sp.sys.start <- Start{
		entityNum:   0,
		entityChan:  0,
		sfx:         sfx,
		origin:      sndOrigin,
		volume:      fvol,
		attenuation: attenuation,
		looping:     true,
	}
}

type Sound struct {
	ID   int
	Name string
}

func (sp *SoundPrecache) set(snds ...Sound) {
	for _, s := range snds {
		sfx := sp.sys.precacheSound(s.Name)
		sp.c[s.ID] = sfx
	}
}
