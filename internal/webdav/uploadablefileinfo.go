package webdav

import (
	"io/fs"
	"os"
	"time"

	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/utils"
)

type uploadableFileInfo struct {
	nzbFileStat          os.FileInfo
	name                 string
	originalFileMetadata usenet.Metadata
}

func NewUploadableFileInfo(metadata usenet.Metadata, name string) (fs.FileInfo, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, err
	}

	fileName := info.Name()

	return &uploadableFileInfo{
		nzbFileStat:          info,
		originalFileMetadata: metadata,
		name:                 utils.ReplaceFileExtension(fileName, metadata.FileExtension),
	}, nil
}

func (fi *uploadableFileInfo) Size() int64 {
	// We need the original file size to display it.
	return fi.originalFileMetadata.FileSize
}

func (fi *uploadableFileInfo) ModTime() time.Time {
	// We need the original file mod time in order to allow comparing when replace a file. Files will never be modified.
	return fi.originalFileMetadata.ModTime
}

func (fi *uploadableFileInfo) IsDir() bool {
	return false
}

func (fi *uploadableFileInfo) Sys() any {
	return nil
}

func (fi *uploadableFileInfo) Name() string {
	return fi.name
}

func (fi *uploadableFileInfo) Mode() fs.FileMode {
	return fi.nzbFileStat.Mode()
}
