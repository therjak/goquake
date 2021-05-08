// SPDX-License-Identifier: GPL-2.0-or-later

package progs

import (
	"testing"
)

func TestEngineStringsEDNewString(t *testing.T) {
	s := "Test"
	p := LoadedProg{}
	idx := p.NewString(s)
	rs, err := p.String(idx)
	if err != nil {
		t.Errorf("Could not return string")
	}
	if rs != s {
		t.Errorf("Strings are not equal: %v, %v", s, rs)
	}
}

func TestEngineStringsSetEngineString(t *testing.T) {
	s := "Test"
	p := LoadedProg{}
	idx := p.AddString(s)
	rs, err := p.String(idx)
	if err != nil {
		t.Errorf("Could not return string")
	}
	if rs != s {
		t.Errorf("Strings are not equal: %v, %v", s, rs)
	}
}

func TestEngineStringsDoubleAdd(t *testing.T) {
	s := "Test"
	p := LoadedProg{}
	// Do not start with an empty string store.
	p.AddString("blub")
	p.AddString("blub2")
	// Now the test:
	idx1 := p.AddString(s)
	idx2 := p.AddString(s)
	idx3 := p.AddString(s)
	if idx1 != idx2 {
		t.Errorf("2. AddString(s) = %d, want %d", idx2, idx1)
	}
	if idx1 != idx3 {
		t.Errorf("3. AddString(s) = %d, want %d", idx3, idx1)
	}
}
