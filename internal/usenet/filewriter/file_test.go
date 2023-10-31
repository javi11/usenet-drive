package filewriter

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-multierror"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"github.com/stretchr/testify/assert"
)

func TestOpenFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()
	fs := osfs.NewMockFileSystem(ctrl)
	cp := connectionpool.NewMockUsenetConnectionPool(ctrl)
	fileSize := int64(100)
	segmentSize := int64(10)
	randomGroup := "alt.binaries.test"
	dryRun := false

	name := "test.mkv"
	flag := os.O_RDONLY
	perm := os.FileMode(0644)
	onClose := func(err error) error { return nil }

	// Call
	f, err := openFile(
		context.Background(),
		name,
		flag,
		perm,
		fileSize,
		segmentSize,
		cp,
		randomGroup,
		log,
		5,
		dryRun,
		onClose,
		fs,
	)
	assert.NoError(t, err)
	assert.Equal(t, name, f.Name())
}

func TestCloseFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()
	fs := osfs.NewMockFileSystem(ctrl)
	cp := connectionpool.NewMockUsenetConnectionPool(ctrl)
	fileSize := int64(100)
	segmentSize := int64(10)
	randomGroup := "alt.binaries.test"
	dryRun := false
	fileNameHash := "test"
	filePath := "test.mkv"
	parts := int64(10)
	poster := "poster"
	fileName := "test.mkv"

	f := &file{
		ctx:              context.Background(),
		maxUploadRetries: 5,
		dryRun:           dryRun,
		cp:               cp,
		fs:               fs,
		log:              log,
		flag:             os.O_WRONLY,
		perm:             os.FileMode(0644),
		nzbMetadata: &nzbMetadata{
			fileNameHash:     fileNameHash,
			filePath:         filePath,
			parts:            parts,
			group:            randomGroup,
			poster:           poster,
			expectedFileSize: fileSize,
		},
		metadata: &usenet.Metadata{
			FileName:      fileName,
			ModTime:       time.Now(),
			FileSize:      0,
			FileExtension: filepath.Ext(fileName),
			ChunkSize:     segmentSize,
		},
	}

	onClosedCalled := false
	onClose := func(_ error) error {
		onClosedCalled = true
		return nil
	}
	merr := &multierror.Group{}
	merr.Go(func() error { return nil })
	closedFile := f
	closedFile.onClose = onClose

	err := f.Close()
	assert.NoError(t, err)
	assert.True(t, onClosedCalled)
}

func TestSystemFileMethods(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	log := slog.Default()
	fs := osfs.NewMockFileSystem(ctrl)
	cp := connectionpool.NewMockUsenetConnectionPool(ctrl)
	fileSize := int64(100)
	segmentSize := int64(10)
	randomGroup := "alt.binaries.test"
	dryRun := false
	fileNameHash := "test"
	filePath := "test.mkv"
	parts := int64(10)
	poster := "poster"
	fileName := "test.mkv"
	modTime := time.Now()

	f := &file{
		ctx:              context.Background(),
		maxUploadRetries: 5,
		dryRun:           dryRun,
		cp:               cp,
		fs:               fs,
		log:              log,
		flag:             os.O_WRONLY,
		perm:             os.FileMode(0644),
		nzbMetadata: &nzbMetadata{
			fileNameHash:     fileNameHash,
			filePath:         filePath,
			parts:            parts,
			group:            randomGroup,
			poster:           poster,
			expectedFileSize: fileSize,
		},
		metadata: &usenet.Metadata{
			FileName:      fileName,
			ModTime:       modTime,
			FileSize:      0,
			FileExtension: filepath.Ext(fileName),
			ChunkSize:     segmentSize,
		},
	}

	t.Run("Chown", func(t *testing.T) {
		uid, gid := 1000, 1000
		err := f.Chown(uid, gid)
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("Chdir", func(t *testing.T) {
		err := f.Chdir()
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("Chmod", func(t *testing.T) {
		mode := os.FileMode(0644)
		err := f.Chmod(mode)
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("Fd", func(t *testing.T) {
		fd := uintptr(0)

		assert.Equal(t, fd, f.Fd())
	})

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, fileName, f.Name())
	})

	t.Run("Readdirnames", func(t *testing.T) {
		_, err := f.Readdirnames(0)
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("SetDeadline", func(t *testing.T) {
		tm := time.Now()

		err := f.SetDeadline(tm)
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("SetReadDeadline", func(t *testing.T) {
		tm := time.Now()
		err := f.SetReadDeadline(tm)
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("SetWriteDeadline", func(t *testing.T) {
		err := f.SetWriteDeadline(time.Now())
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("Sync", func(t *testing.T) {
		err := f.Sync()
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("Truncate", func(t *testing.T) {
		err := f.Truncate(123)
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("Write", func(t *testing.T) {
		_, err := f.Write([]byte("test"))
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("WriteAt", func(t *testing.T) {
		_, err := f.WriteAt([]byte("test"), 0)
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("WriteString", func(t *testing.T) {
		_, err := f.WriteString("test")
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("Seek", func(t *testing.T) {
		offset := int64(0)
		whence := io.SeekStart

		_, err := f.Seek(offset, whence)
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("Stat", func(t *testing.T) {

		info, err := f.Stat()
		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, fileName, info.Name())
		// Without writing any data, the file size should be 0.
		assert.Equal(t, int64(0), info.Size())
		assert.Equal(t, modTime, info.ModTime())
	})
}

func TestReadFrom(t *testing.T) {
	t.Parallel()

	log := slog.Default()
	ctrl := gomock.NewController(t)
	fileSize := int64(100)
	segmentSize := int64(10)
	randomGroup := "alt.binaries.test"
	dryRun := false
	fileNameHash := "test"
	filePath := "test.mkv"
	parts := int64(10)
	poster := "poster"
	fileName := "test.mkv"
	maxUploadRetries := 5

	t.Run("File uploaded", func(t *testing.T) {
		fs := osfs.NewMockFileSystem(ctrl)
		cp := connectionpool.NewMockUsenetConnectionPool(ctrl)
		metadata := &usenet.Metadata{
			FileName:      fileName,
			ModTime:       time.Now(),
			FileSize:      0,
			FileExtension: filepath.Ext(fileName),
			ChunkSize:     segmentSize,
		}

		openedFile := &file{
			ctx:              context.Background(),
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			fs:               fs,
			log:              log,
			flag:             os.O_WRONLY,
			perm:             os.FileMode(0644),
			nzbMetadata: &nzbMetadata{
				fileNameHash:     fileNameHash,
				filePath:         filePath,
				parts:            parts,
				group:            randomGroup,
				poster:           poster,
				expectedFileSize: fileSize,
			},
			metadata: metadata,
		}

		// 100 bytes
		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Post(gomock.Any(), segmentSize).Return(nil).Times(10)
		cp.EXPECT().Get().Return(mockConn, nil).Times(10)
		cp.EXPECT().Free(mockConn).Return(nil).Times(10)

		segments := make([]nzb.NzbSegment, parts)
		for i := int64(0); i < parts; i++ {
			segments[i] = nzb.NzbSegment{
				Bytes: segmentSize,
			}
		}

		fs.EXPECT().WriteFile("test.nzb", gomock.Any(), os.FileMode(0644)).Return(nil)

		n, e := openedFile.ReadFrom(src)
		assert.NoError(t, e)
		assert.Equal(t, int64(100), n)
		assert.Equal(t, metadata.FileSize, n)
	})

	t.Run("Wrong expected file size", func(t *testing.T) {
		fs := osfs.NewMockFileSystem(ctrl)
		cp := connectionpool.NewMockUsenetConnectionPool(ctrl)

		openedFile := &file{
			ctx:              context.Background(),
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			fs:               fs,
			log:              log,
			flag:             os.O_WRONLY,
			perm:             os.FileMode(0644),
			nzbMetadata: &nzbMetadata{
				fileNameHash:     fileNameHash,
				filePath:         filePath,
				parts:            1,
				group:            randomGroup,
				poster:           poster,
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
		}

		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		mockConn := nntpcli.NewMockConnection(ctrl)
		// Due to the async nature of the upload, post can be called 1 or 0 times since the context will be canceled when the error ocurred.
		mockConn.EXPECT().Post(gomock.Any(), segmentSize).Return(nil).AnyTimes()
		cp.EXPECT().Get().Return(mockConn, nil).Times(1)
		cp.EXPECT().Free(mockConn).Return(nil).Times(1)

		_, e := openedFile.ReadFrom(src)
		assert.ErrorIs(t, e, ErrUnexpectedFileSize)
	})

	t.Run("Read stops before the write the expected size", func(t *testing.T) {
		fs := osfs.NewMockFileSystem(ctrl)
		cp := connectionpool.NewMockUsenetConnectionPool(ctrl)

		openedFile := &file{
			ctx:              context.Background(),
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			fs:               fs,
			log:              log,
			flag:             os.O_WRONLY,
			perm:             os.FileMode(0644),
			nzbMetadata: &nzbMetadata{
				fileNameHash:     fileNameHash,
				filePath:         filePath,
				parts:            parts,
				group:            randomGroup,
				poster:           poster,
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
		}

		// Less than 100 bytes
		src := strings.NewReader("Et dignissimos")

		mockConn := nntpcli.NewMockConnection(ctrl)
		// Due to the async nature of the upload, post can be called 1 or 0 times since the context will be canceled when the error ocurred.
		mockConn.EXPECT().Post(gomock.Any(), segmentSize).Return(nil).AnyTimes()
		cp.EXPECT().Get().Return(mockConn, nil).AnyTimes()
		cp.EXPECT().Free(mockConn).Return(nil).AnyTimes()

		_, e := openedFile.ReadFrom(src)
		assert.ErrorIs(t, e, io.ErrShortWrite)
	})

	t.Run("Retry if get connection failed", func(t *testing.T) {
		fs := osfs.NewMockFileSystem(ctrl)
		cp := connectionpool.NewMockUsenetConnectionPool(ctrl)

		openedFile := &file{
			ctx:              context.Background(),
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			fs:               fs,
			log:              log,
			flag:             os.O_WRONLY,
			perm:             os.FileMode(0644),
			nzbMetadata: &nzbMetadata{
				fileNameHash:     fileNameHash,
				filePath:         filePath,
				parts:            parts,
				group:            randomGroup,
				poster:           poster,
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
		}

		// 100 bytes
		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Post(gomock.Any(), segmentSize).Return(nil).Times(10)
		cp.EXPECT().Get().Return(mockConn, syscall.ETIMEDOUT).Times(1)
		cp.EXPECT().Get().Return(mockConn, nil).Times(10)
		cp.EXPECT().Close(mockConn).Return(nil).Times(1)
		cp.EXPECT().Free(mockConn).Return(nil).Times(10)
		fs.EXPECT().WriteFile("test.nzb", gomock.Any(), os.FileMode(0644)).Return(nil)

		n, e := openedFile.ReadFrom(src)
		assert.NoError(t, e)
		assert.Equal(t, int64(100), n)
	})

	t.Run("If max number of retries are exhausted on get connection throw an error", func(t *testing.T) {
		fs := osfs.NewMockFileSystem(ctrl)
		cp := connectionpool.NewMockUsenetConnectionPool(ctrl)

		openedFile := &file{
			ctx:              context.Background(),
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			fs:               fs,
			log:              log,
			flag:             os.O_WRONLY,
			perm:             os.FileMode(0644),
			nzbMetadata: &nzbMetadata{
				fileNameHash:     fileNameHash,
				filePath:         filePath,
				parts:            parts,
				group:            randomGroup,
				poster:           poster,
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
		}

		// 100 bytes
		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		mockConn := nntpcli.NewMockConnection(ctrl)
		cp.EXPECT().Get().Return(mockConn, syscall.ETIMEDOUT).Times(maxUploadRetries)
		cp.EXPECT().Close(mockConn).Return(nil).Times(maxUploadRetries)

		_, err := openedFile.ReadFrom(src)
		assert.ErrorIs(t, err, syscall.ETIMEDOUT)
	})

	t.Run("If error is not retryable get connection, do not retry", func(t *testing.T) {
		fs := osfs.NewMockFileSystem(ctrl)
		cp := connectionpool.NewMockUsenetConnectionPool(ctrl)

		openedFile := &file{
			ctx:              context.Background(),
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			fs:               fs,
			log:              log,
			flag:             os.O_WRONLY,
			perm:             os.FileMode(0644),
			nzbMetadata: &nzbMetadata{
				fileNameHash:     fileNameHash,
				filePath:         filePath,
				parts:            parts,
				group:            randomGroup,
				poster:           poster,
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
		}

		// 100 bytes
		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		e := errors.New("no retryable")
		mockConn := nntpcli.NewMockConnection(ctrl)
		cp.EXPECT().Get().Return(mockConn, e).Times(1)
		cp.EXPECT().Close(mockConn).Return(nil).Times(1)

		_, err := openedFile.ReadFrom(src)
		assert.ErrorIs(t, err, e)
	})

	t.Run("Retry if upload throws a retryable error", func(t *testing.T) {
		fs := osfs.NewMockFileSystem(ctrl)
		cp := connectionpool.NewMockUsenetConnectionPool(ctrl)

		openedFile := &file{
			ctx:              context.Background(),
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			fs:               fs,
			log:              log,
			flag:             os.O_WRONLY,
			perm:             os.FileMode(0644),
			nzbMetadata: &nzbMetadata{
				fileNameHash:     fileNameHash,
				filePath:         filePath,
				parts:            parts,
				group:            randomGroup,
				poster:           poster,
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
		}

		// 100 bytes
		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn2 := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Post(gomock.Any(), segmentSize).Return(net.ErrClosed).Times(1)
		mockConn2.EXPECT().Post(gomock.Any(), segmentSize).Return(nil).Times(10)
		// First connection is closed because of the retryable error
		cp.EXPECT().Get().Return(mockConn, nil).Times(1)
		// Second connection works as expected
		cp.EXPECT().Get().Return(mockConn2, nil).Times(10)
		cp.EXPECT().Free(mockConn2).Return(nil).Times(10)
		fs.EXPECT().WriteFile("test.nzb", gomock.Any(), os.FileMode(0644)).Return(nil)

		n, e := openedFile.ReadFrom(src)
		assert.NoError(t, e)
		assert.Equal(t, int64(100), n)
	})

	t.Run("Retry and recreate segment for partial upload", func(t *testing.T) {
		fs := osfs.NewMockFileSystem(ctrl)
		cp := connectionpool.NewMockUsenetConnectionPool(ctrl)

		openedFile := &file{
			ctx:              context.Background(),
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			fs:               fs,
			log:              log,
			flag:             os.O_WRONLY,
			perm:             os.FileMode(0644),
			nzbMetadata: &nzbMetadata{
				fileNameHash:     fileNameHash,
				filePath:         filePath,
				parts:            parts,
				group:            randomGroup,
				poster:           poster,
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
		}

		// 100 bytes
		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn2 := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Post(gomock.Any(), segmentSize).Return(nntpcli.NntpError{Code: nntpcli.SegmentAlreadyExistsErrCode}).Times(1)
		mockConn2.EXPECT().Post(gomock.Any(), segmentSize).Return(nil).Times(10)
		// First connection is closed because of the retryable error
		cp.EXPECT().Get().Return(mockConn, nil).Times(1)
		// Second connection works as expected
		cp.EXPECT().Get().Return(mockConn2, nil).Times(10)
		cp.EXPECT().Close(mockConn).Return(nil).Times(1)
		cp.EXPECT().Free(mockConn2).Return(nil).Times(10)
		fs.EXPECT().WriteFile("test.nzb", gomock.Any(), os.FileMode(0644)).Return(nil)

		n, e := openedFile.ReadFrom(src)
		assert.NoError(t, e)
		assert.Equal(t, int64(100), n)
	})

	t.Run("Return an error if file write failed", func(t *testing.T) {
		fs := osfs.NewMockFileSystem(ctrl)
		cp := connectionpool.NewMockUsenetConnectionPool(ctrl)

		openedFile := &file{
			ctx:              context.Background(),
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			fs:               fs,
			log:              log,
			flag:             os.O_WRONLY,
			perm:             os.FileMode(0644),
			nzbMetadata: &nzbMetadata{
				fileNameHash:     fileNameHash,
				filePath:         filePath,
				parts:            parts,
				group:            randomGroup,
				poster:           poster,
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
		}

		// 100 bytes
		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Post(gomock.Any(), segmentSize).Return(nil).Times(10)
		cp.EXPECT().Get().Return(mockConn, nil).Times(10)
		cp.EXPECT().Free(mockConn).Return(nil).Times(10)

		fs.EXPECT().WriteFile("test.nzb", gomock.Any(), os.FileMode(0644)).Return(errors.New("error")).Times(1)

		_, err := openedFile.ReadFrom(src)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})

	t.Run("Do not fail if refresh nzb cache failed", func(t *testing.T) {
		fs := osfs.NewMockFileSystem(ctrl)
		cp := connectionpool.NewMockUsenetConnectionPool(ctrl)

		openedFile := &file{
			ctx:              context.Background(),
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			fs:               fs,
			log:              log,
			flag:             os.O_WRONLY,
			perm:             os.FileMode(0644),
			nzbMetadata: &nzbMetadata{
				fileNameHash:     fileNameHash,
				filePath:         filePath,
				parts:            parts,
				group:            randomGroup,
				poster:           poster,
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
		}

		// 100 bytes
		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Post(gomock.Any(), segmentSize).Return(nil).Times(10)
		cp.EXPECT().Get().Return(mockConn, nil).Times(10)
		cp.EXPECT().Free(mockConn).Return(nil).Times(10)
		fs.EXPECT().WriteFile("test.nzb", gomock.Any(), os.FileMode(0644)).Return(nil)

		n, e := openedFile.ReadFrom(src)
		assert.NoError(t, e)
		assert.Equal(t, int64(100), n)
	})

	t.Run("Cancel the upload if file is context is canceled", func(t *testing.T) {
		fs := osfs.NewMockFileSystem(ctrl)
		cp := connectionpool.NewMockUsenetConnectionPool(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		openedFile := &file{
			ctx:              ctx,
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			fs:               fs,
			log:              log,
			flag:             os.O_WRONLY,
			perm:             os.FileMode(0644),
			nzbMetadata: &nzbMetadata{
				fileNameHash:     fileNameHash,
				filePath:         filePath,
				parts:            parts,
				group:            randomGroup,
				poster:           poster,
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
		}

		// 100 bytes
		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Post(gomock.Any(), segmentSize).Return(nil).AnyTimes()
		cp.EXPECT().Get().Return(mockConn, nil).AnyTimes()
		cp.EXPECT().Free(mockConn).Return(nil).AnyTimes()
		wg := &sync.WaitGroup{}

		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := openedFile.ReadFrom(src)
			assert.ErrorIs(t, err, context.Canceled)
		}()

		cancel()

		wg.Wait()
	})
}
