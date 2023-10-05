package usenetfilewriter

import (
	"log/slog"

	connectionpool "github.com/javi11/usenet-drive/internal/usenet/connection-pool"
	corruptednzbsmanager "github.com/javi11/usenet-drive/internal/usenet/corrupted-nzbs-manager"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
)

type Config struct {
	segmentSize   int64
	cp            connectionpool.UsenetConnectionPool
	postGroups    []string
	log           *slog.Logger
	fileAllowlist []string
	nzbLoader     *nzbloader.NzbLoader
	cNzb          corruptednzbsmanager.CorruptedNzbsManager
	dryRun        bool
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{}
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

func WithNzbLoader(nzbLoader *nzbloader.NzbLoader) Option {
	return func(c *Config) {
		c.nzbLoader = nzbLoader
	}
}

func WithCorruptedNzbsManager(cNzb corruptednzbsmanager.CorruptedNzbsManager) Option {
	return func(c *Config) {
		c.cNzb = cNzb
	}
}
