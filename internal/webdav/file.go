package webdav

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
)

type file struct {
	innerFile  *os.File
	fsMutex    sync.RWMutex
	fileReader RemoteFileReader
	onClose    func() error
	log        *slog.Logger
}

func OpenFile(
	name string,
	flag int,
	perm fs.FileMode,
	onClose func() error,
	log *slog.Logger,
	fileReader RemoteFileReader,
) (*file, error) {
	f, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}

	return &file{
		innerFile:  f,
		onClose:    onClose,
		log:        log,
		fileReader: fileReader,
	}, nil
}

func (f *file) Chdir() error {
	return f.innerFile.Chdir()
}

func (f *file) Chmod(mode os.FileMode) error {
	return f.innerFile.Chmod(mode)
}

func (f *file) Chown(uid, gid int) error {
	return f.innerFile.Chown(uid, gid)
}

func (f *file) Close() error {
	err := f.innerFile.Close()
	if err != nil {
		return err
	}

	if f.onClose != nil {
		err := f.onClose()
		if err != nil {
			return err
		}
	}

	return err
}

func (f *file) Fd() uintptr {
	return f.innerFile.Fd()
}

func (f *file) Name() string {
	return f.innerFile.Name()
}

func (f *file) Read(b []byte) (int, error) {
	return f.innerFile.Read(b)
}

func (f *file) ReadAt(b []byte, off int64) (int, error) {
	return f.innerFile.ReadAt(b, off)
}

func (f *file) Readdir(n int) ([]os.FileInfo, error) {
	f.fsMutex.RLock()
	defer f.fsMutex.RUnlock()
	infos, err := f.innerFile.Readdir(n)
	if err != nil {
		return nil, err
	}

	var merr multierror.Group

	for i, info := range infos {
		if info.IsDir() {
			continue
		}

		name := info.Name()
		i := i
		merr.Go(func() error {
			if info == nil {
				return nil
			}
			pathJoin := filepath.Join(f.innerFile.Name(), name)
			ok, s, err := f.fileReader.Stat(pathJoin)
			if err != nil {
				return err
			}

			if ok {
				infos[i] = s
			}

			return nil
		})
	}

	if err := merr.Wait(); err != nil {
		f.log.Error("error reading remote directory", "error", err)

		// Remove nulls from infos
		var filteredInfos []os.FileInfo
		for _, info := range infos {
			if info != nil {
				filteredInfos = append(filteredInfos, info)
			}
		}

		return filteredInfos, nil
	}

	return infos, nil
}

func (f *file) Readdirnames(n int) ([]string, error) {
	return f.innerFile.Readdirnames(n)
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	return f.innerFile.Seek(offset, whence)
}

func (f *file) SetDeadline(t time.Time) error {
	return f.innerFile.SetDeadline(t)
}

func (f *file) SetReadDeadline(t time.Time) error {
	return f.innerFile.SetReadDeadline(t)
}

func (f *file) SetWriteDeadline(t time.Time) error {
	return f.innerFile.SetWriteDeadline(t)
}

func (f *file) Stat() (os.FileInfo, error) {
	return f.innerFile.Stat()
}

func (f *file) Sync() error {
	return f.innerFile.Sync()
}

func (f *file) Truncate(size int64) error {
	return f.innerFile.Truncate(size)
}

func (f *file) Write(b []byte) (int, error) {
	return f.innerFile.Write(b)
}

func (f *file) WriteAt(b []byte, off int64) (int, error) {
	return f.innerFile.WriteAt(b, off)
}

func (f *file) WriteString(s string) (int, error) {
	return f.innerFile.WriteString(s)
}
