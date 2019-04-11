package math

import (
	gmath "math"
)

const (
	Pi = gmath.Pi
)

func Atan2(x, y float32) float32 {
	return float32(gmath.Atan2(float64(x), float64(y)))
}

func Sqrt(x float32) float32 {
	return float32(gmath.Sqrt(float64(x)))
}

func Trunc(x float32) float32 {
	return float32(gmath.Trunc(float64(x)))
}
