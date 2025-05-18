// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"math"

	"goquake/bsp"
	"goquake/cvars"
	"goquake/glh"
	qmath "goquake/math"
	"goquake/math/vec"
	"goquake/mdl"
	"goquake/texture"

	"github.com/go-gl/gl/v4.6-core/gl"
)

func newAliasDrawProgram() (*glh.Program, error) {
	return glh.NewProgram(vertexSourceAliasDrawer, fragmentSourceAliasDrawer)
}

type qAliasDrawer struct {
	// vbo and ebo are stored in mdl.Model
	vao           *glh.VertexArray
	prog          *glh.Program
	projection    int32
	modelview     int32
	blend         int32
	shadeVec      int32
	lightColor    int32
	tex           int32
	fullBrightTex int32
	useOverBright int32
	useFullBright int32
	fogDensity    int32
	fogColor      int32
}

func newAliasDrawer() (*qAliasDrawer, error) {
	d := &qAliasDrawer{}
	d.vao = glh.NewVertexArray()
	var err error
	d.prog, err = newAliasDrawProgram()
	if err != nil {
		return nil, err
	}
	d.projection = d.prog.GetUniformLocation("projection")
	d.modelview = d.prog.GetUniformLocation("modelview")
	d.blend = d.prog.GetUniformLocation("Blend")
	d.shadeVec = d.prog.GetUniformLocation("ShadeVector")
	d.lightColor = d.prog.GetUniformLocation("LightColor")
	d.tex = d.prog.GetUniformLocation("Tex")
	d.fullBrightTex = d.prog.GetUniformLocation("FullbrightTex")
	d.useFullBright = d.prog.GetUniformLocation("UseFullbrightTex")
	d.useOverBright = d.prog.GetUniformLocation("UseOverbright")
	d.fogDensity = d.prog.GetUniformLocation("FogDensity")
	d.fogColor = d.prog.GetUniformLocation("FogColor")

	return d, nil
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
	poseNum := 0
	for i := 0; i < frame; i++ {
		f := &m.Frames[i]
		poseNum += len(f.Group)
	}
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
	if cvars.RLerpModels.Bool() && !(cvars.RLerpModels.Value() != 2 && m.Flags()&mdl.NoLerp != 0) {
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
	} else if e.Origin != e.CurrentOrigin || e.Angles != e.CurrentAngles {
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
		am := func(a float32) float32 {
			if a > 180 {
				return a - 360
			}
			if a < -180 {
				return a + 360
			}
			return a
		}
		d[0] = am(d[0])
		d[1] = am(d[1])
		d[2] = am(d[2])
		l.angles = vec.FMA(e.PreviousAngles, float32(blend), d)
	} else {
		l.origin = e.Origin
		l.angles = e.Angles
	}
}

func (r *qRenderer) cullBrush(e *Entity, model *bsp.Model) bool {
	if e.Angles[0] != 0 || e.Angles[1] != 0 || e.Angles[2] != 0 {
		return r.CullBox(
			vec.Add(e.Origin, vec.Vec3{-model.Radius, -model.Radius, -model.Radius}),
			vec.Add(e.Origin, vec.Vec3{model.Radius, model.Radius, model.Radius}))
	}
	return r.CullBox(
		vec.Add(e.Origin, model.Mins()),
		vec.Add(e.Origin, model.Maxs()))
}

func (r *qRenderer) cullAlias(e *Entity, model *mdl.Model) bool {
	if e.Angles[0] != 0 || e.Angles[2] != 0 {
		return r.CullBox(
			vec.Add(e.Origin, vec.Vec3{-model.Radius, -model.Radius, -model.Radius}),
			vec.Add(e.Origin, vec.Vec3{model.Radius, model.Radius, model.Radius}))
	}
	if e.Angles[1] != 0 {
		return r.CullBox(
			vec.Add(e.Origin, vec.Vec3{-model.YawRadius, -model.YawRadius, model.Mins()[2]}),
			vec.Add(e.Origin, vec.Vec3{model.YawRadius, model.YawRadius, model.Maxs()[2]}))
	}
	return r.CullBox(
		vec.Add(e.Origin, model.Mins()),
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

	skin := e.SkinNum
	if e.SkinNum >= model.SkinCount && e.SkinNum < 0 {
		skin = 0
	}
	var fb *texture.Texture
	anim := int(cl.time * 10)
	t := model.Textures[skin]
	fbt := model.FBTextures[skin]
	tx := t[anim%len(t)]
	if len(fbt) > 0 && cvars.GlFullBrights.Bool() {
		fb = fbt[anim%len(fbt)]
	}

	if !cvars.GlNoColors.Bool() {
		if pt := playerTextures[e]; pt != nil {
			tx = pt
		}
	}

	drawAliasFrame(model, ld, tx, fb, e, alpha, modelview, view.projection)
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

func drawAliasFrame(m *mdl.Model, ld *lerpData, tx, fb *texture.Texture, e *Entity, alpha float32, mv, p qUniform) {
	lightColor := cl.ColorForEntity(e)
	shadeVec := calcShadeVector(e)

	var blend float32
	if ld.pose1 != ld.pose2 {
		blend = float32(ld.blend)
	}
	aliasDrawer.prog.Use()
	m.VertexArrayBuffer.Bind()
	m.VertexElementArrayBuffer.Bind()

	gl.EnableVertexAttribArray(0) // pose1vert
	defer gl.DisableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1) // pose1normal
	defer gl.DisableVertexAttribArray(1)
	gl.EnableVertexAttribArray(2) // pose2vert
	defer gl.DisableVertexAttribArray(2)
	gl.EnableVertexAttribArray(3) // pose2normal
	defer gl.DisableVertexAttribArray(3)
	gl.EnableVertexAttribArray(4) // texcoords
	defer gl.DisableVertexAttribArray(4)

	// layout:
	// 4*uint8 + 4*int8, for each pose and vertex
	// 2*float32 for each vertex

	p1 := uintptr(ld.pose1 * m.VerticeCount * 8)
	p2 := uintptr(ld.pose2 * m.VerticeCount * 8)
	gl.VertexAttribPointerWithOffset(0, 4, gl.UNSIGNED_BYTE, false, 8, p1)
	gl.VertexAttribPointerWithOffset(1, 4, gl.BYTE, true, 8, p1+4)
	gl.VertexAttribPointerWithOffset(2, 4, gl.UNSIGNED_BYTE, false, 8, p2)
	gl.VertexAttribPointerWithOffset(3, 4, gl.BYTE, true, 8, p2+4)
	gl.VertexAttribPointerWithOffset(4, 2, gl.FLOAT, false, 0, uintptr(m.STOffset))

	gl.Uniform1f(aliasDrawer.blend, blend)
	gl.Uniform3f(aliasDrawer.shadeVec, shadeVec[0], shadeVec[1], shadeVec[2])
	gl.Uniform4f(aliasDrawer.lightColor, lightColor[0], lightColor[1], lightColor[2], alpha)
	gl.Uniform1i(aliasDrawer.tex, 0)
	gl.Uniform1i(aliasDrawer.fullBrightTex, 1)
	var useFullBright int32
	if fb != nil {
		useFullBright = 1
	}
	gl.Uniform1i(aliasDrawer.useFullBright, useFullBright)
	var useOverBright int32
	if cvars.GlOverBrightModels.Bool() {
		useOverBright = 1
	}
	gl.Uniform1i(aliasDrawer.useOverBright, useOverBright)
	gl.Uniform1f(aliasDrawer.fogDensity, fog.Density)
	gl.Uniform4f(aliasDrawer.fogColor, fog.Color.R, fog.Color.G, fog.Color.B, 0)
	p.SetAsUniform(aliasDrawer.projection)
	mv.SetAsUniform(aliasDrawer.modelview)

	textureManager.BindUnit(tx, gl.TEXTURE0)
	textureManager.BindUnit(fb, gl.TEXTURE1)

	gl.DrawElements(gl.TRIANGLES, int32(m.IndiceCount), gl.UNSIGNED_SHORT, gl.PtrOffset(0))
}

var aliasDrawer *qAliasDrawer

func CreateAliasDrawer() error {
	var err error
	aliasDrawer, err = newAliasDrawer()
	return err
}
