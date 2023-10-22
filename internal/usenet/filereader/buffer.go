package filereader

//go:generate mockgen -source=./buffer.go -destination=./buffer_mock.go -package=filereader Buffer

import (
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/avast/retry-go"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
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
	size               int
	nzbFile            *nzb.NzbFile
	ptr                int64
	cache              *lru.Cache[string, *yenc.Part]
	cp                 connectionpool.UsenetConnectionPool
	chunkSize          int
	maxDownloadRetries int
	log                *slog.Logger
}

// NewBuffer creates a new data volume based on a buffer
func NewBuffer(nzbFile *nzb.NzbFile, size int, chunkSize int, cp connectionpool.UsenetConnectionPool, log *slog.Logger) (Buffer, error) {
	// Article cache can not be too big since it is stored in memory
	// With 100 the max memory used is 100 * 740kb = 74mb peer stream
	// This is mainly used to not download twice the same article multiple times if was not already
	// full read.
	cache, err := lru.New[string, *yenc.Part](100)
	if err != nil {
		return nil, err
	}

	return &buffer{
		chunkSize:          chunkSize,
		size:               size,
		nzbFile:            nzbFile,
		cache:              cache,
		cp:                 cp,
		maxDownloadRetries: 5,
		log:                log,
	}, nil
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
	v.cache.Purge()
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

	for _, segment := range v.nzbFile.Segments[currentSegment:] {
		if n >= len(p) {
			break
		}

		chunk, err := v.downloadSegment(segment, v.nzbFile.Groups)
		if err != nil {
			if errors.Is(err, ErrCorruptedNzb) {
				return n, err
			}
			break
		}
		beginWriteAt := n
		n += copy(p[beginWriteAt:], chunk.Body[beginReadAt:])
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

	for _, segment := range v.nzbFile.Segments[currentSegment:] {
		if n >= len(p) {
			break
		}
		chunk, err := v.downloadSegment(segment, v.nzbFile.Groups)
		if err != nil {
			break
		}
		beginWriteAt := n
		n += copy(p[beginWriteAt:], chunk.Body[beginReadAt:])
		beginReadAt = 0
	}

	return n, nil
}

func (v *buffer) downloadSegment(segment nzb.NzbSegment, groups []string) (*yenc.Part, error) {
	hit, _ := v.cache.Get(segment.Id)

	var chunk *yenc.Part
	if hit != nil {
		chunk = hit
	} else {
		// Get the connection from the pool
		conn, err := v.cp.Get()
		if err != nil {
			if conn != nil {
				e := v.cp.Close(conn)
				if e != nil {
					v.log.Error("Error closing connection on downloading a file.", "error", e)
				}
			}
			v.log.Error("Error getting nntp connection:", "error", err)

			return nil, err
		}
		defer func() {
			if err = v.cp.Free(conn); err != nil {
				v.log.Error("Error freeing connection on downloading a file.", "error", err)
			}
		}()

		retryErr := retry.Do(func() error {
			err = usenet.FindGroup(conn, groups)
			if err != nil {
				v.log.Error("Error finding nntp group:", "error", err)
				return err
			}

			body, err := conn.Body(fmt.Sprintf("<%v>", segment.Id))
			if err != nil {
				v.log.Error("Error getting nntp article body, marking it as corrupted.", "error", err, "segment", segment.Number)
				return err
			}

			yread, err := yenc.Decode(body)
			if err != nil {
				v.log.Error("Error decoding yenc article body:", "error", err, "segment", segment.Number)
				return err
			}

			chunk = yread
			v.cache.Add(segment.Id, chunk)

			return nil
		},
			retry.Attempts(uint(v.maxDownloadRetries)),
			retry.DelayType(retry.FixedDelay),
			retry.RetryIf(func(err error) bool {
				return connectionpool.IsRetryable(err)
			}),
			retry.OnRetry(func(n uint, err error) {
				v.log.Info("Error downloading segment. Retrying", "error", err, "header", segment.Id, "retry", n)

				err = v.cp.Close(conn)
				if err != nil {
					v.log.Error("Error closing connection.", "error", err)
				}

				c, err := v.cp.Get()
				if err != nil {
					v.log.Error("Error getting connection from pool.", "error", err)
				}

				conn = c
			}),
		)
		if retryErr != nil {
			err := retryErr
			var e retry.Error
			if errors.As(err, &e) {
				err = errors.Join(e.WrappedErrors()...)
			}
			return nil, errors.Join(ErrCorruptedNzb, err)
		}
	}

	return chunk, nil
}
