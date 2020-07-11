package quakelib

// void R_RenderView(void);
import "C"

import (
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/math/vec"

	"github.com/chewxy/math32"
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
}

var (
	view qView
)

func (v *qView) UpdateBlend() {
	cl.updateBlend()
}

//export AddLightBlend
func AddLightBlend(r, g, b, a float32) {
	view.addLightBlend(r, g, b, a)
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

	C.R_RenderView()

	v.polyBlend()
}

func (v *qView) polyBlend() {
	if !cvars.GlPolyBlend.Bool() || v.blendColor.A == 0 {
		return
	}

	textureManager.DisableMultiTexture()
	qRecDrawer.Draw(0, 0, float32(screen.Width), float32(screen.Height), v.blendColor)
}
