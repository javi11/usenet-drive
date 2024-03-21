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
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
	"github.com/javi11/usenet-drive/pkg/nzb"
)

var (
	ErrInvalidWhence  = errors.New("seek: invalid whence")
	ErrSeekNegative   = errors.New("seek: negative position")
	ErrSeekTooFar     = errors.New("seek: too far")
	ErrTimeoutWaiting = errors.New("timeout waiting for download worker")
)

type Buffer interface {
	io.ReaderAt
	io.ReadSeeker
	io.Closer
}

type seekData struct {
	from int
	to   int
}

type buffer struct {
	ctx                context.Context
	fileSize           int
	nzbReader          nzbloader.NzbReader
	nzbGroups          []string
	ptr                int64
	cp                 connectionpool.UsenetConnectionPool
	chunkSize          int64
	dc                 downloadConfig
	log                *slog.Logger
	nextSegment        chan nzb.NzbSegment
	wg                 *sync.WaitGroup
	filePath           string
	seek               chan seekData
	mx                 *sync.RWMutex
	chunkPool          *sync.Pool
	currentDownloading *sync.Map
	chunkCache         ChunkCache
	cancel             context.CancelFunc
}

func NewBuffer(
	ctx context.Context,
	nzbReader nzbloader.NzbReader,
	fileSize int,
	chunkSize int64,
	dc downloadConfig,
	cp connectionpool.UsenetConnectionPool,
	cNzb corruptednzbsmanager.CorruptedNzbsManager,
	filePath string,
	chunkPool *sync.Pool,
	log *slog.Logger,
) (Buffer, error) {
	nzbGroups, err := nzbReader.GetGroups()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	buffer := &buffer{
		ctx:                ctx,
		chunkSize:          chunkSize,
		fileSize:           fileSize,
		nzbReader:          nzbReader,
		nzbGroups:          nzbGroups,
		chunkCache:         &chunkCache{},
		currentDownloading: &sync.Map{},
		cp:                 cp,
		dc:                 dc,
		log:                log,
		nextSegment:        make(chan nzb.NzbSegment),
		wg:                 &sync.WaitGroup{},
		filePath:           filePath,
		seek:               make(chan seekData, 1),
		mx:                 &sync.RWMutex{},
		chunkPool:          chunkPool,
		cancel:             cancel,
	}

	for i := 0; i < dc.maxDownloadWorkers; i++ {
		buffer.wg.Add(1)
		go func() {
			defer buffer.wg.Done()
			buffer.downloadWorker(ctx, cNzb)
		}()
	}

	buffer.wg.Add(1)
	go func() {
		defer buffer.wg.Done()
		buffer.segmentCleaner(ctx)
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
	b.mx.Lock()
	b.ptr = abs
	b.mx.Unlock()
	currentSegmentIndex := b.calculateCurrentSegmentIndex(b.ptr)
	b.seek <- seekData{
		from: previousSegmentIndex,
		to:   currentSegmentIndex,
	}

	return abs, nil
}

func (b *buffer) Close() error {
	b.cancel()
	b.wg.Wait()
	close(b.nextSegment)

	b.chunkCache.DeleteAll(b.chunkPool)
	b.currentDownloading.Range(func(key, value interface{}) bool {
		b.currentDownloading.Delete(key)
		return true
	})

	b.currentDownloading = nil
	b.nzbReader = nil
	b.chunkPool = nil

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
	beginReadAt := max((b.ptr - (int64(currentSegmentIndex) * b.chunkSize)), 0)

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
	beginReadAt := max((off - (int64(currentSegmentIndex) * b.chunkSize)), 0)

	return b.read(p, currentSegmentIndex, beginReadAt)
}

func (b *buffer) calculateCurrentSegmentIndex(offset int64) int {
	return int(float64(offset) / float64(b.chunkSize))
}

func (b *buffer) read(p []byte, currentSegmentIndex int, beginReadAt int64) (int, error) {
	n := 0
	i := 0

	// Send segments to all download workers
	for j := 0; j < b.dc.maxDownloadWorkers; j++ {
		nextSegmentIndex := currentSegmentIndex + j
		if nextSegment, hasMore := b.nzbReader.GetSegment(nextSegmentIndex); hasMore {
			select {
			case b.nextSegment <- nextSegment:
				continue
			case <-b.ctx.Done():
				return n, b.ctx.Err()
			case <-time.After(time.Millisecond * 10):
				continue
			}
		}
	}

	for {
		if n >= len(p) {
			break
		}

		if dm := b.chunkCache.Get(currentSegmentIndex + i); dm != nil {
			nn, err := dm.ReadAt(b.ctx, p[n:], beginReadAt)
			if err != nil && !errors.Is(err, ErrTimeoutWaiting) {
				return n + nn, err
			}

			if err == nil {
				n += nn
				beginReadAt = 0
				i++
				continue
			}
		}

		// Fallback to direct download
		segment, hasMore := b.nzbReader.GetSegment(currentSegmentIndex + i)
		if !hasMore {
			break
		}

		dm := b.chunkPool.Get().(*downloadManager)
		err := b.downloadSegment(b.ctx, segment, b.nzbGroups, dm)
		if err != nil {
			dm.Reset()
			b.chunkPool.Put(dm)
			dm = nil
			return n, fmt.Errorf("error downloading segment: %w", err)
		}

		nn, err := dm.ReadAt(b.ctx, p[n:], beginReadAt)
		if err != nil {
			dm.Reset()
			b.chunkPool.Put(dm)
			dm = nil

			return n + nn, err
		}

		n += nn
		beginReadAt = 0
		i++

		dm.Reset()
		b.chunkPool.Put(dm)
		dm = nil
	}

	return n, nil
}

func (b *buffer) downloadSegment(
	ctx context.Context,
	segment nzb.NzbSegment,
	groups []string,
	dm *downloadManager,
) error {
	// Adjust the size of the chunk to the size of the segment.
	// This is necessary because config can be changed and current article can be greater or smaller than the actual configured chunk size.
	dm.AdjustChunkSize(b.chunkSize)

	var conn connectionpool.Resource
	retryErr := retry.Do(func() error {
		c, err := b.cp.GetDownloadConnection(ctx)
		if err != nil {
			if c != nil {
				b.cp.Close(c)
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

		d, err := nntpConn.Body(segment.Id)
		if err != nil {
			return fmt.Errorf("error getting body: %w", err)
		}

		err = dm.Download(ctx, d)
		if err != nil {
			return fmt.Errorf("error starting download: %w", err)
		}

		b.cp.Free(conn)
		return nil
	},
		retry.Context(ctx),
		retry.Attempts(uint(b.dc.maxDownloadRetries)),
		retry.DelayType(retry.FixedDelay),
		retry.RetryIf(func(err error) bool {
			return nntpcli.IsRetryableError(err) || errors.Is(err, connectionpool.ErrNoProviderAvailable)
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
		var e retry.Error
		if errors.As(err, &e) {
			err = errors.Join(e.WrappedErrors()...)
		}

		if conn != nil {
			if errors.Is(err, context.Canceled) {
				b.cp.Free(conn)
			} else {
				b.cp.Close(conn)
			}
		}

		if nntpcli.IsRetryableError(err) || errors.Is(err, context.Canceled) || errors.Is(err, connectionpool.ErrNoProviderAvailable) {
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
	for {
		select {
		case <-ctx.Done():
			return
		case segment := <-b.nextSegment:
			if _, loaded := b.currentDownloading.LoadOrStore(numberToSegmentIndex(segment.Number), nil); loaded {
				continue
			}

			ctx, cancel := context.WithCancel(ctx)
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case s := <-b.seek:
						if s.to > numberToSegmentIndex(segment.Number) {
							b.currentDownloading.Delete(numberToSegmentIndex(segment.Number))
							cancel()
							return
						}
					}
				}
			}()

			dm := b.chunkPool.Get().(*downloadManager)
			if _, loaded := b.chunkCache.LoadOrStore(numberToSegmentIndex(segment.Number), dm); loaded {
				dm.Reset()
				b.chunkPool.Put(dm)
				dm = nil
				b.currentDownloading.Delete(numberToSegmentIndex(segment.Number))

				cancel()
				continue
			}

			err := b.downloadSegment(ctx, segment, b.nzbGroups, dm)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				dm.Reset()
				b.chunkPool.Put(dm)
				dm = nil
				b.chunkCache.Delete(numberToSegmentIndex(segment.Number))
				b.currentDownloading.Delete(numberToSegmentIndex(segment.Number))
				if errors.Is(err, ErrCorruptedNzb) {
					b.log.Error("Marking file as corrupted:", "error", err, "fileName", b.filePath)
					err := cNzb.Add(b.ctx, b.filePath, err.Error())
					if err != nil {
						b.log.Error("Error adding corrupted nzb to the database:", "error", err)
					}
				}

				cancel()
				continue
			}

			dm = nil
			b.currentDownloading.Delete(numberToSegmentIndex(segment.Number))
			cancel()
		}
	}
}

func (b *buffer) segmentCleaner(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.mx.RLock()
			currentSegmentIndex := b.calculateCurrentSegmentIndex(b.ptr)
			b.mx.RUnlock()
			b.chunkCache.DeleteBefore(currentSegmentIndex, b.chunkPool)
		case s := <-b.seek:
			if s.from > s.to {
				// When seek to previous segments, delete all segments after the current segment
				// leaving the maxDownload workers number as buffer
				b.chunkCache.DeleteAfter(s.from+b.dc.maxDownloadWorkers, b.chunkPool)
			} else {
				// When seek to next segments, delete all segments before the current segment
				// We won't need them anymore
				b.chunkCache.DeleteBefore(s.to, b.chunkPool)
			}
		}
	}
}

func numberToSegmentIndex(segmentNumber int64) int {
	return int(segmentNumber - 1)
}
