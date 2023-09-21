package webdav

import (
	"io/fs"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/utils"
)

type uploadeableFile struct {
	buffer     *uploadBuffer
	innerFile  *os.File
	fsMutex    sync.RWMutex
	log        *slog.Logger
	nzbLoader  *usenet.NzbLoader
	finalSize  int64
	nzbBuilder *usenet.NzbBuilder
}

func OpenUploadeableFile(
	name string,
	flag int,
	perm fs.FileMode,
	finalSize int64,
	uploader usenet.Uploader,
	log *slog.Logger,
	nzbLoader *usenet.NzbLoader,
) (*uploadeableFile, error) {
	fileName := utils.ReplaceFileExtension(name, ".nzb")
	f, err := os.OpenFile(fileName, flag, perm)
	if err != nil {
		return nil, err
	}

	nzbBuilder := uploader.NewNzbBuilder(name, finalSize)

	return &uploadeableFile{
		innerFile:  f,
		log:        log,
		nzbLoader:  nzbLoader,
		finalSize:  finalSize,
		nzbBuilder: nzbBuilder,
	}, nil
}

func (f *uploadeableFile) Chdir() error {
	return f.innerFile.Chdir()
}

func (f *uploadeableFile) Chmod(mode os.FileMode) error {
	return f.innerFile.Chmod(mode)
}

func (f *uploadeableFile) Chown(uid, gid int) error {
	return f.innerFile.Chown(uid, gid)
}

func (f *uploadeableFile) Close() error {
	err := f.innerFile.Close()
	if err != nil {
		return err
	}

	return err
}

func (f *uploadeableFile) Fd() uintptr {
	return f.innerFile.Fd()
}

func (f *uploadeableFile) Name() string {
	return f.innerFile.Name()
}

func (f *uploadeableFile) Read(b []byte) (int, error) {
	return 0, os.ErrPermission
}

func (f *uploadeableFile) ReadAt(b []byte, off int64) (int, error) {
	return 0, os.ErrPermission
}

func (f *uploadeableFile) Readdir(n int) ([]os.FileInfo, error) {
	return []os.FileInfo{}, os.ErrPermission
}

func (f *uploadeableFile) Readdirnames(n int) ([]string, error) {
	return []string{}, os.ErrPermission
}

func (f *uploadeableFile) Seek(offset int64, whence int) (int64, error) {
	return 0, os.ErrPermission
}

func (f *uploadeableFile) SetDeadline(t time.Time) error {
	return os.ErrPermission
}

func (f *uploadeableFile) SetReadDeadline(t time.Time) error {
	return os.ErrPermission
}

func (f *uploadeableFile) SetWriteDeadline(t time.Time) error {
	return os.ErrPermission
}

func (f *uploadeableFile) Stat() (os.FileInfo, error) {
	return nil, os.ErrPermission
}

func (f *uploadeableFile) Sync() error {
	return os.ErrPermission
}

func (f *uploadeableFile) Truncate(size int64) error {
	return os.ErrPermission
}

func (f *uploadeableFile) Write(b []byte) (int, error) {
	f.fsMutex.Lock()
	defer f.fsMutex.Unlock()

	if f.buffer == nil {
		f.buffer = NewUploadBuffer(f.segmentSize)
	}

	n, err := f.buffer.Write(b)
	if err != nil {
		return n, err
	}
	if f.buffer.Len() == int(f.segmentSize) {
		f.upload(f.buffer)

		if len(b) > int(f.segmentSize) {
			nb, err := f.Write(b[n-len(b):])
			if err != nil {
				return nb, err
			}

			return n + nb, nil
		}

		return n, nil
	}

	return n, nil
}

func (f *uploadeableFile) WriteAt(b []byte, off int64) (int, error) {
	return 0, os.ErrPermission
}

func (f *uploadeableFile) WriteString(s string) (int, error) {
	return 0, os.ErrPermission
}

func (f *uploadeableFile) upload(buffer *uploadBuffer) error {

}
