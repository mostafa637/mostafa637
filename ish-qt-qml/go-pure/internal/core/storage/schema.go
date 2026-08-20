package storage

import (
	"context"
	"database/sql"
	"fmt"
)

const schemaVersion = 4

const baseSchema = `
CREATE TABLE meta (id INTEGER UNIQUE DEFAULT 0, db_inode INTEGER);
INSERT INTO meta (db_inode) VALUES (0);
CREATE TABLE stats (inode INTEGER PRIMARY KEY, stat BLOB);
CREATE TABLE paths (path BLOB PRIMARY KEY, inode INTEGER REFERENCES stats(inode));
CREATE INDEX inode_to_path ON paths (inode, path);
CREATE TABLE xattrs (inode INTEGER REFERENCES stats(inode) ON DELETE CASCADE, name BLOB, value BLOB NOT NULL, PRIMARY KEY(inode, name));
PRAGMA user_version = 4;
`

func (s *DB) ensureSchema(ctx context.Context) error {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'meta'`).Scan(&exists)
	switch {
	case err == sql.ErrNoRows:
		if _, err := s.db.ExecContext(ctx, baseSchema); err != nil {
			return fmt.Errorf("storage: create base schema: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("storage: inspect schema: %w", err)
	default:
		return nil
	}
}
