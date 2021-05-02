// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

//#include <stdio.h>
//#include "q_stdinc.h"
//#include "gl_model.h"
//#include "render.h"
//void R_DrawAliasModel(entity_t* e);
//void GL_DrawAliasShadow(entity_t* e);
import "C"

import (
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/math"
	"github.com/therjak/goquake/math/vec"
	"github.com/therjak/goquake/mdl"
)

type lerpData struct {
	pose1  int // lerp between pose1 and pose2
	pose2  int
	blend  float64
	origin vec.Vec3
	angles vec.Vec3
}

func (l *lerpData) setupAliasFrame(e *Entity, m *mdl.Model) {
	frame := e.Frame
	if frame >= len(m.Frames) || frame < 0 {
		frame = 0
	}
	poseNum := 0 // m.Frames[frame].FirstPose // we count within a framegroup and not over all framegroups
	// numPoses := m.Frames[frame].NumPoses
	f := &m.Frames[frame]
	fg := &f.Group
	e.LerpTime = float64(f.Interval)

	numPoses := len(*fg)
	if numPoses > 1 {
		poseNum += int((cl.time / e.LerpTime)) % numPoses
	}
	if e.LerpFlags&lerpResetAnim != 0 {
		e.LerpStart = 0
		e.PreviousPose = poseNum
		e.CurrentPose = poseNum
		e.LerpFlags &^= lerpResetAnim
	} else if e.CurrentPose != poseNum {
		if e.LerpFlags&lerpResetAnim2 != 0 {
			e.LerpStart = 0
			e.PreviousPose = poseNum
			e.CurrentPose = poseNum
			e.LerpFlags &^= lerpResetAnim2
		} else {
			e.LerpStart = cl.time
			e.PreviousPose = e.CurrentPose
			e.CurrentPose = poseNum
		}
	}
	if cvars.RLerpModels.Bool() && (cvars.RLerpModels.Value() == 2 || m.Flags() != mdl.NoLerp) {
		if e.LerpFlags&lerpFinish != 0 && numPoses == 1 {
			l.blend = math.Clamp(0, (cl.time-e.LerpStart)/(e.LerpFinish-e.LerpStart), 1)
		} else {
			l.blend = math.Clamp(0, (cl.time-e.LerpStart)/e.LerpTime, 1)
		}
		l.pose1 = e.PreviousPose
		l.pose2 = e.CurrentPose
	} else {
		l.blend = 1
		l.pose1 = poseNum
		l.pose2 = poseNum
	}
}

func (l *lerpData) setupEntityTransform(e *Entity) {
	if e.LerpFlags&lerpResetMove != 0 {
		e.MoveLerpStart = 0
		e.PreviousOrigin = e.Origin
		e.CurrentOrigin = e.Origin
		e.PreviousAngles = e.Angles
		e.CurrentAngles = e.Angles
		e.LerpFlags &^= lerpResetMove
	} else if e.Origin != e.CurrentOrigin && e.Angles != e.CurrentAngles {
		e.MoveLerpStart = cl.time
		e.PreviousOrigin = e.CurrentOrigin
		e.CurrentOrigin = e.Origin
		e.PreviousAngles = e.CurrentAngles
		e.CurrentAngles = e.Angles
	}

	if cvars.RLerpMove.Bool() && e.LerpFlags&lerpMoveStep != 0 {
		blend := cl.time - e.MoveLerpStart
		if e.LerpFlags&lerpFinish != 0 {
			blend /= e.LerpFinish - e.MoveLerpStart
		} else {
			blend /= 0.1
		}
		blend = math.Clamp(0, blend, 1)

		d := vec.Sub(e.CurrentOrigin, e.PreviousOrigin)
		l.origin = vec.FMA(e.PreviousOrigin, float32(blend), d)

		d = vec.Sub(e.CurrentAngles, e.PreviousAngles)
		d[0] = math.AngleMod32(d[0]) - 180
		d[1] = math.AngleMod32(d[1]) - 180
		d[2] = math.AngleMod32(d[2]) - 180
		l.angles = vec.FMA(e.PreviousAngles, float32(blend), d)
	} else {
		l.origin = e.Origin
		l.angles = e.Angles
	}
}

func (r *qRenderer) cullAlias(e *Entity, model *mdl.Model) bool {
	if e.Angles[0] != 0 || e.Angles[2] != 0 {
		return r.CullBox(
			vec.Sub(e.Origin, vec.Vec3{model.Radius, model.Radius, model.Radius}),
			vec.Add(e.Origin, vec.Vec3{model.Radius, model.Radius, model.Radius}))
	}
	if e.Angles[1] != 0 {
		return r.CullBox(
			vec.Sub(e.Origin, vec.Vec3{model.Radius, model.Radius, model.Mins()[2]}),
			vec.Add(e.Origin, vec.Vec3{model.Radius, model.Radius, model.Maxs()[2]}))
	}
	return r.CullBox(
		vec.Sub(e.Origin, model.Mins()),
		vec.Add(e.Origin, model.Maxs()))
}

func (r *qRenderer) DrawAliasModel(e *Entity, model *mdl.Model) {
	ld := &lerpData{}
	ld.setupAliasFrame(e, model)
	ld.setupEntityTransform(e)
	if r.cullAlias(e, model) {
		return
	}
	C.R_DrawAliasModel(e.ptr)
}

func (r *qRenderer) DrawAliasShadow(e *Entity, model *mdl.Model) {
	C.GL_DrawAliasShadow(e.ptr)
}
