package corruptednzbsmanager

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"
)

type CorruptedNzbsManager interface {
	Add(ctx context.Context, path, error string) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) (Result, error)
}

type Result struct {
	Entries    []cNzb `json:"entries"`
	TotalCount int    `json:"total_count"`
	Offset     int    `json:"offset"`
	Limit      int    `json:"limit"`
}

type cNzb struct {
	ID        int64     `json:"id"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	Error     string    `json:"error"`
}

type corruptedNzbsManager struct {
	db *sql.DB
}

func New(db *sql.DB) (CorruptedNzbsManager, error) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS corrupted_nzbs (
			id INTEGER PRIMARY KEY,
			path TEXT UNIQUE,
			error TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return nil, err
	}

	return &corruptedNzbsManager{db: db}, nil
}

func (q *corruptedNzbsManager) Add(ctx context.Context, path, error string) error {
	stmt, err := q.db.PrepareContext(ctx, "INSERT INTO corrupted_nzbs (path, error) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, path, error)
	if err != nil {
		return err
	}

	return nil
}

func (q *corruptedNzbsManager) Delete(ctx context.Context, id int64) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	row := tx.QueryRowContext(ctx, "SELECT id, path, created_at FROM corrupted_nzbs WHERE id = ?", id)

	var j cNzb
	err = row.Scan(&j.ID, &j.Path, &j.CreatedAt)
	if err != nil {
		tx.Commit()
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM corrupted_nzbs WHERE id = ?", id)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	if _, err := os.Stat(j.Path); err == nil {
		err = os.Remove(j.Path)
		if err != nil {
			return err
		}
	}

	return nil
}

func (q *corruptedNzbsManager) List(ctx context.Context, limit, offset int) (Result, error) {
	// Get the total count of items in the failed_queue table
	var totalCount int
	err := q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM corrupted_nzbs").Scan(&totalCount)
	if err != nil {
		return Result{}, err
	}

	rows, err := q.db.QueryContext(
		ctx,
		fmt.Sprintf("SELECT id, path, created_at, error FROM corrupted_nzbs ORDER BY created_at ASC LIMIT %v OFFSET %v", limit, offset),
	)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()

	var jobs []cNzb = make([]cNzb, 0)
	for rows.Next() {
		var id int
		var path string
		var createdAt time.Time
		var error string
		err = rows.Scan(&id, &path, &createdAt, &error)
		if err != nil {
			return Result{}, err
		}

		jobs = append(jobs, cNzb{
			ID:        int64(id),
			Path:      path,
			CreatedAt: createdAt,
			Error:     error,
		})
	}

	return Result{
		Entries:    jobs,
		TotalCount: totalCount,
		Offset:     offset,
		Limit:      limit,
	}, nil
}
