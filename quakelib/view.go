package quakelib

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
