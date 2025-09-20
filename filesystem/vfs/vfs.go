// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package vfs defines types for abstract file system access and provides an
// implementation accessing the file system of the underlying OS.
package vfs // import "golang.org/x/tools/godoc/vfs"

import (
	"io"
	"os"
)

// The FileSystem interface specifies the methods godoc is using
// to access the file system for which it serves documentation.
type FileSystem interface {
	Open(name string) (io.ReadSeekCloser, error)
	Stat(path string) (os.FileInfo, error)
	String() string
}
