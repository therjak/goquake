package filesystem

import (
	"io/ioutil"
	"quake/pack"
	"testing"
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
	b, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("Could not read file: %v", err)
	}
	if string(b) != "this is the first doc\r\n" {
		t.Errorf("contents: %v", string(b))
	}
}

func TestFilesystemOrder(t *testing.T) {
	AddGameDir("testdir")
	f, err := GetFile("doc1.txt")
	if err != nil {
		t.Fatalf("No file doc1: %v", err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("Could not read file: %v", err)
	}
	if string(b) != "this is the first doc 2. version\r\n" {
		t.Errorf("contents: %v", b)
	}
}

func TestFilesystemPak(t *testing.T) {
	AddGameDir("testdir")
	f, err := GetFile("doc2.txt")
	if err != nil {
		t.Fatalf("No file doc2: %v", err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("Could not read file: %v", err)
	}
	if string(b) != "this is the second doc 2. version" {
		t.Errorf("contents: %v", string(b))
	}
}

func TestFilesystemOs(t *testing.T) {
	AddGameDir("testdir")
	f, err := GetFile("doc5.txt")
	if err != nil {
		t.Fatalf("No file doc5: %v", err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("Could not read file: %v", err)
	}
	if string(b) != "good file5\n" {
		t.Errorf("contents: %v", b)
	}

}
