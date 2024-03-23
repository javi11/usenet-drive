package filereader

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

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

	cache := NewMockChunkCache(ctrl)
	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)
	t.Run("TestBuffer_Read_Empty", func(t *testing.T) {
		currentDownloading := &sync.Map{}
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:                context.Background(),
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkSize:          5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log: slog.Default(),
			mx:  &sync.RWMutex{},
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkCache: cache,
		}

		// Test empty read
		p := make([]byte, 0)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_Read_PastEnd", func(t *testing.T) {
		currentDownloading := &sync.Map{}
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:                context.Background(),
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkSize:          5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
		}

		// Test read past end of buffer
		buf.ptr = int64(buf.fileSize)
		p := make([]byte, 100)
		n, err := buf.Read(p)
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_Read_OneSegment", func(t *testing.T) {
		currentDownloading := &sync.Map{}
		t.Cleanup(func() {
			currentDownloading.Range(func(key, value interface{}) bool {
				currentDownloading.Delete(key)
				return true
			})
		})
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:                context.Background(),
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,

			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
		}

		expectedBody := "body1"
		nf := &downloadManager{
			ch:         nil,
			chunk:      []byte(expectedBody),
			downloaded: true,
			mx:         &sync.RWMutex{},
		}
		cache.EXPECT().Get(0).Return(nf).Times(1)

		p := make([]byte, 5)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody), p[:n])
		assert.Equal(t, int64(5), buf.ptr)
	})

	t.Run("TestBuffer_Direct_Download_Reading", func(t *testing.T) {
		currentDownloading := &sync.Map{}
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:                context.Background(),
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkSize:          5,
			dc: downloadConfig{
				maxDownloadRetries: 3,
				maxDownloadWorkers: 0,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
		}

		cache.EXPECT().Get(0).Return(nil).Times(1)
		nzbReader.EXPECT().GetSegment(0).Return(nzb.NzbSegment{Id: "1", Bytes: 5}, true).Times(1)
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Free(mockResource).Times(1)
		expectedBody := "body1"

		mockConn.EXPECT().JoinGroup("group1").Return(nil).Times(1)
		mockConn.EXPECT().Body("1").Return(io.NopCloser(strings.NewReader(expectedBody)), nil).Times(1)

		p := make([]byte, 5)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody), p[:n])
		assert.Equal(t, int64(5), buf.ptr)
	})

	t.Run("TestBuffer_Read_TwoSegments", func(t *testing.T) {
		currentDownloading := &sync.Map{}
		t.Cleanup(func() {
			currentDownloading.Range(func(key, value interface{}) bool {
				currentDownloading.Delete(key)
				return true
			})
		})
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:                context.Background(),
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkSize:          5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
		}

		expectedBody1 := "body1"
		expectedBody2 := "body2"

		nf := &downloadManager{
			ch:         nil,
			chunk:      []byte(expectedBody1),
			downloaded: true,
			mx:         &sync.RWMutex{},
		}
		cache.EXPECT().Get(0).Return(nf).Times(1)
		nf2 := &downloadManager{
			ch:         nil,
			downloaded: true,
			chunk:      []byte(expectedBody2),
			mx:         &sync.RWMutex{},
		}
		cache.EXPECT().Get(1).Return(nf2).Times(1)

		p := make([]byte, 10)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 10, n)
		assert.Equal(t, []byte("body1body2"), p[:n])
		assert.Equal(t, int64(10), buf.ptr)
	})

	t.Run("TestBuffer_Wait_For_Download_Worker", func(t *testing.T) {
		currentDownloading := &sync.Map{}
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:                context.Background(),
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkSize:          5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
		}

		expectedBody := "body1"
		nf := &downloadManager{
			ch:         make(chan bool, 1),
			downloaded: false,
			chunk:      []byte(expectedBody),
			mx:         &sync.RWMutex{},
		}
		cache.EXPECT().Get(0).Return(nf).Times(1)

		go func() {
			time.Sleep(100 * time.Millisecond)
			nf.mx.Lock()
			nf.downloaded = true
			nf.mx.Unlock()
			nf.ch <- true
		}()

		p := make([]byte, 5)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody), p[:n])
		assert.Equal(t, int64(5), buf.ptr)
	})

	t.Run("TestBuffer_Wait_For_Download_Worker_Ctx_Canceled", func(t *testing.T) {
		currentDownloading := &sync.Map{}
		t.Cleanup(func() {
			currentDownloading.Range(func(key, value interface{}) bool {
				currentDownloading.Delete(key)
				return true
			})
		})
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})

		buf := &buffer{
			ctx:                ctx,
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		nf := &downloadManager{
			ch:         make(chan bool, 1),
			downloaded: false,
			mx:         &sync.RWMutex{},
		}
		cache.EXPECT().Get(0).Return(nf).Times(1)
		currentDownloading.Store(0, nf)

		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		p := make([]byte, 5)
		_, err := buf.Read(p)
		assert.Error(t, err, context.Canceled)
	})

	t.Run("TestBuffer_Preload_Next_Segments", func(t *testing.T) {
		currentDownloading := &sync.Map{}
		t.Cleanup(func() {
			currentDownloading.Range(func(key, value interface{}) bool {
				currentDownloading.Delete(key)
				return true
			})
		})
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:                context.Background(),
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
			},
			nextSegment: make(chan nzb.NzbSegment, 1),
			log:         slog.Default(),
			chunkCache:  cache,
			mx:          &sync.RWMutex{},
		}

		nzbReader.EXPECT().GetSegment(0).Return(nzb.NzbSegment{Id: "1", Bytes: 5}, true).Times(2)
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Free(mockResource).Times(1)
		expectedBody := "body1"

		cache.EXPECT().Get(0).Return(nil).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nil).Times(1)
		mockConn.EXPECT().Body("1").Return(io.NopCloser(strings.NewReader(expectedBody)), nil).Times(1)

		go func() {
			time.Sleep(100 * time.Millisecond)
			// wait for nextSegment to be called
			_, ok := <-buf.nextSegment
			assert.Equal(t, ok, true)
		}()

		p := make([]byte, 5)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody), p[:n])
		assert.Equal(t, int64(5), buf.ptr)
	})

}

func TestBuffer_ReadAt(t *testing.T) {
	ctrl := gomock.NewController(t)
	cache := NewMockChunkCache(ctrl)
	defer ctrl.Finish()

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)
	currentDownloading := &sync.Map{}

	t.Run("TestBuffer_ReadAt_Empty", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:                context.Background(),
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
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
			ctx:                context.Background(),
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 100,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
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
			ctx:                context.Background(),
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:        nil,
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		expectedBody := "body1"
		nf := &downloadManager{
			ch:         nil,
			downloaded: true,
			chunk:      []byte(expectedBody),
			mx:         &sync.RWMutex{},
		}
		cache.EXPECT().Get(0).Return(nf).Times(1)

		p := make([]byte, 5)
		n, err := buf.ReadAt(p, 0)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("body1"), p[:n])
	})
}

func TestBuffer_Seek(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cache := NewMockChunkCache(ctrl)

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)
	nzbReader := nzbloader.NewMockNzbReader(ctrl)

	currentDownloading := &sync.Map{}

	t.Run("Test seek start", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:                ctx,
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
			seek:       make(chan seekData, 1),
		}

		// Test seek start
		off, err := buf.Seek(0, io.SeekStart)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), off)

		seekData := <-buf.seek
		assert.Equal(t, 0, seekData.from)
		assert.Equal(t, 0, seekData.to)
	})

	t.Run("Test seek current", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:       ctx,
			fileSize:  3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkSize:          5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
			seek:       make(chan seekData, 1),
		}

		// Test seek current
		off, err := buf.Seek(10, io.SeekCurrent)
		assert.NoError(t, err)
		assert.Equal(t, int64(10), off)

		seekData := <-buf.seek
		assert.Equal(t, 0, seekData.from)
		assert.Equal(t, 2, seekData.to)
	})

	t.Run("Test seek end", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:                ctx,
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
			seek:       make(chan seekData, 1),
		}

		// Test seek end
		off, err := buf.Seek(-10, io.SeekEnd)
		assert.NoError(t, err)
		assert.Equal(t, int64(buf.fileSize-10), off)

		seekData := <-buf.seek
		assert.Equal(t, 0, seekData.from)
		assert.Equal(t, 58, seekData.to)
	})

	t.Run("Test seek invalid whence", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:                ctx,
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
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
			ctx:                ctx,
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		// Test negative position
		_, err := buf.Seek(-1, io.SeekStart)
		assert.True(t, errors.Is(err, ErrSeekNegative))
	})

	t.Run("Test seek too far", func(t *testing.T) {
		buf := &buffer{
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 100,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		// Test too far
		_, err := buf.Seek(int64(buf.fileSize+1), io.SeekStart)
		assert.True(t, errors.Is(err, ErrSeekTooFar))
	})
}

func TestBuffer_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cache := NewMockChunkCache(ctrl)

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)

	currentDownloading := &sync.Map{}

	t.Run("Test close buffer", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:                ctx,
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:         slog.Default(),
			nextSegment: make(chan nzb.NzbSegment),
			cancel:      cancel,
			chunkCache:  cache,
			mx:          &sync.RWMutex{},
			wg:          &sync.WaitGroup{},
		}

		cache.EXPECT().DeleteAll(buf.chunkPool).Times(1)

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
			ctx:                ctx,
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkSize:          100,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
			},
			log:         slog.Default(),
			nextSegment: make(chan nzb.NzbSegment),
			cancel:      cancel,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			wg:         &sync.WaitGroup{},
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		wg := &sync.WaitGroup{}
		cache.EXPECT().DeleteAll(buf.chunkPool).Times(1)

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

	cache := NewMockChunkCache(ctrl)

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)
	currentDownloading := &sync.Map{}

	segment := nzb.NzbSegment{Id: "1", Number: 1, Bytes: 5}
	groups := []string{"group1"}

	t.Run("Test download segment", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:                ctx,
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Free(mockResource).Times(1)
		expectedBody1 := "body1"

		mockConn.EXPECT().JoinGroup("group1").Return(nil).Times(1)
		nf := &downloadManager{
			ch:         make(chan bool, 1),
			downloaded: true,
			chunk:      make([]byte, 0, 5),
			mx:         &sync.RWMutex{},
		}
		t.Cleanup(func() {
			nf.Reset()
		})
		mockConn.EXPECT().Body("1").Return(io.NopCloser(strings.NewReader(expectedBody1)), nil).Times(1)

		err := buf.downloadSegment(context.Background(), segment, groups, nf)
		assert.NoError(t, err)
		assert.Equal(t, []byte("body1"), nf.chunk)
	})

	// Test error getting connection
	t.Run("Test error getting connection", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:                ctx,
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkSize:          5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}
		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(nil, errors.New("error")).Times(1)

		nf := &downloadManager{
			ch:         nil,
			downloaded: true,
			chunk:      make([]byte, 0, 5),
			mx:         &sync.RWMutex{},
		}
		t.Cleanup(func() {
			nf.Reset()
		})
		err := buf.downloadSegment(context.Background(), segment, groups, nf)
		assert.Error(t, err)
	})

	// Test error finding group
	t.Run("Test error finding group, no retryable", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:                ctx,
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(errors.New("error")).Times(1)

		nf := &downloadManager{
			ch:         nil,
			downloaded: true,
			chunk:      make([]byte, 0, 5),
			mx:         &sync.RWMutex{},
		}
		t.Cleanup(func() {
			nf.Reset()
		})
		err := buf.downloadSegment(context.Background(), segment, groups, nf)
		assert.Error(t, err)
	})

	// Test error getting article body
	t.Run("Test error getting article body, no retryable", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:                ctx,
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nil).Times(1)

		mockConn.EXPECT().Body("1").Return(nil, errors.New("some error")).Times(1)

		nf := &downloadManager{
			ch:         make(chan bool, 1),
			downloaded: true,
			chunk:      make([]byte, 0, 5),
			mx:         &sync.RWMutex{},
		}
		t.Cleanup(func() {
			nf.Reset()
		})
		err := buf.downloadSegment(context.Background(), segment, groups, nf)
		assert.ErrorIs(t, err, ErrCorruptedNzb)
	})

	t.Run("Test retrying after a body retirable error", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:                ctx,
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkSize:          5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(2)
		mockResource.EXPECT().CreationTime().Return(time.Now()).Times(1)

		mockConn2 := nntpcli.NewMockConnection(ctrl)
		mockResource2 := connectionpool.NewMockResource(ctrl)
		mockResource2.EXPECT().Value().Return(mockConn2).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)

		mockConn.EXPECT().JoinGroup("group1").Return(nil).Times(1)
		mockConn.EXPECT().Body("1").Return(nil, &textproto.Error{Code: nntpcli.SegmentAlreadyExistsErrCode}).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource2, nil).Times(1)
		mockPool.EXPECT().Free(mockResource2).Times(1)
		mockConn2.EXPECT().JoinGroup("group1").Return(nil).Times(1)

		expectedBody1 := "body1"
		nf := &downloadManager{
			ch:         make(chan bool, 1),
			downloaded: true,
			chunk:      make([]byte, 0, 5),
			mx:         &sync.RWMutex{},
		}
		t.Cleanup(func() {
			nf.Reset()
		})
		mockConn2.EXPECT().Body("1").Return(io.NopCloser(strings.NewReader(expectedBody1)), nil).Times(1)

		err := buf.downloadSegment(context.Background(), segment, groups, nf)
		assert.NoError(t, err)
		assert.Equal(t, []byte("body1"), nf.chunk)
	})

	t.Run("Test retrying after a group retirable error", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:                ctx,
			fileSize:           3 * 100,
			nzbReader:          nzbReader,
			nzbGroups:          []string{"group1"},
			ptr:                0,
			currentDownloading: currentDownloading,
			cp:                 mockPool,
			chunkPool: &sync.Pool{
				New: func() interface{} {
					return NewDownloadManager(5)
				},
			},
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(2)
		mockResource.EXPECT().CreationTime().Return(time.Now()).Times(1)

		mockConn2 := nntpcli.NewMockConnection(ctrl)
		mockResource2 := connectionpool.NewMockResource(ctrl)
		mockResource2.EXPECT().Value().Return(mockConn2).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(textproto.ProtocolError("some error")).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource2, nil).Times(1)
		mockPool.EXPECT().Free(mockResource2).Times(1)
		mockConn2.EXPECT().JoinGroup("group1").Return(nil).Times(1)

		expectedBody1 := "body1"
		nf := &downloadManager{
			ch:         make(chan bool, 1),
			downloaded: true,
			chunk:      make([]byte, 0, 5),
			mx:         &sync.RWMutex{},
		}
		t.Cleanup(func() {
			nf.Reset()
		})
		mockConn2.EXPECT().Body("1").Return(io.NopCloser(strings.NewReader(expectedBody1)), nil).Times(1)

		err := buf.downloadSegment(context.Background(), segment, groups, nf)

		assert.NoError(t, err)
		assert.Equal(t, []byte("body1"), nf.chunk)
	})
}
