package filereader

import (
	"log/slog"

	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	status "github.com/javi11/usenet-drive/internal/usenet/statusreporter"
	"github.com/javi11/usenet-drive/pkg/osfs"
)

type downloadConfig struct {
	maxDownloadRetries int
	maxDownloadWorkers int
	maxBufferSizeInMb  int
}

type Config struct {
	cp                 connectionpool.UsenetConnectionPool
	log                *slog.Logger
	cNzb               corruptednzbsmanager.CorruptedNzbsManager
	fs                 osfs.FileSystem
	maxDownloadRetries int
	maxDownloadWorkers int
	maxBufferSizeInMb  int
	segmentSize        int64
	debug              bool
	sr                 status.StatusReporter
}

func (c *Config) getDownloadConfig() downloadConfig {
	return downloadConfig{
		maxDownloadRetries: c.maxDownloadRetries,
		maxDownloadWorkers: c.maxDownloadWorkers,
		maxBufferSizeInMb:  c.maxBufferSizeInMb,
	}
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		debug:              false,
		fs:                 osfs.New(),
		maxDownloadRetries: 8,
		maxDownloadWorkers: 10,
		maxBufferSizeInMb:  512,
	}
}

func WithDebug(debug bool) Option {
	return func(c *Config) {
		c.debug = debug
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

func WithMaxDownloadWorkers(maxDownloadWorkers int) Option {
	return func(c *Config) {
		c.maxDownloadWorkers = maxDownloadWorkers
	}
}

func WithMaxBufferSizeInMb(maxBufferSizeInMb int) Option {
	return func(c *Config) {
		c.maxBufferSizeInMb = maxBufferSizeInMb
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

func WithCorruptedNzbsManager(cNzb corruptednzbsmanager.CorruptedNzbsManager) Option {
	return func(c *Config) {
		c.cNzb = cNzb
	}
}

func WithStatusReporter(sr status.StatusReporter) Option {
	return func(c *Config) {
		c.sr = sr
	}
}
