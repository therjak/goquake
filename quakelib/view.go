package quakelib

// void R_RenderView(void);
// extern float v_blend[4];
import "C"

import (
	"quake/cvars"
	"quake/math/vec"

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

type qView struct{}

var (
	view qView
)

func (v *qView) UpdateBlend() {
	cl.updateBlend()
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

	V_PolyBlend(&C.v_blend[0])
}
