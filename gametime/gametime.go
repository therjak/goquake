// SPDX-License-Identifier: GPL-2.0-or-later

package gametime

import (
	"goquake/math"
	"time"
)

var (
	startTime = time.Now()
)

type GameTime struct {
	time       float64
	oldTime    float64
	frameTime  float64
	frameCount int
}

func (h *GameTime) Reset() {
	h.frameTime = 0.1
}

func (h *GameTime) Time() float64      { return h.time }
func (h *GameTime) OldTime() float64   { return h.oldTime }
func (h *GameTime) FrameTime() float64 { return h.frameTime }
func (h *GameTime) FrameCount() int    { return h.frameCount }
func (h *GameTime) FrameIncrease()     { h.frameCount++ }

type Update struct {
	TimeDemo  bool
	TimeScale float64
	FrameRate float64
	MaxFPS    float64
}

// UpdateTime updates the host time.
// Returns false if it would exceed max fps
func (h *GameTime) UpdateTime(u Update) bool {
	h.time = time.Since(startTime).Seconds()
	maxFPS := math.Clamp(10.0, u.MaxFPS, 1000.0)
	if !u.TimeDemo && (h.time-h.oldTime < 1/maxFPS) {
		return false
	}
	h.frameTime = h.time - h.oldTime
	h.oldTime = h.time

	if u.TimeScale > 0 {
		h.frameTime *= u.TimeScale
	} else if u.FrameRate > 0 {
		h.frameTime = u.FrameRate
	} else {
		h.frameTime = math.Clamp(0.001, h.frameTime, 0.1)
	}
	return true
}
