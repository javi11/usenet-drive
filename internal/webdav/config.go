package webdav

import (
	"log/slog"

	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
	"github.com/javi11/usenet-drive/internal/usenet"
)

type Config struct {
	NzbPath             string
	cp                  usenet.UsenetConnectionPool
	queue               uploadqueue.UploadQueue
	log                 *slog.Logger
	uploadFileWhitelist []string
	nzbLoader           *usenet.NzbLoader
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{}
}

func WithNzbPath(nzbPath string) Option {
	return func(c *Config) {
		c.NzbPath = nzbPath
	}
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

func WithUploadFileWhitelist(uploadFileWhitelist []string) Option {
	return func(c *Config) {
		c.uploadFileWhitelist = uploadFileWhitelist
	}
}

func WithNzbLoader(nzbLoader *usenet.NzbLoader) Option {
	return func(c *Config) {
		c.nzbLoader = nzbLoader
	}
}
