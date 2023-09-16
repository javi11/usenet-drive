package uploader

import (
	"log/slog"

	"github.com/javi11/usenet-drive/internal/config"
)

type Config struct {
	dryRun      bool
	providers   []config.UsenetProvider
	nyuuPath    string
	articleSize string
	log         *slog.Logger
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		articleSize: "750K",
		dryRun:      false,
	}
}

func WithProviders(providers []config.UsenetProvider) Option {
	return func(c *Config) {
		c.providers = providers
	}
}

func WithNyuuPath(nyuuPath string) Option {
	return func(c *Config) {
		c.nyuuPath = nyuuPath
	}
}

func WithArticleSize(articleSize string) Option {
	return func(c *Config) {
		c.articleSize = articleSize
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(c *Config) {
		c.log = log
	}
}

func WithDryRun(dryRun bool) Option {
	return func(c *Config) {
		c.dryRun = dryRun
	}
}
