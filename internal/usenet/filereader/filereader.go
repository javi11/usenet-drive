package filereader

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"golang.org/x/net/webdav"
)

type fileReader struct {
	cp    connectionpool.UsenetConnectionPool
	log   *slog.Logger
	cNzb  corruptednzbsmanager.CorruptedNzbsManager
	fs    osfs.FileSystem
	dc    downloadConfig
	cache Cache
}

func NewFileReader(options ...Option) (*fileReader, error) {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	cache, err := NewCache(int(config.segmentSize), config.cacheSizeInMB, config.debug)
	if err != nil {
		return nil, err
	}

	return &fileReader{
		cp:    config.cp,
		log:   config.log,
		cNzb:  config.cNzb,
		fs:    config.fs,
		dc:    config.getDownloadConfig(),
		cache: cache,
	}, nil
}

func (fr *fileReader) OpenFile(ctx context.Context, path string, flag int, perm fs.FileMode, onClose func() error) (bool, webdav.File, error) {
	return openFile(
		ctx,
		path,
		flag,
		perm,
		fr.cp,
		fr.log.With("filename", path),
		onClose,
		fr.cNzb,
		fr.fs,
		fr.dc,
		fr.cache,
	)
}

func (fr *fileReader) Stat(path string) (bool, fs.FileInfo, error) {
	var stat fs.FileInfo
	if !isNzbFile(path) {
		originalFile := getOriginalNzb(fr.fs, path)
		if originalFile != nil {
			// If the file is a masked call the original nzb file
			path = filepath.Join(filepath.Dir(path), originalFile.Name())
			stat = originalFile
		} else {
			return false, nil, nil
		}
	} else {
		s, err := fr.fs.Stat(path)
		if err != nil {
			return true, nil, err
		}

		stat = s
	}

	// If file is a nzb file return a custom file that will mask the nzb
	fi, err := NewFileInfoWithStat(
		fr.fs,
		path,
		fr.log,
		stat,
	)
	if err != nil {
		return true, nil, os.ErrNotExist
	}
	return true, fi, nil
}
