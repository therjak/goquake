package filesystem

//TODO(therjak): the pack files are never closed and ns is never cleaned. There should be an option.

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/therjak/goquake/pack"
	"golang.org/x/tools/godoc/vfs"
)

var (
	ns = vfs.NewNameSpace()
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

func AddGameDir(dir string) {
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
	return ns.Stat(path)
}

func GetFile(name string) (QFile, error) {
	return ns.Open(filepath.Join("/", name))
}

func GetFileContents(name string) ([]byte, error) {
	file, err := GetFile(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return ioutil.ReadAll(file)
}
