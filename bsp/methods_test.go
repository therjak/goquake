// SPDX-License-Identifier: GPL-2.0-or-later
package bsp

import (
	"bytes"
	"testing"
)

func TestVisDecompress(t *testing.T) {
	m := Model{
		Leafs: make([](*MLeaf), 12*8),
	}
	in := []byte{0x7, 0x0, 0x5, 0x5, 0x0, 0x3, 0x1, 0x1}
	got := m.DecompressVis(in)
	want := []byte{0x7, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x0, 0x0, 0x0, 0x1, 0x1}
	if bytes.Compare(got, want) != 0 {
		t.Errorf("Decompress(%v) = %v, want %v", in, got, want)
	}
}
