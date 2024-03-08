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

	cache := NewMockCache(ctrl)
	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)
	t.Run("TestBuffer_Read_Empty", func(t *testing.T) {
		currentDownloading := &currentDownloadingMap{}
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:                 context.Background(),
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			directDownloadChunk: make([]byte, 5),
			cp:                  mockPool,
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		// Test empty read
		p := make([]byte, 0)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_Read_PastEnd", func(t *testing.T) {
		currentDownloading := &currentDownloadingMap{}
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:                 context.Background(),
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		// Test read past end of buffer
		buf.ptr = int64(buf.fileSize)
		p := make([]byte, 100)
		n, err := buf.Read(p)
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_Read_OneSegment", func(t *testing.T) {
		currentDownloading := &currentDownloadingMap{}
		t.Cleanup(func() {
			currentDownloading.Range(func(key, value interface{}) bool {
				currentDownloading.Delete(key)
				return true
			})
		})
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:                 context.Background(),
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		nzbReader.EXPECT().GetSegment(0).Return(nzb.NzbSegment{Id: "1", Bytes: 5}, true).Times(1)
		expectedBody := "body1"
		nf := &downloadNotifier{
			ch:         nil,
			downloaded: true,
		}
		cache.EXPECT().Get("1").Return([]byte(expectedBody), nil).Times(1)
		currentDownloading.Store(0, nf)

		p := make([]byte, 5)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody), p[:n])
		assert.Equal(t, int64(5), buf.ptr)
	})

	t.Run("TestBuffer_Direct_Download_Reading", func(t *testing.T) {
		currentDownloading := &currentDownloadingMap{}
		t.Cleanup(func() {
			currentDownloading.Range(func(key, _ interface{}) bool {
				currentDownloading.Delete(key)
				return true
			})
		})
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:                 context.Background(),
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 3,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		nzbReader.EXPECT().GetSegment(0).Return(nzb.NzbSegment{Id: "1", Bytes: 5}, true).Times(1)
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Free(mockResource).Times(1)
		expectedBody := "body1"

		mockConn.EXPECT().JoinGroup("group1").Return(nil).Times(1)
		mockConn.EXPECT().Body("1", gomock.Any()).Do(func(_ any, chunk []byte) {
			copy(chunk, []byte(expectedBody))
		}).Return(nil).Times(1)

		p := make([]byte, 5)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody), p[:n])
		assert.Equal(t, int64(5), buf.ptr)
	})

	t.Run("TestBuffer_Read_TwoSegments", func(t *testing.T) {
		currentDownloading := &currentDownloadingMap{}
		t.Cleanup(func() {
			currentDownloading.Range(func(key, value interface{}) bool {
				currentDownloading.Delete(key)
				return true
			})
		})
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:                 context.Background(),
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		expectedBody1 := "body1"
		expectedBody2 := "body2"

		nzbReader.EXPECT().GetSegment(0).Return(nzb.NzbSegment{Id: "1", Bytes: 5}, true).Times(1)
		nzbReader.EXPECT().GetSegment(1).Return(nzb.NzbSegment{Id: "2", Bytes: 5}, true).Times(1)
		nf := &downloadNotifier{
			ch:         nil,
			downloaded: true,
		}
		cache.EXPECT().Get("1").Return([]byte(expectedBody1), nil).Times(1)
		currentDownloading.Store(0, nf)
		nf2 := &downloadNotifier{
			ch:         nil,
			downloaded: true,
		}
		cache.EXPECT().Get("2").Return([]byte(expectedBody2), nil).Times(1)
		currentDownloading.Store(1, nf2)

		p := make([]byte, 10)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 10, n)
		assert.Equal(t, []byte("body1body2"), p[:n])
		assert.Equal(t, int64(10), buf.ptr)
	})

	t.Run("TestBuffer_Wait_For_Download_Worker", func(t *testing.T) {
		currentDownloading := &currentDownloadingMap{}
		t.Cleanup(func() {
			currentDownloading.Range(func(key, value interface{}) bool {
				currentDownloading.Delete(key)
				return true
			})
		})
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:                 context.Background(),
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		expectedBody := "body1"

		nzbReader.EXPECT().GetSegment(0).Return(nzb.NzbSegment{Id: "1", Bytes: 5}, true).Times(1)
		nf := &downloadNotifier{
			ch:         make(chan bool, 1),
			downloaded: false,
		}
		cache.EXPECT().Get("1").Return([]byte(expectedBody), nil).Times(1)
		currentDownloading.Store(0, nf)

		go func() {
			time.Sleep(100 * time.Millisecond)
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
		currentDownloading := &currentDownloadingMap{}
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
			ctx:                 ctx,
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		nzbReader.EXPECT().GetSegment(0).Return(nzb.NzbSegment{Id: "1", Bytes: 5}, true).Times(1)
		nf := &downloadNotifier{
			ch:         make(chan bool, 1),
			downloaded: false,
		}
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
		currentDownloading := &currentDownloadingMap{}
		t.Cleanup(func() {
			currentDownloading.Range(func(key, value interface{}) bool {
				currentDownloading.Delete(key)
				return true
			})
		})
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		buf := &buffer{
			ctx:                 context.Background(),
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
			},
			nextSegment: make(chan nzb.NzbSegment, 1),
			log:         slog.Default(),
			chunkCache:  cache,
			mx:          &sync.RWMutex{},
		}

		expectedBody := "body1"

		nzbReader.EXPECT().GetSegment(0).Return(nzb.NzbSegment{Id: "1", Bytes: 5}, true).Times(2)
		nf := &downloadNotifier{
			ch:         nil,
			downloaded: true,
		}
		cache.EXPECT().Get("1").Return([]byte(expectedBody), nil).Times(1)
		currentDownloading.Store(0, nf)

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
	cache := NewMockCache(ctrl)
	defer ctrl.Finish()

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)
	currentDownloading := &currentDownloadingMap{}

	t.Run("TestBuffer_ReadAt_Empty", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:                 context.Background(),
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
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
			ctx:                 context.Background(),
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 100),
			chunkSize:           100,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
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
			cp:                 mockPool,
			chunkSize:          5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:        nil,
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		nzbReader.EXPECT().GetSegment(0).Return(nzb.NzbSegment{Id: "1", Bytes: 5}, true).Times(1)
		expectedBody := "body1"
		nf := &downloadNotifier{
			ch:         nil,
			downloaded: true,
		}
		cache.EXPECT().Get("1").Return([]byte(expectedBody), nil).Times(1)
		currentDownloading.Store(0, nf)

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
	cache := NewMockCache(ctrl)

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)
	nzbReader := nzbloader.NewMockNzbReader(ctrl)

	currentDownloading := &currentDownloadingMap{}

	t.Run("Test seek start", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:                 ctx,
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
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
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
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
			ctx:                 ctx,
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
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
			ctx:                 ctx,
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
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
			ctx:                 ctx,
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
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
			chunkSize:          100,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 1,
				maxBufferSizeInMb:  30,
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
	cache := NewMockCache(ctrl)

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)

	currentDownloading := &currentDownloadingMap{}

	t.Run("Test close buffer", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:                 ctx,
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:         slog.Default(),
			nextSegment: make(chan nzb.NzbSegment),
			close:       make(chan struct{}),
			chunkCache:  cache,
			mx:          &sync.RWMutex{},
			wg:          &sync.WaitGroup{},
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
				maxBufferSizeInMb:  30,
			},
			log:         slog.Default(),
			nextSegment: make(chan nzb.NzbSegment),
			close:       make(chan struct{}),
			wg:          &sync.WaitGroup{},
			chunkCache:  cache,
			mx:          &sync.RWMutex{},
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

	cache := NewMockCache(ctrl)

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)
	currentDownloading := &currentDownloadingMap{}

	segment := nzb.NzbSegment{Id: "1", Number: 1, Bytes: 5}
	groups := []string{"group1"}

	t.Run("Test download segment", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:                 ctx,
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Free(mockResource).Times(1)
		expectedBody1 := "body1"

		mockConn.EXPECT().JoinGroup("group1").Return(nil).Times(1)
		mockConn.EXPECT().Body("1", gomock.Any()).Do(func(_ any, chunk []byte) {
			copy(chunk, []byte(expectedBody1))
		}).Return(nil).Times(1)

		part := make([]byte, 5)
		err := buf.downloadSegment(context.Background(), segment, groups, part)
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
				maxBufferSizeInMb:  30,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}
		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(nil, errors.New("error")).Times(1)

		part := make([]byte, 5)
		err := buf.downloadSegment(context.Background(), segment, groups, part)
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
			ctx:                 ctx,
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(errors.New("error")).Times(1)

		part := make([]byte, 5)
		err := buf.downloadSegment(context.Background(), segment, groups, part)
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
			ctx:                 ctx,
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nil).Times(1)

		mockConn.EXPECT().Body("1", gomock.Any()).Return(errors.New("some error")).Times(1)

		part := make([]byte, 5)
		err := buf.downloadSegment(context.Background(), segment, groups, part)
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
				maxBufferSizeInMb:  30,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(2)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(2)
		mockResource.EXPECT().CreationTime().Return(time.Now()).Times(1)

		mockConn2 := nntpcli.NewMockConnection(ctrl)
		mockConn2.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource2 := connectionpool.NewMockResource(ctrl)
		mockResource2.EXPECT().Value().Return(mockConn2).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)

		mockConn.EXPECT().JoinGroup("group1").Return(nil).Times(1)
		mockConn.EXPECT().Body("1", gomock.Any()).Return(&textproto.Error{Code: nntpcli.SegmentAlreadyExistsErrCode}).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource2, nil).Times(1)
		mockPool.EXPECT().Free(mockResource2).Times(1)
		mockConn2.EXPECT().JoinGroup("group1").Return(nil).Times(1)

		expectedBody1 := "body1"
		mockConn2.EXPECT().Body("1", gomock.Any()).DoAndReturn(func(_ any, chunk []byte) error {
			copy(chunk, []byte(expectedBody1))

			return nil
		}).Times(1)

		part := make([]byte, 5)
		err := buf.downloadSegment(context.Background(), segment, groups, part)
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
			ctx:                 ctx,
			fileSize:            3 * 100,
			nzbReader:           nzbReader,
			nzbGroups:           []string{"group1"},
			ptr:                 0,
			currentDownloading:  currentDownloading,
			cp:                  mockPool,
			directDownloadChunk: make([]byte, 5),
			chunkSize:           5,
			dc: downloadConfig{
				maxDownloadRetries: 5,
				maxDownloadWorkers: 0,
				maxBufferSizeInMb:  30,
			},
			log:        slog.Default(),
			chunkCache: cache,
			mx:         &sync.RWMutex{},
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(2)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(2)
		mockResource.EXPECT().CreationTime().Return(time.Now()).Times(1)

		mockConn2 := nntpcli.NewMockConnection(ctrl)
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
		mockConn2.EXPECT().Body("1", gomock.Any()).Do(func(_ any, chunk []byte) {
			copy(chunk, []byte(expectedBody1))
		}).Return(nil).Times(1)

		part := make([]byte, 5)
		err := buf.downloadSegment(context.Background(), segment, groups, part)

		assert.NoError(t, err)
		assert.NotNil(t, part)
		assert.Equal(t, []byte("body1"), part)
	})
}
