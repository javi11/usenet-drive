package webdav

import (
	"log/slog"

	rclonecli "github.com/javi11/usenet-drive/pkg/rclone-cli"
)

type Config struct {
	rootPath           string
	log                *slog.Logger
	fileWriter         RemoteFileWriter
	fileReader         RemoteFileReader
	rcloneCli          rclonecli.RcloneRcClient
	refreshRcloneCache bool
}

type Option func(*Config)

func WithRcloneCli(rcloneCli rclonecli.RcloneRcClient) Option {
	return func(c *Config) {
		c.rcloneCli = rcloneCli
		c.refreshRcloneCache = true
	}
}

func defaultConfig() *Config {
	return &Config{}
}

func WithFileWriter(fileWriter RemoteFileWriter) Option {
	return func(c *Config) {
		c.fileWriter = fileWriter
	}
}

func WithFileReader(fileReader RemoteFileReader) Option {
	return func(c *Config) {
		c.fileReader = fileReader
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(c *Config) {
		c.log = log
	}
}

func WithRootPath(rootPath string) Option {
	return func(c *Config) {
		c.rootPath = rootPath
	}
}
