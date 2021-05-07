// SPDX-License-Identifier: GPL-2.0-or-later

package glh

import (
	"math"

	"github.com/go-gl/gl/v4.6-core/gl"
)

type Matrix struct {
	m [16]float32
}

func deg2rad(deg float32) float64 {
	return (float64(deg) / 180) * math.Pi
}

func sincos(t float64) (float32, float32) {
	s, c := math.Sincos(t)
	return float32(s), float32(c)
}

func Identity() *Matrix {
	return &Matrix{
		m: [16]float32{
			1, 0, 0, 0, // 0 - 3
			0, 1, 0, 0, // 4 - 7
			0, 0, 1, 0, // 8 - 11
			0, 0, 0, 1, // 12 - 15
		},
	}
}

func (m *Matrix) SetAsUniform(id int32) {
	gl.UniformMatrix4fv(id, 1, false, &m.m[0])
}

func (m *Matrix) Translate(x, y, z float32) {
	// 1, 0, 0, x
	// 0, 1, 0, y
	// 0, 0, 1, z
	// 0, 0, 0, 1 * m.m
	n := [16]float32{
		m.m[0] + x*m.m[12], m.m[1] + x*m.m[13], m.m[2] + x*m.m[14], m.m[3] + x*m.m[15],
		m.m[4] + y*m.m[12], m.m[5] + y*m.m[13], m.m[6] + y*m.m[14], m.m[7] + y*m.m[15],
		m.m[8] + z*m.m[12], m.m[9] + z*m.m[13], m.m[10] + y*m.m[14], m.m[11] + z*m.m[15],
		m.m[12], m.m[13], m.m[14], m.m[15],
	}
	m.m = n
}

func (m *Matrix) RotateX(degree float32) {
	sin, cos := sincos(deg2rad(degree))
	// 1, 0, 0, 0
	// 0, cos, -sin, 0
	// 0, sin, cos 0
	// 0, 0, 0, 1 * m.m
	n := [16]float32{
		m.m[0], m.m[1], m.m[2], m.m[3],
		cos*m.m[4] - sin*m.m[8], cos*m.m[5] - sin*m.m[9], cos*m.m[6] - sin*m.m[10], cos*m.m[7] - sin*m.m[11],
		sin*m.m[4] + cos*m.m[8], sin*m.m[5] + cos*m.m[9], sin*m.m[6] + cos*m.m[10], sin*m.m[7] + cos*m.m[11],
		m.m[12], m.m[13], m.m[14], m.m[15],
	}
	m.m = n
}

func (m *Matrix) RotateY(degree float32) {
	sin, cos := sincos(deg2rad(degree))
	// cos, 0, sin, 0
	// 0, 1, 0, 0
	// -sin, 0, cos, 0
	// 0, 0, 0, 1 * m.m
	n := [16]float32{
		cos*m.m[0] + sin*m.m[8], cos*m.m[1] + sin*m.m[9], cos*m.m[2] + sin*m.m[10], cos*m.m[3] + sin*m.m[11],
		m.m[4], m.m[5], m.m[6], m.m[7],
		-sin*m.m[0] + cos*m.m[8], -sin*m.m[1] + cos*m.m[9], -sin*m.m[2] + cos*m.m[10], -sin*m.m[3] + cos*m.m[11],
		m.m[12], m.m[13], m.m[14], m.m[15],
	}
	m.m = n
}

func (m *Matrix) RotateZ(degree float32) {
	sin, cos := sincos(deg2rad(degree))
	// cos, -sin, 0, 0
	// sin, cos, 0, 0
	// 0, 0, 1, 0
	// 0, 0, 0, 1 * m.m
	n := [16]float32{
		cos*m.m[0] - sin*m.m[4], cos*m.m[1] - sin*m.m[5], cos*m.m[2] - sin*m.m[6], cos*m.m[3] - sin*m.m[7],
		sin*m.m[0] + cos*m.m[4], sin*m.m[1] + cos*m.m[5], sin*m.m[2] + cos*m.m[6], sin*m.m[3] + cos*m.m[7],
		m.m[8], m.m[9], m.m[10], m.m[11],
		m.m[12], m.m[13], m.m[14], m.m[15],
	}
	m.m = n

}

func (m *Matrix) Scale(x, y, z float32) {
	// x, 0, 0, 0
	// 0, y, 0, 0
	// 0, 0, z, 0
	// 0, 0, 0, 1 * m.m
	n := [16]float32{
		x * m.m[0], x * m.m[1], x * m.m[2], x * m.m[3],
		y * m.m[4], y * m.m[5], y * m.m[6], y * m.m[7],
		z * m.m[8], z * m.m[9], z * m.m[10], z * m.m[11],
		m.m[12], m.m[13], m.m[14], m.m[15],
	}
	m.m = n
}
