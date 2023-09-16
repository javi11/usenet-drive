package webdav

import (
	"log/slog"

	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
	"github.com/javi11/usenet-drive/internal/usenet"
)

type Config struct {
	rootPath            string
	tmpPath             string
	cp                  usenet.UsenetConnectionPool
	queue               uploadqueue.UploadQueue
	log                 *slog.Logger
	uploadFileAllowlist []string
	nzbLoader           *usenet.NzbLoader
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{}
}

func WithUsenetConnectionPool(cp usenet.UsenetConnectionPool) Option {
	return func(c *Config) {
		c.cp = cp
	}
}

func WithUploadQueue(queue uploadqueue.UploadQueue) Option {
	return func(c *Config) {
		c.queue = queue
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(c *Config) {
		c.log = log
	}
}

func WithUploadFileAllowlist(uploadFileAllowlist []string) Option {
	return func(c *Config) {
		c.uploadFileAllowlist = uploadFileAllowlist
	}
}

func WithNzbLoader(nzbLoader *usenet.NzbLoader) Option {
	return func(c *Config) {
		c.nzbLoader = nzbLoader
	}
}

func WithRootPath(rootPath string) Option {
	return func(c *Config) {
		c.rootPath = rootPath
	}
}

func WithTmpPath(tmpPath string) Option {
	return func(c *Config) {
		c.tmpPath = tmpPath
	}
}
