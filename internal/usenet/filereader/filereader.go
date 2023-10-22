package filereader

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"golang.org/x/net/webdav"
)

type fileReader struct {
	cp        connectionpool.UsenetConnectionPool
	log       *slog.Logger
	nzbLoader nzbloader.NzbLoader
	cNzb      corruptednzbsmanager.CorruptedNzbsManager
	fs        osfs.FileSystem
	dc        downloadConfig
}

func NewFileReader(options ...Option) *fileReader {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	return &fileReader{
		cp:        config.cp,
		log:       config.log,
		nzbLoader: config.nzbLoader,
		cNzb:      config.cNzb,
		fs:        config.fs,
		dc:        config.dc,
	}
}

func (fr *fileReader) OpenFile(ctx context.Context, path string, flag int, perm fs.FileMode, onClose func() error) (bool, webdav.File, error) {
	return openFile(
		ctx,
		path,
		flag,
		perm,
		fr.cp,
		fr.log,
		onClose,
		fr.nzbLoader,
		fr.cNzb,
		fr.fs,
		fr.dc,
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
		path,
		fr.log,
		fr.nzbLoader,
		stat,
	)
	if err != nil {
		return true, nil, os.ErrNotExist
	}
	return true, fi, nil
}
