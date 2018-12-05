package math

import "math"

// AngleMod32 changes an angle to be within 0-360 degrees
func AngleMod32(a float32) float32 {
	return float32(AngleMod(float64(a)))
}

// AngleMod changes an angle to be within 0-360 degrees
func AngleMod(a float64) float64 {
	return a - math.Floor(a/360)*360
}
