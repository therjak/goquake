// SPDX-License-Identifier: GPL-2.0-or-later

package pack

import (
	"io"
	"testing"
)

const (
	pakFile = "pak1.pak"
)

func TestPak(t *testing.T) {
	p, err := NewPackReader(pakFile)
	if err != nil {
		t.Fatalf("could not open %s: %v", pakFile, err)
	}
	defer p.Close()
	if p.String() != pakFile {
		t.Errorf("pack String error: want %v got %v", pakFile, p.String())
	}
	f1, err := p.Open("doc1.txt")
	if err != nil {
		t.Error("Got no file 'doc1.txt")
	}
	b1, err := io.ReadAll(f1)
	if err != nil {
		t.Fatalf("Could not read f1: %v", err)
	}
	if string(b1) != "this is the first doc 2. version\r\n" {
		t.Errorf("f1 contents is '%v'", b1)
	}
	_, err = p.Open("doc4.txt")
	if err != nil {
		t.Error("Got no file 'doc4.txt")
	}
	f5, err := p.Open("testdir/doc4.txt")
	if err != nil {
		t.Error("Got no file 'testdir/doc4.txt")
	}
	b5, err := io.ReadAll(f5)
	if err != nil {
		t.Fatalf("Could not read f5: %v", err)
	}
	if string(b5) != `this is the fourth doc 2. version` {
		t.Errorf("f5 contents is '%v'", string(b5))
	}

}
