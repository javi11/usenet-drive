package usenet

import (
	"context"
	"os"

	"github.com/chrisfarms/nzb"
	lru "github.com/hashicorp/golang-lru/v2"
	corruptednzbsmanager "github.com/javi11/usenet-drive/internal/corrupted-nzbs-manager"
)

type nzbCache struct {
	Nzb      *nzb.Nzb
	Metadata Metadata
}

type NzbLoader struct {
	cache *lru.Cache[string, *nzbCache]
	cNzb  corruptednzbsmanager.CorruptedNzbsManager
}

func NewNzbLoader(maxCacheSize int, cNzb corruptednzbsmanager.CorruptedNzbsManager) (*NzbLoader, error) {
	cache, err := lru.New[string, *nzbCache](maxCacheSize)
	if err != nil {
		return nil, err
	}

	return &NzbLoader{
		cache: cache,
		cNzb:  cNzb,
	}, nil
}

func (n *NzbLoader) LoadFromFile(name string) (*nzbCache, error) {
	if nzb, ok := n.cache.Get(name); ok {
		return nzb, nil
	}

	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	nzb, err := nzb.New(file)
	if err != nil {
		if err = n.cNzb.Add(context.Background(), name, err.Error()); err != nil {
			return nil, err
		}
	}
	metadata, err := LoadMetadataFromNzb(nzb)
	if err != nil {
		if err = n.cNzb.Add(context.Background(), name, err.Error()); err != nil {
			return nil, err
		}
	}

	nzbCache := &nzbCache{
		Nzb:      nzb,
		Metadata: metadata,
	}

	n.cache.Add(name, nzbCache)

	return nzbCache, nil
}

func (n *NzbLoader) LoadFromFileReader(f *os.File) (*nzbCache, error) {
	if nzb, ok := n.cache.Get(f.Name()); ok {
		return nzb, nil
	}

	nzb, err := nzb.New(f)
	if err != nil {
		if err = n.cNzb.Add(context.Background(), f.Name(), err.Error()); err != nil {
			return nil, err
		}
	}
	metadata, err := LoadMetadataFromNzb(nzb)
	if err != nil {
		if err = n.cNzb.Add(context.Background(), f.Name(), err.Error()); err != nil {
			return nil, err
		}
	}

	nzbCache := &nzbCache{
		Nzb:      nzb,
		Metadata: metadata,
	}

	n.cache.Add(f.Name(), nzbCache)

	return nzbCache, nil
}

func (n *NzbLoader) EvictFromCache(name string) bool {
	if n.cache.Contains(name) {
		return n.cache.Remove(name)
	}

	return false
}

func (n *NzbLoader) RefreshCachedNzb(name string, nzb *nzb.Nzb) (bool, error) {
	metadata, err := LoadMetadataFromNzb(nzb)
	if err != nil {
		return false, err
	}

	return n.cache.Add(name, &nzbCache{
		Nzb:      nzb,
		Metadata: metadata,
	}), nil
}
