package usenet

import (
	"io"
)

// Buf is a Buffer working on a slice of bytes.
type uploadBuffer struct {
	io.Writer
	chunkSize int64
	buffer    []byte
	ptr       int
}

// NewBuffer creates a new data volume based on a buffer
func NewUploadBuffer(chunkSize int64) *uploadBuffer {
	return &uploadBuffer{
		chunkSize: chunkSize,
		buffer:    make([]byte, chunkSize),
		ptr:       0,
	}
}

func (v *uploadBuffer) Write(b []byte) (int, error) {
	n := copy(v.buffer[v.ptr:], b)
	v.ptr += n

	return n, nil
}

func (v *uploadBuffer) Size() int {
	return v.ptr
}

func (v *uploadBuffer) Clear() {
	clear(v.buffer)
	v.ptr = 0
}

func (v *uploadBuffer) Dump() []byte {
	b := v.buffer
	v.Clear()

	return b
}
