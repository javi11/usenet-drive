package filereader

import (
	"log/slog"

	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/osfs"
)

type downloadConfig struct {
	maxDownloadRetries       int
	maxAheadDownloadSegments int
}

type Config struct {
	cp                       connectionpool.UsenetConnectionPool
	log                      *slog.Logger
	nzbLoader                nzbloader.NzbLoader
	cNzb                     corruptednzbsmanager.CorruptedNzbsManager
	fs                       osfs.FileSystem
	maxDownloadRetries       int
	maxAheadDownloadSegments int
	segmentSize              int64
	cacheSizeInMB            int
	debug                    bool
}

func (c *Config) getDownloadConfig() downloadConfig {
	return downloadConfig{
		maxDownloadRetries:       c.maxDownloadRetries,
		maxAheadDownloadSegments: c.maxAheadDownloadSegments,
	}
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		debug:                    false,
		fs:                       osfs.New(),
		maxDownloadRetries:       8,
		maxAheadDownloadSegments: 1,
		cacheSizeInMB:            512,
	}
}

func WithDebug(debug bool) Option {
	return func(c *Config) {
		c.debug = debug
	}
}

func WithCacheSize(cacheSize int) Option {
	return func(c *Config) {
		c.cacheSizeInMB = cacheSize
	}
}

func WithSegmentSize(segmentSize int64) Option {
	return func(c *Config) {
		c.segmentSize = segmentSize
	}
}

func WithMaxDownloadRetries(maxDownloadRetries int) Option {
	return func(c *Config) {
		c.maxDownloadRetries = maxDownloadRetries
	}
}

func WithMaxAheadDownloadSegments(maxAheadDownloadSegments int) Option {
	return func(c *Config) {
		c.maxAheadDownloadSegments = maxAheadDownloadSegments
	}
}

func WithFileSystem(fs osfs.FileSystem) Option {
	return func(c *Config) {
		c.fs = fs
	}
}

func WithConnectionPool(cp connectionpool.UsenetConnectionPool) Option {
	return func(c *Config) {
		c.cp = cp
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(c *Config) {
		c.log = log
	}
}

func WithNzbLoader(nzbLoader nzbloader.NzbLoader) Option {
	return func(c *Config) {
		c.nzbLoader = nzbLoader
	}
}

func WithCorruptedNzbsManager(cNzb corruptednzbsmanager.CorruptedNzbsManager) Option {
	return func(c *Config) {
		c.cNzb = cNzb
	}
}
