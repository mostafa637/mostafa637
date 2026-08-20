package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"

	"modernc.org/sqlite"
)

var (
	ErrNotFound  = errors.New("storage: not found")
	ErrExists    = errors.New("storage: already exists")
	ErrNoData    = errors.New("storage: no data")
	ErrInvariant = errors.New("storage: invariant violation")
)

var registerChangePrefixOnce struct {
	sync.Once
	err error
}

// DB owns the iSH fakefs metadata database. A single database/sql connection is
// used deliberately: it mirrors the serialized sqlite mutex used by the C
// implementation and keeps PRAGMA state stable across operations.
type DB struct {
	db *sql.DB
}

func registerChangePrefix() error {
	registerChangePrefixOnce.Do(func() {
		registerChangePrefixOnce.err = sqlite.RegisterScalarFunction(
			"change_prefix",
			3,
			func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
				return changePrefix(args)
			},
		)
	})
	return registerChangePrefixOnce.err
}

func Open(ctx context.Context, path string) (*DB, error) {
	if err := registerChangePrefix(); err != nil {
		return nil, fmt.Errorf("storage: register change_prefix: %w", err)
	}
	if path == "" {
		path = ":memory:"
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("storage: open %q: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &DB{db: db}
	if err := store.configure(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *DB) configure(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("storage: ping: %w", err)
	}
	for _, pragma := range []string{
		"PRAGMA busy_timeout = 1000",
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
	} {
		if _, err := s.db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("storage: %s: %w", pragma, err)
		}
	}
	if err := s.ensureSchema(ctx); err != nil {
		return err
	}
	if err := s.migrate(ctx); err != nil {
		return err
	}
	return nil
}

func (s *DB) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *DB) SQLDB() *sql.DB {
	return s.db
}

func (s *DB) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
