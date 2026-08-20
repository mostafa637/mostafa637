package storage

import (
	"context"
	"database/sql"
	"fmt"
)

func (s *DB) migrate(ctx context.Context) error {
	var version int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("storage: read user_version: %w", err)
	}
	if version > schemaVersion {
		return fmt.Errorf("storage: unsupported user_version %d (latest supported %d)", version, schemaVersion)
	}
	if version == schemaVersion {
		return nil
	}

	err := s.withTx(ctx, func(tx *sql.Tx) error {
		for version < schemaVersion {
			switch version {
			case 0:
				if _, err := tx.ExecContext(ctx, "CREATE INDEX inode_to_path ON paths (inode, path)"); err != nil {
					return fmt.Errorf("storage: migrate v0->v1: %w", err)
				}
			case 1:
				statements := []string{
					"CREATE TABLE paths_new (path BLOB PRIMARY KEY, inode INTEGER REFERENCES stats(inode))",
					"INSERT INTO paths_new SELECT * FROM paths WHERE EXISTS (SELECT 1 FROM stats WHERE inode = paths.inode)",
					"DROP TABLE paths",
					"ALTER TABLE paths_new RENAME TO paths",
					"CREATE INDEX inode_to_path ON paths (inode, path)",
					"DELETE FROM stats WHERE NOT EXISTS (SELECT 1 FROM paths WHERE inode = stats.inode)",
					"CREATE TRIGGER delete_path AFTER DELETE ON paths WHEN NOT EXISTS (SELECT 1 FROM paths WHERE inode = old.inode) BEGIN DELETE FROM stats WHERE NOT EXISTS (SELECT 1 FROM paths WHERE inode = old.inode) AND inode = old.inode; END",
				}
				for _, statement := range statements {
					if _, err := tx.ExecContext(ctx, statement); err != nil {
						return fmt.Errorf("storage: migrate v1->v2 statement %q: %w", statement, err)
					}
				}
			case 2:
				if _, err := tx.ExecContext(ctx, "DROP TRIGGER delete_path"); err != nil {
					return fmt.Errorf("storage: migrate v2->v3: %w", err)
				}
			case 3:
				if _, err := tx.ExecContext(ctx, "CREATE TABLE xattrs (inode INTEGER REFERENCES stats(inode) ON DELETE CASCADE, name BLOB, value BLOB NOT NULL, PRIMARY KEY(inode, name))"); err != nil {
					return fmt.Errorf("storage: migrate v3->v4: %w", err)
				}
			default:
				return fmt.Errorf("storage: invalid migration version %d", version)
			}
			version++
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
			return fmt.Errorf("storage: set user_version: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
