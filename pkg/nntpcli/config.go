package nntpcli

import (
	"log/slog"
	"time"
)

type Config struct {
	timeout time.Duration
	log     *slog.Logger
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		timeout: time.Duration(5) * time.Second,
		log:     slog.Default(),
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.timeout = timeout
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(c *Config) {
		c.log = log
	}
}
