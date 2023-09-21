package webdav

import (
	"io"
)

// Buf is a Buffer working on a slice of bytes.
type uploadBuffer struct {
	io.Writer
	io.Closer
	chunkSize int64
	buffer    []byte
}

// NewBuffer creates a new data volume based on a buffer
func NewUploadBuffer(chunkSize int64) *uploadBuffer {
	return &uploadBuffer{
		chunkSize: chunkSize,
		buffer:    make([]byte, chunkSize),
	}
}

// Close the buffer. Currently no effect.
func (v *uploadBuffer) Close() error {
	v.buffer = nil
	return nil
}

func (v *uploadBuffer) Write(b []byte) (int, error) {
	n := copy(v.buffer, b)

	return n, nil
}

func (v *uploadBuffer) Len() int {
	return len(v.buffer)
}

func (v *uploadBuffer) Bytes() []byte {
	return v.buffer
}
