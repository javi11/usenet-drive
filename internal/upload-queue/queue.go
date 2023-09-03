package uploadqueue

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
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
	GetFailedJobs(ctx context.Context, limit, offset int) ([]sqllitequeue.Job, error)
	GetPendingJobs(ctx context.Context, limit, offset int) ([]sqllitequeue.Job, error)
	DeleteFailedJob(ctx context.Context, id int64) error
	RetryJob(ctx context.Context, id int64) error
	Close(ctx context.Context) error
	GetJobsInProgress() map[int64]sqllitequeue.Job
}

type uploadQueue struct {
	engine           sqllitequeue.SqlQueue
	uploader         uploader.Uploader
	activeJobs       map[int64]sqllitequeue.Job
	maxActiveUploads int
	log              *slog.Logger
	mx               *sync.Mutex
	closed           bool
}

func NewUploadQueue(options ...Option) UploadQueue {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	return &uploadQueue{
		engine:           config.SqlLiteEngine,
		uploader:         config.Uploader,
		maxActiveUploads: config.MaxActiveUploads,
		log:              config.Log,
		mx:               &sync.Mutex{},
		closed:           false,
	}
}

func (q *uploadQueue) AddJob(ctx context.Context, filePath string) error {
	q.log.InfoContext(ctx, "Adding file %s to upload queue", filePath)
	return q.engine.Enqueue(ctx, filePath)
}

func (q *uploadQueue) ProcessJob(ctx context.Context, job sqllitequeue.Job) error {
	log := q.log.With("job_id", job.ID).With("file_path", job.Data)

	log.InfoContext(ctx, "Uploading file...")

	nzbFilePath, err := q.uploader.UploadFile(ctx, job.Data)
	if err != nil {
		if os.IsNotExist(err) {
			// Corrupted files
			log.ErrorContext(ctx, "File does not exist, removing job...")
			if err != nil {
				return err
			}
		}

		log.ErrorContext(ctx, "Failed to upload file: %v. Adding to failed queue...", "err", err)
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
	q.log.InfoContext(ctx, fmt.Sprintf("Upload queue started with interval of %v seconds...", interval.Seconds()))

	ticker := time.NewTicker(interval)

	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.mx.Lock()
			if q.closed {
				q.mx.Unlock()
				return
			}

			inProgress := len(q.activeJobs)
			if inProgress < q.maxActiveUploads {
				jobs, err := q.engine.Dequeue(ctx, q.maxActiveUploads-inProgress)
				if err != nil {
					q.log.InfoContext(ctx, "Failed to dequeue jobs", "err", err)
					continue
				}

				if len(jobs) == 0 {
					continue
				}

				q.log.InfoContext(ctx, fmt.Sprintf("Processing %d jobs...", len(jobs)))
				var merr multierror.Group

				for _, job := range jobs {
					q.mx.Lock()
					q.activeJobs[job.ID] = job
					q.mx.Unlock()
					job := job

					merr.Go(func() error {
						defer func() {
							q.mx.Lock()
							delete(q.activeJobs, job.ID)
							q.mx.Unlock()
						}()
						return q.ProcessJob(ctx, job)
					})
				}

				err = merr.Wait().ErrorOrNil()
				if err != nil {
					q.log.ErrorContext(ctx, "Failed to process jobs", "err", err)
				}
			}
		}
	}
}

func (q *uploadQueue) GetFailedJobs(ctx context.Context, limit, offset int) ([]sqllitequeue.Job, error) {
	return q.engine.GetFailedJobs(ctx, limit, offset)
}

func (q *uploadQueue) GetPendingJobs(ctx context.Context, limit, offset int) ([]sqllitequeue.Job, error) {
	return q.engine.GetPendingJobs(ctx, limit, offset)
}

func (q *uploadQueue) DeleteFailedJob(ctx context.Context, id int64) error {
	err := q.engine.DeleteFailedJob(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrJobNotFound
		}
		return err
	}

	return nil
}

func (q *uploadQueue) RetryJob(ctx context.Context, id int64) error {
	job, err := q.engine.DequeueFailedJobById(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
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

func (q *uploadQueue) GetJobsInProgress() map[int64]sqllitequeue.Job {
	return q.activeJobs
}

func (q *uploadQueue) Close(ctx context.Context) error {
	q.mx.Lock()
	defer q.mx.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true

	// Mark all active jobs as failed with an error of closed
	for _, job := range q.activeJobs {
		job.Error = "upload failed: queue closed"
		err := q.engine.PushToFailedQueue(ctx, job.Data, "upload failed: queue closed")
		if err != nil {
			return err
		}
	}

	return nil
}
