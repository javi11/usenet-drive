package corruptednzbsmanager

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"github.com/stretchr/testify/assert"
)

func TestCorruptedNzbsManager_Add(t *testing.T) {
	ctrl := gomock.NewController(t)
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	fs := osfs.NewMockFileSystem(ctrl)

	manager := New(db, fs)

	ctx := context.Background()

	// Test Add
	mock.ExpectPrepare("INSERT OR IGNORE INTO corrupted_nzbs").
		ExpectExec().
		WithArgs("test.nzb", "error").
		WillReturnResult(sqlmock.NewResult(1, 1))
	err = manager.Add(ctx, "test", "error")
	assert.NoError(t, err)
}

func TestCorruptedNzbsManager_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	fs := osfs.NewMockFileSystem(ctrl)
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	manager := New(db, fs)

	ctx := context.Background()

	// Test Delete
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, path, created_at FROM corrupted_nzbs WHERE id = ?").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path", "created_at"}).
			AddRow(1, "test.nzb", time.Now()))
	mock.ExpectExec("DELETE FROM corrupted_nzbs WHERE id = ?").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	fs.EXPECT().Stat("test.nzb").Return(osfs.NewMockFileInfo(ctrl), nil)
	fs.EXPECT().Remove("test.nzb").Return(nil)

	err = manager.Delete(ctx, 1)
	assert.NoError(t, err)
}

func TestCorruptedNzbsManager_DiscardByPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	fs := osfs.NewMockFileSystem(ctrl)
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	manager := New(db, fs)

	ctx := context.Background()

	// Test DiscardByPath
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, path, created_at FROM corrupted_nzbs WHERE id = ?").
		WithArgs("test").
		WillReturnRows(sqlmock.NewRows([]string{"id", "path", "created_at"}).
			AddRow(1, "test.nzb", time.Now()))
	mock.ExpectExec("DELETE FROM corrupted_nzbs WHERE id = ?").
		WithArgs("test").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	cnzb, err := manager.DiscardByPath(ctx, "test")
	assert.NoError(t, err)
	assert.NotNil(t, cnzb)
}

func TestCorruptedNzbsManager_Discard(t *testing.T) {
	ctrl := gomock.NewController(t)
	fs := osfs.NewMockFileSystem(ctrl)
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	manager := New(db, fs)

	ctx := context.Background()

	// Test Discard
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, path, created_at FROM corrupted_nzbs WHERE id = ?").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path", "created_at"}).
			AddRow(1, "test.nzb", time.Now()))
	mock.ExpectExec("DELETE FROM corrupted_nzbs WHERE id = \\?").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	cnzb, err := manager.Discard(ctx, 1)
	assert.NoError(t, err)
	assert.NotNil(t, cnzb)
}

func TestCorruptedNzbsManager_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	fs := osfs.NewMockFileSystem(ctrl)
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	manager := New(db, fs)

	ctx := context.Background()

	// Test Update
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, path, created_at FROM corrupted_nzbs WHERE path = \\?").
		WithArgs("old.nzb").
		WillReturnRows(sqlmock.NewRows([]string{"id", "path", "created_at"}).
			AddRow(1, "old.nzb", time.Now()))
	mock.ExpectExec("UPDATE corrupted_nzbs SET path = \\? WHERE id = \\?").
		WithArgs("new.nzb", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	err = manager.Update(ctx, "old.nzb", "new.nzb")
	assert.NoError(t, err)
}

func TestCorruptedNzbsManager_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	fs := osfs.NewMockFileSystem(ctrl)
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	manager := New(db, fs)

	ctx := context.Background()

	// Test List
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM corrupted_nzbs").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
	mock.ExpectQuery("SELECT id, path, created_at, error FROM corrupted_nzbs").
		WillReturnRows(sqlmock.NewRows([]string{"id", "path", "created_at", "error"}).
			AddRow(1, "test.nzb", time.Now(), "error"))
	result, err := manager.List(ctx, 10, 0, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, 1, len(result.Entries))
}

func TestCorruptedNzbsManager_GetFileContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	fs := osfs.NewMockFileSystem(ctrl)
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	defer db.Close()

	manager := New(db, fs)
	assert.NoError(t, err)

	ctx := context.Background()

	// Test GetFileContent
	mock.ExpectQuery("SELECT path FROM corrupted_nzbs WHERE id = ?").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"path"}).AddRow("test.nzb"))

	expectedContent := []byte("test")
	f := osfs.NewMockFile(ctrl)
	f.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (int, error) {
		copy(p, expectedContent)
		return len(expectedContent), io.EOF
	})
	f.EXPECT().Close().Return(nil)
	fs.EXPECT().Open("test.nzb").Return(f, nil)

	content, err := manager.GetFileContent(ctx, 1)
	assert.NoError(t, err)
	defer content.Close()
	assert.NoError(t, err)
	actualContent, err := io.ReadAll(content)
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, actualContent)
	assert.NoError(t, mock.ExpectationsWereMet())
}
