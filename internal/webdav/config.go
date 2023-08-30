package webdav

import (
	"log"

	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
)

type Config struct {
	NzbPath             string
	ServerPort          string
	cp                  UsenetConnectionPool
	queue               uploadqueue.UploadQueue
	log                 *log.Logger
	uploadFileWhitelist []string
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		ServerPort: "8080",
	}
}

func WithNzbPath(nzbPath string) Option {
	return func(c *Config) {
		c.NzbPath = nzbPath
	}
}

func WithServerPort(serverPort string) Option {
	return func(c *Config) {
		c.ServerPort = serverPort
	}
}

func WithUsenetConnectionPool(cp UsenetConnectionPool) Option {
	return func(c *Config) {
		c.cp = cp
	}
}

func WithUploadQueue(queue uploadqueue.UploadQueue) Option {
	return func(c *Config) {
		c.queue = queue
	}
}

func WithLogger(log *log.Logger) Option {
	return func(c *Config) {
		c.log = log
	}
}

func WithUploadFileWhitelist(uploadFileWhitelist []string) Option {
	return func(c *Config) {
		c.uploadFileWhitelist = uploadFileWhitelist
	}
}
