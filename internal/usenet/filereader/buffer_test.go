package filereader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log/slog"
	"net/textproto"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/javi11/usenet-drive/pkg/yenc"
)

func TestBuffer_Read(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)

	cache := NewMockCache(ctrl)
	t.Run("TestBuffer_Read_Empty", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:       context.Background(),
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}

		// Test empty read
		p := make([]byte, 0)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_Read_PastEnd", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:       context.Background(),
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}

		// Test read past end of buffer
		buf.ptr = int64(buf.size)
		p := make([]byte, 100)
		n, err := buf.Read(p)
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_Read_OneSegment", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		nzbReader.EXPECT().GetSegment(0).Return(&nzb.NzbSegment{Id: "1", Number: 1, Bytes: 750000}, true).Times(1)

		buf := &buffer{
			ctx:       context.Background(),
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}

		// Test read one segment
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Free(mockResource).Times(1)

		expectedBody := "body1"
		buff, err := generateYencBuff(expectedBody)
		require.NoError(t, err)

		cache.EXPECT().Set("1", gomock.Any()).Return(nil).Times(1)
		cache.EXPECT().Get("1").Return(nil, errors.New("not found")).Times(1)
		mockConn.EXPECT().Body("<1>").Return(buff, nil).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nntpcli.Group{}, nil).Times(1)

		p := make([]byte, 5)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody), p[:n])
		assert.Equal(t, int64(5), buf.ptr)
	})

	t.Run("TestBuffer_Read_PreloadOneSegment", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		nzbReader.EXPECT().GetGroups().Return([]string{"group1"}, nil).Times(1)
		nzbReader.EXPECT().GetSegment(0).Return(&nzb.NzbSegment{Id: "1", Number: 1, Bytes: 750000}, true).Times(1)
		nzbReader.EXPECT().GetSegment(1).Return(&nzb.NzbSegment{Id: "2", Number: 2, Bytes: 750000}, true).Times(2)
		nzbReader.EXPECT().GetSegment(2).Return(&nzb.NzbSegment{Id: "3", Number: 3, Bytes: 750000}, true).Times(1)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})

		buf, err := NewBuffer(ctx, nzbReader, 3*100, 5, downloadConfig{
			maxDownloadRetries:       5,
			maxAheadDownloadSegments: 1,
		}, mockPool, cache, slog.Default())
		assert.NoError(t, err)

		// Test read one segment
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(3)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(3)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(3)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(3)
		mockPool.EXPECT().Free(mockResource).Times(3)

		expectedBody := "body1"
		buff, err := generateYencBuff(expectedBody)
		assert.NoError(t, err)

		expectedBody2 := "body2"
		buff2, err := generateYencBuff(expectedBody2)
		assert.NoError(t, err)

		expectedBody3 := "body3"
		buff3, err := generateYencBuff(expectedBody3)
		assert.NoError(t, err)

		mockConn.EXPECT().Body("<1>").Return(buff, nil).Times(1)
		mockConn.EXPECT().Body("<2>").Return(buff2, nil).Times(1)
		mockConn.EXPECT().Body("<3>").Return(buff3, nil).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nntpcli.Group{}, nil).Times(3)

		// Segment 1 and 2 are loaded in parallel due to the preload
		cache.EXPECT().Has("2").Return(false).Times(1)
		cache.EXPECT().Get("1").Return(nil, errors.New("not found")).Times(1)
		cache.EXPECT().Get("2").Return(nil, errors.New("not found")).Times(1)

		cache.EXPECT().Set("1", gomock.Any()).Return(nil).Times(1)
		cache.EXPECT().Set("2", gomock.Any()).Return(nil).Times(1)

		p := make([]byte, 5)
		n, err := buf.Read(p)

		// wait preload to finish
		time.Sleep(1000 * time.Millisecond)

		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody), p[:n])

		// Segment 3 is loaded in parallel due to the preload
		cache.EXPECT().Has("3").Return(false).Times(1)
		cache.EXPECT().Get("3").Return(nil, errors.New("not found")).Times(1)
		cache.EXPECT().Set("3", gomock.Any()).Return(nil).Times(1)

		cache.EXPECT().Get("2").Return(bytes.NewBufferString(expectedBody2).Bytes(), nil).Times(1)

		n, err = buf.Read(p)
		assert.NoError(t, err)

		// wait 3th preload to finish
		time.Sleep(1000 * time.Millisecond)

		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody2), p[:n])
	})

	t.Run("TestBuffer_Read_TwoSegments", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		nzbReader.EXPECT().GetSegment(0).Return(&nzb.NzbSegment{Id: "1", Number: 1, Bytes: 750000}, true).Times(1)
		nzbReader.EXPECT().GetSegment(1).Return(&nzb.NzbSegment{Id: "2", Number: 2, Bytes: 750000}, true).Times(1)

		buf := &buffer{
			ctx:       context.Background(),
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(2)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(2)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(2)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(2)
		mockPool.EXPECT().Free(mockResource).Times(2)

		expectedBody1 := "body1"
		buff1, err := generateYencBuff(expectedBody1)
		require.NoError(t, err)

		expectedBody2 := "body2"
		buff2, err := generateYencBuff(expectedBody2)
		require.NoError(t, err)

		cache.EXPECT().Get("1").Return(nil, errors.New("not found")).Times(1)
		cache.EXPECT().Set("1", gomock.Any()).Return(nil).Times(1)
		mockConn.EXPECT().Body("<1>").Return(buff1, nil).Times(1)

		cache.EXPECT().Get("2").Return(nil, errors.New("not found")).Times(1)
		cache.EXPECT().Set("2", gomock.Any()).Return(nil).Times(1)
		mockConn.EXPECT().Body("<2>").Return(buff2, nil).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nntpcli.Group{}, nil).Times(2)

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
	cache := NewMockCache(ctrl)

	t.Run("TestBuffer_ReadAt_Empty", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		buf := &buffer{
			ctx:       context.Background(),
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
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
			ctx:       context.Background(),
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 100,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}

		// Test read past end of buffer
		p := make([]byte, 100)
		n, err := buf.ReadAt(p, int64(buf.size))
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_ReadAt_OneSegment", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		nzbReader.EXPECT().GetSegment(0).Return(&nzb.NzbSegment{Id: "1", Number: 1, Bytes: 750000}, true).Times(1)
		buf := &buffer{
			ctx:       context.Background(),
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: nil,
		}

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Free(mockResource).Times(1)

		expectedBody1 := "body1"
		buff, err := generateYencBuff(expectedBody1)
		require.NoError(t, err)

		cache.EXPECT().Get("1").Return(nil, errors.New("not found")).Times(1)
		cache.EXPECT().Set("1", gomock.Any()).Return(nil).Times(1)
		mockConn.EXPECT().Body("<1>").Return(buff, nil).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nntpcli.Group{}, nil).Times(1)

		p := make([]byte, 5)
		n, err := buf.ReadAt(p, 0)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("body1"), p[:n])
	})

	t.Run("TestBuffer_Read_PreloadOneSegment", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		nzbReader.EXPECT().GetGroups().Return([]string{"group1"}, nil).Times(1)
		nzbReader.EXPECT().GetSegment(1).Return(&nzb.NzbSegment{Id: "2", Number: 2, Bytes: 750000}, true).Times(1)
		// Preload make the segment 2 to be loaded twice
		nzbReader.EXPECT().GetSegment(2).Return(&nzb.NzbSegment{Id: "3", Number: 3, Bytes: 750000}, true).Times(2)
		nzbReader.EXPECT().GetSegment(3).Return(nil, false).Times(1)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})

		buf, err := NewBuffer(ctx, nzbReader, 3*100, 5, downloadConfig{
			maxDownloadRetries:       5,
			maxAheadDownloadSegments: 1,
		}, mockPool, cache, slog.Default())
		assert.NoError(t, err)

		// Test read one segment
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(2)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(2)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(2)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(2)
		mockPool.EXPECT().Free(mockResource).Times(2)

		expectedBody2 := "body2"
		buff2, err := generateYencBuff(expectedBody2)
		assert.NoError(t, err)

		expectedBody3 := "body3"
		buff3, err := generateYencBuff(expectedBody3)
		assert.NoError(t, err)

		mockConn.EXPECT().Body("<2>").Return(buff2, nil).Times(1)
		mockConn.EXPECT().Body("<3>").Return(buff3, nil).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nntpcli.Group{}, nil).Times(2)

		// Segment 1 and 2 are loaded in parallel due to the preload
		cache.EXPECT().Has("3").Return(false).Times(1)
		cache.EXPECT().Get("2").Return(nil, errors.New("not found")).Times(1)
		cache.EXPECT().Get("3").Return(nil, errors.New("not found")).Times(1)

		cache.EXPECT().Set("2", gomock.Any()).Return(nil).Times(1)
		cache.EXPECT().Set("3", gomock.Any()).Return(nil).Times(1)

		p := make([]byte, 5)
		n, err := buf.ReadAt(p, 5)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody2), p[:n])

		// wait preload to finish
		time.Sleep(1000 * time.Millisecond)

		cache.EXPECT().Get("3").Return(bytes.NewBufferString(expectedBody3).Bytes(), nil).Times(1)

		n, err = buf.ReadAt(p, 10)
		assert.NoError(t, err)

		// wait 3th preload to finish
		time.Sleep(1000 * time.Millisecond)

		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody3), p[:n])
	})

	t.Run("TestBuffer_ReadAt_TwoSegments", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		nzbReader.EXPECT().GetSegment(1).Return(&nzb.NzbSegment{Id: "2", Number: 2, Bytes: 750000}, true).Times(1)
		nzbReader.EXPECT().GetSegment(2).Return(&nzb.NzbSegment{Id: "3", Number: 3, Bytes: 750000}, true).Times(1)

		buf := &buffer{
			ctx:       context.Background(),
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}

		// Test read two segments
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(2)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(2)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(2)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(2)
		mockPool.EXPECT().Free(mockResource).Times(2)

		expectedBody1 := "body2"
		buff1, err := generateYencBuff(expectedBody1)
		require.NoError(t, err)

		expectedBody2 := "body3"
		buff2, err := generateYencBuff(expectedBody2)
		require.NoError(t, err)

		cache.EXPECT().Get("2").Return(nil, errors.New("not found")).Times(1)
		cache.EXPECT().Set("2", gomock.Any()).Return(nil).Times(1)
		mockConn.EXPECT().Body("<2>").Return(buff1, nil).Times(1)

		cache.EXPECT().Get("3").Return(nil, errors.New("not found")).Times(1)
		cache.EXPECT().Set("3", gomock.Any()).Return(nil).Times(1)
		mockConn.EXPECT().Body("<3>").Return(buff2, nil).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nntpcli.Group{}, nil).Times(2)

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

	cache := NewMockCache(ctrl)

	t.Run("Test seek start", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
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
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
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
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}

		// Test seek end
		off, err := buf.Seek(-10, io.SeekEnd)
		assert.NoError(t, err)
		assert.Equal(t, int64(buf.size-10), off)
	})

	t.Run("Test seek invalid whence", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
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
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}

		// Test negative position
		_, err := buf.Seek(-1, io.SeekStart)
		assert.True(t, errors.Is(err, ErrSeekNegative))
	})

	t.Run("Test seek too far", func(t *testing.T) {
		buf := &buffer{
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 100,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}

		// Test too far
		_, err := buf.Seek(int64(buf.size+1), io.SeekStart)
		assert.True(t, errors.Is(err, ErrSeekTooFar))
	})
}

func TestBuffer_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)

	cache := NewMockCache(ctrl)

	t.Run("Test close buffer", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		nzbReader.EXPECT().Close().Return().Times(1)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log:         slog.Default(),
			closed:      make(chan bool),
			nextSegment: make(chan *nzb.NzbSegment),
		}

		err := buf.Close()
		assert.NoError(t, err)
	})

	t.Run("Test close buffer with download ahead", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)
		nzbReader.EXPECT().Close().Return().Times(1)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})

		closed := make(chan bool)

		buf := &buffer{
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 100,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 1,
			},
			log:         slog.Default(),
			closed:      closed,
			nextSegment: make(chan *nzb.NzbSegment),
			wg:          &sync.WaitGroup{},
		}

		wg := &sync.WaitGroup{}

		wg.Add(1)
		go func() {
			defer wg.Done()
			// wait for close to be called
			c := <-closed

			assert.Equal(t, c, true)
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
	cache := NewMockCache(ctrl)

	segment := &nzb.NzbSegment{Id: "1", Number: 1, Bytes: 750000}
	groups := []string{"group1"}

	t.Run("Test download segment", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Free(mockResource).Times(1)
		expectedBody1 := "body1"
		buff, err := generateYencBuff(expectedBody1)
		require.NoError(t, err)

		cache.EXPECT().Get("1").Return(nil, errors.New("not found")).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nntpcli.Group{}, nil).Times(1)
		mockConn.EXPECT().Body("<1>").Return(buff, nil).Times(1)
		cache.EXPECT().Set("1", gomock.Any()).Return(nil).Times(1)

		part, err := buf.downloadSegment(context.Background(), segment, groups)
		assert.NoError(t, err)
		assert.Equal(t, []byte("body1"), part)
	})

	t.Run("Test cached segment", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}

		mockConn := nntpcli.NewMockConnection(ctrl)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(0)

		expectedBody1 := "body1"

		cache.EXPECT().Get("1").Return(bytes.NewBufferString(expectedBody1).Bytes(), nil).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(0)
		partCached, err := buf.downloadSegment(context.Background(), segment, groups)
		assert.NoError(t, err)
		assert.Equal(t, []byte("body1"), partCached)
	})

	// Test error getting connection
	t.Run("Test error getting connection", func(t *testing.T) {
		nzbReader := nzbloader.NewMockNzbReader(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(func() {
			cancel()
		})
		buf := &buffer{
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}
		cache.EXPECT().Get("1").Return(nil, errors.New("not found")).Times(1)
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
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       1,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		cache.EXPECT().Get("1").Return(nil, errors.New("not found")).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nntpcli.Group{}, errors.New("error")).Times(1)

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
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nntpcli.Group{}, nil).Times(1)

		cache.EXPECT().Get("1").Return(nil, errors.New("not found")).Times(1)
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
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockConn2 := nntpcli.NewMockConnection(ctrl)
		mockConn2.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn2.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource2 := connectionpool.NewMockResource(ctrl)
		mockResource2.EXPECT().Value().Return(mockConn2).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)

		cache.EXPECT().Get("1").Return(nil, errors.New("not found")).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nntpcli.Group{}, nil).Times(1)
		mockConn.EXPECT().Body("<1>").Return(nil, &textproto.Error{Code: nntpcli.SegmentAlreadyExistsErrCode}).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource2, nil).Times(1)
		mockPool.EXPECT().Free(mockResource2).Times(1)
		mockConn2.EXPECT().JoinGroup("group1").Return(nntpcli.Group{}, nil).Times(1)

		expectedBody1 := "body1"
		buff, err := generateYencBuff(expectedBody1)
		require.NoError(t, err)

		mockConn2.EXPECT().Body("<1>").Return(buff, nil).Times(1)
		cache.EXPECT().Set("1", gomock.Any()).Return(nil).Times(1)

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
			ctx:       ctx,
			size:      3 * 100,
			nzbReader: nzbReader,
			nzbGroups: []string{"group1"},
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: slog.Default(),
		}
		mockConn := nntpcli.NewMockConnection(ctrl)
		mockConn.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource := connectionpool.NewMockResource(ctrl)
		mockResource.EXPECT().Value().Return(mockConn).Times(1)

		mockConn2 := nntpcli.NewMockConnection(ctrl)
		mockConn2.EXPECT().CurrentJoinedGroup().Return("").Times(1)
		mockConn2.EXPECT().Provider().Return(nntpcli.Provider{JoinGroup: true}).Times(1)
		mockResource2 := connectionpool.NewMockResource(ctrl)
		mockResource2.EXPECT().Value().Return(mockConn2).Times(1)

		cache.EXPECT().Get("1").Return(nil, errors.New("not found")).Times(1)
		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource, nil).Times(1)
		mockPool.EXPECT().Close(mockResource).Times(1)
		mockConn.EXPECT().JoinGroup("group1").Return(nntpcli.Group{}, textproto.ProtocolError("some error")).Times(1)

		mockPool.EXPECT().GetDownloadConnection(gomock.Any()).Return(mockResource2, nil).Times(1)
		mockPool.EXPECT().Free(mockResource2).Times(1)
		mockConn2.EXPECT().JoinGroup("group1").Return(nntpcli.Group{}, nil).Times(1)

		expectedBody1 := "body1"
		buff, err := generateYencBuff(expectedBody1)
		require.NoError(t, err)

		mockConn2.EXPECT().Body("<1>").Return(buff, nil).Times(1)
		cache.EXPECT().Set("1", gomock.Any()).Return(nil).Times(1)
		part, err := buf.downloadSegment(context.Background(), segment, groups)

		assert.NoError(t, err)
		assert.NotNil(t, part)
		assert.Equal(t, []byte("body1"), part)
	})

}

func generateYencBuff(s string) (*bytes.Buffer, error) {
	body := []byte(s)
	buff := &bytes.Buffer{}
	buff.WriteString(fmt.Sprintf("=ybegin part=1 total=1 line=128 size=%v\r\n", len(body)))
	buff.WriteString(fmt.Sprintf("=ypart begin=1 end=%v\r\n", len(body)))
	err := yenc.Encode(body, buff)
	if err != nil {
		return nil, err
	}
	h := crc32.NewIEEE()
	h.Write(body)
	buff.WriteString(fmt.Sprintf("=yend size=%d part=%d pcrc32=%08X\r\n", len(body), 1, h.Sum32()))

	return buff, nil
}
