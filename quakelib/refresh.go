// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"goquake/math/vec"
)

const (
	PITCH = iota
	YAW
	ROLL
)

type refreshRect struct {
	viewRect    Rect
	viewOrg     vec.Vec3 //r_origin
	viewAngles  vec.Vec3
	fovX        float64
	fovY        float64
	viewForward vec.Vec3 // vpn
	viewRight   vec.Vec3 // vright
	viewUp      vec.Vec3 // vup
}

var (
	qRefreshRect refreshRect
)
