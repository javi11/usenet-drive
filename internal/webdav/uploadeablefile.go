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
	innerFile    *os.File
	fsMutex      sync.RWMutex
	log          *slog.Logger
	nzbLoader    *usenet.NzbLoader
	finalSize    int64
	fileUploader usenet.FileUploader
	onClose      func()
}

func OpenUploadeableFile(
	name string,
	flag int,
	perm fs.FileMode,
	finalSize int64,
	nzbLoader *usenet.NzbLoader,
	uploader usenet.Uploader,
	onClose func(),
	log *slog.Logger,
) (*uploadeableFile, error) {
	fileName := utils.ReplaceFileExtension(name, ".nzb.tmp")
	f, err := os.OpenFile(fileName, flag, perm)
	if err != nil {
		return nil, err
	}

	fileUploader, err := uploader.NewFileUploader(name, finalSize)
	if err != nil {
		return nil, err
	}

	return &uploadeableFile{
		innerFile:    f,
		log:          log,
		nzbLoader:    nzbLoader,
		finalSize:    finalSize,
		fileUploader: fileUploader,
		onClose:      onClose,
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
	nzb := f.fileUploader.Build()
	f.fileUploader.Close()

	err := nzb.WriteIntoFile(f.innerFile)
	if err != nil {
		return err
	}

	if f.onClose != nil {
		f.onClose()
	}

	err = f.innerFile.Close()
	if err != nil {
		return err
	}

	fileName := f.innerFile.Name()
	newFilePath := fileName[:len(fileName)-len(".tmp")]
	err = os.Rename(f.innerFile.Name(), newFilePath)
	if err != nil {
		return err
	}

	_, err = f.nzbLoader.RefreshCachedNzb(newFilePath, nzb)
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
	metadata := f.fileUploader.GetMetadata()
	return NewUploadableFileInfo(metadata, f.innerFile.Name())
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

	n, err := f.fileUploader.Write(b)
	if err != nil {
		return n, err
	}
	return n, nil
}

func (f *uploadeableFile) WriteAt(b []byte, off int64) (int, error) {
	return 0, os.ErrPermission
}

func (f *uploadeableFile) WriteString(s string) (int, error) {
	return 0, os.ErrPermission
}
