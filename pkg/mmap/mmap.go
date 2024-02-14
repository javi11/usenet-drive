//go:generate mockgen -source=./mmap.go -destination=./mmap_mock.go -package=mmap MmapFileData
package mmap

import (
	"fmt"

	"github.com/javi11/usenet-drive/pkg/osfs"
)

type MmapFileData interface {
	Close() error
	File() osfs.File
	Bytes() []byte
}

type mmapFileData struct {
	f osfs.File
	b []byte
}

func MmapFile(f osfs.File) (MmapFileData, error) {
	return MmapFileWithSize(f, 0)
}

func MmapFileWithSize(f osfs.File, size int) (MmapFileData, error) {
	defer func() {
		f.Close()
	}()
	if size <= 0 {
		info, err := f.Stat()
		if err != nil {
			return nil, fmt.Errorf("stat: %w", err)
		}
		size = int(info.Size())
	}

	b, err := mmap(f, size)
	if err != nil {
		return nil, fmt.Errorf("mmap, size %d: %w", size, err)
	}

	return &mmapFileData{f: f, b: b}, nil
}

func (f *mmapFileData) Close() error {
	err0 := munmap(f.b)
	err1 := f.f.Close()

	if err0 != nil {
		return err0
	}
	return err1
}

func (f *mmapFileData) File() osfs.File {
	return f.f
}

func (f *mmapFileData) Bytes() []byte {
	return f.b
}
