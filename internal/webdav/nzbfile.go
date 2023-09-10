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
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/utils"
)

type nzbFile struct {
	name      string
	buffer    Buffer
	innerFile *os.File
	fsMutex   sync.RWMutex
	log       *slog.Logger
	metadata  usenet.Metadata
	nzbLoader *usenet.NzbLoader
}

func OpenNzbFile(
	ctx context.Context,
	name string,
	flag int,
	perm os.FileMode,
	cp usenet.UsenetConnectionPool,
	log *slog.Logger,
	nzbLoader *usenet.NzbLoader,
) (*nzbFile, error) {
	var err error
	file, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}

	n, err := nzbLoader.LoadFromFileReader(file)
	if err != nil {
		log.ErrorContext(ctx, fmt.Sprintf("Error getting loading nzb %s", name), "err", err)
		return nil, os.ErrNotExist
	}

	buffer, err := NewBuffer(n.Nzb.Files[0], int(n.Metadata.FileSize), int(n.Metadata.ChunkSize), cp)
	if err != nil {
		return nil, err
	}

	return &nzbFile{
		innerFile: file,
		buffer:    buffer,
		metadata:  n.Metadata,
		name:      utils.ReplaceFileExtension(name, n.Metadata.FileExtension),
		log:       log,
		nzbLoader: nzbLoader,
	}, nil
}

func (f *nzbFile) Chdir() error {
	return f.innerFile.Chdir()
}

func (f *nzbFile) Chmod(mode os.FileMode) error {
	return f.innerFile.Chmod(mode)
}

func (f *nzbFile) Chown(uid, gid int) error {
	return f.innerFile.Chown(uid, gid)
}

func (f *nzbFile) Close() error {
	if err := f.buffer.Close(); err != nil {
		return err
	}

	return f.innerFile.Close()
}

func (f *nzbFile) Fd() uintptr {
	return f.innerFile.Fd()
}

func (f *nzbFile) Name() string {
	return f.name
}

func (f *nzbFile) Read(b []byte) (n int, err error) {
	f.fsMutex.RLock()
	n, err = f.buffer.Read(b)
	f.fsMutex.RUnlock()
	return
}

func (f *nzbFile) ReadAt(b []byte, off int64) (int, error) {
	f.fsMutex.RLock()
	defer f.fsMutex.RUnlock()

	return f.buffer.ReadAt(b, off)
}

func (f *nzbFile) Readdir(n int) ([]os.FileInfo, error) {
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
				n := filepath.Join(f.innerFile.Name(), info.Name())
				infos[i], err = NewNZBFileInfo(
					n,
					n,
					f.log,
					f.nzbLoader,
				)
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

func (f *nzbFile) Readdirnames(n int) ([]string, error) {
	f.fsMutex.RLock()
	defer f.fsMutex.RUnlock()
	return f.innerFile.Readdirnames(n)
}

func (f *nzbFile) Seek(offset int64, whence int) (n int64, err error) {
	f.fsMutex.RLock()
	n, err = f.buffer.Seek(offset, whence)
	f.fsMutex.RUnlock()
	return
}

func (f *nzbFile) SetDeadline(t time.Time) error {
	return f.innerFile.SetDeadline(t)
}

func (f *nzbFile) SetReadDeadline(t time.Time) error {
	return f.innerFile.SetReadDeadline(t)
}

func (f *nzbFile) SetWriteDeadline(t time.Time) error {
	return os.ErrPermission
}

func (f *nzbFile) Stat() (os.FileInfo, error) {
	f.fsMutex.RLock()
	defer f.fsMutex.RUnlock()

	return NewNZBFileInfoWithMetadata(f.metadata, f.innerFile.Name())
}

func (f *nzbFile) Sync() error {
	return f.innerFile.Sync()
}

func (f *nzbFile) Truncate(size int64) error {
	return os.ErrPermission
}

func (f *nzbFile) Write(b []byte) (int, error) {
	return 0, os.ErrPermission
}

func (f *nzbFile) WriteAt(b []byte, off int64) (int, error) {
	return 0, os.ErrPermission
}

func (f *nzbFile) WriteString(s string) (int, error) {
	return 0, os.ErrPermission
}
