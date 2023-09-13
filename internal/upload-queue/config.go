package uploadqueue

import (
	"log/slog"

	"github.com/javi11/usenet-drive/internal/uploader"
	sqllitequeue "github.com/javi11/usenet-drive/pkg/sqllite-queue"
)

type Config struct {
	qEngine          sqllitequeue.SqlQueue
	uploader         uploader.Uploader
	maxActiveUploads int
	log              *slog.Logger
	fileWhitelist    []string
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{}
}

func WithQueueEngine(queueEngine sqllitequeue.SqlQueue) Option {
	return func(c *Config) {
		c.qEngine = queueEngine
	}
}

func WithUploader(uploader uploader.Uploader) Option {
	return func(c *Config) {
		c.uploader = uploader
	}
}

func WithMaxActiveUploads(maxActiveUploads int) Option {
	return func(c *Config) {
		c.maxActiveUploads = maxActiveUploads
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(c *Config) {
		c.log = log
	}
}

func WithFileWhitelist(fileWhitelist []string) Option {
	return func(c *Config) {
		c.fileWhitelist = fileWhitelist
	}
}
