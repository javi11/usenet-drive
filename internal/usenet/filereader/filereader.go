package filereader

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	status "github.com/javi11/usenet-drive/internal/usenet/statusreporter"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"golang.org/x/net/webdav"
)

type fileReader struct {
	cp        connectionpool.UsenetConnectionPool
	log       *slog.Logger
	cNzb      corruptednzbsmanager.CorruptedNzbsManager
	fs        osfs.FileSystem
	dc        downloadConfig
	chunkPool *bigcache.BigCache
	sr        status.StatusReporter
}

func NewFileReader(options ...Option) (*fileReader, error) {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	engine, err := bigcache.New(context.Background(), bigcache.Config{
		// number of shards (must be a power of 2)
		Shards: 2,

		// time after which entry can be evicted
		LifeWindow: 15 * time.Minute,

		// Interval between removing expired entries (clean up).
		// If set to <= 0 then no action is performed.
		// Setting to < 1 second is counterproductive — bigcache has a one second resolution.
		CleanWindow: 5 * time.Minute,

		// max entry size in bytes, used only in initial memory allocation
		MaxEntrySize: int(config.segmentSize),

		// prints information about additional memory allocation
		Verbose: true,

		// cache will not allocate more memory than this limit, value in MB
		// if value is reached then the oldest entries can be overridden for the new ones
		// 0 value means no size limit
		HardMaxCacheSize: 512,
	})
	if err != nil {
		return nil, err
	}

	return &fileReader{
		cp:        config.cp,
		log:       config.log,
		cNzb:      config.cNzb,
		fs:        config.fs,
		dc:        config.getDownloadConfig(),
		chunkPool: engine,
		sr:        config.sr,
	}, nil
}

func (fr *fileReader) OpenFile(ctx context.Context, path string, onClose func() error) (bool, webdav.File, error) {
	return openFile(
		ctx,
		path,
		fr.cp,
		fr.log.With("filename", path),
		onClose,
		fr.cNzb,
		fr.fs,
		fr.dc,
		fr.chunkPool,
		fr.sr,
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
