package storage

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	XattrCreate  uint32 = 1
	XattrReplace uint32 = 2
)

// SetXattr stores an extended attribute on an inode. The attribute follows the
// inode across hard links and path renames, matching Linux inode semantics.
func (s *DB) SetXattr(ctx context.Context, inode uint64, name string, value []byte, flags uint32) error {
	if s == nil || s.db == nil || inode == 0 || name == "" {
		return ErrInvariant
	}
	if flags != 0 && flags != XattrCreate && flags != XattrReplace {
		return ErrInvariant
	}
	return s.withTx(ctx, func(tx *sql.Tx) error {
		var exists int
		err := tx.QueryRowContext(ctx, "SELECT 1 FROM xattrs WHERE inode = ? AND name = ?", inode, []byte(name)).Scan(&exists)
		switch {
		case err == sql.ErrNoRows:
			exists = 0
		case err != nil:
			return fmt.Errorf("storage: inspect xattr %q on inode %d: %w", name, inode, err)
		}
		if exists != 0 && flags == XattrCreate {
			return ErrExists
		}
		if exists == 0 && flags == XattrReplace {
			return ErrNoData
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO xattrs (inode, name, value) VALUES (?, ?, ?)
			ON CONFLICT(inode, name) DO UPDATE SET value = excluded.value
		`, inode, []byte(name), append([]byte(nil), value...)); err != nil {
			return fmt.Errorf("storage: set xattr %q on inode %d: %w", name, inode, err)
		}
		return nil
	})
}

// GetXattr returns a copy so callers cannot mutate storage-owned data.
func (s *DB) GetXattr(ctx context.Context, inode uint64, name string) ([]byte, error) {
	if s == nil || s.db == nil || inode == 0 || name == "" {
		return nil, ErrInvariant
	}
	var value []byte
	if err := s.db.QueryRowContext(ctx, "SELECT value FROM xattrs WHERE inode = ? AND name = ?", inode, []byte(name)).Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNoData
		}
		return nil, fmt.Errorf("storage: get xattr %q on inode %d: %w", name, inode, err)
	}
	return append([]byte(nil), value...), nil
}

// ListXattrs returns the Linux listxattr wire format: NUL-terminated names.
func (s *DB) ListXattrs(ctx context.Context, inode uint64) ([]byte, error) {
	if s == nil || s.db == nil || inode == 0 {
		return nil, ErrInvariant
	}
	rows, err := s.db.QueryContext(ctx, "SELECT name FROM xattrs WHERE inode = ? ORDER BY name", inode)
	if err != nil {
		return nil, fmt.Errorf("storage: list xattrs on inode %d: %w", inode, err)
	}
	defer rows.Close()
	var result []byte
	for rows.Next() {
		var name []byte
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("storage: scan xattr on inode %d: %w", inode, err)
		}
		result = append(result, name...)
		result = append(result, 0)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: iterate xattrs on inode %d: %w", inode, err)
	}
	return result, nil
}

func (s *DB) RemoveXattr(ctx context.Context, inode uint64, name string) error {
	if s == nil || s.db == nil || inode == 0 || name == "" {
		return ErrInvariant
	}
	result, err := s.db.ExecContext(ctx, "DELETE FROM xattrs WHERE inode = ? AND name = ?", inode, []byte(name))
	if err != nil {
		return fmt.Errorf("storage: remove xattr %q on inode %d: %w", name, inode, err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return ErrNoData
	}
	return nil
}
