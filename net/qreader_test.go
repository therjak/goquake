// SPDX-License-Identifier: GPL-2.0-or-later

package net

import (
	"testing"
)

func TestReadString(t *testing.T) {
	tests := []struct {
		reader     *QReader
		shouldFail bool
		result     string
	}{
		{
			NewQReader([]byte{'h', 'e', 'l', 'l', 'o', 0, 's', 't', 'u', 'f', 'f'}),
			false,
			string([]byte{'h', 'e', 'l', 'l', 'o'}),
		},
		{
			NewQReader([]byte{'h', 'e', 'l', 'l', 'o'}),
			true,
			"",
		},
	}
	for i, tc := range tests {
		s, err := tc.reader.ReadString()
		if err != nil {
			if !tc.shouldFail {
				t.Errorf("Testcase %d should not return error: %v", i, err)
			} else {
				continue
			}
		}
		if tc.shouldFail {
			t.Errorf("Testcase %d should return error", i)
			continue
		}
		if s != tc.result {
			t.Errorf("Testcase %d. got: %v, want %v", i, s, tc.result)
		}
	}
}
