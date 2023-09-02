package uploadqueue

import (
	"log/slog"

	"github.com/javi11/usenet-drive/internal/uploader"
	sqllitequeue "github.com/javi11/usenet-drive/pkg/sqllite-queue"
)

type Config struct {
	SqlLiteEngine    sqllitequeue.SqlQueue
	Uploader         uploader.Uploader
	MaxActiveUploads int
	Log              *slog.Logger
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{}
}

func WithSqlLiteEngine(sqlLiteEngine sqllitequeue.SqlQueue) Option {
	return func(c *Config) {
		c.SqlLiteEngine = sqlLiteEngine
	}
}

func WithUploader(uploader uploader.Uploader) Option {
	return func(c *Config) {
		c.Uploader = uploader
	}
}

func WithMaxActiveUploads(maxActiveUploads int) Option {
	return func(c *Config) {
		c.MaxActiveUploads = maxActiveUploads
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(c *Config) {
		c.Log = log
	}
}
