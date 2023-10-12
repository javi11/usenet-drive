package filereader

import (
	"context"
	"io/fs"
	"log/slog"

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
	}
}

func (fr *fileReader) OpenFile(ctx context.Context, name string, flag int, perm fs.FileMode, onClose func() error) (bool, webdav.File, error) {
	return openFile(
		ctx,
		name,
		flag,
		perm,
		fr.cp,
		fr.log,
		onClose,
		fr.nzbLoader,
		fr.cNzb,
		fr.fs,
	)
}

func (fr *fileReader) Stat(name string) (bool, fs.FileInfo, error) {
	if !isNzbFile(name) {
		originalName := getOriginalNzb(fr.fs, name)
		if originalName != "" {
			// If the file is a masked call the original nzb file
			name = originalName
		} else {
			return false, nil, nil
		}
	}

	// If file is a nzb file return a custom file that will mask the nzb
	fi, err := NewFileInfo(
		name,
		fr.log,
		fr.nzbLoader,
		fr.fs,
	)
	if err != nil {
		return true, nil, err
	}
	return true, fi, nil
}
