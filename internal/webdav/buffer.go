package webdav

import (
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"sync"

	"github.com/chrisfarms/nzb"
	"github.com/chrisfarms/yenc"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/javi11/usenet-drive/internal/usenet"
)

// Buffer is a usable block of data similar to a file
type Buffer interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer
}

// Buf is a Buffer working on a slice of bytes.
type Buf struct {
	size               int
	nzbFile            *nzb.NzbFile
	ptr                int64
	cache              *lru.Cache[string, *yenc.Part]
	cp                 usenet.UsenetConnectionPool
	mx                 sync.RWMutex
	chunkSize          int
	maxDownloadRetries int
}

// NewBuffer creates a new data volume based on a buffer
func NewBuffer(nzbFile *nzb.NzbFile, size int, chunkSize int, cp usenet.UsenetConnectionPool) (*Buf, error) {
	cache, err := lru.New[string, *yenc.Part](len(nzbFile.Segments))
	if err != nil {
		return nil, err
	}

	return &Buf{
		chunkSize:          chunkSize,
		size:               size,
		nzbFile:            nzbFile,
		cache:              cache,
		cp:                 cp,
		maxDownloadRetries: 5,
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
func (v *Buf) Seek(offset int64, whence int) (int64, error) {
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
func (v *Buf) Close() error {
	return nil
}

// Read reads len(p) byte from the Buffer starting at the current offset.
// It returns the number of bytes read and an error if any.
// Returns io.EOF error if pointer is at the end of the Buffer.
func (v *Buf) Read(p []byte) (int, error) {
	n := 0

	if len(p) == 0 {
		return n, nil
	}
	if v.ptr >= int64(v.size) {
		return n, io.EOF
	}

	currentSegment := int(float64(v.ptr) / float64(v.chunkSize))
	beginReadAt := Max((int(v.ptr) - (currentSegment * v.chunkSize)), 0)

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
	v.ptr += int64(n)

	return n, nil
}

// ReadAt reads len(b) bytes from the Buffer starting at byte offset off.
// It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b).
// At end of file, that error is io.EOF.
func (v *Buf) ReadAt(p []byte, off int64) (int, error) {
	n := 0

	if len(p) == 0 {
		return n, nil
	}
	if off >= int64(v.size) {
		return n, io.EOF
	}

	currentSegment := int(float64(off) / float64(v.chunkSize))
	beginReadAt := Max((int(off) - (currentSegment * v.chunkSize)), 0)

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

func (v *Buf) downloadSegment(segment nzb.NzbSegment, groups []string, retryes int) (*yenc.Part, error) {
	v.mx.RLock()
	hit, _ := v.cache.Get(segment.Id)
	v.mx.RUnlock()
	var chunk *yenc.Part
	if hit != nil {
		chunk = hit
	} else {
		// Get the connection from the pool
		conn, err := v.cp.Get()
		if err != nil {
			v.cp.Close(conn)
			fmt.Fprintln(os.Stderr, "nntp error:", err)
			return nil, err
		}
		defer v.cp.Free(conn)

		err = usenet.FindGroup(conn, groups)
		if err != nil {
			if _, ok := err.(*textproto.Error); ok {
				// if err is a response error free the connection
				// Probably an that can not be solved by retrying
				v.cp.Free(conn)
			} else {
				v.cp.Close(conn)
				if retryes < v.maxDownloadRetries {
					return v.downloadSegment(segment, groups, retryes+1)
				}
			}
			fmt.Fprintln(os.Stderr, "nntp error:", err)
			return nil, err
		}

		body, err := conn.Body(fmt.Sprintf("<%v>", segment.Id))
		if err != nil {
			if _, ok := err.(*textproto.Error); ok {
				// if err is a response error free the connection
				// Probably an that can not be solved by retrying
				v.cp.Free(conn)
			} else {
				v.cp.Close(conn)
				if retryes < v.maxDownloadRetries {
					return v.downloadSegment(segment, groups, retryes+1)
				}
			}
			fmt.Fprintln(os.Stderr, "nntp error:", err)
			return nil, err
		}
		yread, err := yenc.Decode(body)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return nil, err
		}

		chunk = yread
		v.mx.Lock()
		v.cache.Add(segment.Id, chunk)
		v.mx.Unlock()
	}

	return chunk, nil
}
