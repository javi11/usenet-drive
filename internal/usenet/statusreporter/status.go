//go:generate mockgen -source=./status.go -destination=./status_mock.go -package=status StatusReporter
package status

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Kind string

const (
	Upload          Kind = "upload"
	Download        Kind = "download"
	NoReportTimeout      = 30 * time.Second
)

type reporterData struct {
	id uuid.UUID
	td *TimeData
}

type TimeData struct {
	Milliseconds int64
	Bytes        int64
}

type status struct {
	Kind         Kind
	Path         string
	tds          []*TimeData
	TotalBytes   int64
	CurrentSpeed float64
	LastUpdate   time.Time
}

type StatusReporter interface {
	AddTimeData(id uuid.UUID, data *TimeData)
	StartUpload(id uuid.UUID, path string)
	StartDownload(id uuid.UUID, path string)
	FinishUpload(id uuid.UUID)
	FinishDownload(id uuid.UUID)
	Start(ctx context.Context, ticker *time.Ticker)
	GetStatus() map[uuid.UUID]*status
}

type statusReporter struct {
	status   map[uuid.UUID]*status
	reporter chan *reporterData
	mx       *sync.RWMutex
}

func NewStatusReporter() StatusReporter {
	return &statusReporter{
		status:   make(map[uuid.UUID]*status),
		reporter: make(chan *reporterData, 10000),
		mx:       &sync.RWMutex{},
	}
}

func (s *statusReporter) Start(ctx context.Context, ticker *time.Ticker) {
	for t := range ticker.C {
		s.mx.Lock()
		stamp := t.UnixNano() / 1e6
		for _, status := range s.status {
			status.tds = append(status.tds, &TimeData{stamp, 0})
		}
		s.mx.Unlock()

		// Fetch any new TimeData entries
		var breakNow bool
		for {
			breakNow = false

			select {
			case <-ctx.Done():
				return
			case td := <-s.reporter:
				s.mx.Lock()
				// New item, add it to our list
				status := s.status[td.id]
				if status == nil {
					s.mx.Unlock()
					continue
				}

				status.tds = append(status.tds, td.td)
				status.TotalBytes += int64(td.td.Bytes)
				status.LastUpdate = time.Now()
				s.mx.Unlock()
			default:
				// Nothing else in the channel, done for now
				breakNow = true
			}

			if breakNow {
				break
			}
		}

		s.mx.Lock()
		for key, status := range s.status {
			// If we haven't received any updates in a while, remove it
			if status.LastUpdate.Before(time.Now().Add(-NoReportTimeout)) {
				delete(s.status, key)
				continue
			}
			tds := status.tds
			if len(tds) > 0 {
				active := float64(tds[len(tds)-1].Milliseconds-tds[0].Milliseconds) / 1000

				if active == 0 {
					status.CurrentSpeed = 0
					continue
				}

				totalBytes := int64(0)
				for _, td := range tds {
					totalBytes += td.Bytes
				}
				speed := float64(totalBytes) / float64(active)
				status.CurrentSpeed = math.Abs(speed)
			}

			// Trim slice to only use the last 5 seconds
			earliest := stamp - 5000
			start := 0
			for i, td := range tds {
				if td.Milliseconds >= earliest {
					start = i
					break
				}
			}

			status.tds = tds[start:]
		}
		s.mx.Unlock()
	}
}

func (s *statusReporter) AddTimeData(id uuid.UUID, data *TimeData) {
	s.reporter <- &reporterData{
		id: id,
		td: data,
	}
}

func (s *statusReporter) StartUpload(id uuid.UUID, path string) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.status[id] = &status{
		Kind: Upload,
		Path: path,
	}
}

func (s *statusReporter) StartDownload(id uuid.UUID, path string) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.status[id] = &status{
		Kind: Download,
		Path: path,
	}
}

func (s *statusReporter) FinishUpload(id uuid.UUID) {
	s.mx.Lock()
	defer s.mx.Unlock()

	delete(s.status, id)
}

func (s *statusReporter) FinishDownload(id uuid.UUID) {
	s.mx.Lock()
	defer s.mx.Unlock()

	delete(s.status, id)
}

func (s *statusReporter) GetStatus() map[uuid.UUID]*status {
	s.mx.RLock()
	defer s.mx.RUnlock()

	statusCopy := s.status

	return statusCopy
}
