package nzbloader

//go:generate mockgen -source=./nzbloader.go -destination=./nzbloader_mock.go -package=nzbloader NzbLoader

import (
	"context"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/javi11/usenet-drive/pkg/osfs"
)

type NzbLoader interface {
	LoadFromFile(name string) (*NzbCache, error)
	LoadFromFileReader(f osfs.File) (*NzbCache, error)
	EvictFromCache(name string) bool
	RefreshCachedNzb(name string, nzb *nzb.Nzb) (bool, error)
}

type NzbCache struct {
	Nzb      *nzb.Nzb
	Metadata *usenet.Metadata
}

type nzbLoader struct {
	cache     *lru.Cache[string, *NzbCache]
	cNzb      corruptednzbsmanager.CorruptedNzbsManager
	fs        osfs.FileSystem
	nzbParser nzb.NzbParser
}

func NewNzbLoader(
	maxCacheSize int,
	cNzb corruptednzbsmanager.CorruptedNzbsManager,
	fs osfs.FileSystem,
	nzbParser nzb.NzbParser,
) (NzbLoader, error) {
	cache, err := lru.New[string, *NzbCache](maxCacheSize)
	if err != nil {
		return nil, err
	}

	return &nzbLoader{
		cache:     cache,
		cNzb:      cNzb,
		fs:        fs,
		nzbParser: nzbParser,
	}, nil
}

func (n *nzbLoader) LoadFromFile(name string) (*NzbCache, error) {
	if nzb, ok := n.cache.Get(name); ok {
		return nzb, nil
	}

	file, err := n.fs.Open(name)
	if err != nil {
		return nil, err
	}

	return n.loadNzb(file)
}

func (n *nzbLoader) LoadFromFileReader(f osfs.File) (*NzbCache, error) {
	if nzb, ok := n.cache.Get(f.Name()); ok {
		return nzb, nil
	}

	return n.loadNzb(f)
}

func (n *nzbLoader) EvictFromCache(name string) bool {
	if n.cache.Contains(name) {
		return n.cache.Remove(name)
	}

	return false
}

func (n *nzbLoader) RefreshCachedNzb(name string, nzb *nzb.Nzb) (bool, error) {
	metadata, err := usenet.LoadMetadataFromNzb(nzb)
	if err != nil {
		return false, err
	}

	return n.cache.Add(name, &NzbCache{
		Nzb:      nzb,
		Metadata: metadata,
	}), nil
}

func (n *nzbLoader) loadNzb(f osfs.File) (*NzbCache, error) {
	nzb, err := n.nzbParser.Parse(f)
	if err != nil {
		if e := n.cNzb.Add(context.Background(), f.Name(), err.Error()); e != nil {
			return nil, e
		}

		return nil, err
	}
	metadata, err := usenet.LoadMetadataFromNzb(nzb)
	if err != nil {
		if e := n.cNzb.Add(context.Background(), f.Name(), err.Error()); e != nil {
			return nil, e
		}

		return nil, err
	}

	nzbCache := &NzbCache{
		Nzb:      nzb,
		Metadata: metadata,
	}

	n.cache.Add(f.Name(), nzbCache)

	return nzbCache, nil
}
