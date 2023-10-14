package filereader

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/osfs"
)

type nzbFileInfo struct {
	nzbFileStat          os.FileInfo
	name                 string
	originalFileMetadata usenet.Metadata
}

func NewFileInfo(
	name string,
	log *slog.Logger,
	nzbLoader nzbloader.NzbLoader,
	fs osfs.FileSystem,
) (fs.FileInfo, error) {
	var nzbFileStat os.FileInfo
	var metadata usenet.Metadata
	var eg multierror.Group

	eg.Go(func() error {
		n, err := nzbLoader.LoadFromFile(name)
		if err != nil {
			log.Error(fmt.Sprintf("Error getting file %s, this file will be ignored", name), "error", err)
			return err
		}

		metadata = n.Metadata

		return nil
	})

	eg.Go(func() error {
		info, err := fs.Stat(name)
		nzbFileStat = info
		if err != nil {
			return err
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, os.ErrNotExist
	}

	fileName := nzbFileStat.Name()

	return &nzbFileInfo{
		nzbFileStat:          nzbFileStat,
		originalFileMetadata: metadata,
		name:                 usenet.ReplaceFileExtension(fileName, metadata.FileExtension),
	}, nil
}

func NeFileInfoWithMetadata(
	metadata usenet.Metadata,
	name string,
	fs osfs.FileSystem,
) (fs.FileInfo, error) {
	info, err := fs.Stat(name)
	if err != nil {
		return nil, err
	}

	fileName := info.Name()

	return &nzbFileInfo{
		nzbFileStat:          info,
		originalFileMetadata: metadata,
		name:                 usenet.ReplaceFileExtension(fileName, metadata.FileExtension),
	}, nil
}

func NewFileInfoWithStat(
	path string,
	log *slog.Logger,
	nzbLoader nzbloader.NzbLoader,
	nzbFileStat os.FileInfo,
) (fs.FileInfo, error) {
	var metadata usenet.Metadata

	n, err := nzbLoader.LoadFromFile(path)
	if err != nil {
		log.Error(fmt.Sprintf("Error getting file %s, this file will be ignored", path), "error", err)
		return nil, err
	}

	metadata = n.Metadata

	fileName := nzbFileStat.Name()

	return &nzbFileInfo{
		nzbFileStat:          nzbFileStat,
		originalFileMetadata: metadata,
		name:                 usenet.ReplaceFileExtension(fileName, metadata.FileExtension),
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
