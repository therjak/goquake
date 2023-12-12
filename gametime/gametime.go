// SPDX-License-Identifier: GPL-2.0-or-later

package gametime

import (
	"goquake/cvars"
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

// UpdateTime updates the host time.
// Returns false if it would exceed max fps
func (h *GameTime) UpdateTime(timedemo bool) bool {
	h.time = time.Since(startTime).Seconds()
	maxFPS := math.Clamp(10.0, float64(cvars.HostMaxFps.Value()), 1000.0)
	if !timedemo && (h.time-h.oldTime < 1/maxFPS) {
		return false
	}
	h.frameTime = h.time - h.oldTime
	h.oldTime = h.time

	if cvars.HostTimeScale.Value() > 0 {
		h.frameTime *= float64(cvars.HostTimeScale.Value())
	} else if cvars.HostFrameRate.Value() > 0 {
		h.frameTime = float64(cvars.HostFrameRate.Value())
	} else {
		h.frameTime = math.Clamp(0.001, h.frameTime, 0.1)
	}
	return true
}
