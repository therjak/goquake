// SPDX-License-Identifier: GPL-2.0-or-later

package history

import (
	"fmt"
	"testing"
)

func TestEmpty(t *testing.T) {
	h := History{}
	want := ""
	got := h.String()
	if got != want {
		t.Errorf("empty String() = %q, want %q", got, want)
	}
}

func TestEmptyUp(t *testing.T) {
	h := History{}
	h.Up()
	want := ""
	got := h.String()
	if got != want {
		t.Errorf("empty after Up String() = %q, want %q", got, want)
	}
}

func TestEmptyDown(t *testing.T) {
	h := History{}
	h.Down()
	want := ""
	got := h.String()
	if got != want {
		t.Errorf("empty after Down String() = %q, want %q", got, want)
	}
}

func TestAdd(t *testing.T) {
	h := History{}
	l0 := "line0"
	l1 := "line1"
	h.Add(l0)
	// expect the 'empty' head
	want := ""
	got := h.String()
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	h.Up()
	want = l0
	got = h.String()
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	h.Add(l1)
	h.Up()
	want = l1
	got = h.String()
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	h.Up()
	want = l0
	got = h.String()
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	h.Up()
	want = l0
	got = h.String()
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func testHistory() *History {
	h := &History{}
	for i := 0; i < 10; i++ {
		h.Add(fmt.Sprintf("line%d", i))
	}
	return h
}

func TestUpDown(t *testing.T) {
	h := testHistory()
	h.Up()
	want := "line9"
	got := h.String()
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	h.Up()
	want = "line8"
	got = h.String()
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	h.Down()
	want = "line9"
	got = h.String()
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
