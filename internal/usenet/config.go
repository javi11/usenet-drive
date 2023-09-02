package usenet

import (
	"crypto/tls"
	"fmt"
	"log/slog"
)

type Config struct {
	host           string
	port           int
	username       string
	password       string
	tls            bool
	tlsConfig      *tls.Config
	maxConnections int
	log            *slog.Logger
}

func (c *Config) getConnectionString() string {
	return fmt.Sprintf("%s:%d", c.host, c.port)
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		tls: false,
	}
}

func WithHost(host string) Option {
	return func(c *Config) {
		c.host = host
	}
}

func WithPort(port int) Option {
	return func(c *Config) {
		c.port = port
	}
}

func WithUsername(username string) Option {
	return func(c *Config) {
		c.username = username
	}
}

func WithPassword(password string) Option {
	return func(c *Config) {
		c.password = password
	}
}

func WithTLS(tls bool) Option {
	return func(c *Config) {
		c.tls = tls
	}
}

func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(c *Config) {
		c.tlsConfig = tlsConfig
	}
}

func WithMaxConnections(maxConnections int) Option {
	return func(c *Config) {
		c.maxConnections = maxConnections
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(c *Config) {
		c.log = log
	}
}
