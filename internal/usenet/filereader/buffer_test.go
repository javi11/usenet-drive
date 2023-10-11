package filereader

import (
	"errors"
	"io"
	"log/slog"
	"net/textproto"
	"testing"

	"github.com/golang/mock/gomock"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/javi11/usenet-drive/pkg/yenc"
)

func TestBuffer_Read_Empty(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test empty read
	p := make([]byte, 0)
	n, err := buf.Read(p)
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestBuffer_Read_PastEnd(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test read past end of buffer
	buf.ptr = int64(buf.size)
	p := make([]byte, 100)
	n, err := buf.Read(p)
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 0, n)
}

func TestBuffer_Read_OneSegment(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test read one segment
	mockConn := connectionpool.NewMockNntpConnection(ctrl)
	mockPool.EXPECT().Get().Return(mockConn, nil)
	mockPool.EXPECT().Free(mockConn).Return(nil)
	mockConn.EXPECT().Body("<1>").Return([]byte("=ybegin line=128 size=100\nbody1\n=yend\n"), nil)
	p := make([]byte, 100)
	n, err := buf.Read(p)
	assert.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, []byte("body1\n"), p[:n])
	assert.Equal(t, int64(6), buf.ptr)
}

func TestBuffer_Read_TwoSegments(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test read two segments
	mockConn2 := connectionpool.NewMockNntpConnection(ctrl)
	mockPool.EXPECT().Get().Return(mockConn2, nil)
	mockPool.EXPECT().Free(mockConn2).Return(nil)
	mockConn2.EXPECT().Body("<2>").Return([]byte("=ybegin line=128 size=100\nbody2\n=yend\n"), nil)
	mockConn2.EXPECT().Body("<3>").Return([]byte("=ybegin line=128 size=100\nbody3\n=yend\n"), nil)
	p := make([]byte, 100)
	n, err := buf.Read(p)
	assert.NoError(t, err)
	assert.Equal(t, 94, n)
	assert.Equal(t, []byte("body1\nbody2\nbody3\n"), p[:n])
	assert.Equal(t, int64(100), buf.ptr)
}

func TestBuffer_ReadAt_Empty(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test empty read
	p := make([]byte, 0)
	n, err := buf.ReadAt(p, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestBuffer_ReadAt_PastEnd(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test read past end of buffer
	p := make([]byte, 100)
	n, err := buf.ReadAt(p, int64(buf.size))
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 0, n)
}

func TestBuffer_ReadAt_OneSegment(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test read one segment
	mockConn := connectionpool.NewMockNntpConnection(ctrl)
	mockPool.EXPECT().Get().Return(mockConn, nil)
	mockPool.EXPECT().Free(mockConn).Return(nil)
	mockConn.EXPECT().Body("<1>").Return([]byte("=ybegin line=128 size=100\nbody1\n=yend\n"), nil)
	p := make([]byte, 100)
	n, err := buf.ReadAt(p, 0)
	assert.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, []byte("body1\n"), p[:n])
}

func TestBuffer_ReadAt_TwoSegments(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test read two segments
	mockConn2 := connectionpool.NewMockNntpConnection(ctrl)
	mockPool.EXPECT().Get().Return(mockConn2, nil)
	mockPool.EXPECT().Free(mockConn2).Return(nil)
	mockConn2.EXPECT().Body("<2>").Return([]byte("=ybegin line=128 size=100\nbody2\n=yend\n"), nil)
	mockConn2.EXPECT().Body("<3>").Return([]byte("=ybegin line=128 size=100\nbody3\n=yend\n"), nil)
	p := make([]byte, 100)
	n, err := buf.ReadAt(p, 6)
	assert.NoError(t, err)
	assert.Equal(t, 94, n)
	assert.Equal(t, []byte("body1\nbody2\nbody3\n"), p[:n])
}

func TestBuffer_Seek_Start(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test seek start
	off, err := buf.Seek(0, io.SeekStart)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), off)
}

func TestBuffer_Seek_Current(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test seek current
	off, err := buf.Seek(10, io.SeekCurrent)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), off)
}

func TestBuffer_Seek_End(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test seek end
	off, err := buf.Seek(-10, io.SeekEnd)
	assert.NoError(t, err)
	assert.Equal(t, int64(buf.size-10), off)
}

func TestBuffer_Seek_InvalidWhence(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test invalid whence
	_, err = buf.Seek(0, 3)
	assert.True(t, errors.Is(err, errors.New("Seek: invalid whence")))
}

func TestBuffer_Seek_NegativePosition(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test negative position
	_, err = buf.Seek(-1, io.SeekStart)
	assert.True(t, errors.Is(err, errors.New("Seek: negative position")))
}

func TestBuffer_Seek_TooFar(t *testing.T) {
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test too far
	_, err = buf.Seek(int64(buf.size+1), io.SeekStart)
	assert.True(t, errors.Is(err, errors.New("Seek: too far")))
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
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
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
		Groups: []string{"group1", "group2"},
	}

	cache, err := lru.New[string, *yenc.Part](100)
	require.NoError(t, err)

	buf := &buffer{
		size:               3 * 100,
		nzbFile:            nzbFile,
		ptr:                0,
		cache:              cache,
		cp:                 mockPool,
		chunkSize:          100,
		maxDownloadRetries: 5,
		log:                nil,
	}

	// Test successful download
	mockConn := connectionpool.NewMockNntpConnection(ctrl)
	mockPool.EXPECT().Get().Return(mockConn, nil)
	mockPool.EXPECT().Free(mockConn).Return(nil)
	mockConn.EXPECT().Body("<1>").Return([]byte("=ybegin line=128 size=100\nbody1\n=yend\n"), nil)
	part, err := buf.downloadSegment(nzbFile.Segments[0], nzbFile.Groups, 0)
	assert.NoError(t, err)
	assert.Equal(t, []byte("body1\n"), part.Body)

	// Test error getting connection
	mockPool.EXPECT().Get().Return(nil, errors.New("error"))
	_, err = buf.downloadSegment(nzbFile.Segments[0], nzbFile.Groups, 0)
	assert.Error(t, err)

	// Test error finding group
	mockConn2 := connectionpool.NewMockNntpConnection(ctrl)
	mockPool.EXPECT().Get().Return(mockConn2, nil)
	mockPool.EXPECT().Free(mockConn2).Return(nil)
	mockConn2.EXPECT().Body("<2>").Return(nil, &textproto.Error{Code: 411})
	_, err = buf.downloadSegment(nzbFile.Segments[1], nzbFile.Groups, 0)
	assert.Error(t, err)

	// Test error getting article body
	mockConn3 := connectionpool.NewMockNntpConnection(ctrl)
	mockPool.EXPECT().Get().Return(mockConn3, nil)
	mockPool.EXPECT().Free(mockConn3).Return(nil)
	mockConn3.EXPECT().Body("<3>").Return(nil, &textproto.Error{Code: 430})
	_, err = buf.downloadSegment(nzbFile.Segments[2], nzbFile.Groups, 0)
	assert.Error(t, err)

	// Test corrupted article body
	mockConn4 := connectionpool.NewMockNntpConnection(ctrl)
	mockPool.EXPECT().Get().Return(mockConn4, nil)
	mockPool.EXPECT().Free(mockConn4).Return(nil)
	mockConn4.EXPECT().Body("<4>").Return([]byte("=ybegin line=128 size=100\nbody1\n=yend\n"), nil)
	_, err = buf.downloadSegment(nzb.NzbSegment{Id: "4", Number: 4}, nzbFile.Groups, 0)
	assert.True(t, errors.Is(err, ErrCorruptedNzb))
}

func TestNewBuffer(t *testing.T) {
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

	log := &slog.Logger{}

	buf, err := NewBuffer(nzbFile, 3*100, 100, mockPool, log)
	assert.NoError(t, err)
	assert.NotNil(t, buf)
	assert.Equal(t, 100, buf.chunkSize)
	assert.Equal(t, 3*100, buf.size)
	assert.Equal(t, nzbFile, buf.nzbFile)
	assert.NotNil(t, buf.cache)
	assert.Equal(t, mockPool, buf.cp)
	assert.Equal(t, 5, buf.maxDownloadRetries)
	assert.Equal(t, log, buf.log)
}
