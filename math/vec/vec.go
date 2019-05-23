package vec

import (
	"github.com/chewxy/math32"
)

type Vec3 struct {
	X, Y, Z float32
}

func VFromA(a [3]float32) Vec3 {
	return Vec3{a[0], a[1], a[2]}
}

func (v *Vec3) Array() [3]float32 {
	return [3]float32{v.X, v.Y, v.Z}
}

func (v *Vec3) Idx(i int) float32 {
	switch i {
	default:
		return v.X
	case 1:
		return v.Y
	case 2:
		return v.Z
	}
}

// Length returns the length of the vector
func (v *Vec3) Length() float32 {
	return math32.Sqrt(Dot(*v, *v))
}

// Add returns a + b
func Add(a, b Vec3) Vec3 {
	return Vec3{
		X: a.X + b.X,
		Y: a.Y + b.Y,
		Z: a.Z + b.Z,
	}
}

// Sub returns a - b
func Sub(a, b Vec3) Vec3 {
	return Vec3{
		X: a.X - b.X,
		Y: a.Y - b.Y,
		Z: a.Z - b.Z,
	}
}

// Scale returns the vector multiplied by the skalar s
func (v Vec3) Scale(s float32) Vec3 {
	return Vec3{
		X: v.X * s,
		Y: v.Y * s,
		Z: v.Z * s,
	}
}

// Normalize returns the normalized vector
func (v *Vec3) Normalize() Vec3 {
	l := v.Length()
	if l == 0 {
		return Vec3{}
	}
	return v.Scale(1 / l)
}

// Dot returns a dot b
func Dot(a Vec3, b Vec3) float32 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

// DoublePrecDot return a dot b calculated in double precision
func DoublePrecDot(a Vec3, b Vec3) float32 {
	p := func(x, y float32) float64 {
		return float64(x) * float64(y)
	}
	return float32(p(a.X, b.X) + p(a.Y, b.Y) + p(a.Z, b.Z))
}

// Cross returns a cross b
func Cross(a, b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

// Lerp computes a weighted average between two points
func Lerp(a, b Vec3, frac float32) Vec3 {
	fi := 1 - frac
	return Vec3{
		fi*a.X + frac*b.X,
		fi*a.Y + frac*b.Y,
		fi*a.Z + frac*b.Z,
	}
}

// Equal returns a == b
func Equal(a Vec3, b Vec3) bool {
	return a.X == b.X && a.Y == b.Y && a.Z == b.Z
}

func minmax(a, b float32) (float32, float32) {
	if a < b {
		return a, b
	}
	return b, a
}

func MinMax(a, b Vec3) (Vec3, Vec3) {
	var r, s Vec3
	r.X, s.X = minmax(a.X, b.X)
	r.Y, s.Y = minmax(a.Y, b.Y)
	r.Z, s.Z = minmax(a.Z, b.Z)
	return r, s
}

func AngleVectors(angles Vec3) (forward, right, up Vec3) {
	deg := math32.Pi * 2 / 360
	sp, cp := math32.Sincos(angles.X * deg) // PITCH
	sy, cy := math32.Sincos(angles.Y * deg) // YAW
	sr, cr := math32.Sincos(angles.Z * deg) // ROLL

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
