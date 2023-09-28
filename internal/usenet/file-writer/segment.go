package usenetfilewriter

import (
	"io"
)

// Buf is a Buffer working on a slice of bytes.
type segmentBuffer struct {
	io.Writer
	chunkSize int64
	buffer    []byte
	ptr       int
}

// NewBuffer creates a new data volume based on a buffer
func NewSegmentBuffer(chunkSize int64) *segmentBuffer {
	return &segmentBuffer{
		chunkSize: chunkSize,
		buffer:    make([]byte, chunkSize),
		ptr:       0,
	}
}

func (v *segmentBuffer) Write(b []byte) (int, error) {
	n := copy(v.buffer[v.ptr:], b)
	v.ptr += n

	return n, nil
}

func (v *segmentBuffer) Size() int {
	return v.ptr
}

func (v *segmentBuffer) Clear() {
	clear(v.buffer)
	v.ptr = 0
}

func (v *segmentBuffer) Bytes() []byte {
	return v.buffer
}
