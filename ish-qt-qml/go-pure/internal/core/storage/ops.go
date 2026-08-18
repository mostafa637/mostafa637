package storage

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	pathGetInodeSQL = `SELECT inode FROM paths WHERE path = ?`
	pathReadStatSQL = `SELECT inode, stat FROM stats NATURAL JOIN paths WHERE path = ?`
)

func (s *DB) PathGetInode(ctx context.Context, path string) (uint64, error) {
	return pathGetInode(ctx, s.db, path)
}

func pathGetInode(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, path string) (uint64, error) {
	var inode uint64
	err := q.QueryRowContext(ctx, pathGetInodeSQL, []byte(path)).Scan(&inode)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("storage: get inode for %q: %w", path, err)
	}
	return inode, nil
}

func (s *DB) PathReadStat(ctx context.Context, path string) (stat Stat, inode uint64, exists bool, err error) {
	var blob []byte
	err = s.db.QueryRowContext(ctx, pathReadStatSQL, []byte(path)).Scan(&inode, &blob)
	if err == sql.ErrNoRows {
		return Stat{}, 0, false, nil
	}
	if err != nil {
		return Stat{}, 0, false, fmt.Errorf("storage: read stat for %q: %w", path, err)
	}
	stat, err = decodeStat(blob)
	if err != nil {
		return Stat{}, 0, false, fmt.Errorf("storage: read stat for %q: %w", path, err)
	}
	return stat, inode, true, nil
}

func (s *DB) PathCreate(ctx context.Context, path string, stat Stat) (uint64, error) {
	blob, err := stat.MarshalBinary()
	if err != nil {
		return 0, err
	}
	var inode int64
	err = s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, "INSERT INTO stats (stat) VALUES (?)", blob)
		if err != nil {
			return fmt.Errorf("storage: create stat for %q: %w", path, err)
		}
		inode, err = result.LastInsertId()
		if err != nil {
			return fmt.Errorf("storage: read created inode for %q: %w", path, err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT OR REPLACE INTO paths (path, inode) VALUES (?, ?)", []byte(path), inode); err != nil {
			return fmt.Errorf("storage: create path %q: %w", path, err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return uint64(inode), nil
}

func (s *DB) InodeReadStat(ctx context.Context, inode uint64) (Stat, error) {
	var blob []byte
	err := s.db.QueryRowContext(ctx, "SELECT stat FROM stats WHERE inode = ?", inode).Scan(&blob)
	if err == sql.ErrNoRows {
		return Stat{}, ErrNotFound
	}
	if err != nil {
		return Stat{}, fmt.Errorf("storage: read inode %d: %w", inode, err)
	}
	stat, err := decodeStat(blob)
	if err != nil {
		return Stat{}, fmt.Errorf("storage: read inode %d: %w", inode, err)
	}
	return stat, nil
}

func (s *DB) InodeWriteStat(ctx context.Context, inode uint64, stat Stat) error {
	blob, err := stat.MarshalBinary()
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, "UPDATE stats SET stat = ? WHERE inode = ?", blob, inode)
	if err != nil {
		return fmt.Errorf("storage: write inode %d: %w", inode, err)
	}
	if n, err := result.RowsAffected(); err == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *DB) PathLink(ctx context.Context, src, dst string) error {
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		inode, err := pathGetInode(ctx, tx, src)
		if err != nil {
			return err
		}
		if inode == 0 {
			return fmt.Errorf("storage: link %q -> %q: %w", src, dst, ErrNotFound)
		}
		if _, err := tx.ExecContext(ctx, "INSERT OR REPLACE INTO paths (path, inode) VALUES (?, ?)", []byte(dst), inode); err != nil {
			return fmt.Errorf("storage: link %q -> %q: %w", src, dst, err)
		}
		return nil
	})
	return err
}

func (s *DB) PathUnlink(ctx context.Context, path string) (uint64, error) {
	var inode uint64
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		var err error
		inode, err = pathGetInode(ctx, tx, path)
		if err != nil {
			return err
		}
		if inode == 0 {
			return fmt.Errorf("storage: unlink %q: %w", path, ErrNotFound)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM paths WHERE path = ?", []byte(path)); err != nil {
			return fmt.Errorf("storage: unlink %q: %w", path, err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return inode, nil
}

func (s *DB) PathRename(ctx context.Context, src, dst string) error {
	srcBytes := []byte(src)
	srcSlash := append(append([]byte(nil), srcBytes...), '/')
	srcZero := append(append([]byte(nil), srcBytes...), '0')
	_, err := s.db.ExecContext(ctx, `
		UPDATE OR REPLACE paths
		SET path = change_prefix(path, ?, ?)
		WHERE (path >= ? AND path < ?) OR path = ?
	`, len(srcBytes), []byte(dst), srcSlash, srcZero, srcBytes)
	if err != nil {
		return fmt.Errorf("storage: rename %q -> %q: %w", src, dst, err)
	}
	return nil
}

func (s *DB) PathsFromInode(ctx context.Context, inode uint64) ([][]byte, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT path FROM paths WHERE inode = ? ORDER BY path", inode)
	if err != nil {
		return nil, fmt.Errorf("storage: paths for inode %d: %w", inode, err)
	}
	defer rows.Close()

	var paths [][]byte
	for rows.Next() {
		var path []byte
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("storage: scan path for inode %d: %w", inode, err)
		}
		paths = append(paths, append([]byte(nil), path...))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: iterate paths for inode %d: %w", inode, err)
	}
	return paths, nil
}

func (s *DB) TryCleanupInode(ctx context.Context, inode uint64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM stats
		WHERE inode = ? AND NOT EXISTS (SELECT 1 FROM paths WHERE inode = stats.inode)
	`, inode)
	if err != nil {
		return fmt.Errorf("storage: cleanup inode %d: %w", inode, err)
	}
	return nil
}
