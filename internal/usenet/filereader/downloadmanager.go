package filereader

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/neilotoole/streamcache"
)

type downloadManager struct {
	ch           chan bool
	downloaded   bool
	chunk        []byte
	reader       io.ReadCloser
	readerOffset int64
	mx           *sync.RWMutex
}

func NewDownloadManager(chunkSize int64) *downloadManager {
	return &downloadManager{
		chunk: make([]byte, chunkSize),
		ch:    make(chan bool, 1),
		mx:    &sync.RWMutex{},
	}
}

func (d *downloadManager) AdjustChunkSize(chunkSize int64) {
	cz := int(chunkSize)

	if cz > len(d.chunk) {
		d.grow(cz - len(d.chunk))
	} else if cz < len(d.chunk) {
		d.reduce(cz)
	}
}

func (d *downloadManager) Reset() {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.downloaded = false
	if d.reader != nil {
		d.reader.Close()
		d.reader = nil
	}
	if d.ch != nil {
		d.ch <- false
		close(d.ch)
	}

	d.readerOffset = 0
	d.ch = make(chan bool, 1)
}

func (d *downloadManager) Download(ctx context.Context, reader io.ReadCloser) error {
	defer reader.Close()
	s := streamcache.New(reader)
	// We need to keep reading even if the file is closed to free the connection
	directReader := s.NewReader(ctx)
	downloadReader := s.NewReader(ctx)
	defer downloadReader.Close()
	s.Seal()

	d.mx.Lock()
	d.reader = directReader
	d.ch <- true
	close(d.ch)
	d.ch = nil
	d.mx.Unlock()

	_, err := io.ReadFull(downloadReader, d.chunk)
	// Final segments has less bytes than chunkSize. Do not error if it's the case
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return fmt.Errorf("error getting body: %w", err)
	}

	d.mx.Lock()
	d.downloaded = true
	d.mx.Unlock()

	return nil
}

func (d *downloadManager) ReadAt(ctx context.Context, p []byte, off int64) (int, error) {
	d.mx.RLock()
	if d.downloaded {
		d.mx.RUnlock()
		n := copy(p, d.chunk[off:])
		return n, nil
	}

	if d.reader == nil {
		d.mx.RUnlock()
		err := d.waitForDownloadNotification(ctx)
		if err != nil {
			return 0, err
		}

		return d.ReadAt(ctx, p, off)
	}

	if off > 0 && d.readerOffset == 0 {
		d.mx.RUnlock()
		buff := make([]byte, off)
		nn, err := io.ReadFull(d.reader, buff)
		if err != nil {
			return 0, err
		}
		d.mx.Lock()
		d.readerOffset += int64(nn)
		d.mx.Unlock()
		d.mx.RLock()
	}

	if d.downloaded {
		d.mx.RUnlock()
		n := copy(p, d.chunk[off:])

		return n, nil
	}

	d.mx.RUnlock()
	n, err := io.ReadFull(d.reader, p)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return n, err
	}

	d.mx.Lock()
	d.readerOffset += int64(n)
	d.mx.Unlock()

	return n, nil
}

func (d *downloadManager) grow(size int) {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.chunk = append(d.chunk, make([]byte, size)...)
}

func (d *downloadManager) reduce(size int) {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.chunk = d.chunk[:size]
}

func (d *downloadManager) waitForDownloadNotification(ctx context.Context) error {
	select {
	case <-d.wait():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Millisecond * 600):
		return ErrTimeoutWaiting
	}
}

func (d *downloadManager) wait() <-chan bool {
	d.mx.RLock()
	defer d.mx.RUnlock()
	if d.ch != nil {
		return d.ch
	}

	return nil
}
