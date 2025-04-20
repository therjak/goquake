package snd

import "goquake/math/vec"

type SoundPrecache struct {
	sys *SndSys
	c   []int
}

func (sys *SndSys) NewPrecache(snds ...Sound) *SoundPrecache {
	s := &SoundPrecache{
		sys: sys,
		c:   make([]int, 0),
	}
	s.set(snds...)
	return s
}

func (sp *SoundPrecache) Start(entnum int, entchannel int, sfx int, sndOrigin vec.Vec3, fvol float32, attenuation float32, looping bool) {
	if sp.sys == nil {
		return
	}
	sp.sys.start <- Start{
		entityNum:   entnum,
		entityChan:  entchannel,
		sfx:         sp.c[sfx],
		origin:      sndOrigin,
		volume:      fvol,
		attenuation: attenuation,
		looping:     looping,
	}
}

func (sp *SoundPrecache) clear() {
	sp.c = sp.c[:0]
	// TODO: actually clear the precache in sys
}

type Sound struct {
	ID   int
	Name string
}

func (sp *SoundPrecache) add(s Sound) {
	if sp.sys == nil {
		return
	}
	sfx := sp.sys.precacheSound(s.Name)
	sp.c = append(sp.c, sfx)
}

func (sp *SoundPrecache) set(snds ...Sound) {
	sp.clear()
	for _, s := range snds {
		sp.add(s)
	}
}
