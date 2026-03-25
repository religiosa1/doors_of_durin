// Package repository provides abstraction over DB access
package repository

import (
	"context"
	"embed"
	"errors"
	"io"
	"io/fs"
	"log/slog"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

var (
	ErrRecordNotFound   = errors.New("record not found")
	ErrUniqueConstraint = errors.New("unique constraint violation")
)

const (
	MaxPageSize     uint = 200
	DefaultPageSize uint = 20
)

type DB struct {
	DB *sqlx.DB
}

func New(dbFileName string, logger *slog.Logger) (*DB, error) {
	if dbFileName == "" {
		return nil, nil
	}
	db := DB{}
	var dsn string
	if dbFileName == ":memory:" {
		// In-memory SQLite requires a single connection: each connection gets its
		// own separate database instance, so the pool must not open more than one.
		dsn = "file::memory:?mode=memory&_foreign_keys=1"
	} else {
		dsn = dbFileName + "?_journal_mode=WAL&_foreign_keys=1&_busy_timeout=5000&_cache_size=2000&_synchronous=NORMAL"
	}
	d, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	if dbFileName == ":memory:" {
		d.SetMaxOpenConns(1)
	}
	db.DB = d
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	err = db.open(logger)
	if err != nil {
		return nil, err
	}
	return &db, nil
}

//go:embed migrations
var migrationsFS embed.FS

// open and migrate if necessary the db
func (d DB) open(logger *slog.Logger) error {
	migrationsSubFS, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		d.DB.DB,
		migrationsSubFS,
		goose.WithSlog(logger),
	)
	if err != nil {
		return err
	}
	_, err = provider.Up(context.Background())
	return err
}

func (d *DB) Close() (err error) {
	if d.DB != nil {
		err = d.DB.Close()
	}
	d.DB = nil
	return err
}
