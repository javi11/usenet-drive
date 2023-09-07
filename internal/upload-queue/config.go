package uploadqueue

import (
	"log/slog"

	"github.com/javi11/usenet-drive/internal/uploader"
	sqllitequeue "github.com/javi11/usenet-drive/pkg/sqllite-queue"
)

type Config struct {
	sqlLiteEngine    sqllitequeue.SqlQueue
	uploader         uploader.Uploader
	maxActiveUploads int
	log              *slog.Logger
	fileWhitelist    []string
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{}
}

func WithSqlLiteEngine(sqlLiteEngine sqllitequeue.SqlQueue) Option {
	return func(c *Config) {
		c.sqlLiteEngine = sqlLiteEngine
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
