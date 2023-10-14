package filereader

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/javi11/usenet-drive/internal/test"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"github.com/stretchr/testify/assert"
)

func TestOpenFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()
	mockNzbLoader := nzbloader.NewMockNzbLoader(ctrl)
	mockCNzb := corruptednzbsmanager.NewMockCorruptedNzbsManager(ctrl)
	fs := osfs.NewMockFileSystem(ctrl)
	cp := connectionpool.NewMockUsenetConnectionPool(ctrl)

	t.Run("Not nzb file", func(t *testing.T) {
		name := "test.txt"
		flag := os.O_RDONLY
		perm := os.FileMode(0644)
		onClose := func() error { return nil }

		fs.EXPECT().Stat("test.nzb").Return(nil, os.ErrNotExist).Times(1)
		fs.EXPECT().IsNotExist(os.ErrNotExist).Return(true).Times(1)

		_, _, err := openFile(context.Background(), name, flag, perm, cp, log, onClose, mockNzbLoader, mockCNzb, fs)
		assert.NoError(t, err)
	})

	t.Run("Is a Nzb file but do not exists", func(t *testing.T) {
		name := "test.nzb"
		flag := os.O_RDONLY
		perm := os.FileMode(0644)
		onClose := func() error { return nil }

		fs.EXPECT().OpenFile(name, flag, perm).Return(nil, os.ErrNotExist).Times(1)

		_, _, err := openFile(context.Background(), name, flag, perm, cp, log, onClose, mockNzbLoader, mockCNzb, fs)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("Is a Nzb file", func(t *testing.T) {
		name := "test.nzb"
		flag := os.O_RDONLY
		perm := os.FileMode(0644)
		onClose := func() error { return nil }
		mockFile := osfs.NewMockFile(ctrl)
		today := time.Now()

		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)

		fs.EXPECT().OpenFile(name, flag, perm).Return(mockFile, nil).Times(1)
		mockNzbLoader.EXPECT().LoadFromFileReader(mockFile).Return(&nzbloader.NzbCache{
			Metadata: usenet.Metadata{
				FileExtension: ".mkv",
				FileSize:      123,
				ChunkSize:     456,
				FileName:      "file2.mkv",
				ModTime:       today,
			},
			Nzb: nzb,
		}, nil).Times(1)

		// Call
		ok, file, err := openFile(context.Background(), name, flag, perm, cp, log, onClose, mockNzbLoader, mockCNzb, fs)

		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "test.mkv", file.name)
	})

	t.Run("Is a Nzb file masked", func(t *testing.T) {
		name := "test.mkv"
		flag := os.O_RDONLY
		perm := os.FileMode(0644)
		onClose := func() error { return nil }
		mockFile := osfs.NewMockFile(ctrl)
		today := time.Now()

		fsStatMock := osfs.NewMockFileInfo(ctrl)
		fsStatMock.EXPECT().Name().Return("test.nzb").Times(1)

		fs.EXPECT().Stat("test.nzb").Return(fsStatMock, nil).Times(1)
		fs.EXPECT().IsNotExist(nil).Return(false).Times(1)

		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)

		fs.EXPECT().OpenFile("test.nzb", flag, perm).Return(mockFile, nil).Times(1)
		mockNzbLoader.EXPECT().LoadFromFileReader(mockFile).Return(&nzbloader.NzbCache{
			Metadata: usenet.Metadata{
				FileExtension: ".mkv",
				FileSize:      123,
				ChunkSize:     456,
				FileName:      "file2.mkv",
				ModTime:       today,
			},
			Nzb: nzb,
		}, nil).Times(1)

		// Call
		ok, file, err := openFile(context.Background(), name, flag, perm, cp, log, onClose, mockNzbLoader, mockCNzb, fs)

		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "test.mkv", file.name)
	})

	t.Run("Nzb file with corrupted metadata", func(t *testing.T) {
		name := "test.nzb"
		flag := os.O_RDONLY
		perm := os.FileMode(0644)
		onClose := func() error { return nil }
		mockFile := osfs.NewMockFile(ctrl)

		fs.EXPECT().OpenFile("test.nzb", flag, perm).Return(mockFile, nil).Times(1)
		mockNzbLoader.EXPECT().LoadFromFileReader(mockFile).Return(nil, ErrCorruptedNzb).Times(1)

		ok, _, err := openFile(context.Background(), name, flag, perm, cp, log, onClose, mockNzbLoader, mockCNzb, fs)
		assert.ErrorIs(t, err, os.ErrNotExist)
		// File exists but is corrupted
		assert.True(t, ok)
	})

	t.Run("Error opening the file", func(t *testing.T) {
		name := "test.nzb"
		flag := os.O_RDONLY
		perm := os.FileMode(0644)
		onClose := func() error { return nil }

		fs.EXPECT().OpenFile("test.nzb", flag, perm).Return(nil, os.ErrPermission).Times(1)

		ok, _, err := openFile(context.Background(), name, flag, perm, cp, log, onClose, mockNzbLoader, mockCNzb, fs)
		assert.ErrorIs(t, err, os.ErrPermission)
		// File should be an nzb at this point but we cannot open it
		assert.True(t, ok)
	})

}

func TestCloseFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()
	mockNzbLoader := nzbloader.NewMockNzbLoader(ctrl)
	mockCNzb := corruptednzbsmanager.NewMockCorruptedNzbsManager(ctrl)
	fs := osfs.NewMockFileSystem(ctrl)
	mockFile := osfs.NewMockFile(ctrl)
	mockBuffer := NewMockBuffer(ctrl)
	onClosedCalled := false
	f := &file{
		name:      "test.nzb",
		buffer:    mockBuffer,
		innerFile: mockFile,
		fsMutex:   sync.RWMutex{},
		log:       log,
		metadata:  usenet.Metadata{},
		nzbLoader: mockNzbLoader,
		onClose: func() error {
			onClosedCalled = true
			return nil
		},
		cNzb: mockCNzb,
		fs:   fs,
	}
	t.Run("Error", func(t *testing.T) {
		mockFile.EXPECT().Close().Return(os.ErrPermission)
		mockBuffer.EXPECT().Close().Return(nil)

		err := f.Close()
		assert.Equal(t, os.ErrPermission, err)

		assert.True(t, onClosedCalled)
	})

	t.Run("Success", func(t *testing.T) {
		mockFile.EXPECT().Close().Return(nil)
		mockBuffer.EXPECT().Close().Return(nil)

		err := f.Close()
		assert.NoError(t, err)

		assert.True(t, onClosedCalled)
	})

	t.Run("NoOnCloseFunction", func(t *testing.T) {
		f.onClose = nil
		mockFile.EXPECT().Close().Return(nil)
		mockBuffer.EXPECT().Close().Return(nil)

		err := f.Close()
		assert.NoError(t, err)
	})

}

func TestRead(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()
	mockNzbLoader := nzbloader.NewMockNzbLoader(ctrl)
	mockCNzb := corruptednzbsmanager.NewMockCorruptedNzbsManager(ctrl)
	fs := osfs.NewMockFileSystem(ctrl)
	mockFile := osfs.NewMockFile(ctrl)
	mockBuffer := NewMockBuffer(ctrl)

	t.Run("Read success", func(t *testing.T) {
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		b := []byte("test")
		n := len(b)

		mockBuffer.EXPECT().Read(b).Return(n, nil)

		n2, err := f.Read(b)
		assert.NoError(t, err)
		assert.Equal(t, n, n2)
	})

	t.Run("Mark file as corrupted on read error", func(t *testing.T) {
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		b := []byte("test")
		n := len(b)

		mockBuffer.EXPECT().Read(b).Return(n, ErrCorruptedNzb)
		mockCNzb.EXPECT().Add(context.Background(), "test.nzb", "corrupted nzb").Return(nil)

		n2, err := f.Read(b)
		assert.Equal(t, n, n2)
		assert.Equal(t, io.ErrUnexpectedEOF, err)
	})
}

func TestReadAt(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()
	mockNzbLoader := nzbloader.NewMockNzbLoader(ctrl)
	mockCNzb := corruptednzbsmanager.NewMockCorruptedNzbsManager(ctrl)
	fs := osfs.NewMockFileSystem(ctrl)
	mockFile := osfs.NewMockFile(ctrl)
	mockBuffer := NewMockBuffer(ctrl)

	t.Run("ReadAt success", func(t *testing.T) {
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		b := []byte("test")
		n := len(b)
		offset := int64(10)

		mockBuffer.EXPECT().ReadAt(b, offset).Return(n, nil)

		n2, err := f.ReadAt(b, offset)
		assert.NoError(t, err)
		assert.Equal(t, n, n2)
	})

	t.Run("Mark file as corrupted on read at error", func(t *testing.T) {
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		b := []byte("test")
		n := len(b)
		offset := int64(10)

		mockBuffer.EXPECT().ReadAt(b, offset).Return(n, ErrCorruptedNzb)
		mockCNzb.EXPECT().Add(context.Background(), "test.nzb", "corrupted nzb").Return(nil)

		n2, err := f.ReadAt(b, offset)
		assert.Equal(t, n, n2)
		assert.Equal(t, io.ErrUnexpectedEOF, err)
	})
}

func TestReadDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()
	mockNzbLoader := nzbloader.NewMockNzbLoader(ctrl)
	mockCNzb := corruptednzbsmanager.NewMockCorruptedNzbsManager(ctrl)
	fs := osfs.NewMockFileSystem(ctrl)
	mockFile := osfs.NewMockFile(ctrl)
	mockBuffer := NewMockBuffer(ctrl)

	t.Run("Readdir returns one txt one nzb", func(t *testing.T) {
		fileInfo1 := osfs.NewMockFileInfo(ctrl)
		fileInfo2 := osfs.NewMockFileInfo(ctrl)

		infos := []os.FileInfo{
			fileInfo1,
			fileInfo2,
		}

		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		mockFile.EXPECT().Readdir(0).Return(infos, nil).Times(1)
		mockFile.EXPECT().Name().Return("folder").Times(1)

		// Every directory file needs to be stat'ed this means stat for every infos
		fileInfo1.EXPECT().Name().Return("file1.txt").Times(1)
		fileInfo2.EXPECT().Name().Return("file2.nzb").Times(2)

		// Txt file is not a masked nzb file
		fs.EXPECT().Stat("file1.nzb").Return(nil, os.ErrNotExist).Times(1)
		fs.EXPECT().IsNotExist(os.ErrNotExist).Return(true).Times(1)

		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)

		today := time.Now()
		mockNzbLoader.EXPECT().LoadFromFile("folder/file2.nzb").Return(&nzbloader.NzbCache{
			Metadata: usenet.Metadata{
				FileExtension: ".mkv",
				FileSize:      123,
				ChunkSize:     456,
				FileName:      "file2.mkv",
				ModTime:       today,
			},
			Nzb: nzb,
		}, nil)

		result, err := f.Readdir(0)
		assert.NoError(t, err)
		// We should receive txt file as is
		assert.Equal(t, fileInfo1, result[0])
		// We should receive the nzb file with the real uploaded file extensions size and modtime
		assert.Equal(t, "file2.mkv", result[1].Name())
		assert.Equal(t, 123, int(result[1].Size()))
		assert.Equal(t, today, result[1].ModTime())
	})

	t.Run("There are one masked nzb file", func(t *testing.T) {
		fileInfo1 := osfs.NewMockFileInfo(ctrl)
		fileInfo2 := osfs.NewMockFileInfo(ctrl)

		infos := []os.FileInfo{
			fileInfo1,
			fileInfo2,
		}

		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		mockFile.EXPECT().Readdir(0).Return(infos, nil).Times(1)
		mockFile.EXPECT().Name().Return("folder").Times(1)

		// Every directory file needs to be stat'ed this means stat for every infos
		fileInfo1.EXPECT().Name().Return("file1.txt").Times(1)
		fileInfo2.EXPECT().Name().Return("file2-masked.mkv").Times(1)

		// Txt file is not a masked nzb file
		fs.EXPECT().Stat("file1.nzb").Return(nil, os.ErrNotExist).Times(1)
		fs.EXPECT().IsNotExist(os.ErrNotExist).Return(true).Times(1)

		// mkv file is a masked nzb file
		fileInfoMaskedFile := osfs.NewMockFileInfo(ctrl)
		fileInfoMaskedFile.EXPECT().Name().Return("file2-masked.nzb").Times(2)
		fs.EXPECT().Stat("file2-masked.nzb").Return(fileInfoMaskedFile, nil).Times(1)
		fs.EXPECT().IsNotExist(nil).Return(false).Times(1)

		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)
		today := time.Now()
		mockNzbLoader.EXPECT().LoadFromFile("folder/file2-masked.nzb").Return(&nzbloader.NzbCache{
			Metadata: usenet.Metadata{
				FileExtension: ".mkv",
				FileSize:      123,
				ChunkSize:     456,
				FileName:      "file2.mkv",
				ModTime:       today,
			},
			Nzb: nzb,
		}, nil)

		// Call
		result, err := f.Readdir(0)
		assert.NoError(t, err)

		// We should receive txt file as is
		assert.Equal(t, fileInfo1, result[0])
		// We should receive the nzb file with the real uploaded file extensions size and modtime
		assert.Equal(t, "file2-masked.mkv", result[1].Name())
		assert.Equal(t, 123, int(result[1].Size()))
		assert.Equal(t, today, result[1].ModTime())
	})

	t.Run("Readdir ignores corrupted metadata files", func(t *testing.T) {
		fileInfo1 := osfs.NewMockFileInfo(ctrl)
		fileInfo2 := osfs.NewMockFileInfo(ctrl)

		infos := []os.FileInfo{
			fileInfo1,
			fileInfo2,
		}

		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		mockFile.EXPECT().Readdir(0).Return(infos, nil).Times(1)
		mockFile.EXPECT().Name().Return("folder").Times(1)

		// Every directory file needs to be stat'ed this means stat for every infos
		fileInfo1.EXPECT().Name().Return("file1.txt").Times(1)
		fileInfo2.EXPECT().Name().Return("file2.nzb").Times(1)

		// Txt file is not a masked nzb file
		fs.EXPECT().Stat("file1.nzb").Return(nil, os.ErrNotExist).Times(1)
		fs.EXPECT().IsNotExist(os.ErrNotExist).Return(true).Times(1)

		mockNzbLoader.EXPECT().LoadFromFile("folder/file2.nzb").Return(nil, ErrCorruptedNzb)

		result, err := f.Readdir(0)
		assert.NoError(t, err)

		assert.Len(t, result, 1)
		// We should receive txt file as is
		assert.Equal(t, fileInfo1, result[0])
	})
}

func TestSystemFileMethods(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()
	mockNzbLoader := nzbloader.NewMockNzbLoader(ctrl)
	mockCNzb := corruptednzbsmanager.NewMockCorruptedNzbsManager(ctrl)
	fs := osfs.NewMockFileSystem(ctrl)
	mockFile := osfs.NewMockFile(ctrl)
	mockBuffer := NewMockBuffer(ctrl)

	t.Run("Chown", func(t *testing.T) {
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}
		mockFile.EXPECT().Chown(1000, 1000).Return(nil)

		uid, gid := 1000, 1000
		err := f.Chown(uid, gid)
		assert.NoError(t, err)
	})

	t.Run("Chdir", func(t *testing.T) {
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}
		mockFile.EXPECT().Chdir().Return(nil)

		err := f.Chdir()
		assert.NoError(t, err)
	})

	t.Run("Chmod", func(t *testing.T) {
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}
		mockFile.EXPECT().Chmod(os.FileMode(0644)).Return(nil)

		mode := os.FileMode(0644)
		err := f.Chmod(mode)
		assert.NoError(t, err)
	})

	t.Run("Fd", func(t *testing.T) {
		fd := uintptr(123)
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		mockFile.EXPECT().Fd().Return(fd)

		assert.Equal(t, fd, f.Fd())
	})

	t.Run("Name", func(t *testing.T) {
		name := "test.nzb"
		f := &file{
			name:      name,
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		assert.Equal(t, name, f.Name())
	})

	t.Run("Readdirnames", func(t *testing.T) {
		names := []string{"file1", "file2"}
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		mockFile.EXPECT().Readdirnames(0).Return(names, nil)

		names2, err := f.Readdirnames(0)
		assert.NoError(t, err)
		assert.Equal(t, names, names2)
	})

	t.Run("SetDeadline", func(t *testing.T) {
		tm := time.Now()
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		mockFile.EXPECT().SetDeadline(tm).Return(nil)

		err := f.SetDeadline(tm)
		assert.NoError(t, err)
	})

	t.Run("SetReadDeadline", func(t *testing.T) {
		tm := time.Now()
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		mockFile.EXPECT().SetReadDeadline(tm).Return(nil)

		err := f.SetReadDeadline(tm)
		assert.NoError(t, err)
	})

	t.Run("SetWriteDeadline", func(t *testing.T) {
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		err := f.SetWriteDeadline(time.Now())
		assert.Equal(t, os.ErrPermission, err)
	})

	t.Run("Sync", func(t *testing.T) {
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		mockFile.EXPECT().Sync().Return(nil)

		err := f.Sync()
		assert.NoError(t, err)
	})

	t.Run("Truncate", func(t *testing.T) {
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		err := f.Truncate(123)
		assert.Equal(t, os.ErrPermission, err)
	})

	t.Run("Write", func(t *testing.T) {
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		n, err := f.Write([]byte("test"))
		assert.Equal(t, 0, n)
		assert.Equal(t, os.ErrPermission, err)
	})

	t.Run("WriteAt", func(t *testing.T) {
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		n, err := f.WriteAt([]byte("test"), 0)
		assert.Equal(t, 0, n)
		assert.Equal(t, os.ErrPermission, err)
	})

	t.Run("WriteString", func(t *testing.T) {
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		n, err := f.WriteString("test")
		assert.Equal(t, 0, n)
		assert.Equal(t, os.ErrPermission, err)
	})

	t.Run("Seek", func(t *testing.T) {
		offset := int64(0)
		whence := io.SeekStart
		n := int64(123)
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata:  usenet.Metadata{},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		mockBuffer.EXPECT().Seek(offset, whence).Return(n, nil)

		n2, err := f.Seek(offset, whence)
		assert.NoError(t, err)
		assert.Equal(t, n, n2)
	})

	t.Run("Stat", func(t *testing.T) {
		today := time.Now()
		f := &file{
			name:      "test.nzb",
			buffer:    mockBuffer,
			innerFile: mockFile,
			fsMutex:   sync.RWMutex{},
			log:       log,
			metadata: usenet.Metadata{
				FileExtension: ".mkv",
				FileSize:      123,
				ChunkSize:     456,
				FileName:      "test.mkv",
				ModTime:       today,
			},
			nzbLoader: mockNzbLoader,
			onClose:   func() error { return nil },
			cNzb:      mockCNzb,
			fs:        fs,
		}

		mockFsStat := osfs.NewMockFileInfo(ctrl)
		mockFsStat.EXPECT().Name().Return("test.nzb").Times(1)

		mockFile.EXPECT().Name().Return("folder/test.nzb").Times(1)
		fs.EXPECT().Stat("folder/test.nzb").Return(mockFsStat, nil).Times(1)

		info, err := f.Stat()
		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, "test.mkv", info.Name())
		assert.Equal(t, int64(123), info.Size())
		assert.Equal(t, today, info.ModTime())
	})
}