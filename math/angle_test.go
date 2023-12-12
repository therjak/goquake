// SPDX-License-Identifier: GPL-2.0-or-later

package math

import (
	"testing"
)

func TestAngleInside(t *testing.T) {
	var a float64 = 180
	got := AngleMod(a)
	if got != a {
		t.Errorf("AngleMod(%v) = %v want 180", a, got)
	}
}

func TestAngleInside2(t *testing.T) {
	var a float64 = 66.6666
	got := AngleMod(a)
	if got != a {
		t.Errorf("AngleMod(%v) = %v want %v", a, got, a)
	}
}

func TestAngleOver(t *testing.T) {
	var a float64 = 180 + 360
	got := AngleMod(a)
	if got != 180 {
		t.Errorf("AngleMod(%v) = %v want 180", a, got)
	}
}

func TestAngleUnder(t *testing.T) {
	var a float64 = 180 - 360
	got := AngleMod(a)
	if got != 180 {
		t.Errorf("AngleMod(%v) = %v want 180", a, got)
	}
}

func TestAngleLower(t *testing.T) {
	var a float64 = 0
	got := AngleMod(a)
	if got != 0 {
		t.Errorf("AngleMod(%v) = %v want 0", a, got)
	}
}

func TestAngleUpper(t *testing.T) {
	var a float64 = 360
	got := AngleMod(a)
	if got != 0 {
		t.Errorf("AngleMod(%v) = %v want 0", a, got)
	}
}
