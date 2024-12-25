// SPDX-License-Identifier: GPL-2.0-or-later

package filesystem

import (
	"io"
	"testing"

	"goquake/pack"
)

func TestPackFileSystem(t *testing.T) {
	p, err := pack.NewPackReader("testdir/pak0.pak")
	if err != nil {
		t.Fatalf("Could not open pak: %v", err)
	}
	pfs := packFileSystem{p}
	f, err := pfs.Open("doc1.txt")
	if err != nil {
		t.Fatalf("Could not open doc1: %v", err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("Could not read file: %v", err)
	}
	if string(b) != "this is the first doc\r\n" {
		t.Errorf("contents: %v", string(b))
	}
}

func TestFilesystemOrder(t *testing.T) {
	UseGameDir("testdir")
	f, err := Open("doc1.txt")
	if err != nil {
		t.Fatalf("No file doc1: %v", err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("Could not read file: %v", err)
	}
	if string(b) != "this is the first doc 2. version\r\n" {
		t.Errorf("contents: %v", b)
	}
}

func TestFilesystemPak(t *testing.T) {
	UseGameDir("testdir")
	f, err := Open("doc2.txt")
	if err != nil {
		t.Fatalf("No file doc2: %v", err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("Could not read file: %v", err)
	}
	if string(b) != "this is the second doc 2. version" {
		t.Errorf("contents: %v", string(b))
	}
}

func TestFilesystemOs(t *testing.T) {
	UseGameDir("testdir")
	f, err := Open("doc5.txt")
	if err != nil {
		t.Fatalf("No file doc5: %v", err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("Could not read file: %v", err)
	}
	if string(b) != "good file5\n" {
		t.Errorf("contents: %v", b)
	}

}

func TestExt(t *testing.T) {
	for _, tt := range []struct {
		in  string
		out string
	}{
		{"blub", ""},
		{"blub.txt", ".txt"},
		{"blah.c/blub", ""},
		{"blah.c/blub.h", ".h"},
		{"blah.c\\blub", ""},
		{"blah.c\\blub.h", ".h"},
	} {
		t.Run(tt.in, func(t *testing.T) {
			s := Ext(tt.in)
			if s != tt.out {
				t.Errorf("got %q, want %q", s, tt.out)
			}
		})
	}
}

func TestStripExt(t *testing.T) {
	for _, tt := range []struct {
		in  string
		out string
	}{
		{"blub", "blub"},
		{"blub.txt", "blub"},
		{"blah.c/blub", "blah.c/blub"},
		{"blah.c/blub.h", "blah.c/blub"},
		{"blah.c\\blub", "blah.c\\blub"},
		{"blah.c\\blub.h", "blah.c\\blub"},
	} {
		t.Run(tt.in, func(t *testing.T) {
			s := StripExt(tt.in)
			if s != tt.out {
				t.Errorf("got %q, want %q", s, tt.out)
			}
		})
	}

}
