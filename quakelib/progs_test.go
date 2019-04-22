package quakelib

import (
	"testing"
)

func TestEngineStringsEDNewString(t *testing.T) {
	s := "Test"
	idx := EDNewString(s)
	rs := PRGetString(idx)
	if *rs != s {
		t.Errorf("Strings are not equal: %v, %v", s, *rs)
	}
}

func TestEngineStringsSetEngineString(t *testing.T) {
	s := "Test"
	idx := PRSetEngineString(s)
	rs := PRGetString(idx)
	if *rs != s {
		t.Errorf("Strings are not equal: %v, %v", s, *rs)
	}
}
