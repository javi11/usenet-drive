package filereader

import (
	"sync"
)

type downloadNotifier struct {
	ch         chan bool
	downloaded bool
	Chunk      []byte
}

func (d *downloadNotifier) Grow(size int) {
	d.Chunk = append(d.Chunk, make([]byte, size)...)
}

func (d *downloadNotifier) Reduce(size int) {
	d.Chunk = d.Chunk[:size]
}

func (d *downloadNotifier) IsDownloaded() bool {
	return d.downloaded
}

func (d *downloadNotifier) Wait() <-chan bool {
	if d.ch != nil {
		return d.ch
	}

	return nil
}

func (d *downloadNotifier) Start() {
	d.ch = make(chan bool, 1)
}

func (d *downloadNotifier) Reset() {
	d.downloaded = false
	if d.ch != nil {
		d.ch <- false
		close(d.ch)
		d.ch = nil
	}
}

func (d *downloadNotifier) Notify() {
	if d.ch != nil {
		d.downloaded = true
		d.ch <- true
		close(d.ch)
		d.ch = nil
	}
}

type currentDownloadingMap struct {
	sync.Map
}

func (cd *currentDownloadingMap) Get(segmentIndex int) *downloadNotifier {
	v, ok := cd.Load(segmentIndex)
	if !ok {
		return nil
	}

	return v.(*downloadNotifier)
}

func (cd *currentDownloadingMap) DeleteBefore(segmentIndex int, chunkPool *sync.Pool) {
	cd.Range(func(key, value interface{}) bool {
		if key.(int) < segmentIndex {
			nf := value.(*downloadNotifier)
			nf.Reset()
			chunkPool.Put(nf)
			cd.Delete(key)
		}

		return true
	})
}

func (cd *currentDownloadingMap) DeleteAfter(segmentIndex int, chunkPool *sync.Pool) {
	cd.Range(func(key, value interface{}) bool {
		if key.(int) > segmentIndex {
			nf := value.(*downloadNotifier)
			nf.Reset()
			chunkPool.Put(nf)
			cd.Delete(key)
		}

		return true
	})
}

func (cd *currentDownloadingMap) DeleteAll(chunkPool *sync.Pool) {
	cd.Range(func(key, value interface{}) bool {
		nf := value.(*downloadNotifier)
		nf.Reset()
		chunkPool.Put(nf)
		cd.Delete(key)
		return true
	})
}

func (cd *currentDownloadingMap) Len() int {
	length := 0
	cd.Range(func(key, value interface{}) bool {
		length++
		return true
	})

	return length
}
