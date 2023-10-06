package filewriter

import (
	"io/fs"
	"time"

	"github.com/javi11/usenet-drive/internal/usenet"
)

type fileInfo struct {
	name     string
	metadata usenet.Metadata
}

func NewFileInfo(metadata usenet.Metadata, name string) (fs.FileInfo, error) {
	return &fileInfo{
		metadata: metadata,
		name:     usenet.ReplaceFileExtension(name, metadata.FileExtension),
	}, nil
}

func (fi *fileInfo) Size() int64 {
	// We need the original file size to display it.
	return fi.metadata.FileSize
}

func (fi *fileInfo) ModTime() time.Time {
	// We need the original file mod time in order to allow comparing when replace a file. Files will never be modified.
	return fi.metadata.ModTime
}

func (fi *fileInfo) IsDir() bool {
	return false
}

func (fi *fileInfo) Sys() any {
	return nil
}

func (fi *fileInfo) Name() string {
	return fi.name
}

func (fi *fileInfo) Mode() fs.FileMode {
	return fs.ModeType
}
