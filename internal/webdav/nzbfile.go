package webdav

import (
	"os"
	"sync"
	"time"

	"github.com/javi11/usenet-drive/internal/domain"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/utils"
)

type NzbFile struct {
	name   string
	size   int64
	buffer Buffer
	*os.File
	mutex *sync.RWMutex
}

func OpenNzbFile(name string, flag int, perm os.FileMode, cp usenet.UsenetConnectionPool, rwMutex *sync.RWMutex) (*NzbFile, error) {
	var err error

	file, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}

	nzbFile, err := parseNzbFile(file)
	if err != nil {
		return nil, err
	}

	metadata, err := domain.LoadFromNzb(nzbFile)
	if err != nil {
		return nil, err
	}

	return &NzbFile{
		File:   file,
		mutex:  rwMutex,
		buffer: NewBuffer(nzbFile.Files[0], int(metadata.FileSize), int(metadata.ChunkSize), cp),
		size:   metadata.FileSize,
		name:   utils.ReplaceFileExtension(name, metadata.FileExtension),
	}, nil
}

func (f *NzbFile) Chdir() error {
	return f.File.Chdir()
}

func (f *NzbFile) Chmod(mode os.FileMode) error {
	return f.File.Chmod(mode)
}

func (f *NzbFile) Chown(uid, gid int) error {
	return f.File.Chown(uid, gid)
}

func (f *NzbFile) Close() error {
	return f.File.Close()
}

func (f *NzbFile) Fd() uintptr {
	return f.File.Fd()
}

func (f *NzbFile) Name() string {
	return f.name
}

func (f *NzbFile) Read(b []byte) (n int, err error) {
	f.mutex.RLock()
	n, err = f.buffer.Read(b)
	f.mutex.RUnlock()
	return
}

func (f *NzbFile) ReadAt(b []byte, off int64) (n int, err error) {
	f.mutex.RLock()
	n, err = f.buffer.ReadAt(b, off)
	f.mutex.RUnlock()
	return
}

func (f *NzbFile) Readdir(n int) ([]os.FileInfo, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	infos, err := f.File.Readdir(n)
	if err != nil {
		return nil, err
	}

	for i, info := range infos {
		if isNzbFile(info.Name()) {
			infos[i], _ = NewFileInfoWithMetadata(info.Name())
		}
	}

	return infos, nil
}

func (f *NzbFile) Readdirnames(n int) ([]string, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.File.Readdirnames(n)
}

func (f *NzbFile) Seek(offset int64, whence int) (n int64, err error) {
	f.mutex.RLock()
	n, err = f.buffer.Seek(offset, whence)
	f.mutex.RUnlock()
	return
}

func (f *NzbFile) SetDeadline(t time.Time) error {
	return f.File.SetDeadline(t)
}

func (f *NzbFile) SetReadDeadline(t time.Time) error {
	return f.File.SetReadDeadline(t)
}

func (f *NzbFile) SetWriteDeadline(t time.Time) error {
	return f.File.SetWriteDeadline(t)
}

func (f *NzbFile) Stat() (os.FileInfo, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	if isNzbFile(f.File.Name()) {
		return NewFileInfoWithMetadata(f.File.Name())
	}

	return f.File.Stat()
}

func (f *NzbFile) Sync() error {
	return f.File.Sync()
}

func (f *NzbFile) Truncate(size int64) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return f.File.Truncate(size)
}

func (f *NzbFile) Write(b []byte) (int, error) {
	return f.File.Write(b)
}

func (f *NzbFile) WriteAt(b []byte, off int64) (int, error) {
	return f.File.WriteAt(b, off)
}

func (f *NzbFile) WriteString(s string) (int, error) {
	return f.File.WriteString(s)
}
