package snd

import "goquake/math/vec"

type SoundPrecache struct {
	sys *SndSys
	c   []int
}

func (sys *SndSys) NewPrecache() *SoundPrecache {
	return &SoundPrecache{
		sys: sys,
		c:   make([]int, 0),
	}
}

func (sp *SoundPrecache) Start(entnum int, entchannel int, sfx int, sndOrigin vec.Vec3, fvol float32, attenuation float32, looping bool) {
	sp.sys.Start(entnum, entchannel, sp.c[sfx], sndOrigin, fvol, attenuation, looping)
}

func (sp *SoundPrecache) Clear() {
	sp.c = sp.c[:0]
	// TODO: actually clear the precache in sys
}

func (sp *SoundPrecache) Add(s string) {
	sfx := sp.sys.PrecacheSound(s)
	sp.c = append(sp.c, sfx)
}

func (sp *SoundPrecache) Set(snds ...string) {
	sp.Clear()
	for _, s := range snds {
		sp.Add(s)
	}
}
