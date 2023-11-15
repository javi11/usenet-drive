//go:generate mockgen -source=./status.go -destination=./status_mock.go -package=status StatusReporter
package status

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
)

type Kind string

const (
	Upload   Kind = "upload"
	Download Kind = "download"
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
}

func NewStatusReporter() StatusReporter {
	return &statusReporter{
		status:   make(map[uuid.UUID]*status),
		reporter: make(chan *reporterData, 10000),
	}
}

func (s *statusReporter) Start(ctx context.Context, ticker *time.Ticker) {
	for t := range ticker.C {
		stamp := t.UnixNano() / 1e6
		for _, status := range s.status {
			status.tds = append(status.tds, &TimeData{stamp, 0})
		}

		// Fetch any new TimeData entries
		var breakNow bool
		for {
			breakNow = false

			select {
			case <-ctx.Done():
				return
			case td := <-s.reporter:
				// New item, add it to our list
				status := s.status[td.id]
				if status == nil {
					continue
				}

				status.tds = append(status.tds, td.td)
				status.TotalBytes += int64(td.td.Bytes)
			default:
				// Nothing else in the channel, done for now
				breakNow = true
			}

			if breakNow {
				break
			}
		}

		for _, status := range s.status {
			tds := status.tds
			if len(tds) > 0 {
				active := float64(tds[len(tds)-1].Milliseconds-tds[0].Milliseconds) / 1000
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
	}
}

func (s *statusReporter) AddTimeData(id uuid.UUID, data *TimeData) {
	s.reporter <- &reporterData{
		id: id,
		td: data,
	}
}

func (s *statusReporter) StartUpload(id uuid.UUID, path string) {
	s.status[id] = &status{
		Kind: Upload,
		Path: path,
	}
}

func (s *statusReporter) StartDownload(id uuid.UUID, path string) {
	s.status[id] = &status{
		Kind: Download,
		Path: path,
	}
}

func (s *statusReporter) FinishUpload(id uuid.UUID) {
	delete(s.status, id)
}

func (s *statusReporter) FinishDownload(id uuid.UUID) {
	delete(s.status, id)
}

func (s *statusReporter) GetStatus() map[uuid.UUID]*status {
	return s.status
}
