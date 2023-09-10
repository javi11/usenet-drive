package uploadqueue

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/javi11/usenet-drive/internal/uploader"
	"github.com/javi11/usenet-drive/internal/utils"
	sqllitequeue "github.com/javi11/usenet-drive/pkg/sqllite-queue"
)

type UploadQueue interface {
	Start(ctx context.Context, interval time.Duration)
	AddJob(ctx context.Context, filePath string) error
	ProcessJob(ctx context.Context, job sqllitequeue.Job) error
	GetFailedJobs(ctx context.Context, limit, offset int) (sqllitequeue.Result, error)
	GetPendingJobs(ctx context.Context, limit, offset int) (sqllitequeue.Result, error)
	DeleteFailedJob(ctx context.Context, id int64) error
	DeletePendingJob(ctx context.Context, id int64) error
	RetryJob(ctx context.Context, id int64) error
	Close(ctx context.Context) error
	GetJobsInProgress() []sqllitequeue.Job
	GetActiveJobLog(id int64) (string, error)
}

type uploadQueue struct {
	rootPath         string
	tmpPath          string
	engine           sqllitequeue.SqlQueue
	uploader         uploader.Uploader
	activeJobs       map[int64]sqllitequeue.Job
	maxActiveUploads int
	log              *slog.Logger
	mx               *sync.RWMutex
	closed           bool
	fileWhitelist    []string
}

func NewUploadQueue(options ...Option) UploadQueue {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	return &uploadQueue{
		engine:           config.sqlLiteEngine,
		uploader:         config.uploader,
		maxActiveUploads: config.maxActiveUploads,
		log:              config.log,
		mx:               &sync.RWMutex{},
		closed:           false,
		activeJobs:       make(map[int64]sqllitequeue.Job, 0),
		fileWhitelist:    config.fileWhitelist,
	}
}

func (q *uploadQueue) AddJob(ctx context.Context, filePath string) error {
	q.log.InfoContext(ctx, "Adding file %s to upload queue", filePath)
	return q.engine.Enqueue(ctx, filePath)
}

func (q *uploadQueue) ProcessJob(ctx context.Context, job sqllitequeue.Job) error {
	log := q.log.With("job_id", job.ID).With("file_path", job.Data)

	q.log.InfoContext(ctx, "Adding file(s) to upload queue...")

	// Check if filePath is a directory
	fileInfo, err := os.Stat(job.Data)
	if err != nil {
		if os.IsNotExist(err) {
			// Corrupted files
			log.ErrorContext(ctx, "File does not exist, removing job...")
			if err != nil {
				return err
			}
		}

	}

	if fileInfo.IsDir() {
		q.log.InfoContext(ctx, "File is a directory, adding all files to upload queue...")
		// Walk through the directory and add all files to the queue
		err = filepath.Walk(job.Data, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Check if the file is allowed
			if !utils.HasAllowedExtension(info.Name(), q.fileWhitelist) {
				q.log.InfoContext(ctx, fmt.Sprintf("File %s ignored, extension not allowed", path))
				return nil
			}

			// Add the file to the queue
			err = q.engine.Enqueue(ctx, path)
			if err != nil {
				return err
			}

			q.log.InfoContext(ctx, fmt.Sprintf("Added file %s to upload queue", path))
			return nil
		})

		if err != nil {
			log.ErrorContext(ctx, "Error adding the directory files...", "err", err)
			return err
		}

		return nil
	}

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

	log.InfoContext(ctx, "File uploaded successfully")

	// Remove .tmp extension from nzbFilePath
	newFilePath := nzbFilePath[:len(nzbFilePath)-len(uploader.TmpExtension)]

	log.DebugContext(ctx, "Renaming nzb file and removing original file", "old_path", nzbFilePath, "new_path", newFilePath)
	err = os.Rename(nzbFilePath, newFilePath)
	if err != nil {
		log.ErrorContext(ctx, "Failed to rename nzb file", "err", err)
		return err
	}

	if originalFile, err := os.Readlink(job.Data); err == nil {
		// If the file is a symlink, remove the original file
		if err := os.Remove(originalFile); err != nil {
			return err
		}
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
			q.mx.RLock()
			if q.closed {
				q.mx.RLocker()
				return
			}
			q.mx.RUnlock()

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

func (q *uploadQueue) GetFailedJobs(ctx context.Context, limit, offset int) (sqllitequeue.Result, error) {
	return q.engine.GetFailedJobs(ctx, limit, offset)
}

func (q *uploadQueue) GetPendingJobs(ctx context.Context, limit, offset int) (sqllitequeue.Result, error) {
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

func (q *uploadQueue) DeletePendingJob(ctx context.Context, id int64) error {
	err := q.engine.DeletePendingJob(ctx, id)
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

func (q *uploadQueue) GetJobsInProgress() []sqllitequeue.Job {
	jobs := make([]sqllitequeue.Job, 0, len(q.activeJobs))
	for _, job := range q.activeJobs {
		jobs = append(jobs, job)
	}

	return jobs
}

func (q *uploadQueue) GetActiveJobLog(id int64) (string, error) {
	q.mx.RLock()
	defer q.mx.RUnlock()

	for _, job := range q.activeJobs {
		if job.ID == id {
			return q.uploader.GetActiveUploadLog(job.Data)
		}
	}

	return "", ErrJobNotFound
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
