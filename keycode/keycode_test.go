// SPDX-License-Identifier: GPL-2.0-or-later

package keycode

import (
	"testing"
)

func TestKeyToString(t *testing.T) {
	tests := []struct {
		key KeyCode
		str string
	}{
		{TAB, "TAB"},
		{PAUSE, "PAUSE"},
		{0x30, "0"},
		{0x3B, ";"},
		{0x41, "A"},
		{0x60, "`"},
		{0x61, "a"},
		{0x7E, "~"},
		{0x7F, "BACKSPACE"},
		{0x80, "UPARROW"},
	}
	for _, test := range tests {
		if got := KeyToString(test.key); got != test.str {
			t.Errorf("KeyToString(%d) = %s; want %s", test.key, got, test.str)
		}
	}
}

func TestStringToKey(t *testing.T) {
	tests := []struct {
		key KeyCode
		str string
	}{
		{TAB, "TAB"},
		{PAUSE, "PAUSE"},
		{0x30, "0"},
		{0x3B, ";"},
		{0x41, "A"},
		{0x60, "`"},
		{0x61, "a"},
		{0x7E, "~"},
		{0x7F, "BACKSPACE"},
		{0x80, "UPARROW"},
	}
	for _, test := range tests {
		if got := StringToKey(test.str); got != test.key {
			t.Errorf("KeyToString(%s) = %d; want %d", test.str, got, test.key)
		}
	}
}
