package filereader

//go:generate mockgen -source=./buffer.go -destination=./buffer_mock.go -package=filereader Buffer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/avast/retry-go"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/mnightingale/rapidyenc"
)

var (
	ErrInvalidWhence = errors.New("seek: invalid whence")
	ErrSeekNegative  = errors.New("seek: negative position")
	ErrSeekTooFar    = errors.New("seek: too far")
)

const defaultBufSize = 4096

type Buffer interface {
	io.ReaderAt
	io.ReadSeeker
	io.Closer
}

// Buf is a Buffer working on a slice of bytes.
type buffer struct {
	ctx                context.Context
	size               int
	nzbReader          nzbloader.NzbReader
	nzbGroups          []string
	ptr                int64
	cache              Cache
	cp                 connectionpool.UsenetConnectionPool
	chunkSize          int
	dc                 downloadConfig
	log                *slog.Logger
	nextSegment        chan nzb.NzbSegment
	wg                 *sync.WaitGroup
	currentDownloading *sync.Map
	decoder            *rapidyenc.Decoder
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
		ctx:                ctx,
		chunkSize:          chunkSize,
		size:               size,
		nzbReader:          nzbReader,
		nzbGroups:          nzbGroups,
		cache:              cache,
		cp:                 cp,
		dc:                 dc,
		log:                log,
		nextSegment:        make(chan nzb.NzbSegment),
		wg:                 &sync.WaitGroup{},
		currentDownloading: &sync.Map{},
		decoder:            rapidyenc.NewDecoder(defaultBufSize),
	}

	if dc.maxAheadDownloadSegments > 0 {
		for i := 0; i < dc.maxAheadDownloadSegments; i++ {
			buffer.wg.Add(1)
			go func() {
				defer buffer.wg.Done()
				buffer.downloadBoost(ctx)
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
		abs = int64(b.size) + offset
	default:
		return 0, ErrInvalidWhence
	}
	if abs < 0 {
		return 0, ErrSeekNegative
	}
	if abs > int64(b.size) {
		return 0, ErrSeekTooFar
	}
	b.ptr = abs

	return abs, nil
}

// Close the buffer. Currently no effect.
func (b *buffer) Close() error {
	close(b.nextSegment)

	b.decoder.Reset()
	b.decoder = nil

	if b.dc.maxAheadDownloadSegments > 0 {
		b.wg.Wait()
	}

	b.nzbReader = nil
	b.currentDownloading.Range(func(key any, _ any) bool {
		b.currentDownloading.Delete(key)
		return true
	})
	b.currentDownloading = nil
	b.cache = nil

	return nil
}

// Read reads len(p) byte from the Buffer starting at the current offset.
// It returns the number of bytes read and an error if any.
// Returns io.EOF error if pointer is at the end of the Buffer.
func (b *buffer) Read(p []byte) (int, error) {
	n := 0

	if len(p) == 0 {
		return n, nil
	}
	if b.ptr >= int64(b.size) {
		return n, io.EOF
	}

	currentSegmentIndex := int(float64(b.ptr) / float64(b.chunkSize))
	beginReadAt := max((int(b.ptr) - (currentSegmentIndex * b.chunkSize)), 0)

	for i := 0; ; i++ {
		if n >= len(p) {
			break
		}

		segment, hasMore := b.nzbReader.GetSegment(currentSegmentIndex + i)
		if !hasMore {
			break
		}

		nextSegmentIndex := currentSegmentIndex + i + 1
		// Preload next segments
		for j := 0; j < b.dc.maxAheadDownloadSegments; j++ {
			nextSegmentIndex := nextSegmentIndex + j
			// Preload next segments
			if nextSegment, hasMore := b.nzbReader.GetSegment(nextSegmentIndex); hasMore && !b.cache.Has(nextSegment.Id) {
				b.nextSegment <- nextSegment
			}
		}

		chunk, err := b.getSegment(b.ctx, segment, b.nzbGroups, b.decoder)
		if err != nil {
			// If nzb is corrupted stop reading
			if errors.Is(err, ErrCorruptedNzb) {
				return n, fmt.Errorf("error downloading segment: %w", err)
			}
			break
		}
		beginWriteAt := n
		n += copy(p[beginWriteAt:], chunk[beginReadAt:])
		chunk = nil
		beginReadAt = 0
	}
	b.ptr += int64(n)

	return n, nil
}

// ReadAt reads len(b) bytes from the Buffer starting at byte offset off.
// It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b).
// At end of file, that error is io.EOF.
func (b *buffer) ReadAt(p []byte, off int64) (int, error) {
	n := 0

	if len(p) == 0 {
		return n, nil
	}
	if off >= int64(b.size) {
		return n, io.EOF
	}

	currentSegmentIndex := int(float64(off) / float64(b.chunkSize))
	beginReadAt := max((int(off) - (currentSegmentIndex * b.chunkSize)), 0)

	for i := 0; ; i++ {
		if n >= len(p) {
			break
		}

		segment, hasMore := b.nzbReader.GetSegment(currentSegmentIndex + i)
		if !hasMore {
			break
		}

		nextSegmentIndex := currentSegmentIndex + i + 1
		// Preload next segments
		for j := 0; j < b.dc.maxAheadDownloadSegments; j++ {
			nextSegmentIndex := nextSegmentIndex + j
			// Preload next segments
			if nextSegment, hasMore := b.nzbReader.GetSegment(nextSegmentIndex); hasMore && !b.cache.Has(nextSegment.Id) {
				b.nextSegment <- nextSegment
			}
		}

		chunk, err := b.getSegment(b.ctx, segment, b.nzbGroups, b.decoder)
		if err != nil {
			// If nzb is corrupted stop reading
			if errors.Is(err, ErrCorruptedNzb) {
				return n, fmt.Errorf("error downloading segment: %w", err)
			}
			break
		}
		beginWriteAt := n
		n += copy(p[beginWriteAt:], chunk[beginReadAt:])
		chunk = nil
		beginReadAt = 0
	}

	return n, nil
}

func (b *buffer) getSegment(ctx context.Context, segment nzb.NzbSegment, groups []string, decoder *rapidyenc.Decoder) ([]byte, error) {
	hit := b.cache.Get(segment.Id)
	if hit != nil {
		return hit, nil
	}

	chunk, err := b.downloadSegment(ctx, segment, groups, decoder)
	if err != nil {
		return nil, err
	}

	b.cache.Set(segment.Id, chunk)

	return chunk, err
}

func (b *buffer) downloadSegment(ctx context.Context, segment nzb.NzbSegment, groups []string, decoder *rapidyenc.Decoder) ([]byte, error) {
	var chunk []byte
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

		body, err := nntpConn.Body(fmt.Sprintf("<%v>", segment.Id))
		if err != nil {
			return fmt.Errorf("error getting body: %w", err)
		}

		defer decoder.Reset()
		decoder.SetReader(body)

		chunk, err = io.ReadAll(decoder)
		if err != nil {
			return fmt.Errorf("error decoding the body: %w", err)
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
			b.log.DebugContext(ctx, "Retrying download", "error", err, "segment", segment.Id, "retry", n)

			if conn != nil {
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

		return nil, errors.Join(ErrCorruptedNzb, err)
	}

	return chunk, nil
}

func (b *buffer) downloadBoost(ctx context.Context) {
	decoder := rapidyenc.NewDecoder(defaultBufSize)
	defer func() {
		decoder.Reset()
		decoder = nil
	}()

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

			defer b.currentDownloading.Delete(segment.Number)
			if b.cache.Has(segment.Id) {
				continue
			}

			chunk, err := b.downloadSegment(ctx, segment, b.nzbGroups, decoder)
			if err != nil && !errors.Is(err, context.Canceled) {
				b.log.Error("Error downloading segment.", "error", err, "segment", segment.Number)
			}

			if err == nil {
				b.cache.Set(segment.Id, chunk)
			}
		}
	}
}
