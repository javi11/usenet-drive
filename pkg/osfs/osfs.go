package osfs

//go:generate mockgen -source=./osfs.go -destination=./osfs_mock.go -package=osfs FileSystem,File,FileInfo

import (
	"io"
	"os"
	"time"
)

type File interface {
	Fd() uintptr
	io.Closer
	io.ReaderAt
	io.Seeker
	io.WriterAt
	io.ReadCloser
	Stat() (os.FileInfo, error)
	Sync() error
}

type FileInfo interface {
	Name() string
	Size() int64
	Mode() os.FileMode
	ModTime() time.Time
	IsDir() bool
}

type FileSystem interface {
	Lstat(name string) (FileInfo, error)
	OpenFile(name string, flag int, perm os.FileMode) (File, error)
	ReadDir(name string) ([]os.DirEntry, error)
	Readlink(name string) (string, error)
	Remove(name string) error
	Mkdir(name string, perm os.FileMode) error
	Rename(oldName, newName string) error
	Stat(name string) (FileInfo, error)
	Open(name string) (File, error)
}

type osFS struct{}

func New() FileSystem {
	return &osFS{}
}

func (*osFS) Lstat(n string) (FileInfo, error)        { return os.Lstat(n) }
func (*osFS) ReadDir(n string) ([]os.DirEntry, error) { return os.ReadDir(n) }
func (*osFS) Readlink(n string) (string, error)       { return os.Readlink(n) }
func (*osFS) OpenFile(n string, f int, p os.FileMode) (File, error) {
	return os.OpenFile(n, f, p)
}
func (*osFS) RemoveAll(path string) error               { return os.RemoveAll(path) }
func (*osFS) Remove(path string) error                  { return os.Remove(path) }
func (*osFS) Mkdir(path string, perm os.FileMode) error { return os.Mkdir(path, perm) }
func (*osFS) Rename(oldName, newName string) error      { return os.Rename(oldName, newName) }
func (*osFS) Stat(name string) (FileInfo, error)        { return os.Stat(name) }
func (*osFS) Open(name string) (File, error)            { return os.Open(name) }
