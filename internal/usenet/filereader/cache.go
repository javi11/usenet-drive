package filereader

//go:generate mockgen -source=./cache.go -destination=./cache_mock.go -package=filereader Cache
import (
	"context"
	"time"

	"github.com/allegro/bigcache/v3"
)

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, entry []byte) error
	Delete(key string) error
	Reset() error
	Len() int
	Close() error
	Has(key string) bool
}

type cache struct {
	engine *bigcache.BigCache
}

func NewCache(segmentSize, maxCacheSize int, debug bool) (Cache, error) {
	// Download cache
	config := bigcache.Config{
		// number of shards (must be a power of 2)
		Shards: 4,

		// time after which entry can be evicted
		LifeWindow: 15 * time.Minute,

		// Interval between removing expired entries (clean up).
		// If set to <= 0 then no action is performed.
		// Setting to < 1 second is counterproductive — bigcache has a one second resolution.
		CleanWindow: 5 * time.Minute,

		// max entry size in bytes, used only in initial memory allocation
		MaxEntrySize: segmentSize,

		// prints information about additional memory allocation
		Verbose: debug,

		// cache will not allocate more memory than this limit, value in MB
		// if value is reached then the oldest entries can be overridden for the new ones
		// 0 value means no size limit
		HardMaxCacheSize: maxCacheSize,
	}

	engine, err := bigcache.New(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return &cache{
		engine: engine,
	}, nil
}

func (c *cache) Get(key string) ([]byte, error) {
	return c.engine.Get(key)
}

func (c *cache) Set(key string, entry []byte) error {
	return c.engine.Set(key, entry)
}

func (c *cache) Delete(key string) error {
	return c.engine.Delete(key)
}

func (c *cache) Reset() error {
	return c.engine.Reset()
}

func (c *cache) Len() int {
	return c.engine.Len()
}

func (c *cache) Close() error {
	return c.engine.Close()
}

func (c *cache) Has(key string) bool {
	_, err := c.engine.Get(key)
	return err == nil
}
