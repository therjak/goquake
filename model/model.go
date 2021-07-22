// SPDX-License-Identifier: GPL-2.0-or-later

package model

import (
	"goquake/math/vec"
)

const (
	EntityEffectBrightField = 1 << iota
	EntityEffectMuzzleFlash // 2
	EntityEffectBrightLight // 4
	EntityEffectDimLight    // 8
)

const (
	EntityEffectRocket  = 1 << iota
	EntityEffectGrenade // 2
	EntityEffectGib     // 4
	EntityEffectRotate  // 8
	EntityEffectTracer  // 16
	EntityEffectZomGib  // 32
	EntityEffectTracer2 // 64
	EntityEffectTracer3 // 128
)

const (
	MAX_MODELS = 2048
)

type Model interface {
	Name() string
	Mins() vec.Vec3
	Maxs() vec.Vec3
	Flags() int
}
