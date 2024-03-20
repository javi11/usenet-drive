package filereader

import "sync"

type chunkCache struct {
	sync.Map
}

func (cd *chunkCache) Get(segmentIndex int) *downloadManager {
	v, ok := cd.Load(segmentIndex)
	if !ok {
		return nil
	}

	return v.(*downloadManager)
}

func (cd *chunkCache) DeleteBefore(segmentIndex int, chunkPool *sync.Pool) {
	cd.Range(func(key, value interface{}) bool {
		if key.(int) < segmentIndex-1 {
			nf := value.(*downloadManager)
			nf.Reset()
			chunkPool.Put(nf)
			cd.Delete(key)
		}

		return true
	})
}

func (cd *chunkCache) DeleteAfter(segmentIndex int, chunkPool *sync.Pool) {
	cd.Range(func(key, value interface{}) bool {
		if key.(int) > segmentIndex {
			nf := value.(*downloadManager)
			nf.Reset()
			chunkPool.Put(nf)
			cd.Delete(key)
		}

		return true
	})
}

func (cd *chunkCache) DeleteAll(chunkPool *sync.Pool) {
	cd.Range(func(key, value interface{}) bool {
		nf := value.(*downloadManager)
		nf.Reset()
		chunkPool.Put(nf)
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