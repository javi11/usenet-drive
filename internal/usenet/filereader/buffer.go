package filereader

//go:generate mockgen -source=./buffer.go -destination=./buffer_mock.go -package=filereader Buffer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/avast/retry-go"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
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
	ctx         context.Context
	size        int
	nzbReader   nzbloader.NzbReader
	nzbGroups   []string
	ptr         int64
	cache       Cache
	cp          connectionpool.UsenetConnectionPool
	chunkSize   int
	dc          downloadConfig
	log         *slog.Logger
	closed      chan bool
	nextSegment chan *nzb.NzbSegment
	wg          *sync.WaitGroup
}

// NewBuffer creates a new data volume based on a buffer
func NewBuffer(
	ctx context.Context,
	nzbReader nzbloader.NzbReader,
	size int,
	chunkSize int,
	dc downloadConfig,
	cp connectionpool.UsenetConnectionPool,
	cache Cache,
	log *slog.Logger,
) (Buffer, error) {

	nzbGroups, err := nzbReader.GetGroups()
	if err != nil {
		return nil, err
	}

	buffer := &buffer{
		ctx:         ctx,
		chunkSize:   chunkSize,
		size:        size,
		nzbReader:   nzbReader,
		nzbGroups:   nzbGroups,
		cache:       cache,
		cp:          cp,
		dc:          dc,
		log:         log,
		nextSegment: make(chan *nzb.NzbSegment),
		closed:      make(chan bool),
		wg:          &sync.WaitGroup{},
	}

	if dc.maxAheadDownloadSegments > 0 {
		buffer.wg.Add(1)
		go buffer.downloadBoost(ctx)
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

	close(v.closed)
	close(v.nextSegment)
	v.nzbReader.Close()

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

	currentSegmentIndex := int(float64(v.ptr) / float64(v.chunkSize))
	beginReadAt := max((int(v.ptr) - (currentSegmentIndex * v.chunkSize)), 0)

	for i := 0; ; i++ {
		if n >= len(p) {
			break
		}

		segment, hasMore := v.nzbReader.GetSegment(currentSegmentIndex + i)
		if !hasMore {
			break
		}

		nextSegmentIndex := currentSegmentIndex + i + 1
		// Preload next segments
		for j := 0; j < v.dc.maxAheadDownloadSegments; j++ {
			nextSegmentIndex := nextSegmentIndex + j
			// Preload next segments
			if nextSegment, hasMore := v.nzbReader.GetSegment(nextSegmentIndex); hasMore && !v.cache.Has(nextSegment.Id) {

				v.nextSegment <- nextSegment
			}
		}

		chunk, err := v.downloadSegment(v.ctx, segment, v.nzbGroups)
		if err != nil {
			// If nzb is corrupted stop reading
			if errors.Is(err, ErrCorruptedNzb) {
				return n, fmt.Errorf("error downloading segment: %w", err)
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

	currentSegmentIndex := int(float64(off) / float64(v.chunkSize))
	beginReadAt := max((int(off) - (currentSegmentIndex * v.chunkSize)), 0)

	for i := 0; ; i++ {
		if n >= len(p) {
			break
		}

		segment, hasMore := v.nzbReader.GetSegment(currentSegmentIndex + i)
		if !hasMore {
			break
		}

		nextSegmentIndex := currentSegmentIndex + i + 1
		// Preload next segments
		for j := 0; j < v.dc.maxAheadDownloadSegments; j++ {
			nextSegmentIndex := nextSegmentIndex + j
			// Preload next segments
			if nextSegment, hasMore := v.nzbReader.GetSegment(nextSegmentIndex); hasMore && !v.cache.Has(nextSegment.Id) {
				v.nextSegment <- nextSegment
			}
		}

		chunk, err := v.downloadSegment(v.ctx, segment, v.nzbGroups)
		if err != nil {
			// If nzb is corrupted stop reading
			if errors.Is(err, ErrCorruptedNzb) {
				return n, fmt.Errorf("error downloading segment: %w", err)
			}
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
		var conn connectionpool.Resource
		retryErr := retry.Do(func() error {
			c, err := v.cp.GetDownloadConnection(ctx)
			if err != nil {
				if conn != nil {
					v.cp.Close(conn)
					conn = nil
				}

				if errors.Is(err, context.Canceled) {
					return err
				}

				v.log.ErrorContext(ctx, "Error getting nntp connection:", "error", err, "segment", segment.Number)

				return fmt.Errorf("error getting nntp connection: %w", err)
			}
			conn = c
			nntpConn := conn.Value()

			if nntpConn.ProviderOptions().JoinGroup {
				err = usenet.JoinGroup(nntpConn, groups)
				if err != nil {
					return fmt.Errorf("error joining group: %w", err)
				}
			}

			body, err := nntpConn.Body(fmt.Sprintf("<%v>", segment.Id))
			if err != nil {
				return fmt.Errorf("error getting body: %w", err)
			}

			yread, err := yenc.Decode(body)
			if err != nil {
				return retry.Unrecoverable(fmt.Errorf("error decoding yenc: %w", err))
			}

			chunk = yread.Body

			v.cp.Free(conn)
			conn = nil

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
					v.cp.Close(conn)
					conn = nil
				}
			}),
		)
		if retryErr != nil {
			err := retryErr

			if conn != nil {
				v.cp.Close(conn)
				conn = nil
			}

			var e retry.Error
			if errors.As(err, &e) {
				err = errors.Join(e.WrappedErrors()...)
			}

			if errors.Is(err, context.Canceled) ||
				errors.Is(err, io.EOF) ||
				errors.Is(err, io.ErrUnexpectedEOF) {
				return nil, err
			}

			if _, ok := err.(net.Error); ok {
				// Net errors are retryable
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

func (v *buffer) downloadBoost(ctx context.Context) {
	defer v.wg.Done()

	var mx sync.RWMutex
	currentDownloading := make(map[int64]bool)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case <-v.closed:
			return
		case segment, ok := <-v.nextSegment:
			if !ok {
				return
			}

			mx.RLock()
			if len(currentDownloading) >= v.dc.maxAheadDownloadSegments || currentDownloading[segment.Number] {
				mx.RUnlock()
				continue
			}
			mx.RUnlock()

			mx.Lock()
			currentDownloading[segment.Number] = true
			mx.Unlock()

			v.wg.Add(1)
			go func() {
				defer v.wg.Done()
				_, err := v.downloadSegment(ctx, segment, v.nzbGroups)
				if err != nil && !errors.Is(err, context.Canceled) {
					v.log.Error("Error downloading segment.", "error", err, "segment", segment.Number)
				}

				mx.Lock()
				delete(currentDownloading, segment.Number)
				mx.Unlock()
			}()
		}
	}
}
