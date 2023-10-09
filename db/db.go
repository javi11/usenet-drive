package db

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func NewDB(dbPath string) (*sql.DB, error) {
	sqlLite, err := goose.OpenDBWithDriver("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	// setup database

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, err
	}

	if err := goose.Up(sqlLite, "migrations"); err != nil {
		return nil, err
	}

	return sqlLite, nil
}
