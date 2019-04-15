package math

import (
	gmath "math"
)

func Lerp(a, b, frac float32) float32 {
	return (1-frac)*a + frac*b
}

func RoundToEven(x float32) float32 {
	return float32(gmath.RoundToEven(float64(x)))
}
