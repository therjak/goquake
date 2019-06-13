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
