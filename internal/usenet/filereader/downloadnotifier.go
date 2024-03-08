package filereader

import "sync"

type downloadNotifier struct {
	ch         chan bool
	downloaded bool
	Chunk      []byte
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

func (cd *currentDownloadingMap) DeleteBefore(segmentIndex int) {
	cd.Range(func(key, value interface{}) bool {
		if key.(int) < segmentIndex {
			cd.Delete(key)
		}

		return true
	})
}

func (cd *currentDownloadingMap) DeleteAfter(segmentIndex int) {
	cd.Range(func(key, value interface{}) bool {
		if key.(int) > segmentIndex {
			cd.Delete(key)
		}

		return true
	})
}
