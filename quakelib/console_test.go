// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"reflect"
	"testing"
	"time"

	qsnd "goquake/snd"
)

func TestConsolePrintNotInitialized(t *testing.T) {
	c := &qconsole{
		initialized: false,
	}
	c.Print("hello")
	if len(c.origText) != 0 {
		t.Errorf("Print on uninitialized console should not print anything, got: %v", c.origText)
	}
}

func TestConsolePrintEmpty(t *testing.T) {
	c := &qconsole{
		initialized: true,
	}
	c.print("")
	if len(c.origText) != 0 {
		t.Errorf("print empty string should not print anything, got: %v", c.origText)
	}
}

func TestConsolePrintAppendLines(t *testing.T) {
	c := &qconsole{
		initialized: true,
	}

	// 1. Print a line without newline
	c.print("hello")
	expected := []string{"hello"}
	if !reflect.DeepEqual(c.origText, expected) {
		t.Errorf("expected %v, got %v", expected, c.origText)
	}

	// 2. Print another line without newline (should append to the last line)
	c.print(" world")
	expected = []string{"hello world"}
	if !reflect.DeepEqual(c.origText, expected) {
		t.Errorf("expected %v, got %v", expected, c.origText)
	}

	// 3. Print a line ending with newline
	c.print("!\n")
	expected = []string{"hello world!\n"}
	if !reflect.DeepEqual(c.origText, expected) {
		t.Errorf("expected %v, got %v", expected, c.origText)
	}

	// 4. Print after a newline (should start a new line)
	c.print("new line")
	expected = []string{"hello world!\n", "new line"}
	if !reflect.DeepEqual(c.origText, expected) {
		t.Errorf("expected %v, got %v", expected, c.origText)
	}

	// 5. Print multiple lines with newlines in between
	c.print("\nsecond line\nthird line")
	expected = []string{"hello world!\n", "new line\n", "second line\n", "third line"}
	if !reflect.DeepEqual(c.origText, expected) {
		t.Errorf("expected %v, got %v", expected, c.origText)
	}
}

func TestConsolePrintTimes(t *testing.T) {
	c := &qconsole{
		initialized: true,
	}

	// Since times only records up to 4 elements, let's verify how it gets populated
	// Step 1: append 1 line. Times should have 1 non-zero value at the end.
	c.print("one\n")
	nonZeroCount := 0
	for _, tm := range c.times {
		if !tm.IsZero() {
			nonZeroCount++
		}
	}
	if nonZeroCount != 1 {
		t.Errorf("expected 1 non-zero time, got %v", nonZeroCount)
	}

	// Step 2: append 4 lines at once. Times should be completely filled with non-zero values.
	c.print("two\nthree\nfour\nfive\n")
	for i, tm := range c.times {
		if tm.IsZero() {
			t.Errorf("expected time %d to be non-zero", i)
		}
	}
}

func TestConsolePrintSpecialChars(t *testing.T) {
	// Backup and mock defaultSounds to prevent nil pointer panics
	origDefaultSounds := defaultSounds
	defaultSounds = &qsnd.SoundPrecache{}
	t.Cleanup(func() {
		defaultSounds = origDefaultSounds
	})

	c := &qconsole{
		initialized: true,
	}

	// Test \x01: Play talk sound (which our mocked SoundPrecache will ignore gracefully)
	// and mask characters with 128 (high bit set)
	c.print("\x01abc\n")
	if len(c.origText) != 1 {
		t.Fatalf("expected 1 line, got %d", len(c.origText))
	}
	// 'a' is 97. 97 | 128 = 225.
	// '\n' should NOT be masked.
	expectedLine := string([]byte{97 | 128, 98 | 128, 99 | 128, '\n'})
	if c.origText[0] != expectedLine {
		t.Errorf("expected %q, got %q", expectedLine, c.origText[0])
	}

	// Reset console
	c.Clear()

	// Test \x02: Mask characters with 128 without playing sound
	c.print("\x02def\n")
	if len(c.origText) != 1 {
		t.Fatalf("expected 1 line, got %d", len(c.origText))
	}
	expectedLine2 := string([]byte{100 | 128, 101 | 128, 102 | 128, '\n'})
	if c.origText[0] != expectedLine2 {
		t.Errorf("expected %q, got %q", expectedLine2, c.origText[0])
	}
}

func TestConsolePrintf(t *testing.T) {
	c := &qconsole{
		initialized: true,
	}
	c.Printf("hello %s %d", "world", 42)
	expected := []string{"hello world 42"}
	if !reflect.DeepEqual(c.origText, expected) {
		t.Errorf("expected %v, got %v", expected, c.origText)
	}
}

func TestConsoleClear(t *testing.T) {
	c := &qconsole{
		initialized: true,
		origText:    []string{"a", "b"},
		backScroll:  5,
	}
	c.Clear()
	if len(c.origText) != 0 || c.backScroll != 0 {
		t.Errorf("Clear failed, got origText=%v, backScroll=%v", c.origText, c.backScroll)
	}
}

func TestConsoleCenterPrint(t *testing.T) {
	c := &qconsole{
		initialized: true,
		lineWidth:   40,
	}

	// 1. Text length less than width (40)
	// "hello" length is 5.
	// wl = (40 - 5) / 2 = 17.
	// Padding should be 17 spaces.
	c.centerPrint("hello")
	expected := []string{"                 hello\n"}
	if !reflect.DeepEqual(c.origText, expected) {
		t.Errorf("expected %q, got %q", expected, c.origText)
	}

	// Reset console
	c.Clear()

	// 2. Text containing escaped \n (rendered as "\\n")
	// "abc\ndef" -> parts: "abc", "def"
	// for "abc" (len 3): wl = (40 - 3) / 2 = 18.
	// for "def" (len 3): wl = 18.
	c.centerPrint("abc\\ndef")
	expected2 := []string{"                  abc\n", "                  def\n"}
	if !reflect.DeepEqual(c.origText, expected2) {
		t.Errorf("expected %q, got %q", expected2, c.origText)
	}
}

func TestGlobalConsoleCenterPrint(t *testing.T) {
	// Backup global console settings
	origInitialized := console.initialized
	origOrigText := console.origText
	origTimes := console.times
	origLastCenter := console.lastCenter
	origLineWidth := console.lineWidth

	console.initialized = true
	console.origText = nil
	console.times = [4]time.Time{}
	console.lastCenter = ""
	console.lineWidth = 40 // match quakeBar length

	t.Cleanup(func() {
		console.initialized = origInitialized
		console.origText = origOrigText
		console.times = origTimes
		console.lastCenter = origLastCenter
		console.lineWidth = origLineWidth
	})

	console.CenterPrint("hello")

	// We expect:
	// 1. quakeBar (the first bar)
	// 2. centered text: "                 hello\n"
	// 3. quakeBar (the second bar)
	if len(console.origText) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(console.origText), console.origText)
	}
	if console.origText[0] != quakeBar {
		t.Errorf("expected first line to be quakeBar, got %q", console.origText[0])
	}
	expectedCenter := "                 hello\n"
	if console.origText[1] != expectedCenter {
		t.Errorf("expected second line to be %q, got %q", expectedCenter, console.origText[1])
	}
	if console.origText[2] != quakeBar {
		t.Errorf("expected third line to be quakeBar, got %q", console.origText[2])
	}
}
