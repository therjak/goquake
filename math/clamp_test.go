// SPDX-License-Identifier: GPL-2.0-or-later

package math

import (
	"testing"
)

func TestClampMin(t *testing.T) {
	v := Clamp(1, 0, 10)
	if v != 1 {
		t.Errorf("Clamp(1,0,10) = %v", v)
	}
}

func TestClampMan(t *testing.T) {
	v := Clamp(1, 100, 10)
	if v != 10 {
		t.Errorf("Clamp(1,100,10) = %v", v)
	}
}

func TestClampVal(t *testing.T) {
	v := Clamp(1, 5, 10)
	if v != 5 {
		t.Errorf("Clamp(1,5,10) = %v", v)
	}
}
