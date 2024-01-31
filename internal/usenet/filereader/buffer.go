package filereader

//go:generate mockgen -source=./buffer.go -destination=./buffer_mock.go -package=filereader Buffer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/bool64/cache"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
	"github.com/javi11/usenet-drive/pkg/nzb"
)

var (
	ErrInvalidWhence = errors.New("seek: invalid whence")
	ErrSeekNegative  = errors.New("seek: negative position")
	ErrSeekTooFar    = errors.New("seek: too far")
)

const toMb = 1e+6

type Buffer interface {
	io.ReaderAt
	io.ReadSeeker
	io.Closer
}

// Buf is a Buffer working on a slice of bytes.
type buffer struct {
	ctx                    context.Context
	fileSize               int
	nzbReader              nzbloader.NzbReader
	nzbGroups              []string
	ptr                    int64
	segmentsBuffer         *cache.ShardedMapOf[[]byte]
	cp                     connectionpool.UsenetConnectionPool
	chunkSize              int
	dc                     downloadConfig
	log                    *slog.Logger
	nextSegment            chan nzb.NzbSegment
	wg                     *sync.WaitGroup
	currentDownloading     *sync.Map
	filePath               string
	downloadRetryTimeoutMs int
}

// NewBuffer creates a new data volume based on a buffer
func NewBuffer(
	ctx context.Context,
	nzbReader nzbloader.NzbReader,
	fileSize int,
	chunkSize int,
	dc downloadConfig,
	cp connectionpool.UsenetConnectionPool,
	cNzb corruptednzbsmanager.CorruptedNzbsManager,
	filePath string,
	log *slog.Logger,
) (Buffer, error) {
	nzbGroups, err := nzbReader.GetGroups()
	if err != nil {
		return nil, err
	}

	bufferLimit := uint64(dc.maxBufferSizeInMb) * toMb

	c := cache.NewShardedMapOf[[]byte](func(cfg *cache.Config) {
		cfg.TimeToLive = 13 * time.Minute
		cfg.Logger = cache.NewLogger(func(ctx context.Context, msg string, keysAndValues ...interface{}) {
			log.DebugContext(ctx, "cache failed: %s %v", msg, keysAndValues)
		}, nil, nil, nil)
		cfg.DeleteExpiredAfter = 1 * time.Minute
		cfg.DeleteExpiredJobInterval = 10 * time.Minute
		cfg.HeapInUseSoftLimit = bufferLimit
		cfg.EvictFraction = 0.1
		cfg.SysMemSoftLimit = bufferLimit
	})

	retyTimeout := time.Duration(dc.maxDownloadRetries) * time.Second
	buffer := &buffer{
		ctx:                    ctx,
		chunkSize:              chunkSize,
		fileSize:               fileSize,
		nzbReader:              nzbReader,
		nzbGroups:              nzbGroups,
		segmentsBuffer:         c,
		cp:                     cp,
		dc:                     dc,
		log:                    log,
		nextSegment:            make(chan nzb.NzbSegment, 200),
		wg:                     &sync.WaitGroup{},
		currentDownloading:     &sync.Map{},
		filePath:               filePath,
		downloadRetryTimeoutMs: int(retyTimeout.Milliseconds()),
	}

	if dc.maxDownloadWorkers > 0 {
		for i := 0; i < dc.maxDownloadWorkers; i++ {
			buffer.wg.Add(1)
			go func() {
				defer buffer.wg.Done()
				buffer.downloadWorker(ctx, cNzb)
			}()
		}
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
func (b *buffer) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart: // Relative to the origin of the file
		abs = offset
	case io.SeekCurrent: // Relative to the current offset
		abs = int64(b.ptr) + offset
	case io.SeekEnd: // Relative to the end
		abs = int64(b.fileSize) + offset
	default:
		return 0, ErrInvalidWhence
	}
	if abs < 0 {
		return 0, ErrSeekNegative
	}
	if abs > int64(b.fileSize) {
		return 0, ErrSeekTooFar
	}
	b.ptr = abs

	return abs, nil
}

// Close the buffer. Currently no effect.
func (b *buffer) Close() error {
	close(b.nextSegment)

	if b.dc.maxDownloadWorkers > 0 {
		b.wg.Wait()
	}

	b.segmentsBuffer = nil
	b.nzbReader = nil
	b.currentDownloading = nil

	return nil
}

// Read reads len(p) byte from the Buffer starting at the current offset.
// It returns the number of bytes read and an error if any.
// Returns io.EOF error if pointer is at the end of the Buffer.
func (b *buffer) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if b.ptr >= int64(b.fileSize) {
		return 0, io.EOF
	}

	currentSegmentIndex := int(float64(b.ptr) / float64(b.chunkSize))
	beginReadAt := max((int(b.ptr) - (currentSegmentIndex * b.chunkSize)), 0)

	return b.read(p, currentSegmentIndex, beginReadAt)
}

// ReadAt reads len(b) bytes from the Buffer starting at byte offset off.
// It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b).
// At end of file, that error is io.EOF.
func (b *buffer) ReadAt(p []byte, off int64) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if off >= int64(b.fileSize) {
		return 0, io.EOF
	}

	currentSegmentIndex := int(float64(off) / float64(b.chunkSize))
	beginReadAt := max((int(off) - (currentSegmentIndex * b.chunkSize)), 0)

	return b.read(p, currentSegmentIndex, beginReadAt)
}

func (b *buffer) read(p []byte, currentSegmentIndex, beginReadAt int) (int, error) {
	n := 0
	br := beginReadAt

	// Preload next segments
	for j := 0; j < b.dc.maxDownloadWorkers; j++ {
		nextSegmentIndex := currentSegmentIndex + j
		if _, ok := b.segmentsBuffer.Load([]byte(fmt.Sprint(nextSegmentIndex))); !ok {
			if nextSegment, hasMore := b.nzbReader.GetSegment(nextSegmentIndex); hasMore {
				b.nextSegment <- nextSegment
			}
		}
	}

	i := 0
	retries := 0
	for {
		if n >= len(p) {
			b.ptr += int64(n)

			return n, nil
		}

		segment, ok := b.segmentsBuffer.Load([]byte(fmt.Sprint(currentSegmentIndex + i)))
		if !ok {

			if retries >= b.downloadRetryTimeoutMs {
				// Last try to direct download a segment
				if nextSegment, hasMore := b.nzbReader.GetSegment(currentSegmentIndex + i); hasMore {
					s, err := b.downloadSegment(b.ctx, nextSegment, b.nzbGroups)
					if err != nil {
						return n, fmt.Errorf("error downloading segment: %w", err)
					}

					segment = s
				} else {
					b.log.WarnContext(b.ctx, "Timeout waiting for chunk", "segment", currentSegmentIndex+i)
					return n, io.ErrNoProgress
				}
			} else {
				time.Sleep(1 * time.Millisecond)
				retries++
				continue
			}
		}
		i++
		chunk := segment

		beginWriteAt := n
		n += copy(p[beginWriteAt:], chunk[br:])
		chunk = nil
		br = 0
		retries = 0
	}
}

func (b *buffer) downloadSegment(ctx context.Context, segment nzb.NzbSegment, groups []string) ([]byte, error) {
	chunk := make([]byte, segment.Bytes)
	var conn connectionpool.Resource
	retryErr := retry.Do(func() error {
		c, err := b.cp.GetDownloadConnection(ctx)
		if err != nil {
			if conn != nil {
				b.cp.Close(conn)
				conn = nil
			}

			if errors.Is(err, context.Canceled) {
				return err
			}

			b.log.ErrorContext(ctx, "Error getting nntp connection:", "error", err, "segment", segment.Number)

			return fmt.Errorf("error getting nntp connection: %w", err)
		}
		conn = c
		nntpConn := conn.Value()

		if nntpConn.Provider().JoinGroup {
			err = usenet.JoinGroup(nntpConn, groups)
			if err != nil {
				return fmt.Errorf("error joining group: %w", err)
			}
		}

		chunk, err = nntpConn.Body(fmt.Sprintf("<%v>", segment.Id))
		if err != nil {
			return fmt.Errorf("error getting body: %w", err)
		}

		b.cp.Free(conn)
		conn = nil

		return nil
	},
		retry.Context(ctx),
		retry.Attempts(uint(b.dc.maxDownloadRetries)),
		retry.DelayType(retry.FixedDelay),
		retry.RetryIf(func(err error) bool {
			return nntpcli.IsRetryableError(err)
		}),
		retry.OnRetry(func(n uint, err error) {
			b.log.DebugContext(ctx,
				"Retrying download",
				"error", err,
				"segment", segment.Id,
				"retry", n,
			)

			if conn != nil {
				b.log.DebugContext(ctx,
					"Closing connection",
					"error", err,
					"segment", segment.Id,
					"retry", n,
					"error_connection_host", conn.Value().Provider().Host,
					"error_connection_created_at", conn.CreationTime(),
				)

				b.cp.Close(conn)
				conn = nil
			}
		}),
	)
	if retryErr != nil {
		err := retryErr

		if conn != nil {
			b.cp.Close(conn)
			conn = nil
		}

		var e retry.Error
		if errors.As(err, &e) {
			err = errors.Join(e.WrappedErrors()...)
		}

		if nntpcli.IsRetryableError(err) || errors.Is(err, context.Canceled) {
			// do not mark file as corrupted if it's a retryable error
			return nil, err
		}

		b.log.DebugContext(ctx,
			"All download retries exhausted",
			"error", retryErr,
			"segment", segment.Id,
		)

		return nil, errors.Join(ErrCorruptedNzb, err)
	}

	return chunk, nil
}

func (b *buffer) downloadWorker(ctx context.Context, cNzb corruptednzbsmanager.CorruptedNzbsManager) {
	for {
		select {
		case <-ctx.Done():
			return
		case segment, ok := <-b.nextSegment:
			if !ok {
				return
			}

			if _, loaded := b.currentDownloading.LoadOrStore(segment.Number, true); loaded {
				continue
			}

			chunk, err := b.downloadSegment(ctx, segment, b.nzbGroups)
			if err != nil && !errors.Is(err, context.Canceled) {
				if errors.Is(err, ErrCorruptedNzb) {
					b.log.Error("Marking file as corrupted:", "error", err, "fileName", b.filePath)
					err := cNzb.Add(b.ctx, b.filePath, err.Error())
					if err != nil {
						b.log.Error("Error adding corrupted nzb to the database:", "error", err)
					}
				}
			}

			if err == nil {
				segmentIndex := fmt.Sprint(segment.Number - 1)
				b.segmentsBuffer.Store([]byte(segmentIndex), chunk)
			}

			b.currentDownloading.Delete(segment.Number)
		}
	}
}
