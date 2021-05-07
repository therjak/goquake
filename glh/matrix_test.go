// SPDX-License-Identifier: GPL-2.0-or-later

package glh

import "testing"

const (
	e = 1.e-15
)

func eq(a, b [16]float32) bool {
	for i := range a {
		if a[i]-b[i] > e {
			return false
		}
	}
	return true
}

func TestIdentity(t *testing.T) {
	m := Identity()
	if !eq(m.m, [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}) {
		t.Errorf("Identity broken: %v", m.m)
	}
}

func TestTranslate(t *testing.T) {
	m := Identity()
	m.Translate(2, 3, 5)
	if !eq(m.m, [16]float32{
		1, 0, 0, 2,
		0, 1, 0, 3,
		0, 0, 1, 5,
		0, 0, 0, 1,
	}) {
		t.Errorf("Identity.Translate(2,3,5) = %v", m.m)
	}
}

func TestScale(t *testing.T) {
	m := Identity()
	m.Scale(2, 3, 5)
	if !eq(m.m, [16]float32{
		2, 0, 0, 0,
		0, 3, 0, 0,
		0, 0, 5, 0,
		0, 0, 0, 1,
	}) {
		t.Errorf("Identity.Scale(2,3,5) = %v", m.m)
	}
}

func TestRotateX(t *testing.T) {
	m := Identity()
	m.RotateX(90)
	if !eq(m.m, [16]float32{
		1, 0, 0, 0,
		0, 0, -1, 0,
		0, 1, 0, 0,
		0, 0, 0, 1,
	}) {
		t.Errorf("Identity.RotateX(90) = %v", m.m)
	}
}

func TestRotateY(t *testing.T) {
	m := Identity()
	m.RotateY(90)
	if !eq(m.m, [16]float32{
		0, 0, 1, 0,
		0, 1, 0, 0,
		-1, 0, 0, 0,
		0, 0, 0, 1,
	}) {
		t.Errorf("Identity.RotateY(90) = %v", m.m)
	}
}

func TestRotateZ(t *testing.T) {
	m := Identity()
	m.RotateZ(90)
	if !eq(m.m, [16]float32{
		0, -1, 0, 0,
		1, 0, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}) {
		t.Errorf("Identity.RotateZ(90) = %v", m.m)
	}
}
