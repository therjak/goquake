// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

// void R_SetupView(void);
// void R_SetupScene(void);
// void R_RenderScene(void);
import "C"

import (
	"goquake/bsp"
	"goquake/cvars"
	"goquake/glh"
	"goquake/math/vec"
	"log"
	"math"

	"github.com/chewxy/math32"
	"github.com/go-gl/gl/v4.6-core/gl"
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
		v.setup()
		v.renderScene()
		if cvars.RPos.Bool() {
			printPosition()
		}
	}

	v.polyBlend()
}

func (v *qView) setup() {
	C.R_SetupView()

	qRefreshRect.viewForward, qRefreshRect.viewRight, qRefreshRect.viewUp = vec.AngleVectors(qRefreshRect.viewAngles)

	viewLeaf.Update(cl.worldModel, qRefreshRect.viewOrg)

	cl.setContentsColor(viewLeaf.current.Contents())
	v.blendColor = cl.calcBlend()

	r_fovx := qRefreshRect.fovX
	r_fovy := qRefreshRect.fovY
	if cvars.RWaterWarp.Bool() {
		l, err := cl.worldModel.PointInLeaf(qRefreshRect.viewOrg)
		if err != nil {
			log.Printf("renderScene, PointInLeaf: %v", err)
		}
		switch l.Contents() {
		case bsp.CONTENTS_WATER, bsp.CONTENTS_SLIME, bsp.CONTENTS_LAVA:
			// variance is a percentage of width, where width = 2 * tan(fov / 2)
			// otherwise the effect is too dramatic at high FOV and too subtle at low
			// FOV.  what a mess!
			t := math.Sin(cl.time*1.5) * 0.03
			x := r_fovx * piDiv360
			y := r_fovy * piDiv360
			r_fovx = math.Atan(math.Tan(x)*(0.97+t)) * piDiv360Inv
			r_fovy = math.Atan(math.Tan(y)*(1.03-t)) * piDiv360Inv
		}
	}

	v.projection = glh.Frustum(r_fovx, r_fovy, cvars.GlFarClip.Value())
	renderer.SetFrustum(float32(r_fovx), float32(r_fovy))

	v.modelView = glh.Identity()
	v.modelView.RotateX(-90)
	v.modelView.RotateZ(90)
	v.modelView.RotateX(-qRefreshRect.viewAngles[2])
	v.modelView.RotateY(-qRefreshRect.viewAngles[0])
	v.modelView.RotateZ(-qRefreshRect.viewAngles[1])
	v.modelView.Translate(-qRefreshRect.viewOrg[0], -qRefreshRect.viewOrg[1], -qRefreshRect.viewOrg[2])

	MarkSurfaces()
	renderer.cullSurfaces(cl.worldModel)

	qCanvas.Set(CANVAS_DEFAULT)
	statusbar.MarkChanged()
	screen.ResetTileClearUpdates()
	clearGl()
}

func clearGl() {
	var cb uint32 = gl.DEPTH_BUFFER_BIT
	/*
		  gl_stencilbits := SDL_GL_GetAttribute(SDL_GL_STENCIL_SIZE)
			if gl_stencilbits {
				cb |= gl.STENCIL_BUFFER_BIT
			}
	*/
	if cvars.GlClear.Bool() {
		cb |= gl.COLOR_BUFFER_BIT
	}
	gl.Clear(cb)
}

func setupGl() {
	if cvars.GlCull.Bool() {
		gl.Enable(gl.CULL_FACE)
	} else {
		gl.Disable(gl.CULL_FACE)
	}
	gl.Disable(gl.BLEND)
	gl.Enable(gl.DEPTH_TEST)
}

const (
	piDiv360    = math.Pi / 360
	piDiv360Inv = 360 / math.Pi
)

func (v *qView) renderScene() {
	const alphaPass = true

	// setup scene
	if !cvars.GlFlashBlend.Bool() {
		markLights(cl.worldModel.Node)
	}
	R_AnimateLight()
	renderer.frameCount++

	gl.Viewport(
		int32(qRefreshRect.viewRect.x),
		int32(screen.Height-qRefreshRect.viewRect.y-qRefreshRect.viewRect.height),
		int32(qRefreshRect.viewRect.width),
		int32(qRefreshRect.viewRect.height))

	setupGl()

	sky.Draw()
	// TODO: enable fog?
	renderer.DrawWorld(cl.worldModel, v.modelView)
	renderer.DrawEntitiesOnList(!alphaPass)
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
