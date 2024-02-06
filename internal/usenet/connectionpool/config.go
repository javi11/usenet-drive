package connectionpool

import (
	"log/slog"
	"time"

	"github.com/javi11/usenet-drive/internal/config"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
)

type Config struct {
	downloadProviders []config.UsenetProvider
	uploadProviders   []config.UsenetProvider
	log               *slog.Logger
	fakeConnections   bool
	cli               nntpcli.Client
	maxConnectionTTL  time.Duration
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		fakeConnections:  false,
		maxConnectionTTL: 10 * time.Minute,
	}
}

func WithClient(cli nntpcli.Client) Option {
	return func(c *Config) {
		c.cli = cli
	}
}

func WithDownloadProviders(providers []config.UsenetProvider) Option {
	return func(c *Config) {
		c.downloadProviders = providers
	}
}

func WithUploadProviders(providers []config.UsenetProvider) Option {
	return func(c *Config) {
		c.uploadProviders = providers
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(c *Config) {
		c.log = log
	}
}

func WithFakeConnections(fakeConnections bool) Option {
	return func(c *Config) {
		c.fakeConnections = fakeConnections
	}
}

func WithMaxConnectionTTL(maxConnectionTTL time.Duration) Option {
	return func(c *Config) {
		c.maxConnectionTTL = maxConnectionTTL
	}
}
