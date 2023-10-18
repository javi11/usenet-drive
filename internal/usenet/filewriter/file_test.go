package filewriter

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-multierror"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"github.com/stretchr/testify/assert"
)

func TestOpenFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()
	mockNzbLoader := nzbloader.NewMockNzbLoader(ctrl)
	fs := osfs.NewMockFileSystem(ctrl)
	cp := connectionpool.NewMockUsenetConnectionPool(ctrl)
	fileSize := int64(100)
	segmentSize := int64(10)
	randomGroup := "alt.binaries.test"
	dryRun := false

	name := "test.mkv"
	flag := os.O_RDONLY
	perm := os.FileMode(0644)
	onClose := func() error { return nil }

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
		mockNzbLoader,
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
	mockNzbLoader := nzbloader.NewMockNzbLoader(ctrl)
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
		maxUploadRetries: 5,
		dryRun:           dryRun,
		cp:               cp,
		nzbLoader:        mockNzbLoader,
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
			segments:         make([]nzb.NzbSegment, parts),
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
	t.Run("Error uploading a file", func(t *testing.T) {
		merr := &multierror.Group{}
		merr.Go(func() error { return errors.New("error uploading file") })
		closedFile := f
		closedFile.merr = merr

		err := f.Close()
		assert.Equal(t, io.ErrUnexpectedEOF, err)
	})

	t.Run("Upload was canceled  resulting on 0 bytes segments", func(t *testing.T) {
		merr := &multierror.Group{}
		merr.Go(func() error { return nil })
		closedFile := f
		closedFile.merr = merr
		segments := make([]nzb.NzbSegment, parts)
		segments[0] = nzb.NzbSegment{
			Bytes: 0,
		}
		closedFile.nzbMetadata.segments = segments

		err := f.Close()
		assert.Equal(t, io.ErrUnexpectedEOF, err)
	})

	t.Run("Error writing the file", func(t *testing.T) {
		merr := &multierror.Group{}
		merr.Go(func() error { return nil })
		closedFile := f
		closedFile.merr = merr
		segments := make([]nzb.NzbSegment, parts)
		for i := int64(0); i < parts; i++ {
			segments[i] = nzb.NzbSegment{
				Bytes: segmentSize,
			}
		}

		closedFile.nzbMetadata.segments = segments

		fs.EXPECT().WriteFile("test.nzb", gomock.Any(), os.FileMode(0644)).Return(errors.New("error writing file"))

		err := f.Close()
		assert.Equal(t, io.ErrUnexpectedEOF, err)
	})

	t.Run("Success", func(t *testing.T) {
		onClosedCalled := false
		onClose := func() error {
			onClosedCalled = true
			return nil
		}
		merr := &multierror.Group{}
		merr.Go(func() error { return nil })
		closedFile := f
		closedFile.merr = merr
		closedFile.onClose = onClose
		segments := make([]nzb.NzbSegment, parts)
		for i := int64(0); i < parts; i++ {
			segments[i] = nzb.NzbSegment{
				Bytes: segmentSize,
			}
		}

		closedFile.nzbMetadata.segments = segments

		fs.EXPECT().WriteFile("test.nzb", gomock.Any(), os.FileMode(0644)).Return(nil)
		mockNzbLoader.EXPECT().RefreshCachedNzb("test.nzb", gomock.Any()).Return(true, nil)

		err := f.Close()
		assert.NoError(t, err)
		assert.True(t, onClosedCalled)
	})

	t.Run("An error refreshing nzb cache should not stop the close to succeed", func(t *testing.T) {
		onClosedCalled := false
		onClose := func() error {
			onClosedCalled = true
			return nil
		}
		merr := &multierror.Group{}
		merr.Go(func() error { return nil })
		closedFile := f
		closedFile.merr = merr
		closedFile.onClose = onClose
		segments := make([]nzb.NzbSegment, parts)
		for i := int64(0); i < parts; i++ {
			segments[i] = nzb.NzbSegment{
				Bytes: segmentSize,
			}
		}

		closedFile.nzbMetadata.segments = segments

		fs.EXPECT().WriteFile("test.nzb", gomock.Any(), os.FileMode(0644)).Return(nil)
		mockNzbLoader.EXPECT().RefreshCachedNzb("test.nzb", gomock.Any()).Return(false, errors.New("error refreshing nzb cache"))

		err := f.Close()
		assert.NoError(t, err)
		assert.True(t, onClosedCalled)
	})

}

func TestSystemFileMethods(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()
	mockNzbLoader := nzbloader.NewMockNzbLoader(ctrl)
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
		maxUploadRetries: 5,
		dryRun:           dryRun,
		cp:               cp,
		nzbLoader:        mockNzbLoader,
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
			segments:         make([]nzb.NzbSegment, parts),
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
	ctrl := gomock.NewController(t)
	log := slog.Default()
	mockNzbLoader := nzbloader.NewMockNzbLoader(ctrl)
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
	maxUploadRetries := 5

	t.Run("File uploaded", func(t *testing.T) {
		openedFile := &file{
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			nzbLoader:        mockNzbLoader,
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
				segments:         make([]nzb.NzbSegment, parts),
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
			merr: &multierror.Group{},
		}

		// 100 bytes
		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockConn.EXPECT().Post(gomock.Any()).Return(nil).Times(10)
		cp.EXPECT().Get().Return(mockConn, nil).Times(10)
		cp.EXPECT().Free(mockConn).Return(nil).Times(10)

		n, e := openedFile.ReadFrom(src)
		assert.NoError(t, e)
		assert.Equal(t, int64(100), n)

		err := openedFile.merr.Wait().ErrorOrNil()
		assert.NoError(t, err)
	})

	t.Run("Unexpected file size", func(t *testing.T) {
		openedFile := &file{
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			nzbLoader:        mockNzbLoader,
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
				segments:         make([]nzb.NzbSegment, parts),
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
			merr: &multierror.Group{},
		}

		src := strings.NewReader("this is the input")

		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockConn.EXPECT().Post(gomock.Any()).Return(nil).Times(1)
		cp.EXPECT().Get().Return(mockConn, nil).Times(1)
		cp.EXPECT().Free(mockConn).Return(nil).Times(1)

		_, e := openedFile.ReadFrom(src)
		assert.ErrorIs(t, e, ErrUnexpectedFileSize)

		err := openedFile.merr.Wait().ErrorOrNil()
		assert.NoError(t, err)
	})

	t.Run("Retry if get connection failed", func(t *testing.T) {
		openedFile := &file{
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			nzbLoader:        mockNzbLoader,
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
				segments:         make([]nzb.NzbSegment, parts),
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
			merr: &multierror.Group{},
		}

		// 100 bytes
		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockConn.EXPECT().Post(gomock.Any()).Return(nil).Times(10)
		cp.EXPECT().Get().Return(mockConn, errors.New("test")).Times(1)
		cp.EXPECT().Get().Return(mockConn, nil).Times(10)
		cp.EXPECT().Close(mockConn).Return(nil).Times(1)
		cp.EXPECT().Free(mockConn).Return(nil).Times(10)

		n, e := openedFile.ReadFrom(src)
		assert.NoError(t, e)
		assert.Equal(t, int64(100), n)

		err := openedFile.merr.Wait().ErrorOrNil()
		assert.NoError(t, err)
	})

	t.Run("If max number of retries are exhausted throw an error", func(t *testing.T) {
		openedFile := &file{
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			nzbLoader:        mockNzbLoader,
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
				segments:         make([]nzb.NzbSegment, parts),
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
			merr: &multierror.Group{},
		}

		// 100 bytes
		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		e := errors.New("test")
		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		cp.EXPECT().Get().Return(mockConn, e).Times(maxUploadRetries + 1)
		cp.EXPECT().Close(mockConn).Return(nil).Times(maxUploadRetries + 1)

		_, err := openedFile.ReadFrom(src)
		assert.ErrorIs(t, err, e)
	})

	t.Run("Retry if upload throws a retryable error", func(t *testing.T) {
		openedFile := &file{
			maxUploadRetries: maxUploadRetries,
			dryRun:           dryRun,
			cp:               cp,
			nzbLoader:        mockNzbLoader,
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
				segments:         make([]nzb.NzbSegment, parts),
				expectedFileSize: fileSize,
			},
			metadata: &usenet.Metadata{
				FileName:      fileName,
				ModTime:       time.Now(),
				FileSize:      0,
				FileExtension: filepath.Ext(fileName),
				ChunkSize:     segmentSize,
			},
			merr: &multierror.Group{},
		}

		// 100 bytes
		src := strings.NewReader("Et dignissimos incidunt ipsam molestiae occaecati. Fugit quo autem corporis occaecati sint. lorem it")

		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockConn2 := connectionpool.NewMockNntpConnection(ctrl)
		mockConn.EXPECT().Post(gomock.Any()).Return(syscall.EPIPE).Times(1)
		mockConn2.EXPECT().Post(gomock.Any()).Return(nil).Times(10)
		// First connection is closed because of the retryable error
		cp.EXPECT().Get().Return(mockConn, nil).Times(1)
		// Second connection works as expected
		cp.EXPECT().Get().Return(mockConn2, nil).Times(10)
		cp.EXPECT().Close(mockConn).Return(nil).Times(1)
		cp.EXPECT().Free(mockConn2).Return(nil).Times(10)

		n, e := openedFile.ReadFrom(src)
		assert.NoError(t, e)
		assert.Equal(t, int64(100), n)

		err := openedFile.merr.Wait().ErrorOrNil()
		assert.NoError(t, err)
	})

}
