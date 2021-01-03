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

func Round(x float32) float32 {
	return float32(gmath.Round(float64(x)))
}

// RSqrt return aprox 1/sqrt(x) for 0<=x
func RSqrt(x float32) float32 {
	x2 := x * 0.5
	i := gmath.Float32bits(x)
	i = 0x5f375a86 - (i >> 1)
	y := gmath.Float32frombits(i)
	y *= 1.5 - (x2 * y * y)
	return y
}

// Sqrt return aprox sqrt(x) for 0<=x
func Sqrt(x float32) float32 {
	i := gmath.Float32bits(x)
	i = (i >> 1) + (0x3f800000 >> 1)
	y := gmath.Float32frombits(i)
	return (y*y + x) / (2 * y)
}
