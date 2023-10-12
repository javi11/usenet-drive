package filereader

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSystemFileMethods(t *testing.T) {
	ctx := context.Background()
	cp := new(mockConnectionPool)
	log := slog.New()
	nzbLoader := new(mockNzbLoader)
	cNzb := new(mockCorruptedNzbsManager)
	fs := new(mockFileSystem)

	f := &file{
		name:      "test.nzb",
		buffer:    new(mockBuffer),
		innerFile: new(mockFile),
		fsMutex:   sync.RWMutex{},
		log:       log,
		metadata:  usenet.Metadata{},
		nzbLoader: nzbLoader,
		onClose:   func() error { return nil },
		cNzb:      cNzb,
		fs:        fs,
	}

	t.Run("Chdir", func(t *testing.T) {
		err := f.Chdir()
		assert.NoError(t, err)
	})

	t.Run("Chmod", func(t *testing.T) {
		mode := os.FileMode(0644)
		err := f.Chmod(mode)
		assert.NoError(t, err)
	})

	t.Run("Chown", func(t *testing.T) {
		uid, gid := 1000, 1000
		err := f.Chown(uid, gid)
		assert.NoError(t, err)
	})

	t.Run("Close", func(t *testing.T) {
		buffer := new(mockBuffer)
		onClose := func() error { return nil }
		innerFile := new(mockFile)
		innerFile.On("Close").Return(nil)
		f := &file{
			buffer:    buffer,
			onClose:   onClose,
			innerFile: innerFile,
		}

		buffer.On("Close").Return(nil)

		err := f.Close()
		assert.NoError(t, err)

		buffer.AssertCalled(t, "Close")
		innerFile.AssertCalled(t, "Close")
	})

	t.Run("Fd", func(t *testing.T) {
		fd := uintptr(123)
		innerFile := new(mockFile)
		innerFile.On("Fd").Return(fd)
		f := &file{
			innerFile: innerFile,
		}

		assert.Equal(t, fd, f.Fd())

		innerFile.AssertCalled(t, "Fd")
	})

	t.Run("Name", func(t *testing.T) {
		name := "test.nzb"
		f := &file{
			name: name,
		}

		assert.Equal(t, name, f.Name())
	})

	t.Run("Read", func(t *testing.T) {
		b := []byte("test")
		n := len(b)
		buffer := new(mockBuffer)
		buffer.On("Read", b).Return(n, nil)
		f := &file{
			buffer:  buffer,
			fsMutex: sync.RWMutex{},
			log:     log,
			name:    "test.nzb",
			cNzb:    cNzb,
		}

		n2, err := f.Read(b)
		assert.NoError(t, err)
		assert.Equal(t, n, n2)

		buffer.AssertCalled(t, "Read", b)
	})

	t.Run("ReadAt", func(t *testing.T) {
		b := []byte("test")
		off := int64(0)
		n := len(b)
		buffer := new(mockBuffer)
		buffer.On("ReadAt", b, off).Return(n, nil)
		f := &file{
			buffer:  buffer,
			fsMutex: sync.RWMutex{},
			log:     log,
			name:    "test.nzb",
			cNzb:    cNzb,
		}

		n2, err := f.ReadAt(b, off)
		assert.NoError(t, err)
		assert.Equal(t, n, n2)

		buffer.AssertCalled(t, "ReadAt", b, off)
	})

	t.Run("Readdir", func(t *testing.T) {
		dirname := "test"
		infos := []os.FileInfo{
			new(mockFileInfo),
			new(mockFileInfo),
		}
		innerFile := new(mockFile)
		innerFile.On("Readdir", 0).Return(infos, nil)
		fs := new(mockFileSystem)
		fs.On("Stat", mock.Anything).Return(new(mockFileInfo), nil)
		f := &file{
			innerFile: innerFile,
			fs:        fs,
		}

		infos2, err := f.Readdir(0)
		assert.NoError(t, err)
		assert.Equal(t, infos, infos2)

		innerFile.AssertCalled(t, "Readdir", 0)
	})

	t.Run("Readdirnames", func(t *testing.T) {
		dirname := "test"
		names := []string{"file1", "file2"}
		innerFile := new(mockFile)
		innerFile.On("Readdirnames", 0).Return(names, nil)
		f := &file{
			innerFile: innerFile,
		}

		names2, err := f.Readdirnames(0)
		assert.NoError(t, err)
		assert.Equal(t, names, names2)

		innerFile.AssertCalled(t, "Readdirnames", 0)
	})

	t.Run("Seek", func(t *testing.T) {
		offset := int64(0)
		whence := io.SeekStart
		n := int64(123)
		buffer := new(mockBuffer)
		buffer.On("Seek", offset, whence).Return(n, nil)
		f := &file{
			buffer:  buffer,
			fsMutex: sync.RWMutex{},
		}

		n2, err := f.Seek(offset, whence)
		assert.NoError(t, err)
		assert.Equal(t, n, n2)

		buffer.AssertCalled(t, "Seek", offset, whence)
	})

	t.Run("SetDeadline", func(t *testing.T) {
		tm := time.Now()
		innerFile := new(mockFile)
		innerFile.On("SetDeadline", tm).Return(nil)
		f := &file{
			innerFile: innerFile,
		}

		err := f.SetDeadline(tm)
		assert.NoError(t, err)

		innerFile.AssertCalled(t, "SetDeadline", tm)
	})

	t.Run("SetReadDeadline", func(t *testing.T) {
		tm := time.Now()
		innerFile := new(mockFile)
		innerFile.On("SetReadDeadline", tm).Return(nil)
		f := &file{
			innerFile: innerFile,
		}

		err := f.SetReadDeadline(tm)
		assert.NoError(t, err)

		innerFile.AssertCalled(t, "SetReadDeadline", tm)
	})

	t.Run("SetWriteDeadline", func(t *testing.T) {
		f := &file{}

		err := f.SetWriteDeadline(time.Now())
		assert.Equal(t, os.ErrPermission, err)
	})

	t.Run("Stat", func(t *testing.T) {
		name := "test.nzb"
		metadata := usenet.Metadata{
			FileExtension: ".nzb",
			FileSize:      123,
			ChunkSize:     456,
		}
		fs := new(mockFileSystem)
		fs.On("Stat", name).Return(new(mockFileInfo), nil)
		innerFile := new(mockFile)
		innerFile.On("Name").Return(name)
		nzbLoader := new(mockNzbLoader)
		nzbLoader.On("LoadFromFileReader", mock.Anything).Return(&nzbloader.Nzb{}, nil)
		f := &file{
			name:      name,
			metadata:  metadata,
			fs:        fs,
			innerFile: innerFile,
			nzbLoader: nzbLoader,
		}

		info, err := f.Stat()
		assert.NoError(t, err)
		assert.NotNil(t, info)

		fs.AssertCalled(t, "Stat", name)
		innerFile.AssertCalled(t, "Name")
		nzbLoader.AssertCalled(t, "LoadFromFileReader", mock.Anything)
	})

	t.Run("Sync", func(t *testing.T) {
		innerFile := new(mockFile)
		innerFile.On("Sync").Return(nil)
		f := &file{
			innerFile: innerFile,
		}

		err := f.Sync()
		assert.NoError(t, err)

		innerFile.AssertCalled(t, "Sync")
	})

	t.Run("Truncate", func(t *testing.T) {
		f := &file{}

		err := f.Truncate(123)
		assert.Equal(t, os.ErrPermission, err)
	})

	t.Run("Write", func(t *testing.T) {
		f := &file{}

		n, err := f.Write([]byte("test"))
		assert.Equal(t, 0, n)
		assert.Equal(t, os.ErrPermission, err)
	})

	t.Run("WriteAt", func(t *testing.T) {
		f := &file{}

		n, err := f.WriteAt([]byte("test"), 0)
		assert.Equal(t, 0, n)
		assert.Equal(t, os.ErrPermission, err)
	})

	t.Run("WriteString", func(t *testing.T) {
		f := &file{}

		n, err := f.WriteString("test")
		assert.Equal(t, 0, n)
		assert.Equal(t, os.ErrPermission, err)
	})
}
