package vec

import (
	"github.com/chewxy/math32"
)

type Vec3 [3]float32

func VFromA(a [3]float32) Vec3 {
	return Vec3{a[0], a[1], a[2]}
}

// Length returns the length of the vector
func (v Vec3) Length() float32 {
	return math32.Sqrt(Dot(v, v))
}

// Add returns a + b
func Add(a, b Vec3) Vec3 {
	return Vec3{
		a[0] + b[0],
		a[1] + b[1],
		a[2] + b[2],
	}
}

// Sub returns a - b
func Sub(a, b Vec3) Vec3 {
	return Vec3{
		a[0] - b[0],
		a[1] - b[1],
		a[2] - b[2],
	}
}

// Scale returns the vector multiplied by the skalar s
func (v Vec3) Scale(s float32) Vec3 {
	return Vec3{
		v[0] * s,
		v[1] * s,
		v[2] * s,
	}
}

// Normalize returns the normalized vector
func (v Vec3) Normalize() Vec3 {
	l := v.Length()
	if l == 0 {
		return Vec3{}
	}
	return v.Scale(1 / l)
}

// Dot returns a dot b
func Dot(a Vec3, b Vec3) float32 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}

// DoublePrecDot return a dot b calculated in double precision
func DoublePrecDot(a Vec3, b Vec3) float32 {
	p := func(x, y float32) float64 {
		return float64(x) * float64(y)
	}
	return float32(p(a[0], b[0]) + p(a[1], b[1]) + p(a[2], b[2]))
}

// Cross returns a cross b
func Cross(a, b Vec3) Vec3 {
	return Vec3{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}

// Lerp computes a weighted average between two points
func Lerp(a, b Vec3, frac float32) Vec3 {
	fi := 1 - frac
	return Vec3{
		fi*a[0] + frac*b[0],
		fi*a[1] + frac*b[1],
		fi*a[2] + frac*b[2],
	}
}

func minmax(a, b float32) (float32, float32) {
	if a < b {
		return a, b
	}
	return b, a
}

func MinMax(a, b Vec3) (Vec3, Vec3) {
	var r, s Vec3
	r[0], s[0] = minmax(a[0], b[0])
	r[1], s[1] = minmax(a[1], b[1])
	r[2], s[2] = minmax(a[2], b[2])
	return r, s
}

func AngleVectors(angles Vec3) (forward, right, up Vec3) {
	deg := math32.Pi * 2 / 360
	sp, cp := math32.Sincos(angles[0] * deg) // PITCH
	sy, cy := math32.Sincos(angles[1] * deg) // YAW
	sr, cr := math32.Sincos(angles[2] * deg) // ROLL

	forward = Vec3{cp * cy, cp * sy, -sp}
	right = Vec3{
		(-1*sr*sp*cy + -1*cr*-sy),
		(-1*sr*sp*sy + -1*cr*cy),
		-1 * sr * cp,
	}
	up = Vec3{
		(cr*sp*cy + -sr*-sy),
		(cr*sp*sy + -sr*cy),
		cr * cp,
	}
	return
}
