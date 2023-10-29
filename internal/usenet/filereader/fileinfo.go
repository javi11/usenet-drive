package filereader

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"time"

	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/osfs"
)

type nzbFileInfo struct {
	nzbFileStat          os.FileInfo
	name                 string
	originalFileMetadata usenet.Metadata
}

func NeFileInfoWithMetadata(
	path string,
	metadata usenet.Metadata,
	fs osfs.FileSystem,
) (fs.FileInfo, error) {
	info, err := fs.Stat(path)
	if err != nil {
		return nil, err
	}

	name := info.Name()

	return &nzbFileInfo{
		nzbFileStat:          info,
		originalFileMetadata: metadata,
		name:                 usenet.ReplaceFileExtension(name, metadata.FileExtension),
	}, nil
}

func NewFileInfoWithStat(
	fs osfs.FileSystem,
	path string,
	log *slog.Logger,
	nzbFileStat os.FileInfo,
) (fs.FileInfo, error) {
	var metadata usenet.Metadata

	f, err := fs.Open(path)
	if err != nil {
		log.Error(fmt.Sprintf("Error getting file %s, this file will be ignored", path), "error", err)
		return nil, err
	}

	reader := nzbloader.NewNzbReader(f)

	metadata, err = reader.GetMetadata()
	if err != nil {
		log.Error(fmt.Sprintf("Error getting metadata for file %s, this file will be ignored", path), "error", err)
		return nil, err
	}

	name := nzbFileStat.Name()

	return &nzbFileInfo{
		nzbFileStat:          nzbFileStat,
		originalFileMetadata: metadata,
		name:                 usenet.ReplaceFileExtension(name, metadata.FileExtension),
	}, nil
}

func (fi *nzbFileInfo) Size() int64 {
	// We need the original file size to display it.
	return fi.originalFileMetadata.FileSize
}

func (fi *nzbFileInfo) ModTime() time.Time {
	// We need the original file mod time in order to allow comparing when replace a file. Files will never be modified.
	return fi.originalFileMetadata.ModTime
}

func (fi *nzbFileInfo) IsDir() bool {
	return false
}

func (fi *nzbFileInfo) Sys() any {
	return fi.nzbFileStat.Sys()
}

func (fi *nzbFileInfo) Name() string {
	return fi.name
}

func (fi *nzbFileInfo) Mode() fs.FileMode {
	return fi.nzbFileStat.Mode()
}
