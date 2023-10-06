package corruptednzbsmanager

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/utils"
)

type Filters struct {
	Path      utils.Filter `json:"path"`
	CreatedAt utils.Filter `json:"created_at"`
	Error     utils.Filter `json:"error"`
}

type SortBy struct {
	Path      utils.SortByDirection `json:"path"`
	CreatedAt utils.SortByDirection `json:"created_at"`
	Error     utils.SortByDirection `json:"error"`
}

type CorruptedNzbsManager interface {
	Add(ctx context.Context, path, error string) error
	Delete(ctx context.Context, path string) error
	Discard(ctx context.Context, path string) error
	Update(ctx context.Context, oldPath, newPath string) error
	List(ctx context.Context, limit, offset int, filters *Filters, sortBy *SortBy) (Result, error)
	GetFileContent(ctx context.Context, id int) (io.ReadCloser, error)
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
	stmt, err := q.db.PrepareContext(ctx, "INSERT OR IGNORE INTO corrupted_nzbs (path, error) VALUES (?, ?)")
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

func (q *corruptedNzbsManager) Delete(ctx context.Context, path string) error {
	err := q.Discard(ctx, path)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		return os.Remove(path)
	}

	return nil
}

func (q *corruptedNzbsManager) Discard(ctx context.Context, path string) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	row := tx.QueryRowContext(ctx, "SELECT id, path, created_at FROM corrupted_nzbs WHERE path = ?", path)

	var j cNzb
	err = row.Scan(&j.ID, &j.Path, &j.CreatedAt)
	if err != nil {
		tx.Commit()
		return nil
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM corrupted_nzbs WHERE path = ?", path)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (q *corruptedNzbsManager) Update(ctx context.Context, oldPath, newPath string) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	row := tx.QueryRowContext(ctx, "SELECT id, path, created_at FROM corrupted_nzbs WHERE path = ?", oldPath)

	var j cNzb
	err = row.Scan(&j.ID, &j.Path, &j.CreatedAt)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "UPDATE corrupted_nzbs SET path = ? WHERE id = ?", newPath, j.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (q *corruptedNzbsManager) List(ctx context.Context, limit, offset int, filters *Filters, sortBy *SortBy) (Result, error) {

	sqlFilterBuilder := utils.NewSqlFilterBuilder()
	var queryParams []any

	// Build the WHERE clause for the query based on the filters
	if filters != nil {
		if filters.Path.Value != "" {
			queryParams = append(queryParams, sqlFilterBuilder.AddFilter("path", filters.Path))
		}
		if filters.CreatedAt.Value != "" {
			queryParams = append(queryParams, sqlFilterBuilder.AddFilter("created_at", filters.CreatedAt))
		}
		if filters.Error.Value != "" {
			queryParams = append(queryParams, sqlFilterBuilder.AddFilter("error", filters.Error))
		}
	}

	// Build the ORDER BY clause for the query based on the sortBy
	if sortBy != nil {
		if sortBy.Path != "" {
			sqlFilterBuilder.AddSortBy("path", sortBy.Path)
		}
		if sortBy.CreatedAt != "" {
			sqlFilterBuilder.AddSortBy("created_at", sortBy.CreatedAt)
		}
		if sortBy.Error != "" {
			sqlFilterBuilder.AddSortBy("error", sortBy.Error)
		}
	} else {
		sqlFilterBuilder.AddSortBy("created_at", utils.SortByDirectionDesc)
	}

	filter := sqlFilterBuilder.Build()

	// Get the total count of items in the failed_queue table
	var totalCount int
	err := q.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM corrupted_nzbs %s", filter), queryParams...).Scan(&totalCount)
	if err != nil {
		return Result{}, err
	}

	queryParams = append(queryParams, limit, offset)

	rows, err := q.db.QueryContext(
		ctx,
		fmt.Sprintf("SELECT id, path, created_at, error FROM corrupted_nzbs %s LIMIT ? OFFSET ?", filter),
		queryParams...,
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
			Path:      usenet.ReplaceFileExtension(path, ".nzb"),
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

func (q *corruptedNzbsManager) GetFileContent(ctx context.Context, id int) (io.ReadCloser, error) {
	var path string
	err := q.db.QueryRowContext(ctx, "SELECT path FROM corrupted_nzbs WHERE id = ?", id).Scan(&path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return file, nil
}
