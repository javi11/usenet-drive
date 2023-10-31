package filewriter

import (
	"context"
	"io/fs"
	"log/slog"
	"math/rand"
	"path/filepath"
	"strings"

	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"golang.org/x/net/webdav"
)

type fileWriter struct {
	segmentSize      int64
	cp               connectionpool.UsenetConnectionPool
	postGroups       []string
	log              *slog.Logger
	fileAllowlist    []string
	nzbWriter        nzbloader.NzbWriter
	cNzb             corruptednzbsmanager.CorruptedNzbsManager
	dryRun           bool
	fs               osfs.FileSystem
	maxUploadRetries int
}

func NewFileWriter(options ...Option) *fileWriter {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	return &fileWriter{
		segmentSize:      config.segmentSize,
		cp:               config.cp,
		postGroups:       config.postGroups,
		log:              config.log,
		fileAllowlist:    config.fileAllowlist,
		nzbWriter:        config.nzbWriter,
		cNzb:             config.cNzb,
		dryRun:           config.dryRun,
		fs:               config.fs,
		maxUploadRetries: config.maxUploadRetries,
	}
}

func (u *fileWriter) OpenFile(
	ctx context.Context,
	filePath string,
	fileSize int64,
	flag int,
	perm fs.FileMode,
	onClose func(err error) error,
) (webdav.File, error) {
	randomGroup := u.postGroups[rand.Intn(len(u.postGroups))]

	return openFile(
		ctx,
		filePath,
		flag,
		perm,
		fileSize,
		u.segmentSize,
		u.cp,
		randomGroup,
		u.log,
		u.maxUploadRetries,
		u.dryRun,
		onClose,
		u.fs,
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
		err := u.fs.RemoveAll(maskFile)
		if err != nil {
			return false, err
		}

		_, err = u.cNzb.DiscardByPath(ctx, maskFile)
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
			err := u.nzbWriter.UpdateMetadata(originalName, nzb.UpdateableMetadata{
				FileExtension: filepath.Ext(newFileName),
			})
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

	err := u.fs.Rename(fileName, newFileName)
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
	_, err := u.fs.Stat(originalName)
	if u.fs.IsNotExist(err) {
		return ""
	}

	return originalName
}
