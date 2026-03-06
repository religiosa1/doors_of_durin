// Package repository provides abstraction over DB access
package repository

import (
	"embed"
	"errors"

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

func New(dbFileName string) (*DB, error) {
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
	err = db.open()
	if err != nil {
		return nil, err
	}
	return &db, nil
}

//go:embed migrations
var migrationsFS embed.FS

// open and migrate if necessary the db
func (d DB) open() error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.Up(d.DB.DB, "migrations")
}

func (d *DB) Close() (err error) {
	if d.DB != nil {
		err = d.DB.Close()
	}
	d.DB = nil
	return err
}
