//go:generate mockgen -source=./chunkcache.go -destination=./chunkcache_mock.go -package=filereader ChunkCache

package filereader

import "sync"

type ChunkCache interface {
	Get(segmentIndex int) *downloadManager
	DeleteBefore(segmentIndex int)
	DeleteAfter(segmentIndex int)
	Close()
	Len() int
	LoadOrStore(key any, value any) (any, bool)
	Delete(segmentIndex any)
}

type chunkCache struct {
	sync.Map
	pool *sync.Pool
}

func NewChunkCache(p *sync.Pool) ChunkCache {
	return &chunkCache{
		pool: p,
	}
}

func (cd *chunkCache) Get(segmentIndex int) *downloadManager {
	v, ok := cd.Load(segmentIndex)
	if !ok {
		return nil
	}

	return v.(*downloadManager)
}

func (cd *chunkCache) DeleteBefore(segmentIndex int) {
	cd.Range(func(key, value interface{}) bool {
		if key.(int) < segmentIndex {
			nf := value.(*downloadManager)
			nf.mx.RLock()
			if nf.reading {
				nf.mx.RUnlock()
				return true
			}
			nf.mx.RUnlock()

			nf.Reset()
			cd.pool.Put(nf)
			cd.Delete(key)
		}

		return true
	})
}

func (cd *chunkCache) DeleteAfter(segmentIndex int) {
	cd.Range(func(key, value interface{}) bool {
		if key.(int) > segmentIndex {
			nf := value.(*downloadManager)
			nf.mx.RLock()
			if nf.reading {
				nf.mx.RUnlock()
				return true
			}
			nf.mx.RUnlock()

			nf.Reset()
			cd.pool.Put(nf)
			cd.Delete(key)
		}

		return true
	})
}

func (cd *chunkCache) Close() {
	cd.Range(func(key, value interface{}) bool {
		nf := value.(*downloadManager)
		nf.Reset()
		cd.pool.Put(nf)
		cd.Delete(key)
		return true
	})
}

func (cd *chunkCache) Len() int {
	length := 0
	cd.Range(func(key, value interface{}) bool {
		length++
		return true
	})

	return length
}
