package sqllitequeue

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type SqlQueue interface {
	Enqueue(ctx context.Context, data string) error
	Dequeue(ctx context.Context, limit int) ([]Job, error)
	Delete(ctx context.Context, id int64) error
	PushToFailedQueue(ctx context.Context, data string, error string) error
	GetFailedJobs(ctx context.Context, limit, offset int) (Result, error)
	DeleteFailedJob(ctx context.Context, id int64) error
	GetPendingJobs(ctx context.Context, limit, offset int) (Result, error)
	DequeueFailedJobById(ctx context.Context, id int64) (Job, error)
	DeletePendingJob(ctx context.Context, id int64) error
}

type Result struct {
	Entries    []Job `json:"entries"`
	TotalCount int   `json:"totalCount"`
	Offset     int   `json:"offset"`
	Limit      int   `json:"limit"`
}

type Job struct {
	ID        int64     `json:"id"`
	Data      string    `json:"data"`
	CreatedAt time.Time `json:"createdAt"`
	Error     string    `json:"error,omitempty"`
}

type sQLiteQueue struct {
	db *sql.DB
}

func NewSQLiteQueue(db *sql.DB) (SqlQueue, error) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS queue (
			id INTEGER PRIMARY KEY,
			data TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS failed_queue (
			id INTEGER PRIMARY KEY,
			data TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			error TEXT
		);
	`)
	if err != nil {
		return nil, err
	}

	return &sQLiteQueue{db: db}, nil
}

func (q *sQLiteQueue) Enqueue(ctx context.Context, data string) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO queue (data) VALUES (?)")
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx, data)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (q *sQLiteQueue) Dequeue(ctx context.Context, limit int) ([]Job, error) {
	if limit < 1 {
		return nil, errors.New("limit must be greater than 0")
	}

	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf("SELECT id, data, created_at FROM queue ORDER BY created_at ASC LIMIT %v", limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job = make([]Job, 0)
	for rows.Next() {
		var id int
		var data string
		var createdAt time.Time
		err = rows.Scan(&id, &data, &createdAt)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, Job{
			ID:        int64(id),
			Data:      data,
			CreatedAt: createdAt,
		})

		_, err := tx.ExecContext(ctx, "DELETE FROM queue WHERE id = ?", id)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

func (q *sQLiteQueue) DequeueFailedJobById(ctx context.Context, id int64) (Job, error) {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, err
	}

	row := tx.QueryRowContext(ctx, "SELECT id, data, created_at, error FROM failed_queue WHERE id = ?", id)

	var j Job
	err = row.Scan(&j.ID, &j.Data, &j.CreatedAt, &j.Error)
	if err != nil {
		return Job{}, err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM failed_queue WHERE id = ?", id)
	if err != nil {
		return Job{}, err
	}

	err = tx.Commit()
	if err != nil {
		return Job{}, err
	}

	return j, nil
}

func (q *sQLiteQueue) Delete(ctx context.Context, id int64) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM queue WHERE id = ?", id)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (q *sQLiteQueue) PushToFailedQueue(ctx context.Context, data string, error string) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO failed_queue (data, error) VALUES (?, ?)")
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx, data, error)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (q *sQLiteQueue) GetFailedJobs(ctx context.Context, limit, offset int) (Result, error) {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, err
	}

	// Get the total count of items in the failed_queue table
	var totalCount int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM failed_queue").Scan(&totalCount)
	if err != nil {
		return Result{}, err
	}

	rows, err := tx.QueryContext(
		ctx,
		fmt.Sprintf("SELECT id, data, created_at, error FROM failed_queue ORDER BY created_at ASC LIMIT %v OFFSET %v", limit, offset),
	)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()

	var jobs []Job = make([]Job, 0)
	for rows.Next() {
		var id int
		var data string
		var createdAt time.Time
		var error string
		err = rows.Scan(&id, &data, &createdAt, &error)
		if err != nil {
			return Result{}, err
		}

		jobs = append(jobs, Job{
			ID:        int64(id),
			Data:      data,
			CreatedAt: createdAt,
			Error:     error,
		})
	}

	err = tx.Commit()
	if err != nil {
		return Result{}, err
	}

	return Result{
		Entries:    jobs,
		TotalCount: totalCount,
		Offset:     offset,
		Limit:      limit,
	}, nil
}

func (q *sQLiteQueue) DeleteFailedJob(ctx context.Context, id int64) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM failed_queue WHERE id = ?", id)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (q *sQLiteQueue) DeletePendingJob(ctx context.Context, id int64) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM queue WHERE id = ?", id)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (q *sQLiteQueue) GetPendingJobs(ctx context.Context, limit, offset int) (Result, error) {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, err
	}

	// Get the total count of items in the failed_queue table
	var totalCount int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue").Scan(&totalCount)
	if err != nil {
		return Result{}, err
	}

	rows, err := tx.QueryContext(
		ctx,
		fmt.Sprintf("SELECT id, data, created_at FROM queue ORDER BY created_at ASC LIMIT %v OFFSET %v", limit, offset),
	)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()

	var jobs []Job = make([]Job, 0)
	for rows.Next() {
		var id int
		var data string
		var createdAt time.Time
		err = rows.Scan(&id, &data, &createdAt)
		if err != nil {
			return Result{}, err
		}

		jobs = append(jobs, Job{
			ID:        int64(id),
			Data:      data,
			CreatedAt: createdAt,
		})
	}

	err = tx.Commit()
	if err != nil {
		return Result{}, err
	}

	return Result{
		Entries:    jobs,
		TotalCount: totalCount,
		Offset:     offset,
		Limit:      limit,
	}, nil
}
