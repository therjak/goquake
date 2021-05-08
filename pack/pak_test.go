// SPDX-License-Identifier: GPL-2.0-or-later

package pack

import (
	"io/ioutil"
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
	if p.String() != pakFile {
		t.Errorf("pack String error: want %v got %v", pakFile, p.String())
	}
	f1 := p.GetFile("doc1.txt")
	if f1 == nil {
		t.Error("Got no file 'doc1.txt")
	}
	b1, err := ioutil.ReadAll(f1)
	if err != nil {
		t.Fatalf("Could not read f1: %v", err)
	}
	if string(b1) != "this is the first doc 2. version\r\n" {
		t.Errorf("f1 contents is '%v'", b1)
	}
	f4 := p.GetFile("doc4.txt")
	if f4 == nil {
		t.Error("Got no file 'doc4.txt")
	}
	f5 := p.GetFile("testdir/doc4.txt")
	if f5 == nil {
		t.Error("Got no file 'testdir/doc4.txt")
	}
	b5, err := ioutil.ReadAll(f5)
	if err != nil {
		t.Fatalf("Could not read f5: %v", err)
	}
	if string(b5) != `this is the fourth doc 2. version` {
		t.Errorf("f5 contents is '%v'", string(b5))
	}

}
