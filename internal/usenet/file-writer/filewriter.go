package usenetfilewriter

import (
	"io/fs"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/javi11/usenet-drive/internal/usenet"
	connectionpool "github.com/javi11/usenet-drive/internal/usenet/connection-pool"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/nzb"
)

type fileWriter struct {
	segmentSize   int64
	cp            connectionpool.UsenetConnectionPool
	postGroups    []string
	log           *slog.Logger
	fileAllowlist []string
	nzbLoader     nzbloader.NzbLoader
}

func NewFileWriter(options ...Option) *fileWriter {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	return &fileWriter{
		segmentSize:   config.segmentSize,
		cp:            config.cp,
		postGroups:    config.postGroups,
		log:           config.log,
		fileAllowlist: config.fileAllowlist,
		nzbLoader:     config.nzbLoader,
	}
}

func (u *fileWriter) OpenFile(
	fileName string,
	fileSize int64,
	flag int,
	perm fs.FileMode,
	onClose func() error,
) (*file, error) {
	randomGroup := u.postGroups[rand.Intn(len(u.postGroups))]

	return openFile(
		fileSize,
		u.segmentSize,
		fileName,
		u.cp,
		randomGroup,
		flag,
		perm,
		u.log,
		onClose,
	)
}

func (u *fileWriter) HasAllowedFileExtension(fileName string) bool {
	if len(u.fileAllowlist) == 0 {
		return true
	}

	for _, allowedFile := range u.fileAllowlist {
		if filepath.Ext(fileName) == allowedFile {
			return true
		}
	}

	return false
}

func (u *fileWriter) RemoveFile(fileName string) (bool, error) {
	if maskFile := u.getOriginalNzb(fileName); maskFile != "" {
		err := os.RemoveAll(maskFile)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (u *fileWriter) RenameFile(fileName string, newFileName string) (bool, error) {
	originalName := u.getOriginalNzb(fileName)
	if originalName != "" {
		// In case you want to update the file extension we need to update it in the original nzb file
		if filepath.Ext(newFileName) != filepath.Ext(fileName) {
			c, err := u.nzbLoader.LoadFromFile(originalName)
			if err != nil {
				return false, err
			}

			n := c.Nzb.UpdateMetadada(nzb.UpdateableMetadata{
				FileExtension: filepath.Ext(newFileName),
			})
			b, err := c.Nzb.ToBytes()
			if err != nil {
				return false, err
			}

			err = os.WriteFile(originalName, b, 0766)
			if err != nil {
				return false, err
			}

			// Refresh the cache
			_, err = u.nzbLoader.RefreshCachedNzb(originalName, n)
			if err != nil {
				return false, err
			}
		}
		// If the file is a masked call the original nzb file
		fileName = originalName
		newFileName = usenet.ReplaceFileExtension(newFileName, ".nzb")

		if newFileName == fileName {
			return true, nil
		}
	}

	err := os.Rename(fileName, newFileName)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (u *fileWriter) getOriginalNzb(name string) string {
	originalName := usenet.ReplaceFileExtension(name, ".nzb")
	_, err := os.Stat(originalName)
	if os.IsNotExist(err) {
		return ""
	}

	return originalName
}
