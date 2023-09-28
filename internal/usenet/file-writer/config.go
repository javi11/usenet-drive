package usenetfilewriter

import (
	"log/slog"

	connectionpool "github.com/javi11/usenet-drive/internal/usenet/connection-pool"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
)

type Config struct {
	segmentSize   int64
	cp            connectionpool.UsenetConnectionPool
	postGroups    []string
	log           *slog.Logger
	fileAllowlist []string
	nzbLoader     nzbloader.NzbLoader
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{}
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
