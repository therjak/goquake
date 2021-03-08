// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"

import (
	"github.com/therjak/goquake/math/vec"
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
	fovX        float32
	fovY        float32
	viewForward vec.Vec3 // vpn
	viewRight   vec.Vec3 // vright
	viewUp      vec.Vec3 // vup
}

var (
	qRefreshRect refreshRect
)

//export R_Refdef_fov_x
func R_Refdef_fov_x() float32 {
	return qRefreshRect.fovX
}

//export R_Refdef_fov_y
func R_Refdef_fov_y() float32 {
	return qRefreshRect.fovY
}

//export R_Refdef_vrect_x
func R_Refdef_vrect_x() int {
	return qRefreshRect.viewRect.x
}

//export R_Refdef_vrect_y
func R_Refdef_vrect_y() int {
	return qRefreshRect.viewRect.y
}

//export R_Refdef_vrect_width
func R_Refdef_vrect_width() int {
	return qRefreshRect.viewRect.width
}

//export R_Refdef_vrect_height
func R_Refdef_vrect_height() int {
	return qRefreshRect.viewRect.height
}

//export R_Refdef_vieworg
func R_Refdef_vieworg(i int) float32 {
	return qRefreshRect.viewOrg[i]
}

//export R_Refdef_viewangles
func R_Refdef_viewangles(i int) float32 {
	return qRefreshRect.viewAngles[i]
}

//export R_Refdef_SetViewAngles
func R_Refdef_SetViewAngles(i int, v float32) {
	qRefreshRect.viewAngles[i] = v
}

//export UpdateVpnGo
func UpdateVpnGo() {
	qRefreshRect.viewForward, qRefreshRect.viewRight, qRefreshRect.viewUp = vec.AngleVectors(qRefreshRect.viewAngles)
}
