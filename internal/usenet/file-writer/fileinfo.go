package usenetfilewriter

import (
	"io/fs"
	"time"

	"github.com/javi11/usenet-drive/internal/usenet"
)

type fileInfo struct {
	name                 string
	originalFileMetadata usenet.Metadata
}

func NewFileInfo(metadata usenet.Metadata, name string) (fs.FileInfo, error) {
	return &fileInfo{
		originalFileMetadata: metadata,
		name:                 usenet.ReplaceFileExtension(name, metadata.FileExtension),
	}, nil
}

func (fi *fileInfo) Size() int64 {
	// We need the original file size to display it.
	return fi.originalFileMetadata.FileSize
}

func (fi *fileInfo) ModTime() time.Time {
	// We need the original file mod time in order to allow comparing when replace a file. Files will never be modified.
	return fi.originalFileMetadata.ModTime
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
