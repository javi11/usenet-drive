package filereader

//go:generate mockgen -source=./buffer.go -destination=./buffer_mock.go -package=filereader Buffer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/avast/retry-go"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/javi11/usenet-drive/pkg/yenc"
)

var (
	ErrInvalidWhence = errors.New("seek: invalid whence")
	ErrSeekNegative  = errors.New("seek: negative position")
	ErrSeekTooFar    = errors.New("seek: too far")
)

type Buffer interface {
	io.ReaderAt
	io.ReadSeeker
	io.Closer
}

// Buf is a Buffer working on a slice of bytes.
type buffer struct {
	ctx              context.Context
	size             int
	nzbFile          *nzb.NzbFile
	ptr              int64
	cache            Cache
	cp               connectionpool.UsenetConnectionPool
	chunkSize        int
	dc               downloadConfig
	log              *slog.Logger
	closed           chan bool
	nextSegmentIndex chan int
	wg               *sync.WaitGroup
}

// NewBuffer creates a new data volume based on a buffer
func NewBuffer(
	ctx context.Context,
	nzbFile *nzb.NzbFile,
	size int,
	chunkSize int,
	dc downloadConfig,
	cp connectionpool.UsenetConnectionPool,
	cache Cache,
	log *slog.Logger,
) (Buffer, error) {
	buffer := &buffer{
		ctx:              ctx,
		chunkSize:        chunkSize,
		size:             size,
		nzbFile:          nzbFile,
		cache:            cache,
		cp:               cp,
		dc:               dc,
		log:              log,
		nextSegmentIndex: make(chan int),
		closed:           make(chan bool),
		wg:               &sync.WaitGroup{},
	}

	if dc.maxAheadDownloadSegments > 0 {
		buffer.wg.Add(1)
		go buffer.downloadBoost(ctx, buffer.nextSegmentIndex)
	}

	return buffer, nil
}

// Seek sets the offset for the next Read or Write on the buffer to offset,
// interpreted according to whence:
//
//	0 (os.SEEK_SET) means relative to the origin of the file
//	1 (os.SEEK_CUR) means relative to the current offset
//	2 (os.SEEK_END) means relative to the end of the file
//
// It returns the new offset and an error, if any.
func (v *buffer) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart: // Relative to the origin of the file
		abs = offset
	case io.SeekCurrent: // Relative to the current offset
		abs = int64(v.ptr) + offset
	case io.SeekEnd: // Relative to the end
		abs = int64(v.size) + offset
	default:
		return 0, ErrInvalidWhence
	}
	if abs < 0 {
		return 0, ErrSeekNegative
	}
	if abs > int64(v.size) {
		return 0, ErrSeekTooFar
	}
	v.ptr = abs
	return abs, nil
}

// Close the buffer. Currently no effect.
func (v *buffer) Close() error {
	if v.dc.maxAheadDownloadSegments > 0 {
		v.closed <- true
		v.wg.Wait()
	}

	return nil
}

// Read reads len(p) byte from the Buffer starting at the current offset.
// It returns the number of bytes read and an error if any.
// Returns io.EOF error if pointer is at the end of the Buffer.
func (v *buffer) Read(p []byte) (int, error) {
	n := 0

	if len(p) == 0 {
		return n, nil
	}
	if v.ptr >= int64(v.size) {
		return n, io.EOF
	}

	currentSegment := int(float64(v.ptr) / float64(v.chunkSize))
	beginReadAt := max((int(v.ptr) - (currentSegment * v.chunkSize)), 0)

	for i, segment := range v.nzbFile.Segments[currentSegment:] {
		if n >= len(p) {
			break
		}

		nextSegmentIndex := currentSegment + i + 1
		// Preload next segments
		for j := 0; j < v.dc.maxAheadDownloadSegments; j++ {
			index := nextSegmentIndex + j

			if index >= len(v.nzbFile.Segments) || v.cache.Has(v.nzbFile.Segments[index].Id) {
				break
			}

			// Preload next segments
			v.nextSegmentIndex <- index
		}

		chunk, err := v.downloadSegment(v.ctx, segment, v.nzbFile.Groups)
		if err != nil {
			if errors.Is(err, ErrCorruptedNzb) {
				return n, err
			}
			break
		}
		beginWriteAt := n
		n += copy(p[beginWriteAt:], chunk[beginReadAt:])
		beginReadAt = 0
	}
	v.ptr += int64(n)

	return n, nil
}

// ReadAt reads len(b) bytes from the Buffer starting at byte offset off.
// It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b).
// At end of file, that error is io.EOF.
func (v *buffer) ReadAt(p []byte, off int64) (int, error) {
	n := 0

	if len(p) == 0 {
		return n, nil
	}
	if off >= int64(v.size) {
		return n, io.EOF
	}

	currentSegment := int(float64(off) / float64(v.chunkSize))
	beginReadAt := max((int(off) - (currentSegment * v.chunkSize)), 0)

	for i, segment := range v.nzbFile.Segments[currentSegment:] {
		if n >= len(p) {
			break
		}
		nextSegmentIndex := currentSegment + i + 1
		// Preload next segments
		for j := 0; j < v.dc.maxAheadDownloadSegments; j++ {
			index := nextSegmentIndex + j

			if index >= len(v.nzbFile.Segments) || v.cache.Has(v.nzbFile.Segments[index].Id) {
				break
			}

			// Preload next segments
			v.nextSegmentIndex <- index
		}

		chunk, err := v.downloadSegment(v.ctx, segment, v.nzbFile.Groups)
		if err != nil {
			break
		}
		beginWriteAt := n
		n += copy(p[beginWriteAt:], chunk[beginReadAt:])
		beginReadAt = 0
	}

	return n, nil
}

func (v *buffer) downloadSegment(ctx context.Context, segment *nzb.NzbSegment, groups []string) ([]byte, error) {
	hit, err := v.cache.Get(segment.Id)

	var chunk []byte
	if err == nil {
		chunk = hit
	} else {
		var conn nntpcli.Connection
		segment := segment
		retryErr := retry.Do(func() error {
			c, err := v.cp.Get()
			if err != nil {
				if conn != nil {
					e := v.cp.Close(conn)
					if e != nil {
						v.log.DebugContext(ctx, "Error closing connection on downloading a file.", "error", e)
					}
				}
				v.log.ErrorContext(ctx, "Error getting nntp connection:", "error", err, "segment", segment.Number)

				// Retry
				return syscall.ETIMEDOUT
			}
			conn = c

			err = usenet.FindGroup(conn, groups)
			if err != nil {
				return err
			}

			body, err := conn.Body(fmt.Sprintf("<%v>", segment.Id))
			if err != nil {
				return err
			}

			yread, err := yenc.Decode(body)
			if err != nil {
				return err
			}

			chunk = yread.Body

			if err = v.cp.Free(conn); err != nil {
				v.log.DebugContext(ctx, "Error freeing connection on downloading a file.", "error", err)
			}

			return nil
		},
			retry.Context(ctx),
			retry.Attempts(uint(v.dc.maxDownloadRetries)),
			retry.DelayType(retry.FixedDelay),
			retry.RetryIf(func(err error) bool {
				return nntpcli.IsRetryableError(err)
			}),
			retry.OnRetry(func(n uint, err error) {
				v.log.InfoContext(ctx, "Retrying download", "error", err, "segment", segment.Id, "retry", n)

				if conn != nil {
					err = v.cp.Close(conn)
					if err != nil {
						v.log.DebugContext(ctx, "Error closing connection.", "error", err)
					}
				}
			}),
		)
		if retryErr != nil {
			err := retryErr

			if conn != nil {
				err = v.cp.Close(conn)
				if err != nil {
					v.log.DebugContext(ctx, "Error closing connection.", "error", err)
				}
			}

			var e retry.Error
			if errors.As(err, &e) {
				err = errors.Join(e.WrappedErrors()...)
			}

			if errors.Is(err, context.Canceled) {
				return nil, err
			}

			return nil, errors.Join(ErrCorruptedNzb, err)
		}

		err := v.cache.Set(segment.Id, chunk)
		if err != nil {
			v.log.ErrorContext(ctx, "Error caching segment.", "error", err, "segment", segment.Number)
		}
	}

	return chunk, nil
}

func (v *buffer) downloadBoost(ctx context.Context, nextSegmentIndex chan int) {
	defer v.wg.Done()

	activeDownloaders := atomic.Int32{}
	currentDownloading := map[int]bool{}

	ctx, cancel := context.WithCancelCause(ctx)
	for {
		select {
		case <-ctx.Done():
			cancel(errors.New("context canceled by the client"))
			return
		case <-v.closed:
			cancel(errors.New("file closed"))
			return
		case i := <-nextSegmentIndex:
			if activeDownloaders.Load() >= int32(v.dc.maxAheadDownloadSegments) || currentDownloading[i] {
				continue
			}

			activeDownloaders.Add(1)
			currentDownloading[i] = true
			segment := v.nzbFile.Segments[i]

			v.wg.Add(1)
			go func() {
				defer v.wg.Done()
				_, err := v.downloadSegment(ctx, segment, v.nzbFile.Groups)
				if err != nil && !errors.Is(err, context.Canceled) {
					v.log.Error("Error downloading segment.", "error", err, "segment", segment.Number)
				}

				activeDownloaders.Add(-1)
				currentDownloading[i] = false
			}()
		}
	}
}
