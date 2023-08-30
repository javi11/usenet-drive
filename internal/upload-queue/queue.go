package uploadqueue

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/javi11/usenet-drive/internal/uploader"
	sqllitequeue "github.com/javi11/usenet-drive/pkg/sqllite-queue"
)

type UploadQueue interface {
	Start(ctx context.Context, interval time.Duration)
	AddJob(ctx context.Context, filePath string) error
	ProcessJob(ctx context.Context, job sqllitequeue.Job) error
}

type uploadQueue struct {
	engine           sqllitequeue.SqlQueue
	uploader         uploader.Uploader
	activeUploads    int
	maxActiveUploads int
	log              *log.Logger
	mx               *sync.Mutex
	closed           bool
}

func NewUploadQueue(engine sqllitequeue.SqlQueue, uploader uploader.Uploader, maxActiveUploads int, log *log.Logger) UploadQueue {
	return &uploadQueue{
		engine:           engine,
		uploader:         uploader,
		maxActiveUploads: maxActiveUploads,
		log:              log,
		mx:               &sync.Mutex{},
		closed:           false,
	}
}

func (q *uploadQueue) AddJob(ctx context.Context, filePath string) error {
	q.log.Printf("Adding file %s to upload queue", filePath)
	return q.engine.Enqueue(ctx, filePath)
}

func (q *uploadQueue) ProcessJob(ctx context.Context, job sqllitequeue.Job) error {
	q.log.Printf("Uploading file %v...", job.Data)
	nzbFilePath, err := q.uploader.UploadFile(ctx, job.Data)
	if err != nil {
		if os.IsNotExist(err) {
			// Corrupted files
			q.log.Printf("File %v does not exist, removing job...", job.Data)
			if err != nil {
				return err
			}
		}

		q.log.Printf("Failed to upload file %v: %v. Retrying...", job.Data, err)
		return q.engine.Enqueue(ctx, job.Data)
	}

	// Remove .tmp extension from nzbFilePath
	newFilePath := nzbFilePath[:len(nzbFilePath)-4]

	err = os.Rename(nzbFilePath, newFilePath)
	if err != nil {
		return err
	}

	err = os.Remove(job.Data)
	if err != nil {
		return err
	}

	err = q.engine.Delete(ctx, job.ID)
	if err != nil {
		return err
	}

	return nil
}

func (q *uploadQueue) Start(ctx context.Context, interval time.Duration) {
	q.log.Printf("Starting upload queue with interval of %v seconds...", interval.Seconds())

	ticker := time.NewTicker(interval)

	go func(ticker *time.Ticker) {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if q.activeUploads < q.maxActiveUploads {
					jobs, err := q.engine.Dequeue(ctx, q.maxActiveUploads-q.activeUploads)
					if err != nil {
						q.log.Printf("Failed to dequeue jobs: %v", err)
						continue
					}

					if len(jobs) == 0 {
						q.log.Printf("No jobs to process...")
						continue
					}

					q.log.Printf("Processing %d jobs...", len(jobs))
					var merr multierror.Group

					for _, job := range jobs {
						q.mx.Lock()
						q.activeUploads++
						q.mx.Unlock()
						job := job

						merr.Go(func() error {
							defer func() {
								q.mx.Lock()
								q.activeUploads--
								q.mx.Unlock()
							}()
							return q.ProcessJob(ctx, job)

							return nil
						})
					}

					err = merr.Wait().ErrorOrNil()
					if err != nil {
						q.log.Printf("Failed to process jobs: %v", err)
					}
				}
			}
		}
	}(ticker)
}
