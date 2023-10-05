package usenetfilereader

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/textproto"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/javi11/usenet-drive/internal/usenet"
	connectionpool "github.com/javi11/usenet-drive/internal/usenet/connection-pool"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/javi11/usenet-drive/pkg/yenc"
)

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
func NewBuffer(nzbFile *nzb.NzbFile, size int, chunkSize int, cp connectionpool.UsenetConnectionPool, log *slog.Logger) (*buffer, error) {
	// Article cache can not be too big since it is stored in memory
	// With 100 the max memory used is 100 * 740kb = 74mb peer stream
	// This is mainly used to not redownload the same article multiple times if was not already
	// full readed.
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
		return 0, errors.New("Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("Seek: negative position")
	}
	if abs > int64(v.size) {
		return 0, errors.New("Seek: too far")
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
		chunk, err := v.downloadSegment(segment, v.nzbFile.Groups, 0)
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
		chunk, err := v.downloadSegment(segment, v.nzbFile.Groups, 0)
		if err != nil {
			break
		}
		beginWriteAt := n
		n += copy(p[beginWriteAt:], chunk.Body[beginReadAt:])
		beginReadAt = 0
	}

	return n, nil
}

func (v *buffer) downloadSegment(segment nzb.NzbSegment, groups []string, retryes int) (*yenc.Part, error) {
	hit, _ := v.cache.Get(segment.Id)

	var chunk *yenc.Part
	if hit != nil {
		chunk = hit
	} else {
		// Get the connection from the pool
		conn, err := v.cp.Get()
		if err != nil {
			v.cp.Close(conn)
			v.log.Error("Error getting nntp connection:", "error", err)
			if retryes < v.maxDownloadRetries {
				return v.downloadSegment(segment, groups, retryes+1)
			}

			return nil, err
		}
		defer v.cp.Free(conn)

		err = usenet.FindGroup(conn, groups)
		if err != nil {
			if _, ok := err.(*textproto.Error); !ok {
				if retryes < v.maxDownloadRetries {
					return v.downloadSegment(segment, groups, retryes+1)
				}
			}
			v.log.Error("Error finding nntp group:", "error", err)
			return nil, err
		}

		body, err := conn.Body(fmt.Sprintf("<%v>", segment.Id))
		if err != nil {
			if _, ok := err.(*textproto.Error); !ok {
				if retryes < v.maxDownloadRetries {
					return v.downloadSegment(segment, groups, retryes+1)
				}
			}
			v.log.Error("Error getting nntp article body:", "error", err, "segment", segment.Number)
			return nil, err
		}
		yread, err := yenc.Decode(body)
		if err != nil {
			v.log.Error("Error decoding yenc article body:", "error", err, "segment", segment.Number)
			return nil, errors.Join(ErrCorruptedNzb, err)
		}

		chunk = yread
		v.cache.Add(segment.Id, chunk)
	}

	return chunk, nil
}
