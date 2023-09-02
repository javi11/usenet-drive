package webdav

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/javi11/usenet-drive/internal/domain"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/utils"
)

type NzbFile struct {
	name      string
	size      int64
	buffer    Buffer
	innerFile *os.File
	fsMutex   *sync.RWMutex
	log       *slog.Logger
}

func OpenNzbFile(
	ctx context.Context,
	name string,
	flag int,
	perm os.FileMode,
	cp usenet.UsenetConnectionPool,
	fsMutex *sync.RWMutex,
	log *slog.Logger,
) (*NzbFile, error) {
	var err error

	file, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}

	nzbFile, err := parseNzbFile(file)
	if err != nil {
		log.ErrorContext(ctx, fmt.Sprintf("Error parsing nzb file %s", name), "err", err)
		return nil, os.ErrNotExist
	}

	metadata, err := domain.LoadFromNzb(nzbFile)
	if err != nil {
		log.ErrorContext(ctx, fmt.Sprintf("Error getting metadata from file %s", name), "err", err)
		return nil, os.ErrNotExist
	}
	return &NzbFile{
		innerFile: file,
		fsMutex:   fsMutex,
		buffer:    NewBuffer(nzbFile.Files[0], int(metadata.FileSize), int(metadata.ChunkSize), cp),
		size:      metadata.FileSize,
		name:      utils.ReplaceFileExtension(name, metadata.FileExtension),
		log:       log,
	}, nil
}

func (f *NzbFile) Chdir() error {
	return f.innerFile.Chdir()
}

func (f *NzbFile) Chmod(mode os.FileMode) error {
	return f.innerFile.Chmod(mode)
}

func (f *NzbFile) Chown(uid, gid int) error {
	return f.innerFile.Chown(uid, gid)
}

func (f *NzbFile) Close() error {
	return f.innerFile.Close()
}

func (f *NzbFile) Fd() uintptr {
	return f.innerFile.Fd()
}

func (f *NzbFile) Name() string {
	return f.name
}

func (f *NzbFile) Read(b []byte) (n int, err error) {
	f.fsMutex.RLock()
	n, err = f.buffer.Read(b)
	f.fsMutex.RUnlock()
	return
}

func (f *NzbFile) ReadAt(b []byte, off int64) (n int, err error) {
	f.fsMutex.RLock()
	n, err = f.buffer.ReadAt(b, off)
	f.fsMutex.RUnlock()
	return
}

func (f *NzbFile) Readdir(n int) ([]os.FileInfo, error) {
	f.fsMutex.RLock()
	defer f.fsMutex.RUnlock()
	infos, err := f.innerFile.Readdir(n)
	if err != nil {
		return nil, err
	}

	var merr multierror.Group

	for i, info := range infos {
		if isNzbFile(info.Name()) {
			info := info
			i := i
			merr.Go(func() error {
				infos[i], err = NewFileInfoWithMetadata(filepath.Join(f.innerFile.Name(), info.Name()), f.log)
				if err != nil {
					return err
				}

				return nil
			})
		}
	}

	if err := merr.Wait(); err != nil {
		return removeNzb(infos), nil
	}

	return infos, nil
}

func (f *NzbFile) Readdirnames(n int) ([]string, error) {
	f.fsMutex.RLock()
	defer f.fsMutex.RUnlock()
	return f.innerFile.Readdirnames(n)
}

func (f *NzbFile) Seek(offset int64, whence int) (n int64, err error) {
	f.fsMutex.RLock()
	n, err = f.buffer.Seek(offset, whence)
	f.fsMutex.RUnlock()
	return
}

func (f *NzbFile) SetDeadline(t time.Time) error {
	return f.innerFile.SetDeadline(t)
}

func (f *NzbFile) SetReadDeadline(t time.Time) error {
	return f.innerFile.SetReadDeadline(t)
}

func (f *NzbFile) SetWriteDeadline(t time.Time) error {
	return os.ErrPermission
}

func (f *NzbFile) Stat() (os.FileInfo, error) {
	f.fsMutex.RLock()
	defer f.fsMutex.RUnlock()

	return NewFileInfoWithMetadata(f.innerFile.Name(), f.log)
}

func (f *NzbFile) Sync() error {
	return f.innerFile.Sync()
}

func (f *NzbFile) Truncate(size int64) error {
	return os.ErrPermission
}

func (f *NzbFile) Write(b []byte) (int, error) {
	return 0, os.ErrPermission
}

func (f *NzbFile) WriteAt(b []byte, off int64) (int, error) {
	return 0, os.ErrPermission
}

func (f *NzbFile) WriteString(s string) (int, error) {
	return 0, os.ErrPermission
}
