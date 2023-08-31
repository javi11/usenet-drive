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
	"github.com/mattn/go-sqlite3"
)

type UploadQueue interface {
	Start(ctx context.Context, interval time.Duration)
	AddJob(ctx context.Context, filePath string) error
	ProcessJob(ctx context.Context, job sqllitequeue.Job) error
	GetFailedJobs(ctx context.Context) ([]sqllitequeue.Job, error)
	GetPendingJobs(ctx context.Context) ([]sqllitequeue.Job, error)
	DeleteFailedJob(ctx context.Context, id int64) error
	RetryJob(ctx context.Context, id int64) error
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

		q.log.Printf("Failed to upload file %v: %v. Adding to failed queue...", job.Data, err)
		return q.engine.PushToFailedQueue(ctx, job.Data, err.Error())
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
	q.log.Printf("Upload queue started with interval of %v seconds...", interval.Seconds())

	ticker := time.NewTicker(interval)

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
					})
				}

				err = merr.Wait().ErrorOrNil()
				if err != nil {
					q.log.Printf("Failed to process jobs: %v", err)
				}
			}
		}
	}
}

func (q *uploadQueue) GetFailedJobs(ctx context.Context) ([]sqllitequeue.Job, error) {
	return q.engine.GetFailedJobs(ctx)
}

func (q *uploadQueue) GetPendingJobs(ctx context.Context) ([]sqllitequeue.Job, error) {
	return q.engine.GetPendingJobs(ctx)
}

func (q *uploadQueue) DeleteFailedJob(ctx context.Context, id int64) error {
	err := q.engine.DeleteFailedJob(ctx, id)
	if err != nil {
		if err == sqlite3.ErrNotFound {
			return ErrJobNotFound
		}
		return err
	}

	return nil
}

func (q *uploadQueue) RetryJob(ctx context.Context, id int64) error {
	job, err := q.engine.DequeueFailedJobById(ctx, id)
	if err != nil {
		if err == sqlite3.ErrNotFound {
			return ErrJobNotFound
		}
		return err
	}

	err = q.engine.Enqueue(ctx, job.Data)
	if err != nil {
		return err
	}

	return nil
}
