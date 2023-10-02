package usenetfilereader

import (
	"context"
	"io/fs"
	"log/slog"

	connectionpool "github.com/javi11/usenet-drive/internal/usenet/connection-pool"
	corruptednzbsmanager "github.com/javi11/usenet-drive/internal/usenet/corrupted-nzbs-manager"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"golang.org/x/net/webdav"
)

type fileReader struct {
	cp        connectionpool.UsenetConnectionPool
	log       *slog.Logger
	nzbLoader *nzbloader.NzbLoader
	cNzb      corruptednzbsmanager.CorruptedNzbsManager
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
	}
}

func (fr *fileReader) OpenFile(ctx context.Context, name string, flag int, perm fs.FileMode, onClose func() error) (bool, webdav.File, error) {
	return openFile(ctx, name, flag, perm, fr.cp, fr.log, onClose, fr.nzbLoader, fr.cNzb)
}

func (fr *fileReader) Stat(name string) (bool, fs.FileInfo, error) {
	if !isNzbFile(name) {
		originalName := getOriginalNzb(name)
		if originalName != "" {
			// If the file is a masked call the original nzb file
			name = originalName
		} else {
			return false, nil, nil
		}
	}

	// If file is a nzb file return a custom file that will mask the nzb
	fi, err := NewFileInfo(name, fr.log, fr.nzbLoader)
	if err != nil {
		return true, nil, err
	}
	return true, fi, nil
}
