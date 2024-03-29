// SPDX-License-Identifier: GPL-2.0-or-later

package filesystem

//TODO(therjak): the pack files are never closed and ns is never cleaned. There should be an option.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"goquake/pack"

	"golang.org/x/tools/godoc/vfs"
)

var (
	baseDir string
	//baseNS  = vfs.NewNameSpace()
	gameDir string
	gameNS  vfs.NameSpace
	mutex   sync.RWMutex
)

type QFile interface {
	io.Seeker
	// io.ReaderAt
	io.Reader
	io.Closer
}

type packFileSystem struct {
	p *pack.Pack
}

func (p packFileSystem) RootType(path string) vfs.RootType {
	return ""
}

func (p packFileSystem) Open(path string) (vfs.ReadSeekCloser, error) {
	// inside a pack file there is no 'root'. all files are relative to '.'
	path = strings.TrimPrefix(path, "/")
	f := p.p.GetFile(path)
	if f != nil {
		return f, nil
	}
	return nil, os.ErrNotExist
}

func (p packFileSystem) stat(path string) (os.FileInfo, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (p packFileSystem) Lstat(path string) (os.FileInfo, error) {
	return p.stat(path)
}

func (p packFileSystem) Stat(path string) (os.FileInfo, error) {
	return p.stat(path)
}

func (p packFileSystem) ReadDir(path string) ([]os.FileInfo, error) {
	return nil, fmt.Errorf("Not implemented")
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
	gameDir = filepath.Join(baseDir, "id1")
	gameNS = vfs.NewNameSpace()
	useDir(&gameNS, gameDir)
}

func UseGameDir(dir string) {
	mutex.Lock()
	defer mutex.Unlock()
	gameNS = vfs.NewNameSpace()
	useDir(&gameNS, filepath.Join(baseDir, "id1"))
	gameDir = filepath.Join(baseDir, dir)
	useDir(&gameNS, gameDir)
}

func useDir(ns *vfs.NameSpace, dir string) {
	// 1) add path to the beginning of the search paths list
	// 2) Add pak[i].pak files to the beginning order high number to low number
	// 3) add quakespasm.pak to the beginning
	ns.Bind("/", vfs.OS(dir), "/", vfs.BindBefore)
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

func GetFile(name string) (QFile, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return gameNS.Open(filepath.Join("/", name))
}

func GetFileContents(name string) ([]byte, error) {
	file, err := GetFile(name)
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
