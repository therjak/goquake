package math

import (
	"github.com/chewxy/math32"
	"math"
)

// AngleMod32 changes an angle to be within 0-360 degrees
func AngleMod32(a float32) float32 {
	return a - math32.Floor(a/360)*360
}

// AngleMod changes an angle to be within 0-360 degrees
func AngleMod(a float64) float64 {
	return a - math.Floor(a/360)*360
}
