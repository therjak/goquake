// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

// void R_SetupView(void);
// void R_SetupScene(void);
// void R_DrawWorld(void);
// void R_DrawWorld_Water(void);
// void R_RenderScene(void);
import "C"

import (
	"github.com/chewxy/math32"
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/glh"
	"github.com/therjak/goquake/math/vec"
)

func CalcRoll(angles, velocity vec.Vec3) float32 {
	_, right, _ := vec.AngleVectors(angles)

	side := vec.Dot(velocity, right)
	neg := math32.Signbit(side)
	side = math32.Abs(side)

	r := cvars.ClientRollAngle.Value()
	rs := cvars.ClientRollSpeed.Value()

	if side < rs {
		side *= r / rs
		if neg {
			return -side
		}
		return side
	}
	if neg {
		return -r
	}
	return r
}

type qView struct {
	blendColor Color
	projection *glh.Matrix
	modelView  *glh.Matrix
}

var (
	view qView
)

func (v *qView) UpdateBlend() {
	cl.updateBlend()
}

func (v *qView) addLightBlend(r, g, b, a2 float32) {
	a := v.blendColor.A + a2*(1-v.blendColor.A)
	v.blendColor.A = a
	a2 /= a
	v.blendColor.R = v.blendColor.R*(1-a2) + r*a2
	v.blendColor.G = v.blendColor.G*(1-a2) + g*a2
	v.blendColor.B = v.blendColor.B*(1-a2) + b*a2
}

// The player's clipping box goes from (-16 -16 -24) to (16 16 32) from
// the entity origin, so any view position inside that will be valid
func (v *qView) Render() {
	if console.forceDuplication {
		return
	}
	if cl.intermission != 0 {
		cl.calcIntermissionRefreshRect()
	} else if !cl.paused {
		cl.calcRefreshRect()
	}

	if !cvars.RNoRefresh.Bool() {
		if cl.worldModel == nil {
			Error("R_RenderView: NULL worldmodel")
		}
		if cvars.GlFinish.Bool() {
			gl.Finish()
		}
		C.R_SetupView()
		v.renderScene()
		if cvars.RPos.Bool() {
			printPosition()
		}
	}

	v.polyBlend()
}

func (v *qView) renderScene() {
	const alphaPass = true

	r_fovx := qRefreshRect.fovX
	r_fovy := qRefreshRect.fovY
	// TODO: water/slime/lava modification from R_SetupView
	/*
				if cvars.WaterWarp.Bool() {
					contents := Mod_PointInLeaf(r_origin, cl.worldModel).contents
					switch contents{
					case CONTENTS_WATER,CONTENTS_SLIME,CONTENTS_LAVE:
		      // variance is a percentage of width, where width = 2 * tan(fov / 2)
		      // otherwise the effect is too dramatic at high FOV and too subtle at low
		      // FOV.  what a mess!
		      r_fovx = atan(tan(DEG2RAD(R_Refdef_fov_x()) / 2) *
		                    (0.97 + sin(CL_Time() * 1.5) * 0.03)) *
		               2 / M_PI_DIV_180;
		      r_fovy = atan(tan(DEG2RAD(R_Refdef_fov_y()) / 2) *
		                    (1.03 - sin(CL_Time() * 1.5) * 0.03)) *
		               2 / M_PI_DIV_180;

					}
	*/
	v.projection = glh.Frustum(r_fovx, r_fovy, cvars.GlFarClip.Value())
	v.projection.Print()

	v.modelView = glh.Identity()
	v.modelView.RotateX(-90)
	v.modelView.RotateZ(90)
	v.modelView.RotateX(-qRefreshRect.viewAngles[2])
	v.modelView.RotateY(-qRefreshRect.viewAngles[0])
	v.modelView.RotateZ(-qRefreshRect.viewAngles[1])
	v.modelView.Translate(-qRefreshRect.viewOrg[0], -qRefreshRect.viewOrg[1], -qRefreshRect.viewOrg[2])

	C.R_SetupScene()
	sky.Draw()
	// TODO: enable fog?
	gl.UseProgram(0) // enable fixed pipeline
	C.R_DrawWorld()
	renderer.DrawShadows()
	renderer.DrawEntitiesOnList(!alphaPass)
	C.R_DrawWorld_Water()
	renderer.DrawEntitiesOnList(alphaPass)
	renderer.RenderDynamicLights()
	particlesDraw()
	// TODO: disable fog?
	renderer.DrawWeaponModel()
}

func (v *qView) polyBlend() {
	if !cvars.GlPolyBlend.Bool() || v.blendColor.A == 0 {
		return
	}

	textureManager.DisableMultiTexture()
	qRecDrawer.Draw(0, 0, float32(screen.Width), float32(screen.Height), v.blendColor)
}
