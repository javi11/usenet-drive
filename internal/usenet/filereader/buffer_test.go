package filereader

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/chrisfarms/nntp"
	"github.com/golang/mock/gomock"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/javi11/usenet-drive/pkg/yenc"
)

func TestBuffer_Read(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)

	nzbFile := &nzb.NzbFile{
		Segments: []nzb.NzbSegment{
			{Id: "1", Number: 1},
			{Id: "2", Number: 2},
			{Id: "3", Number: 3},
		},
		Groups: []string{"group1", "group2"},
	}

	cache, err := lru.New[string, *yenc.Part](100)
	require.NoError(t, err)
	t.Run("TestBuffer_Read_Empty", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
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

		// Test empty read
		p := make([]byte, 0)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_Read_PastEnd", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
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

		// Test read past end of buffer
		buf.ptr = int64(buf.size)
		p := make([]byte, 100)
		n, err := buf.Read(p)
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_Read_OneSegment", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
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

		// Test read one segment
		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockPool.EXPECT().Get().Return(mockConn, nil).Times(1)
		mockPool.EXPECT().Free(mockConn).Return(nil).Times(1)

		expectedBody := "body1"
		buff, err := generateYencBuff(expectedBody)
		require.NoError(t, err)

		mockConn.EXPECT().Body("<1>").Return(buff, nil).Times(1)
		mockConn.EXPECT().Group("group1").Return(0, 0, 0, nil).Times(1)

		p := make([]byte, 5)
		n, err := buf.Read(p)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody), p[:n])
		assert.Equal(t, int64(5), buf.ptr)
	})

	t.Run("TestBuffer_Read_PreloadOneSegment", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 5,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 1,
			},
			log: nil,
		}

		// Test read one segment
		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockPool.EXPECT().Get().Return(mockConn, nil).Times(2)
		mockPool.EXPECT().Free(mockConn).Return(nil).Times(2)

		expectedBody := "body1"
		buff, err := generateYencBuff(expectedBody)
		require.NoError(t, err)

		expectedBody2 := "body2"
		buff2, err := generateYencBuff(expectedBody2)
		require.NoError(t, err)

		mockConn.EXPECT().Body("<1>").Return(buff, nil).Times(1)
		mockConn.EXPECT().Body("<2>").Return(buff2, nil).Times(1)
		mockConn.EXPECT().Group("group1").Return(0, 0, 0, nil).Times(2)

		p := make([]byte, 5)
		n, err := buf.Read(p)

		// wait preload to finish
		time.Sleep(1000 * time.Millisecond)

		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(expectedBody), p[:n])
		assert.Equal(t, int64(5), buf.ptr)
		assert.Equal(t, 2, buf.cache.Len())
	})

	t.Run("TestBuffer_Read_TwoSegments", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
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

		mockConn := connectionpool.NewMockNntpConnection(ctrl)

		mockPool.EXPECT().Get().Return(mockConn, nil).Times(2)
		mockPool.EXPECT().Free(mockConn).Return(nil).Times(2)

		expectedBody1 := "body1"
		buff1, err := generateYencBuff(expectedBody1)
		require.NoError(t, err)

		expectedBody2 := "body2"
		buff2, err := generateYencBuff(expectedBody2)
		require.NoError(t, err)

		mockConn.EXPECT().Body("<1>").Return(buff1, nil).Times(1)
		mockConn.EXPECT().Body("<2>").Return(buff2, nil).Times(1)
		mockConn.EXPECT().Group("group1").Return(0, 0, 0, nil).Times(2)

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

	nzbFile := &nzb.NzbFile{
		Segments: []nzb.NzbSegment{
			{Id: "1", Number: 1},
			{Id: "2", Number: 2},
			{Id: "3", Number: 3},
		},
		Groups: []string{"group1", "group2"},
	}

	cache, err := lru.New[string, *yenc.Part](100)
	require.NoError(t, err)

	t.Run("TestBuffer_ReadAt_Empty", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
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

		// Test empty read
		p := make([]byte, 0)
		n, err := buf.ReadAt(p, 0)
		assert.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_ReadAt_PastEnd", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 100,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: nil,
		}

		// Test read past end of buffer
		p := make([]byte, 100)
		n, err := buf.ReadAt(p, int64(buf.size))
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, 0, n)
	})

	t.Run("TestBuffer_ReadAt_OneSegment", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
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

		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockPool.EXPECT().Get().Return(mockConn, nil).Times(1)
		mockPool.EXPECT().Free(mockConn).Return(nil).Times(1)

		expectedBody1 := "body1"
		buff, err := generateYencBuff(expectedBody1)
		require.NoError(t, err)

		mockConn.EXPECT().Body("<1>").Return(buff, nil).Times(1)
		mockConn.EXPECT().Group("group1").Return(0, 0, 0, nil).Times(1)

		p := make([]byte, 5)
		n, err := buf.ReadAt(p, 0)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("body1"), p[:n])
	})

	t.Run("TestBuffer_ReadAt_TwoSegments", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
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

		// Test read two segments
		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockPool.EXPECT().Get().Return(mockConn, nil).Times(2)
		mockPool.EXPECT().Free(mockConn).Return(nil).Times(2)

		expectedBody1 := "body2"
		buff1, err := generateYencBuff(expectedBody1)
		require.NoError(t, err)

		expectedBody2 := "body3"
		buff2, err := generateYencBuff(expectedBody2)
		require.NoError(t, err)

		mockConn.EXPECT().Body("<2>").Return(buff1, nil).Times(1)
		mockConn.EXPECT().Body("<3>").Return(buff2, nil).Times(1)
		mockConn.EXPECT().Group("group1").Return(0, 0, 0, nil).Times(2)

		p := make([]byte, 100)
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

	nzbFile := &nzb.NzbFile{
		Segments: []nzb.NzbSegment{
			{Id: "1", Number: 1},
			{Id: "2", Number: 2},
			{Id: "3", Number: 3},
		},
		Groups: []string{"group1", "group2"},
	}

	cache, err := lru.New[string, *yenc.Part](100)
	require.NoError(t, err)

	t.Run("Test seek start", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
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

		// Test seek start
		off, err := buf.Seek(0, io.SeekStart)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), off)
	})

	t.Run("Test seek current", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
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

		// Test seek current
		off, err := buf.Seek(10, io.SeekCurrent)
		assert.NoError(t, err)
		assert.Equal(t, int64(10), off)
	})

	t.Run("Test seek end", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
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

		// Test seek end
		off, err := buf.Seek(-10, io.SeekEnd)
		assert.NoError(t, err)
		assert.Equal(t, int64(buf.size-10), off)
	})

	t.Run("Test seek invalid whence", func(t *testing.T) {

		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
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

		// Test invalid whence
		_, err = buf.Seek(0, 3)
		assert.True(t, errors.Is(err, ErrInvalidWhence))
	})

	t.Run("Test seek negative position", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})
		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 100,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: nil,
		}

		// Test negative position
		_, err = buf.Seek(-1, io.SeekStart)
		assert.True(t, errors.Is(err, ErrSeekNegative))
	})

	t.Run("Test seek too far", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		buf := &buffer{
			size:      3 * 100,
			nzbFile:   nzbFile,
			ptr:       0,
			cache:     cache,
			cp:        mockPool,
			chunkSize: 100,
			dc: downloadConfig{
				maxDownloadRetries:       5,
				maxAheadDownloadSegments: 0,
			},
			log: nil,
		}

		// Test too far
		_, err = buf.Seek(int64(buf.size+1), io.SeekStart)
		assert.True(t, errors.Is(err, ErrSeekTooFar))
	})
}

func TestBuffer_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)

	nzbFile := &nzb.NzbFile{
		Segments: []nzb.NzbSegment{
			{Id: "1", Number: 1},
			{Id: "2", Number: 2},
			{Id: "3", Number: 3},
		},
		Groups: []string{"group1", "group2"},
	}

	cache, err := lru.New[string, *yenc.Part](100)
	require.NoError(t, err)

	buf := &buffer{
		size:      3 * 100,
		nzbFile:   nzbFile,
		ptr:       0,
		cache:     cache,
		cp:        mockPool,
		chunkSize: 100,
		dc: downloadConfig{
			maxDownloadRetries:       5,
			maxAheadDownloadSegments: 0,
		},
		log: nil,
	}

	err = buf.Close()
	assert.NoError(t, err)
}

func TestBuffer_downloadSegment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPool := connectionpool.NewMockUsenetConnectionPool(ctrl)

	nzbFile := &nzb.NzbFile{
		Segments: []nzb.NzbSegment{
			{Id: "1", Number: 1},
			{Id: "2", Number: 2},
			{Id: "3", Number: 3},
		},
		Groups: []string{"group1"},
	}

	cache, err := lru.New[string, *yenc.Part](100)
	require.NoError(t, err)

	buf := &buffer{
		size:      3 * 100,
		nzbFile:   nzbFile,
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

	t.Run("Test download segment", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockPool.EXPECT().Get().Return(mockConn, nil).Times(1)
		mockPool.EXPECT().Free(mockConn).Return(nil).Times(1)
		expectedBody1 := "body1"
		buff, err := generateYencBuff(expectedBody1)
		require.NoError(t, err)

		mockConn.EXPECT().Group("group1").Return(0, 0, 0, nil).Times(1)
		mockConn.EXPECT().Body("<1>").Return(buff, nil).Times(1)

		part, err := buf.downloadSegment(nzbFile.Segments[0], nzbFile.Groups)
		assert.NoError(t, err)
		assert.Equal(t, []byte("body1"), part.Body)
	})

	t.Run("Test segment cached segment", func(t *testing.T) {
		t.Cleanup(func() {
			cache.Purge()
		})

		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockPool.EXPECT().Get().Return(mockConn, nil).Times(1)
		mockPool.EXPECT().Free(mockConn).Return(nil).Times(1)
		expectedBody1 := "body1"
		buff, err := generateYencBuff(expectedBody1)
		require.NoError(t, err)

		mockConn.EXPECT().Group("group1").Return(0, 0, 0, nil).Times(1)
		mockConn.EXPECT().Body("<1>").Return(buff, nil).Times(1)

		part, err := buf.downloadSegment(nzbFile.Segments[0], nzbFile.Groups)
		assert.NoError(t, err)
		assert.Equal(t, []byte("body1"), part.Body)

		mockPool.EXPECT().Get().Return(mockConn, nil).Times(0)
		partCached, err := buf.downloadSegment(nzbFile.Segments[0], nzbFile.Groups)
		assert.NoError(t, err)
		assert.Equal(t, []byte("body1"), partCached.Body)
		assert.Equal(t, cache.Len(), 1)
	})

	// Test error getting connection
	t.Run("Test error getting connection", func(t *testing.T) {
		mockPool.EXPECT().Get().Return(nil, errors.New("error")).Times(1)

		_, err = buf.downloadSegment(nzbFile.Segments[0], nzbFile.Groups)
		assert.Error(t, err)
	})

	// Test error finding group
	t.Run("Test error finding group", func(t *testing.T) {
		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockPool.EXPECT().Get().Return(mockConn, nil).Times(1)
		mockPool.EXPECT().Free(mockConn).Return(nil).Times(1)
		mockConn.EXPECT().Group("group1").Return(0, 0, 0, errors.New("error")).Times(1)

		_, err = buf.downloadSegment(nzbFile.Segments[1], nzbFile.Groups)
		assert.Error(t, err)
	})

	// Test error getting article body
	t.Run("Test error getting article body", func(t *testing.T) {
		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockPool.EXPECT().Get().Return(mockConn, nil).Times(1)
		mockPool.EXPECT().Free(mockConn).Return(nil).Times(1)
		mockConn.EXPECT().Group("group1").Return(0, 0, 0, nil).Times(1)

		mockConn.EXPECT().Body("<3>").Return(nil, errors.New("some error")).Times(1)
		_, err = buf.downloadSegment(nzbFile.Segments[2], nzbFile.Groups)
		assert.ErrorIs(t, err, ErrCorruptedNzb)
	})

	t.Run("Test retrying after a body retirable error", func(t *testing.T) {
		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockConn2 := connectionpool.NewMockNntpConnection(ctrl)

		mockPool.EXPECT().Get().Return(mockConn, nil).Times(1)
		mockPool.EXPECT().Close(mockConn).Return(nil).Times(1)
		mockConn.EXPECT().Group("group1").Return(0, 0, 0, nil).Times(1)
		mockConn.EXPECT().Body("<3>").Return(nil, nntp.Error{Code: 441}).Times(1)

		mockPool.EXPECT().Get().Return(mockConn2, nil).Times(1)
		mockPool.EXPECT().Free(mockConn2).Return(nil).Times(1)
		mockConn2.EXPECT().Group("group1").Return(0, 0, 0, nil).Times(1)

		expectedBody1 := "body1"
		buff, err := generateYencBuff(expectedBody1)
		require.NoError(t, err)

		mockConn2.EXPECT().Body("<3>").Return(buff, nil).Times(1)
		part, err := buf.downloadSegment(nzbFile.Segments[2], nzbFile.Groups)

		assert.NoError(t, err)
		assert.NotNil(t, part)
		assert.Equal(t, []byte("body1"), part.Body)
	})

	t.Run("Test retrying after a group retirable error", func(t *testing.T) {
		mockConn := connectionpool.NewMockNntpConnection(ctrl)
		mockConn2 := connectionpool.NewMockNntpConnection(ctrl)

		mockPool.EXPECT().Get().Return(mockConn, nil).Times(1)
		mockPool.EXPECT().Close(mockConn).Return(nil).Times(1)
		mockConn.EXPECT().Group("group1").Return(0, 0, 0, nntp.Error{Code: 441}).Times(1)

		mockPool.EXPECT().Get().Return(mockConn2, nil).Times(1)
		mockPool.EXPECT().Free(mockConn2).Return(nil).Times(1)
		mockConn2.EXPECT().Group("group1").Return(0, 0, 0, nil).Times(1)

		expectedBody1 := "body1"
		buff, err := generateYencBuff(expectedBody1)
		require.NoError(t, err)

		mockConn2.EXPECT().Body("<3>").Return(buff, nil).Times(1)
		part, err := buf.downloadSegment(nzbFile.Segments[2], nzbFile.Groups)

		assert.NoError(t, err)
		assert.NotNil(t, part)
		assert.Equal(t, []byte("body1"), part.Body)
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
