// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vfs

import (
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"strings"
)

// A NameSpace is a file system made up of other file systems
// mounted at specific locations in the name space.
//
// The representation is a map from mount point locations
// to the list of file systems mounted at that location.  A traditional
// Unix mount table would use a single file system per mount point,
// but we want to be able to mount multiple file systems on a single
// mount point and have the system behave as if the union of those
// file systems were present at the mount point.
// For example, if the OS file system has a Go installation in
// c:\Go and additional Go path trees in d:\Work1 and d:\Work2, then
// this name space creates the view we want for the godoc server:
//
//	NameSpace{
//		"/": {
//			{old: "/", fs: OS(`c:\Go`), new: "/"},
//		},
//		"/src/pkg": {
//			{old: "/src/pkg", fs: OS(`c:\Go`), new: "/src/pkg"},
//			{old: "/src/pkg", fs: OS(`d:\Work1`), new: "/src"},
//			{old: "/src/pkg", fs: OS(`d:\Work2`), new: "/src"},
//		},
//	}
//
// This is created by executing:
//
//	ns := NameSpace{}
//	ns.Bind("/", OS(`c:\Go`), "/", BindReplace)
//	ns.Bind("/src/pkg", OS(`d:\Work1`), "/src", BindAfter)
//	ns.Bind("/src/pkg", OS(`d:\Work2`), "/src", BindAfter)
//
// A particular mount point entry is a triple (old, fs, new), meaning that to
// operate on a path beginning with old, replace that prefix (old) with new
// and then pass that path to the FileSystem implementation fs.
//
// If you do not explicitly mount a FileSystem at the root mountpoint "/" of the
// NameSpace like above, Stat("/") will return a "not found" error which could
// break typical directory traversal routines. In such cases, use NewNameSpace()
// to get a NameSpace pre-initialized with an emulated empty directory at root.
//
// Given this name space, a ReadDir of /src/pkg/code will check each prefix
// of the path for a mount point (first /src/pkg/code, then /src/pkg, then /src,
// then /), stopping when it finds one.  For the above example, /src/pkg/code
// will find the mount point at /src/pkg:
//
//	{old: "/src/pkg", fs: OS(`c:\Go`), new: "/src/pkg"},
//	{old: "/src/pkg", fs: OS(`d:\Work1`), new: "/src"},
//	{old: "/src/pkg", fs: OS(`d:\Work2`), new: "/src"},
//
// ReadDir will when execute these three calls and merge the results:
//
//	OS(`c:\Go`).ReadDir("/src/pkg/code")
//	OS(`d:\Work1').ReadDir("/src/code")
//	OS(`d:\Work2').ReadDir("/src/code")
//
// Note that the "/src/pkg" in "/src/pkg/code" has been replaced by
// just "/src" in the final two calls.
//
// OS is itself an implementation of a file system: it implements
// OS(`c:\Go`).ReadDir("/src/pkg/code") as ioutil.ReadDir(`c:\Go\src\pkg\code`).
//
// Because the new path is evaluated by fs (here OS(root)), another way
// to read the mount table is to mentally combine fs+new, so that this table:
//
//	{old: "/src/pkg", fs: OS(`c:\Go`), new: "/src/pkg"},
//	{old: "/src/pkg", fs: OS(`d:\Work1`), new: "/src"},
//	{old: "/src/pkg", fs: OS(`d:\Work2`), new: "/src"},
//
// reads as:
//
//	"/src/pkg" -> c:\Go\src\pkg
//	"/src/pkg" -> d:\Work1\src
//	"/src/pkg" -> d:\Work2\src
//
// An invariant (a redundancy) of the name space representation is that
// ns[mtpt][i].old is always equal to mtpt (in the example, ns["/src/pkg"]'s
// mount table entries always have old == "/src/pkg").  The 'old' field is
// useful to callers, because they receive just a []mountedFS and not any
// other indication of which mount point was found.
type NameSpace map[string][]mountedFS

// A mountedFS handles requests for path by replacing
// a prefix 'old' with 'new' and then calling the fs methods.
type mountedFS struct {
	old string
	fs  FileSystem
	new string
}

// hasPathPrefix reports whether x == y or x == y + "/" + more.
func hasPathPrefix(x, y string) bool {
	return x == y || strings.HasPrefix(x, y) && (strings.HasSuffix(y, "/") || strings.HasPrefix(x[len(y):], "/"))
}

// translate translates path for use in m, replacing old with new.
//
// mountedFS{"/src/pkg", fs, "/src"}.translate("/src/pkg/code") == "/src/code".
func (m mountedFS) translate(path string) string {
	path = pathpkg.Clean("/" + path)
	if !hasPathPrefix(path, m.old) {
		panic("translate " + path + " but old=" + m.old)
	}
	return pathpkg.Join(m.new, path[len(m.old):])
}

func (NameSpace) String() string {
	return "ns"
}

// clean returns a cleaned, rooted path for evaluation.
// It canonicalizes the path so that we can use string operations
// to analyze it.
func (NameSpace) clean(path string) string {
	return pathpkg.Clean("/" + path)
}

type BindMode int

const (
	BindReplace BindMode = iota
	BindBefore
	BindAfter
)

// Bind causes references to old to redirect to the path new in newfs.
// If mode is BindReplace, old redirections are discarded.
// If mode is BindBefore, this redirection takes priority over existing ones,
// but earlier ones are still consulted for paths that do not exist in newfs.
// If mode is BindAfter, this redirection happens only after existing ones
// have been tried and failed.
func (ns NameSpace) Bind(old string, newfs FileSystem, new string, mode BindMode) {
	old = ns.clean(old)
	new = ns.clean(new)
	m := mountedFS{old, newfs, new}
	var mtpt []mountedFS
	switch mode {
	case BindReplace:
		mtpt = append(mtpt, m)
	case BindAfter:
		mtpt = append(mtpt, ns.resolve(old)...)
		mtpt = append(mtpt, m)
	case BindBefore:
		mtpt = append(mtpt, m)
		mtpt = append(mtpt, ns.resolve(old)...)
	}

	// Extend m.old, m.new in inherited mount point entries.
	for i := range mtpt {
		m := &mtpt[i]
		if m.old != old {
			if !hasPathPrefix(old, m.old) {
				// This should not happen.  If it does, panic so
				// that we can see the call trace that led to it.
				panic(fmt.Sprintf("invalid Bind: old=%q m={%q, %s, %q}", old, m.old, m.fs.String(), m.new))
			}
			suffix := old[len(m.old):]
			m.old = pathpkg.Join(m.old, suffix)
			m.new = pathpkg.Join(m.new, suffix)
		}
	}

	ns[old] = mtpt
}

// resolve resolves a path to the list of mountedFS to use for path.
func (ns NameSpace) resolve(path string) []mountedFS {
	path = ns.clean(path)
	for {
		if m := ns[path]; m != nil {
			return m
		}
		if path == "/" {
			break
		}
		path = pathpkg.Dir(path)
	}
	return nil
}

// Open implements the FileSystem Open method.
func (ns NameSpace) Open(path string) (io.ReadSeekCloser, error) {
	var err error
	for _, m := range ns.resolve(path) {
		tp := m.translate(path)
		r, err1 := m.fs.Open(tp)
		if err1 == nil {
			return r, nil
		}
		// IsNotExist errors in overlay FSes can mask real errors in
		// the underlying FS, so ignore them if there is another error.
		if err == nil || os.IsNotExist(err) {
			err = err1
		}
	}
	if err == nil {
		err = &os.PathError{Op: "open", Path: path, Err: os.ErrNotExist}
	}
	return nil, err
}

// stat implements the FileSystem Stat and Lstat methods.
func (ns NameSpace) stat(path string, f func(FileSystem, string) (os.FileInfo, error)) (os.FileInfo, error) {
	var err error
	for _, m := range ns.resolve(path) {
		fi, err1 := f(m.fs, m.translate(path))
		if err1 == nil {
			return fi, nil
		}
		if err == nil {
			err = err1
		}
	}
	if err == nil {
		err = &os.PathError{Op: "stat", Path: path, Err: os.ErrNotExist}
	}
	return nil, err
}

func (ns NameSpace) Stat(path string) (os.FileInfo, error) {
	return ns.stat(path, FileSystem.Stat)
}
