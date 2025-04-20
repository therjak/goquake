// SPDX-License-Identifier: GPL-2.0-or-later

package snd

import (
	"goquake/math"
	"goquake/math/vec"

	"github.com/gopxl/beep/v2"
)

type playingSound struct {
	entchannel int // entchannel
	// TODO:
	// entchannel. 0 willingly overrides, 1-7 always overrides
	// 0 auto
	// 1 weapon
	// 2 voice
	// 3 item
	// 4 body
	// 8 no phys add
	entnum             int // entnum
	distanceMultiplier float32
	masterVolume       float64
	origin             vec.Vec3
	done               bool // if done it must no longer be updated
	right              float64
	left               float64
	sound              beep.Streamer
	paused             bool
}

func (s *playingSound) spatialize(listener int, listenerPos, listenerRight vec.Vec3) {
	if listener == s.entnum || s.entnum == -1 {
		s.right = 1
		s.left = 1
	} else {
		v := vec.Sub(s.origin, listenerPos)
		dist := v.Length() * s.distanceMultiplier
		v = v.Normalize()
		dot := vec.Dot(listenerRight, v)
		dist = 1.0 - dist
		lscale := (1.0 - dot) * dist
		rscale := (1.0 + dot) * dist
		s.left = math.Clamp(0, float64(lscale), 1)
		s.right = math.Clamp(0, float64(rscale), 1)
	}
}

func (s *playingSound) Stream(samples [][2]float64) (int, bool) {
	if s.sound == nil {
		return 0, false
	}
	if s.paused {
		clear(samples)
		return len(samples), true
	}
	n, ok := s.sound.Stream(samples)
	for i := range samples[:n] {
		samples[i][0] *= s.left * s.masterVolume
		samples[i][1] *= s.right * s.masterVolume
	}
	return n, ok
}

func (s *playingSound) Err() error {
	if s.sound == nil {
		return nil
	}
	return s.sound.Err()
}
