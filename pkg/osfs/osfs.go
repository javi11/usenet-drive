package osfs

//go:generate mockgen -source=./osfs.go -destination=./osfs_mock.go -package=osfs FileSystem,File,FileInfo

import (
	"io"
	"io/fs"
	"os"
	"syscall"
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
	Name() string
	Readdir(n int) ([]os.FileInfo, error)
	ReadDir(n int) ([]fs.DirEntry, error)
	Readdirnames(n int) ([]string, error)
	SyscallConn() (syscall.RawConn, error)
	Chdir() error
	Chmod(mode os.FileMode) error
	Chown(uid, gid int) error
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
}

type FileInfo interface {
	fs.FileInfo
}

type FileSystem interface {
	Lstat(name string) (os.FileInfo, error)
	OpenFile(name string, flag int, perm os.FileMode) (File, error)
	ReadDir(name string) ([]os.DirEntry, error)
	Readlink(name string) (string, error)
	Remove(name string) error
	RemoveAll(path string) error
	Mkdir(name string, perm os.FileMode) error
	Rename(oldName, newName string) error
	Stat(name string) (fs.FileInfo, error)
	Open(name string) (File, error)
	IsNotExist(err error) bool
	WriteFile(filename string, data []byte, perm os.FileMode) error
}

type osFS struct{}

func New() FileSystem {
	return &osFS{}
}

func (*osFS) Lstat(n string) (fs.FileInfo, error)     { return os.Lstat(n) }
func (*osFS) ReadDir(n string) ([]os.DirEntry, error) { return os.ReadDir(n) }
func (*osFS) Readlink(n string) (string, error)       { return os.Readlink(n) }
func (*osFS) OpenFile(n string, f int, p os.FileMode) (File, error) {
	return os.OpenFile(n, f, p)
}
func (*osFS) RemoveAll(path string) error               { return os.RemoveAll(path) }
func (*osFS) Remove(path string) error                  { return os.Remove(path) }
func (*osFS) Mkdir(path string, perm os.FileMode) error { return os.Mkdir(path, perm) }
func (*osFS) Rename(oldName, newName string) error      { return os.Rename(oldName, newName) }
func (*osFS) Stat(name string) (fs.FileInfo, error)     { return os.Stat(name) }
func (*osFS) Open(name string) (File, error)            { return os.Open(name) }
func (*osFS) IsNotExist(err error) bool                 { return os.IsNotExist(err) }
func (*osFS) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}
