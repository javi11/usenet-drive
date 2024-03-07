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

	"github.com/allegro/bigcache/v3"
	"github.com/avast/retry-go"
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
	segmentsBuffer         *sync.Map
	cp                     connectionpool.UsenetConnectionPool
	chunkSize              int
	dc                     downloadConfig
	log                    *slog.Logger
	nextSegment            chan nzb.NzbSegment
	wg                     *sync.WaitGroup
	filePath               string
	downloadRetryTimeoutMs int
	chunk                  []byte
	close                  chan struct{}
	seek                   chan seekData
	mx                     *sync.RWMutex
	chunkPool              *bigcache.BigCache
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
	chunkPool *bigcache.BigCache,
	log *slog.Logger,
) (Buffer, error) {
	nzbGroups, err := nzbReader.GetGroups()
	if err != nil {
		return nil, err
	}

	retryTimeout := time.Duration(dc.maxDownloadRetries) * time.Second
	buffer := &buffer{
		ctx:                    ctx,
		chunkSize:              chunkSize,
		fileSize:               fileSize,
		nzbReader:              nzbReader,
		nzbGroups:              nzbGroups,
		segmentsBuffer:         &sync.Map{},
		cp:                     cp,
		dc:                     dc,
		log:                    log,
		nextSegment:            make(chan nzb.NzbSegment),
		wg:                     &sync.WaitGroup{},
		filePath:               filePath,
		downloadRetryTimeoutMs: int(retryTimeout.Milliseconds()),
		chunk:                  make([]byte, chunkSize),
		close:                  make(chan struct{}),
		seek:                   make(chan seekData),
		mx:                     &sync.RWMutex{},
		chunkPool:              chunkPool,
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

	buffer.wg.Add(1)
	go func() {
		defer buffer.wg.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-buffer.close:
				return
			case <-ticker.C:
				buffer.mx.RLock()
				currentSegmentIndex := buffer.calculateCurrentSegmentIndex(buffer.ptr)
				buffer.mx.RUnlock()
				buffer.deleteSegmentsBefore(currentSegmentIndex)
			case s := <-buffer.seek:
				if s.from > s.to {
					// When seek to previous segments, delete all segments after the current segment
					// leaving the maxDownload workers number as buffer
					buffer.deleteSegmentsAfter(s.from)
				} else {
					// When seek to next segments, delete all segments before the current segment
					// We won't need them anymore
					buffer.deleteSegmentsBefore(s.to)
				}
			}
		}
	}()

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
	previousSegmentIndex := b.calculateCurrentSegmentIndex(b.ptr)
	currentSegmentIndex := b.calculateCurrentSegmentIndex(b.ptr)
	b.seek <- seekData{
		from: previousSegmentIndex,
		to:   currentSegmentIndex,
	}
	b.mx.Lock()
	b.ptr = abs
	b.mx.Unlock()

	return abs, nil
}

// Close the buffer. Currently no effect.
func (b *buffer) Close() error {
	close(b.close)
	b.wg.Wait()
	close(b.nextSegment)

	b.segmentsBuffer.Range(func(key, value interface{}) bool {
		b.segmentsBuffer.Delete(key)
		return true
	})

	b.chunkPool = nil
	b.segmentsBuffer = nil
	b.nzbReader = nil

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

	currentSegmentIndex := b.calculateCurrentSegmentIndex(b.ptr)
	beginReadAt := max((int(b.ptr) - (currentSegmentIndex * b.chunkSize)), 0)

	n, err := b.read(p, currentSegmentIndex, beginReadAt)
	b.mx.Lock()
	b.ptr += int64(n)
	b.mx.Unlock()

	return n, err
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

	currentSegmentIndex := b.calculateCurrentSegmentIndex(off)
	beginReadAt := max((int(off) - (currentSegmentIndex * b.chunkSize)), 0)

	return b.read(p, currentSegmentIndex, beginReadAt)
}

func (b *buffer) calculateCurrentSegmentIndex(offset int64) int {
	return int(float64(offset) / float64(b.chunkSize))
}

func (b *buffer) read(p []byte, currentSegmentIndex, beginReadAt int) (int, error) {
	n := 0
	i := 0

	// Preload next segments
	for j := 0; j < 20; j++ {
		nextSegmentIndex := currentSegmentIndex + j
		// Preload next segments
		if nextSegment, hasMore := b.nzbReader.GetSegment(nextSegmentIndex); hasMore {
			b.nextSegment <- nextSegment
		}
	}

	for {
		if n >= len(p) {
			break
		}

		segment, hasMore := b.nzbReader.GetSegment(currentSegmentIndex + i)
		if !hasMore {
			break
		}

		var chunk []byte
		if c, ok := b.segmentsBuffer.Load(currentSegmentIndex + i); ok {
			channel := *c.(*chunkCh)
			if !channel.downloaded {
				for {
					select {
					case <-channel.ch:
						goto Downloaded
					case <-b.close:
						return n, io.ErrUnexpectedEOF
					case <-b.ctx.Done():
						return n, b.ctx.Err()
					}
				}
			}

		Downloaded:
			if channel.Err != nil {
				return n, channel.Err
			}

			c, _ := b.chunkPool.Get(segment.Id)
			if c != nil {
				chunk = c
			}
		}

		if chunk == nil {
			c, err := b.getSegment(b.ctx, segment, b.nzbGroups)
			if err != nil {
				return n, fmt.Errorf("error downloading segment: %w", err)
			}

			chunk = c
		}

		n += copy(p[n:], chunk[beginReadAt:])
		beginReadAt = 0
		i++
	}

	return n, nil
}

func (b *buffer) getSegment(ctx context.Context, segment nzb.NzbSegment, groups []string) ([]byte, error) {
	err := b.downloadSegment(ctx, segment, groups, b.chunk)
	if err != nil {
		return nil, err
	}

	return b.chunk, nil
}

func (b *buffer) deleteSegmentsBefore(index int) {
	b.segmentsBuffer.Range(func(key, value interface{}) bool {
		if key.(int) < index {
			b.segmentsBuffer.Delete(key)

		}
		return true
	})
}

func (b *buffer) deleteSegmentsAfter(index int) {
	b.segmentsBuffer.Range(func(key, value interface{}) bool {
		if key.(int) < index+b.dc.maxDownloadWorkers {
			b.segmentsBuffer.Delete(key)
		}
		return true
	})
}

func (b *buffer) downloadSegment(
	ctx context.Context,
	segment nzb.NzbSegment,
	groups []string,
	chunk []byte,
) error {
	var conn connectionpool.Resource
	retryErr := retry.Do(func() error {
		c, err := b.cp.GetDownloadConnection(ctx)
		if err != nil {
			if conn != nil {
				b.cp.Close(conn)
			}

			if errors.Is(err, context.Canceled) {
				return err
			}

			b.log.ErrorContext(ctx, "Error getting nntp connection:", "error", err, "segment", segment.Number)

			return fmt.Errorf("error getting nntp connection: %w", err)
		}
		conn = c
		nntpConn := conn.Value()
		if nntpConn == nil {
			return fmt.Errorf("error getting nntp connection")
		}

		if nntpConn.Provider().JoinGroup {
			err = usenet.JoinGroup(nntpConn, groups)
			if err != nil {
				return fmt.Errorf("error joining group: %w", err)
			}
		}

		err = nntpConn.Body(segment.Id, chunk)
		if err != nil {
			// Final segments has less bytes than chunkSize. Do not error if it's the case
			if err != io.ErrUnexpectedEOF && err != io.EOF {
				return fmt.Errorf("error getting body: %w", err)
			}
		}

		b.cp.Free(conn)

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
			}
		}),
	)
	if retryErr != nil {
		err := retryErr

		if conn != nil {
			b.cp.Close(conn)
		}

		var e retry.Error
		if errors.As(err, &e) {
			err = errors.Join(e.WrappedErrors()...)
		}

		if nntpcli.IsRetryableError(err) || errors.Is(err, context.Canceled) {
			// do not mark file as corrupted if it's a retryable error
			return err
		}

		b.log.DebugContext(ctx,
			"All download retries exhausted",
			"error", retryErr,
			"segment", segment.Id,
		)

		return errors.Join(ErrCorruptedNzb, err)
	}

	return nil
}

func (b *buffer) downloadWorker(ctx context.Context, cNzb corruptednzbsmanager.CorruptedNzbsManager) {
	chunk := make([]byte, b.chunkSize)
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.close:
			return
		case segment := <-b.nextSegment:
			ctx, cancel := context.WithCancel(ctx)
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case s := <-b.seek:
						if s.to > int(segment.Number)-1 {
							cancel()
						}
					case <-b.close:
						cancel()
					}
				}
			}()

			c := &chunkCh{
				ch: make(chan bool, 1),
			}
			if _, loaded := b.segmentsBuffer.LoadOrStore(int(segment.Number-1), c); loaded {
				cancel()
				close(c.ch)
				continue
			}

			err := b.downloadSegment(ctx, segment, b.nzbGroups, chunk)
			if err == nil {
				err = b.chunkPool.Set(segment.Id, chunk)
				if err != nil {
					b.log.Error("Error setting chunk in bigcache", "error", err)
					close(c.ch)
					cancel()
					continue
				}
			}

			c.downloaded = true
			c.ch <- true
			close(c.ch)
			c.Err = err
			if err != nil {
				if errors.Is(err, ErrCorruptedNzb) {
					b.log.Error("Marking file as corrupted:", "error", err, "fileName", b.filePath)
					err := cNzb.Add(b.ctx, b.filePath, err.Error())
					if err != nil {
						b.log.Error("Error adding corrupted nzb to the database:", "error", err)
					}
				}
			}
			c = nil

			cancel()
		}
	}
}

type chunkCh struct {
	Err        error
	ch         chan bool
	downloaded bool
}

type seekData struct {
	from int
	to   int
}
