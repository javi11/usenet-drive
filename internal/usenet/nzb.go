package usenet

import (
	"os"
	"sync"

	"github.com/chrisfarms/nzb"
	lru "github.com/hashicorp/golang-lru/v2"
)

type nzbCache struct {
	Nzb      *nzb.Nzb
	Metadata Metadata
}

type NzbLoader struct {
	cache *lru.Cache[string, *nzbCache]
	mx    sync.RWMutex
}

func NewNzbLoader(maxCacheSize int) (*NzbLoader, error) {
	cache, err := lru.New[string, *nzbCache](maxCacheSize)
	if err != nil {
		return nil, err
	}

	return &NzbLoader{
		cache: cache,
	}, nil
}

func (n *NzbLoader) LoadFromFile(name string) (*nzbCache, error) {
	n.mx.RLock()
	if nzb, ok := n.cache.Get(name); ok {
		n.mx.RUnlock()
		return nzb, nil
	}

	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	nzb, err := nzb.New(file)
	if err != nil {
		return nil, err
	}
	metadata, err := LoadMetadataFromNzb(nzb)
	if err != nil {
		return nil, err
	}

	nzbCache := &nzbCache{
		Nzb:      nzb,
		Metadata: metadata,
	}

	n.mx.RLock()
	n.cache.Add(name, nzbCache)
	n.mx.RUnlock()

	return nzbCache, nil
}

func (n *NzbLoader) LoadFromFileReader(f *os.File) (*nzbCache, error) {
	n.mx.RLock()
	if nzb, ok := n.cache.Get(f.Name()); ok {
		n.mx.RUnlock()
		return nzb, nil
	}

	nzb, err := nzb.New(f)
	if err != nil {
		return nil, err
	}
	metadata, err := LoadMetadataFromNzb(nzb)
	if err != nil {
		return nil, err
	}

	nzbCache := &nzbCache{
		Nzb:      nzb,
		Metadata: metadata,
	}

	n.mx.RLock()
	n.cache.Add(f.Name(), nzbCache)
	n.mx.RUnlock()

	return nzbCache, nil
}
