package uploader

import (
	"log/slog"
)

type Config struct {
	dryRun         bool
	host           string
	port           int
	username       string
	password       string
	groups         []string
	ssl            bool
	maxConnections int
	fileWhiteList  []string
	nyuuPath       string
	articleSize    string
	log            *slog.Logger
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		ssl:         false,
		articleSize: "750K",
		dryRun:      false,
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

func WithGroups(groups []string) Option {
	return func(c *Config) {
		c.groups = groups
	}
}

func WithSSL(ssl bool) Option {
	return func(c *Config) {
		c.ssl = ssl
	}
}

func WithMaxConnections(maxConnections int) Option {
	return func(c *Config) {
		c.maxConnections = maxConnections
	}
}

func WithFileWhiteList(fileWhiteList []string) Option {
	return func(c *Config) {
		c.fileWhiteList = fileWhiteList
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
