// SPDX-License-Identifier: GPL-2.0-or-later

package filesystem

//TODO(therjak): the pack files are never closed and ns is never cleaned. There should be an option.

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"goquake/filesystem/vfs"
	"goquake/pack"
)

// vfs:
// vfs.OS
// vfs.BindBefore
// vfs.NameSpace
// -- NameSpace.Bind
// -- NameSpace.Stat
// -- NameSpace.Open
// vfs.NewNameSpace

var (
	baseDir string
	//baseNS  = vfs.NewNameSpace()
	gameDir string
	gameNS  vfs.NameSpace
	mutex   sync.RWMutex
)

type File interface {
	io.ReadSeekCloser
	io.ReaderAt
}

type packFileSystem struct {
	p *pack.Pack
}

type closer struct {
	*io.SectionReader
}

func (*closer) Close() error {
	return nil
}

type fileInfo struct {
	name string // base name of the file
	size int64  // length in bytes for regular files; system-dependent for others
}

func (f *fileInfo) Name() string {
	return f.name
}
func (f *fileInfo) Size() int64 {
	return f.size
}
func (f *fileInfo) Mode() fs.FileMode {
	return 0
}
func (f *fileInfo) ModTime() time.Time {
	return time.Time{}
}
func (f *fileInfo) IsDir() bool {
	return false
}
func (f *fileInfo) Sys() any {
	return nil
}

func (p packFileSystem) Open(path string) (io.ReadSeekCloser, error) {
	// inside a pack file there is no 'root'. all files are relative to '.'
	path = strings.TrimPrefix(path, "/")
	f, err := p.p.Open(path)
	return &closer{f}, err
}

func (p packFileSystem) stat(path string) (os.FileInfo, error) {
	path = strings.TrimPrefix(path, "/")
	f, err := p.p.Open(path)
	if err != nil {
		return nil, err
	}
	return &fileInfo{
		name: path,
		size: f.Size(),
	}, nil
}

func (p packFileSystem) Stat(path string) (os.FileInfo, error) {
	return p.stat(path)
}

func (p packFileSystem) String() string {
	return p.p.String()
}

func GameDir() string {
	mutex.RLock()
	defer mutex.RUnlock()
	return gameDir
}

func BaseDir() string {
	mutex.RLock()
	defer mutex.RUnlock()
	return baseDir
}

func UseBaseDir(dir string) {
	mutex.Lock()
	defer mutex.Unlock()
	baseDir = dir
	root := filepath.Join(baseDir, "id1")
	gameDir = root
	gameNS = vfs.NameSpace{}
	gameNS.Bind("/", vfs.OS(root), "/", vfs.BindReplace)
	useDir(&gameNS, root)
}

func UseGameDir(dir string) {
	mutex.Lock()
	defer mutex.Unlock()
	gameNS = vfs.NameSpace{}
	root := filepath.Join(baseDir, "id1")
	gameNS.Bind("/", vfs.OS(root), "/", vfs.BindReplace)
	useDir(&gameNS, root)
	gameDir = filepath.Join(baseDir, dir)
	gameNS.Bind("/", vfs.OS(dir), "/", vfs.BindBefore)
	useDir(&gameNS, gameDir)
}

func useDir(ns *vfs.NameSpace, dir string) {
	// 1) Add pak[i].pak files to the beginning order high number to low number
	// 2) add quakespasm.pak to the beginning
	for i := 0; ; i++ {
		pfn := fmt.Sprintf("pak%d.pak", i)
		pfp := filepath.Join(dir, pfn)
		p, err := pack.NewPackReader(pfp)
		if err != nil {
			break
		}
		ns.Bind("/", packFileSystem{p}, "/", vfs.BindBefore)
	}
	qsm := filepath.Join(dir, "quakespasm.pak")
	qsmp, err := pack.NewPackReader(qsm)
	if err == nil {
		ns.Bind("/", packFileSystem{qsmp}, "/", vfs.BindBefore)
	}
}

func Stat(path string) (os.FileInfo, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return gameNS.Stat(path)
}

func Open(name string) (File, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	nf, err := gameNS.Open(filepath.Join("/", name))
	if err != nil {
		return nil, err
	}
	f, ok := nf.(File)
	if !ok {
		f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}

func ReadFile(name string) ([]byte, error) {
	file, err := Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func isSep(c uint8) bool {
	return c == '/' || c == '\\'
}

func Ext(path string) string {
	for i := len(path) - 1; i >= 0 && !isSep(path[i]); i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return ""
}

func StripExt(path string) string {
	for i := len(path) - 1; i >= 0 && !isSep(path[i]); i-- {
		if path[i] == '.' {
			return path[:i]
		}
	}
	return path
}
