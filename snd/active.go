// SPDX-License-Identifier: GPL-2.0-or-later

package snd

import (
	"goquake/math/vec"
)

type channel [8]*playingSound

type aSounds struct {
	// entchannel 0-7
	// 0 is ambient
	ambient []*playingSound
	sounds  map[int]channel
	local   *playingSound
}

func newASounds() *aSounds {
	return &aSounds{
		ambient: make([]*playingSound, 0),
		sounds:  make(map[int]channel),
	}
}

func (a *aSounds) add(p *playingSound) {
	if p.entchannel < 0 {
		a.local = p
		return
	}
	if p.entnum == 0 {
		a.ambient = append(a.ambient, p)
		return
	}
	c, ok := a.sounds[p.entnum]
	if !ok {
		c = channel{}
	}
	c[p.entchannel] = p
	a.sounds[p.entnum] = c
}

func (a *aSounds) stop(entnum, entchannel int) {
	c, ok := a.sounds[entnum]
	if !ok {
		return
	}
	ps := c[entchannel]
	if ps != nil {
		ps.paused = true
		ps.sound = nil
	}
	c[entchannel] = nil
	a.sounds[entnum] = c
}

func (a *aSounds) update(listener int, listenerOrigin, listenerRight vec.Vec3) {
	for _, c := range a.sounds {
		for i := range c {
			s := c[i]
			if s != nil {
				s.spatialize(listener, listenerOrigin, listenerRight)
			}
		}
	}
	for _, s := range a.ambient {
		s.spatialize(listener, listenerOrigin, listenerRight)
	}
	// TODO(therjak): start sounds which became audible
	//                stop sounds which are unaudible
	// ambientsounds to ambient_levels
}
