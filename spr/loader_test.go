// SPDX-License-Identifier: GPL-2.0-or-later
package spr

import "testing"

func TestFloorExact(t *testing.T) {
	v := floor(5)
	if v != 5 {
		t.Errorf("floor(5) = %v", v)
	}
}
func TestFloorClose(t *testing.T) {
	v := floor(4.999)
	if v != 4 {
		t.Errorf("floor(4.999) = %v", v)
	}
}
