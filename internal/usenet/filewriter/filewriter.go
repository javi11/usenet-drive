package filewriter

import (
	"context"
	"io/fs"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"golang.org/x/net/webdav"
)

type fileWriter struct {
	segmentSize   int64
	cp            connectionpool.UsenetConnectionPool
	postGroups    []string
	log           *slog.Logger
	fileAllowlist []string
	nzbLoader     *nzbloader.NzbLoader
	cNzb          corruptednzbsmanager.CorruptedNzbsManager
	dryRun        bool
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
		cNzb:          config.cNzb,
		dryRun:        config.dryRun,
	}
}

func (u *fileWriter) OpenFile(
	ctx context.Context,
	fileName string,
	fileSize int64,
	flag int,
	perm fs.FileMode,
	onClose func() error,
) (webdav.File, error) {
	randomGroup := u.postGroups[rand.Intn(len(u.postGroups))]

	return openFile(
		ctx,
		fileSize,
		u.segmentSize,
		fileName,
		u.cp,
		randomGroup,
		flag,
		perm,
		u.log,
		u.nzbLoader,
		u.dryRun,
		onClose,
	)
}

func (u *fileWriter) HasAllowedFileExtension(fileName string) bool {
	if len(u.fileAllowlist) == 0 {
		return true
	}

	for _, ext := range u.fileAllowlist {
		if strings.HasSuffix(strings.ToLower(fileName), ext) {
			return true
		}
	}

	return false
}

func (u *fileWriter) RemoveFile(ctx context.Context, fileName string) (bool, error) {
	if maskFile := u.getOriginalNzb(fileName); maskFile != "" {
		err := os.RemoveAll(maskFile)
		if err != nil {
			return false, err
		}

		err = u.cNzb.Discard(ctx, fileName)
		if err != nil {
			u.log.ErrorContext(ctx, "Error removing corrupted nzb from list", "error", err)
			return true, nil
		}

		return true, nil
	}

	return false, nil
}

func (u *fileWriter) RenameFile(ctx context.Context, fileName string, newFileName string) (bool, error) {
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

	if !isNzbFile(fileName) {
		return false, nil
	}

	err := os.Rename(fileName, newFileName)
	if err != nil {
		return false, err
	}

	err = u.cNzb.Update(ctx, fileName, newFileName)
	if err != nil {
		u.log.ErrorContext(ctx, "Error updating corrupted nzb", "error", err)
		return true, nil
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
