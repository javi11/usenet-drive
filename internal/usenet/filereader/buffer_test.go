package filereader

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/textproto"
	"sync"
	"testing"
	"time"

	"github.com/bool64/cache"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
	"github.com/javi11/usenet-drive/pkg/nzb"
)

func TestBuffer_Read(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)
	t.Run("TestBuffer_Read_Empty", func(t *testing.T) {
		segmentsBuffer := cache.NewShardedMapOf[[]byte]()
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:            context.Background(),
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		// Test empty read
		p := make([]byte, 0)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_Read_PastEnd", func(t *testing.T) {
		segmentsBuffer := cache.NewShardedMapOf[[]byte]()
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:            context.Background(),
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			downloadRetryTimeoutMs: 1000,
		}

		// Test read past end of buffer
		buf.ptr = int64(buf.fileSize)
		p := make([]byte, 100)
		n, err := buf.Read(p)
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_Read_OneSegment", func(t *testing.T) {
		segmentsBuffer := cache.NewShardedMapOf[[]byte]()
		t.Cleanup(func() {
			segmentsBuffer.DeleteAll(context.Background())
		})
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:            context.Background(),
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		expectedBody := "body1"
		segmentsBuffer.Store([]byte("0"), []byte(expectedBody))

		p := make([]byte, 5)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody), p[:n])
		assert.Equal(t, int64(5), buf.ptr)
	})

	t.Run("TestBuffer_Timeout_Reading", func(t *testing.T) {
		segmentsBuffer := cache.NewShardedMapOf[[]byte]()
		t.Cleanup(func() {
			segmentsBuffer.DeleteAll(context.Background())
		})
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:            context.Background(),
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 3,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		nzbReader.EXPECT().GetSegment(0).Return(nzb.NzbSegment{Id: "1", Bytes: 5}, true).Times(1)
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Free(mockResource).Times(1)
		expectedBody := "body1"

		mockConn.EXPECT().JoinGroup("group1").Return(nil).Times(1)
		mockConn.EXPECT().Body("<1>").Return([]byte(expectedBody), nil).Times(1)

		p := make([]byte, 5)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody), p[:n])
		assert.Equal(t, int64(5), buf.ptr)
	})

	t.Run("TestBuffer_Read_TwoSegments", func(t *testing.T) {
		segmentsBuffer := cache.NewShardedMapOf[[]byte]()
		t.Cleanup(func() {
			segmentsBuffer.DeleteAll(context.Background())
		})
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:            context.Background(),
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		expectedBody1 := "body1"
		expectedBody2 := "body2"

		segmentsBuffer.Store([]byte("0"), []byte(expectedBody1))
		segmentsBuffer.Store([]byte("1"), []byte(expectedBody2))

		p := make([]byte, 10)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 10, n)
		assert.Equal(t, []byte("body1body2"), p[:n])
		assert.Equal(t, int64(10), buf.ptr)
	})
}

func TestBuffer_ReadAt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)
	segmentsBuffer := cache.NewShardedMapOf[[]byte]()

	t.Run("TestBuffer_ReadAt_Empty", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:            context.Background(),
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		// Test empty read
		p := make([]byte, 0)
		n, err := buf.ReadAt(p, 0)
		assert.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_ReadAt_PastEnd", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:            context.Background(),
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      100,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			downloadRetryTimeoutMs: 1000,
		}

		// Test read past end of buffer
		p := make([]byte, 100)
		n, err := buf.ReadAt(p, int64(buf.fileSize))
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_ReadAt_OneSegment", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:            context.Background(),
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
			},
			log:                nil,
			currentDownloading: &sync.Map{},
		}

		expectedBody1 := "body1"
		segmentsBuffer.Store([]byte("0"), []byte(expectedBody1))

		p := make([]byte, 5)
		n, err := buf.ReadAt(p, 0)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("body1"), p[:n])
	})

	t.Run("TestBuffer_ReadAt_TwoSegments", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:            context.Background(),
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		expectedBody1 := "body2"

		expectedBody2 := "body3"

		segmentsBuffer.Store([]byte("1"), []byte(expectedBody1))
		segmentsBuffer.Store([]byte("2"), []byte(expectedBody2))

		p := make([]byte, 9)
		// Special attention to the offset, it will start reading from the second segment since chunkSize is 5
		n, err := buf.ReadAt(p, 6)
		assert.NoError(t, err)
		assert.Equal(t, 9, n)
		assert.Equal(t, []byte("ody2body3"), p[:n])
	})
}

func TestBuffer_Seek(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)
	nzbReader := nzbloader.NewMockNzbReader(ctrl)

	segmentsBuffer := cache.NewShardedMapOf[[]byte]()

	t.Run("Test seek start", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:            ctx,
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		// Test seek start
		off, err := buf.Seek(0, io.SeekStart)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), off)
	})

	t.Run("Test seek current", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:            ctx,
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		// Test seek current
		off, err := buf.Seek(10, io.SeekCurrent)
		assert.NoError(t, err)
		assert.Equal(t, int64(10), off)
	})

	t.Run("Test seek end", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:            ctx,
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		// Test seek end
		off, err := buf.Seek(-10, io.SeekEnd)
		assert.NoError(t, err)
		assert.Equal(t, int64(buf.fileSize-10), off)
	})

	t.Run("Test seek invalid whence", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:            ctx,
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		// Test invalid whence
		_, err := buf.Seek(0, 3)
		assert.True(t, errors.Is(err, ErrInvalidWhence))
	})

	t.Run("Test seek negative position", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:            ctx,
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		// Test negative position
		_, err := buf.Seek(-1, io.SeekStart)
		assert.True(t, errors.Is(err, ErrSeekNegative))
	})

	t.Run("Test seek too far", func(t *testing.T) {
		buf := &buffer{
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      100,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		// Test too far
		_, err := buf.Seek(int64(buf.fileSize+1), io.SeekStart)
		assert.True(t, errors.Is(err, ErrSeekTooFar))
	})
}

func TestBuffer_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)

	segmentsBuffer := cache.NewShardedMapOf[[]byte]()

	t.Run("Test close buffer", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:            ctx,
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,

			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			nextSegment:            make(chan nzb.NzbSegment),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		err := buf.Close()
		assert.NoError(t, err)
	})

	t.Run("Test close buffer with download workers", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})

		buf := &buffer{
			ctx:            ctx,
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      100,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			nextSegment:            make(chan nzb.NzbSegment),
			wg:                     &sync.WaitGroup{},
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		wg := &sync.WaitGroup{}

		wg.Add(1)
		go func() {
			defer wg.Done()
			// wait for nextSegment close to be called
			_, ok := <-buf.nextSegment

			assert.Equal(t, ok, false)
		}()

		err := buf.Close()
		assert.NoError(t, err)

		wg.Wait()
	})
}

func TestBuffer_downloadSegment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)
	segmentsBuffer := cache.NewShardedMapOf[[]byte]()

	segment := nzb.NzbSegment{Id: "1", Number: 1, Bytes: 5}
	groups := []string{"group1"}

	t.Run("Test download segment", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:            ctx,
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Free(mockResource).Times(1)
		expectedBody1 := "body1"

		mockConn.EXPECT().JoinGroup("group1").Return(nil).Times(1)
		mockConn.EXPECT().Body("<1>").Return([]byte(expectedBody1), nil).Times(1)

		part, err := buf.downloadSegment(context.Background(), segment, groups)
		assert.NoError(t, err)
		assert.Equal(t, []byte("body1"), part)
	})

	// Test error getting connection
	t.Run("Test error getting connection", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:            ctx,
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}
		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(nil, errors.New("error")).Times(1)

		_, err := buf.downloadSegment(context.Background(), segment, groups)
		assert.Error(t, err)
	})

	// Test error finding group
	t.Run("Test error finding group", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:            ctx,
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(errors.New("error")).Times(1)

		_, err := buf.downloadSegment(context.Background(), segment, groups)
		assert.Error(t, err)
	})

	// Test error getting article body
	t.Run("Test error getting article body", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:            ctx,
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nil).Times(1)

		mockConn.EXPECT().Body("<1>").Return(nil, errors.New("some error")).Times(1)

		_, err := buf.downloadSegment(context.Background(), segment, groups)
		assert.ErrorIs(t, err, ErrCorruptedNzb)
	})

	t.Run("Test retrying after a body retirable error", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:            ctx,
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(2)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(2)
		mockResource.EXPECT().CreationTime().Return(time.Now()).Times(1)

		mockConn2 := nntpcli.NewMockConnection(ctrl)
		mockConn2.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn2.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource2 := connectionpool.NewMockResource(ctrl)
		mockResource2.EXPECT().Value().Return(mockConn2).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)

		mockConn.EXPECT().JoinGroup("group1").Return(nil).Times(1)
		mockConn.EXPECT().Body("<1>").Return(nil, &textproto.Error{Code: nntpcli.SegmentAlreadyExistsErrCode}).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource2, nil).Times(1)
		mockPool.EXPECT().Free(mockResource2).Times(1)
		mockConn2.EXPECT().JoinGroup("group1").Return(nil).Times(1)

		expectedBody1 := "body1"

		mockConn2.EXPECT().Body("<1>").Return([]byte(expectedBody1), nil).Times(1)

		part, err := buf.downloadSegment(context.Background(), segment, groups)
		assert.NoError(t, err)
		assert.NotNil(t, part)
		assert.Equal(t, []byte("body1"), part)
	})

	t.Run("Test retrying after a group retirable error", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:            ctx,
			fileSize:       3 * 100,
			nzbReader:      nzbReader,
			nzbGroups:      []string{"group1"},
			ptr:            0,
			segmentsBuffer: segmentsBuffer,
			cp:             mockPool,
			chunkSize:      5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:                    slog.Default(),
			currentDownloading:     &sync.Map{},
			downloadRetryTimeoutMs: 1000,
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(2)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(2)
		mockResource.EXPECT().CreationTime().Return(time.Now()).Times(1)

		mockConn2 := nntpcli.NewMockConnection(ctrl)
		mockConn2.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn2.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource2 := connectionpool.NewMockResource(ctrl)
		mockResource2.EXPECT().Value().Return(mockConn2).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(textproto.ProtocolError("some error")).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource2, nil).Times(1)
		mockPool.EXPECT().Free(mockResource2).Times(1)
		mockConn2.EXPECT().JoinGroup("group1").Return(nil).Times(1)

		expectedBody1 := "body1"

		mockConn2.EXPECT().Body("<1>").Return([]byte(expectedBody1), nil).Times(1)

		part, err := buf.downloadSegment(context.Background(), segment, groups)

		assert.NoError(t, err)
		assert.NotNil(t, part)
		assert.Equal(t, []byte("body1"), part)
	})
}
