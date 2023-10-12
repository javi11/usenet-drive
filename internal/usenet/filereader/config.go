package filereader

import (
	"log/slog"

	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/osfs"
)

type Config struct {
	cp        connectionpool.UsenetConnectionPool
	log       *slog.Logger
	nzbLoader nzbloader.NzbLoader
	cNzb      corruptednzbsmanager.CorruptedNzbsManager
	fs        osfs.FileSystem
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		fs: osfs.New(),
	}
}

func WithFileSystem(fs osfs.FileSystem) Option {
	return func(c *Config) {
		c.fs = fs
	}
}

func WithConnectionPool(cp connectionpool.UsenetConnectionPool) Option {
	return func(c *Config) {
		c.cp = cp
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(c *Config) {
		c.log = log
	}
}

func WithNzbLoader(nzbLoader nzbloader.NzbLoader) Option {
	return func(c *Config) {
		c.nzbLoader = nzbLoader
	}
}

func WithCorruptedNzbsManager(cNzb corruptednzbsmanager.CorruptedNzbsManager) Option {
	return func(c *Config) {
		c.cNzb = cNzb
	}
}
