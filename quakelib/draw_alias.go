// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

//#include <stdio.h>
//#include "q_stdinc.h"
//#include "gl_model.h"
//#include "render.h"
//void R_DrawAliasModel(entity_t* e);
import "C"

import (
	"fmt"
	"math"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/glh"
	qmath "github.com/therjak/goquake/math"
	"github.com/therjak/goquake/math/vec"
	"github.com/therjak/goquake/mdl"
	"github.com/therjak/goquake/texture"
)

func newAliasDrawProgram() (*glh.Program, error) {
	return glh.NewProgram(vertexSourceAliasDrawer, fragmentSourceAliasDrawer)
}

type qAliasDrawer struct {
	vao *glh.VertexArray
	vbo *glh.Buffer // this should probably be inside the model
	// ebo *glh.Buffer
	prog       *glh.Program
	projection int32
	modelview  int32
	blend      int32
	shadeVec   int32
	lightColor int32
	tex        int32
	fullBright int32
	overBright int32
	fogDensity int32
	fogColor   int32
	// 4 pose1vert, 3 pose1normal
	// 4 pose2vert, 3 pose2normal
	// 4 texcoords
}

func newAliasDrawer() *qAliasDrawer {
	d := &qAliasDrawer{}
	d.vao = glh.NewVertexArray()
	d.vbo = glh.NewBuffer()
	var err error
	d.prog, err = newAliasDrawProgram()
	if err != nil {
		Error(err.Error())
	}
	d.projection = d.prog.GetUniformLocation("projection")
	d.modelview = d.prog.GetUniformLocation("modelview")
	d.blend = d.prog.GetUniformLocation("Blend")
	d.shadeVec = d.prog.GetUniformLocation("ShadeVector")
	d.lightColor = d.prog.GetUniformLocation("LightColor")
	d.tex = d.prog.GetUniformLocation("Tex")
	d.fullBright = d.prog.GetUniformLocation("FullbrightTex")
	d.overBright = d.prog.GetUniformLocation("UseFullbrightTex")
	d.fogDensity = d.prog.GetUniformLocation("FogDensity")
	d.fogColor = d.prog.GetUniformLocation("FogColor")

	return d
}

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
			l.blend = qmath.Clamp(0, (cl.time-e.LerpStart)/(e.LerpFinish-e.LerpStart), 1)
		} else {
			l.blend = qmath.Clamp(0, (cl.time-e.LerpStart)/e.LerpTime, 1)
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
		blend = qmath.Clamp(0, blend, 1)

		d := vec.Sub(e.CurrentOrigin, e.PreviousOrigin)
		l.origin = vec.FMA(e.PreviousOrigin, float32(blend), d)

		d = vec.Sub(e.CurrentAngles, e.PreviousAngles)
		d[0] = qmath.AngleMod32(d[0]) - 180
		d[1] = qmath.AngleMod32(d[1]) - 180
		d[2] = qmath.AngleMod32(d[2]) - 180
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
	alpha := entAlphaDecode(e.Alpha)
	if alpha == 0 {
		return
	}
	if alpha < 1 {
		gl.DepthMask(false)
		gl.Enable(gl.BLEND)
		defer gl.DepthMask(true)
		defer gl.Disable(gl.BLEND)
	}

	modelview := view.modelView.Copy()
	modelview.Translate(ld.origin[0], ld.origin[1], ld.origin[2])
	modelview.RotateZ(ld.angles[1])
	modelview.RotateY(-ld.angles[0])
	modelview.RotateX(ld.angles[2])
	modelview.Translate(model.Translate[0], model.Translate[1], model.Translate[2])
	modelview.Scale(model.Scale[0], model.Scale[1], model.Scale[2])
	// view.projection

	textureManager.DisableMultiTexture()
	var tx, fb *texture.Texture
	if e.SkinNum < model.SkinCount && e.SkinNum >= 0 {
		anim := int(cl.time * 10)
		t := model.Textures[e.SkinNum]
		fbt := model.FBTextures[e.SkinNum]
		tx = t[anim%len(t)]
		if len(fbt) > 0 {
			fb = fbt[anim%len(fbt)]
		}
	}
	if !cvars.GlNoColors.Bool() {
		// TODO: colored player textures
		// if pt := PlayerTexture(e); pt != nil {
		//   t = pt
		// }
	}
	if !cvars.GlFullBrights.Bool() {
		fb = nil
	}

	drawAliasFrame(model, ld, tx, fb, e, modelview, view.projection)
}

type qUniform interface {
	SetAsUniform(id int32)
}

func calcShadeVector(e *Entity) vec.Vec3 {
	const shadeDotQuant = 16
	quantizedAngle := float64(int(e.Angles[1]*(shadeDotQuant/360.0)) & (shadeDotQuant - 1))
	radiansAngle := (quantizedAngle / 16.0) * 2.0 * math.Pi
	s, c := math.Sincos(-radiansAngle)
	r := vec.Vec3{float32(c), float32(s), 1}
	r.Normalize()
	return r
}

func drawAliasFrame(m *mdl.Model, ld *lerpData, tx, fb *texture.Texture, e *Entity, mv, p qUniform) {
	lightColor := cl.ColorForEntity(e)
	shadeVec := calcShadeVector(e)
	// Now we should have everything needed to call GL_DrawAliasFrame_GLSL
	fmt.Printf("LightColor: %v, %v\n", lightColor, shadeVec)
	// R_SetupAliasLighting(e)
	// GL_DrawAliasFrame_GLSL
	C.R_DrawAliasModel(e.ptr)
}

var aliasDrawer *qAliasDrawer

func CreateAliasDrawer() {
	aliasDrawer = newAliasDrawer()
}

//export PrintMV
func PrintMV() {
	m := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	gl.GetFloatv(0x0BA6, &m[0])
	fmt.Printf("ModelView:\n%v %v %v %v\n%v %v %v %v\n%v %v %v %v\n%v %v %v %v\n",
		m[0], m[4], m[8], m[12],
		m[1], m[5], m[9], m[13],
		m[2], m[6], m[10], m[14],
		m[3], m[7], m[11], m[15],
	)
}

//export PrintP
func PrintP() {
	m := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	gl.GetFloatv(0x0BA7, &m[0])
	fmt.Printf("Projection:\n%v %v %v %v\n%v %v %v %v\n%v %v %v %v\n%v %v %v %v\n",
		m[0], m[4], m[8], m[12],
		m[1], m[5], m[9], m[13],
		m[2], m[6], m[10], m[14],
		m[3], m[7], m[11], m[15],
	)
}
