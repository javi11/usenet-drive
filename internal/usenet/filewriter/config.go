package filewriter

import (
	"log/slog"

	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/osfs"
)

type Config struct {
	segmentSize      int64
	cp               connectionpool.UsenetConnectionPool
	postGroups       []string
	log              *slog.Logger
	fileAllowlist    []string
	nzbLoader        nzbloader.NzbLoader
	cNzb             corruptednzbsmanager.CorruptedNzbsManager
	dryRun           bool
	fs               osfs.FileSystem
	maxUploadRetries int
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		maxUploadRetries: 8,
	}
}

func WithDryRun(dryRun bool) Option {
	return func(c *Config) {
		c.dryRun = dryRun
	}
}

func WithSegmentSize(segmentSize int64) Option {
	return func(c *Config) {
		c.segmentSize = segmentSize
	}
}

func WithConnectionPool(cp connectionpool.UsenetConnectionPool) Option {
	return func(c *Config) {
		c.cp = cp
	}
}

func WithPostGroups(postGroups []string) Option {
	return func(c *Config) {
		c.postGroups = postGroups
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(c *Config) {
		c.log = log
	}
}

func WithFileAllowlist(fileAllowlist []string) Option {
	return func(c *Config) {
		c.fileAllowlist = fileAllowlist
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

func WithFileSystem(fs osfs.FileSystem) Option {
	return func(c *Config) {
		c.fs = fs
	}
}

func WithMaxUploadRetries(maxUploadRetries int) Option {
	return func(c *Config) {
		c.maxUploadRetries = maxUploadRetries
	}
}
