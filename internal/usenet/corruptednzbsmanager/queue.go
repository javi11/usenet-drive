package corruptednzbsmanager

//go:generate mockgen -source=./queue.go -destination=./queue_mock.go -package=corruptednzbsmanager CorruptedNzbsManager

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/utils"
	"github.com/javi11/usenet-drive/pkg/osfs"
)

type Segment struct {
	Number    int       `json:"number"`
	ID        string    `json:"id"`
	UUID      string    `json:"uuid"`
	CreatedAt time.Time `json:"created_at"`
}

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
	Add(ctx context.Context, path string, err error) error
	Delete(ctx context.Context, id int) error
	Discard(ctx context.Context, id int) (*cNzb, error)
	DiscardByPath(ctx context.Context, path string) (*cNzb, error)
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
	Segments  []Segment `json:"segments"`
}

type corruptedNzbsManager struct {
	db *sql.DB
	fs osfs.FileSystem
	mx *sync.Mutex
}

func New(db *sql.DB, fs osfs.FileSystem) CorruptedNzbsManager {
	return &corruptedNzbsManager{db: db, fs: fs, mx: &sync.Mutex{}}
}

func (q *corruptedNzbsManager) Add(ctx context.Context, path string, err error) error {
	q.mx.Lock()
	defer q.mx.Unlock()

	return utils.Transact(q.db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, "SELECT id FROM corrupted_nzbs WHERE path = ? LIMIT 1")
		if err != nil {
			return err
		}
		defer stmt.Close()

		var nzbId int64
		err = stmt.QueryRowContext(ctx, path).Scan(&nzbId)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		if nzbId != 0 {
			stmt, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO corrupted_nzbs (path, error) VALUES (?, ?)")
			if err != nil {
				return err
			}
			defer stmt.Close()

			result, err := stmt.ExecContext(ctx, usenet.ReplaceFileExtension(path, ".nzb"), err.Error())
			if err != nil {
				return err
			}

			nzbId, err = result.LastInsertId()
			if err != nil {
				return err
			}
		}

		if e, ok := err.(*ErrCorruptedNzb); ok {
			segment := e.Segment
			if segment == nil {
				stmt, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO corrupted_nzbs_segments (number, uuid, nzbId) VALUES (?, ?, ?)")
				if err != nil {
					return err
				}
				defer stmt.Close()

				_, err = stmt.ExecContext(ctx, segment.Number, segment.Id, nzbId)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (q *corruptedNzbsManager) Delete(ctx context.Context, id int) error {
	cnzb, err := q.Discard(ctx, id)
	if err != nil {
		return err
	}

	if _, err := q.fs.Stat(cnzb.Path); err == nil {
		return q.fs.Remove(cnzb.Path)
	}

	return nil
}

func (q *corruptedNzbsManager) DiscardByPath(ctx context.Context, path string) (*cNzb, error) {
	q.mx.Lock()
	defer q.mx.Unlock()

	var j cNzb
	err := utils.Transact(q.db, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, "SELECT id, path, created_at FROM corrupted_nzbs WHERE id = ?", path)

		err := row.Scan(&j.ID, &j.Path, &j.CreatedAt)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, "DELETE FROM corrupted_nzbs WHERE id = ?", path)
		if err != nil {
			return err
		}

		return nil
	})

	return &j, err
}

func (q *corruptedNzbsManager) Discard(ctx context.Context, id int) (*cNzb, error) {
	q.mx.Lock()
	defer q.mx.Unlock()
	var j cNzb

	err := utils.Transact(q.db, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, "SELECT id, path, created_at FROM corrupted_nzbs WHERE id = ?", id)
		err := row.Scan(&j.ID, &j.Path, &j.CreatedAt)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, "DELETE FROM corrupted_nzbs WHERE id = ?", id)
		if err != nil {
			return err
		}

		j.Path = usenet.ReplaceFileExtension(j.Path, ".nzb")

		return nil
	})

	return &j, err
}

func (q *corruptedNzbsManager) Update(ctx context.Context, oldPath, newPath string) error {
	q.mx.Lock()
	defer q.mx.Unlock()

	return utils.Transact(q.db, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, "SELECT id, path, created_at FROM corrupted_nzbs WHERE path = ?", oldPath)
		var j cNzb
		err := row.Scan(&j.ID, &j.Path, &j.CreatedAt)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, "UPDATE corrupted_nzbs SET path = ? WHERE id = ?", newPath, j.ID)
		if err != nil {
			return err
		}

		return nil
	})
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
		fmt.Sprintf("SELECT id, path, created_at, error FROM corrupted_nzbs JOIN corrupted_nzbs_segments ON corrupted_nzbs.id = corrupted_nzbs_segments.nzbId %s LIMIT ? OFFSET ?", filter),
		queryParams...,
	)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()

	var nzbs []cNzb = make([]cNzb, 0)
	for rows.Next() {
		var id int
		var path string
		var createdAt time.Time
		var error string
		var segment Segment

		err := rows.Scan(&id, &path, &createdAt, &error, &segment.Number, &segment.ID, &segment.UUID, &segment.CreatedAt)
		if err != nil {
			return Result{}, err
		}

		var nzb cNzb
		for _, n := range nzbs {
			if n.ID == int64(id) {
				nzb = n
			}
		}

		if nzb.ID == 0 {
			nzb = cNzb{
				ID:        int64(id),
				Path:      path,
				CreatedAt: createdAt,
				Error:     error,
			}

			nzbs = append(nzbs, nzb)
		}

		nzb.Segments = append(nzb.Segments, segment)
	}

	return Result{
		Entries:    nzbs,
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

	file, err := q.fs.Open(usenet.ReplaceFileExtension(path, ".nzb"))
	if err != nil {
		return nil, err
	}

	return file, nil
}
