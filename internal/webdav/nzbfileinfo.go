package webdav

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/javi11/usenet-drive/internal/domain"
	"github.com/javi11/usenet-drive/internal/utils"
)

type nzbFileInfoWithMetadata struct {
	nzbFile  os.FileInfo
	name     string
	metadata domain.Metadata
}

func NewFileInfoWithMetadata(name string, log *slog.Logger) (fs.FileInfo, error) {
	var nzbFileInfo os.FileInfo
	var metadata domain.Metadata
	var eg multierror.Group

	eg.Go(func() error {
		file, err := os.OpenFile(name, os.O_RDONLY, 0)
		if err != nil {
			log.Error(fmt.Sprintf("Error opening file %s, this file will be ignored", name), "err", err)
			return err
		}

		nzbFile, err := parseNzbFile(file)
		if err != nil {
			log.Error(fmt.Sprintf("Error parsing nzb file %s, this file will be ignored", name), "err", err)
			return err
		}

		metadata, err = domain.LoadFromNzb(nzbFile)
		if err != nil {
			log.Error(fmt.Sprintf("Error getting metadata from file %s, this file will be ignored", name), "err", err)
			return err
		}

		return nil
	})

	eg.Go(func() error {
		info, err := os.Stat(name)
		nzbFileInfo = info
		if err != nil {
			return err
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, os.ErrNotExist
	}

	fileName := nzbFileInfo.Name()

	return &nzbFileInfoWithMetadata{
		nzbFile:  nzbFileInfo,
		metadata: metadata,
		name:     utils.ReplaceFileExtension(fileName, metadata.FileExtension),
	}, nil
}

func (fi *nzbFileInfoWithMetadata) Size() int64 {
	// We need the original file size to display it.
	return fi.metadata.FileSize
}

func (fi *nzbFileInfoWithMetadata) ModTime() time.Time {
	// We need the original file mod time in order to allow comparing when replace a file. Files will never be modified.
	return fi.metadata.ModTime
}

func (fi *nzbFileInfoWithMetadata) IsDir() bool {
	return false
}

func (fi *nzbFileInfoWithMetadata) Sys() any {
	return nil
}

func (fi *nzbFileInfoWithMetadata) Name() string {
	return fi.name
}

func (fi *nzbFileInfoWithMetadata) Mode() fs.FileMode {
	return fi.nzbFile.Mode()
}
