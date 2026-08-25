package rootfs

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE meta (id INTEGER UNIQUE DEFAULT 0, db_inode INTEGER);
INSERT INTO meta (db_inode) VALUES (0);
CREATE TABLE stats (inode INTEGER PRIMARY KEY, stat BLOB);
CREATE TABLE paths (path BLOB PRIMARY KEY, inode INTEGER REFERENCES stats(inode));
CREATE INDEX inode_to_path ON paths (inode, path);
PRAGMA user_version=3;
`

// IshStat is the 16-byte metadata blob consumed by the iSH fakefs core.
type IshStat struct {
	Mode uint32
	UID  uint32
	GID  uint32
	Rdev uint32
}

func (s IshStat) blob() []byte {
	b := make([]byte, 16)
	binary.LittleEndian.PutUint32(b[0:4], s.Mode)
	binary.LittleEndian.PutUint32(b[4:8], s.UID)
	binary.LittleEndian.PutUint32(b[8:12], s.GID)
	binary.LittleEndian.PutUint32(b[12:16], s.Rdev)
	return b
}

type hardlink struct {
	path   string
	target string
}

// Install imports a bundled iSH root.tar.gz into base. The resulting layout is
// base/data plus base/meta.db, matching fakefs_mount's expected mount source.
// The import is staged first and never removes base itself, which is important
// on Android where base is the application-private data directory.
func Install(ctx context.Context, archive io.Reader, base string) error {
	if archive == nil {
		return errors.New("rootfs: nil archive")
	}
	if base == "" {
		return errors.New("rootfs: empty base path")
	}
	if err := mkdirAllNoStat(base, 0o755); err != nil {
		return fmt.Errorf("rootfs: create base: %w", err)
	}

	stage := filepath.Join(base, ".rootfs-import")
	if err := removeAllNoStat(stage); err != nil {
		return fmt.Errorf("rootfs: clear stage: %w", err)
	}
	if err := mkdirAllNoStat(filepath.Join(stage, "data"), 0o755); err != nil {
		return fmt.Errorf("rootfs: create stage data: %w", err)
	}
	defer func() { _ = removeAllNoStat(stage) }()

	dbPath := filepath.Join(stage, "meta.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("rootfs: open metadata database: %w", err)
	}
	defer db.Close()
	if err := initDB(ctx, db); err != nil {
		return err
	}

	gz, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("rootfs: open gzip stream: %w", err)
	}
	t := tar.NewReader(gz)
	hardlinks := make([]hardlink, 0)
	rootSeen := false
	for {
		h, nextErr := t.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return fmt.Errorf("rootfs: read tar header: %w", nextErr)
		}
		path, err := normalizePath(h.Name)
		if err != nil {
			return fmt.Errorf("rootfs: unsafe path %q: %w", h.Name, err)
		}
		if path == "" {
			rootSeen = true
		}
		if h.Typeflag == tar.TypeLink {
			target, err := normalizePath(h.Linkname)
			if err != nil {
				return fmt.Errorf("rootfs: unsafe hardlink target %q: %w", h.Linkname, err)
			}
			hardlinks = append(hardlinks, hardlink{path: path, target: target})
			continue
		}

		out := dataPath(stage, path)
		mode := uint32(h.Mode)
		if h.Typeflag == tar.TypeDir {
			mode |= 0o40000
		} else if h.Typeflag == tar.TypeSymlink {
			mode |= 0o120000
		} else {
			mode |= 0o100000
		}
		if err := materializeParents(out); err != nil {
			return err
		}

		switch h.Typeflag {
		case tar.TypeDir:
			if err := mkdirAllNoStat(out, 0o755); err != nil {
				return fmt.Errorf("rootfs: mkdir %q: %w", path, err)
			}
		case tar.TypeReg, tar.TypeRegA, tar.TypeSymlink, tar.TypeBlock, tar.TypeChar, tar.TypeFifo:
			// fakefs stores symlink and special-file payloads as ordinary host
			// files and uses the metadata blob to expose their guest type.
			f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				return fmt.Errorf("rootfs: create %q: %w", path, err)
			}
			if h.Typeflag == tar.TypeReg || h.Typeflag == tar.TypeRegA {
				if _, err := io.Copy(f, t); err != nil {
					f.Close()
					return fmt.Errorf("rootfs: copy %q: %w", path, err)
				}
			} else if h.Typeflag == tar.TypeSymlink {
				if _, err := io.WriteString(f, h.Linkname); err != nil {
					f.Close()
					return fmt.Errorf("rootfs: write symlink %q: %w", path, err)
				}
			}
			if err := f.Close(); err != nil {
				return fmt.Errorf("rootfs: close %q: %w", path, err)
			}
		default:
			return fmt.Errorf("rootfs: unsupported tar type %d for %q", h.Typeflag, path)
		}
		if err := insertPath(ctx, db, path, IshStat{Mode: mode, UID: uint32(h.Uid), GID: uint32(h.Gid), Rdev: uint32(h.Devmajor)}); err != nil {
			return err
		}
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("rootfs: close gzip stream: %w", err)
	}

	for _, link := range hardlinks {
		if err := materializeParents(dataPath(stage, link.path)); err != nil {
			return err
		}
		if err := os.Link(dataPath(stage, link.target), dataPath(stage, link.path)); err != nil {
			return fmt.Errorf("rootfs: hardlink %q -> %q: %w", link.path, link.target, err)
		}
		if err := insertHardlink(ctx, db, link.path, link.target); err != nil {
			return err
		}
	}
	if !rootSeen {
		if err := insertPath(ctx, db, "", IshStat{Mode: 0o755}); err != nil {
			return err
		}
	}
	if err := finalizeDB(ctx, db); err != nil {
		return err
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("rootfs: close metadata database: %w", err)
	}

	data := filepath.Join(base, "data")
	meta := filepath.Join(base, "meta.db")
	if err := removeAllNoStat(data); err != nil {
		return fmt.Errorf("rootfs: replace data: %w", err)
	}
	if err := os.Remove(meta); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("rootfs: replace metadata: %w", err)
	}
	if err := os.Rename(filepath.Join(stage, "data"), data); err != nil {
		return fmt.Errorf("rootfs: install data: %w", err)
	}
	if err := os.Rename(filepath.Join(stage, "meta.db"), meta); err != nil {
		return fmt.Errorf("rootfs: install metadata: %w", err)
	}
	return Validate(ctx, base)
}

func initDB(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL;"); err != nil {
		return fmt.Errorf("rootfs: enable WAL: %w", err)
	}
	if _, err := db.ExecContext(ctx, "BEGIN;"); err != nil {
		return fmt.Errorf("rootfs: begin metadata transaction: %w", err)
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("rootfs: create metadata schema: %w", err)
	}
	return nil
}

func finalizeDB(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "COMMIT;"); err != nil {
		return fmt.Errorf("rootfs: commit metadata: %w", err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE);"); err != nil {
		return fmt.Errorf("rootfs: checkpoint metadata: %w", err)
	}
	return nil
}

func insertPath(ctx context.Context, db *sql.DB, path string, stat IshStat) error {
	result, err := db.ExecContext(ctx, "INSERT INTO stats (stat) VALUES (?)", stat.blob())
	if err != nil {
		return fmt.Errorf("rootfs: insert stat for %q: %w", path, err)
	}
	inode, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("rootfs: read inode for %q: %w", path, err)
	}
	if _, err := db.ExecContext(ctx, "INSERT OR REPLACE INTO paths (path, inode) VALUES (?, ?)", []byte(path), inode); err != nil {
		return fmt.Errorf("rootfs: insert path %q: %w", path, err)
	}
	return nil
}

func insertHardlink(ctx context.Context, db *sql.DB, path, target string) error {
	if _, err := db.ExecContext(ctx, "INSERT OR REPLACE INTO paths (path, inode) VALUES (?, (SELECT inode FROM paths WHERE path = ? LIMIT 1))", []byte(path), []byte(target)); err != nil {
		return fmt.Errorf("rootfs: insert hardlink %q -> %q: %w", path, target, err)
	}
	return nil
}

func materializeParents(path string) error {
	if err := mkdirAllNoStat(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("rootfs: create parent %q: %w", filepath.Dir(path), err)
	}
	return nil
}

// mkdirAllNoStat creates a path without os.MkdirAll. The latter performs
// path-based Stat calls while walking existing components; on Android x86_64
// those become the legacy lstat syscall rejected by the platform seccomp
// policy. Existing components are checked through an open descriptor instead.
func mkdirAllNoStat(path string, perm os.FileMode) error {
	clean := filepath.Clean(path)
	if clean == "." || clean == string(filepath.Separator) {
		return nil
	}
	current := string(filepath.Separator)
	for _, part := range strings.Split(strings.TrimPrefix(clean, current), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		if err := os.Mkdir(current, perm); err == nil {
			continue
		} else if !errors.Is(err, os.ErrExist) {
			return err
		}
		f, err := os.Open(current)
		if err != nil {
			return err
		}
		info, statErr := f.Stat()
		closeErr := f.Close()
		if statErr != nil {
			return statErr
		}
		if closeErr != nil {
			return closeErr
		}
		if !info.IsDir() {
			return fmt.Errorf("%s exists and is not a directory", current)
		}
	}
	return nil
}

// removeAllNoStat removes the importer staging tree without path-based
// lstat. Staging entries are created by this importer, so descriptor-based
// directory traversal is sufficient and keeps Android's seccomp policy happy.
func removeAllNoStat(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	info, statErr := f.Stat()
	if statErr != nil {
		_ = f.Close()
		return statErr
	}
	if !info.IsDir() {
		closeErr := f.Close()
		if closeErr != nil {
			return closeErr
		}
		return os.Remove(path)
	}
	names, readErr := f.Readdirnames(-1)
	closeErr := f.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	for _, name := range names {
		if err := removeAllNoStat(filepath.Join(path, name)); err != nil {
			return err
		}
	}
	return os.Remove(path)
}

func dataPath(stage, path string) string {
	return filepath.Join(stage, "data", strings.TrimPrefix(path, "/"))
}

func normalizePath(raw string) (string, error) {
	p := strings.ReplaceAll(raw, "\\", "/")
	for strings.HasPrefix(p, "./") {
		p = strings.TrimPrefix(p, "./")
	}
	p = strings.TrimLeft(p, "/")
	parts := strings.Split(strings.TrimRight(p, "/"), "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return "", errors.New("path traversal")
		}
		clean = append(clean, part)
	}
	if len(clean) == 0 {
		return "", nil
	}
	return "/" + strings.Join(clean, "/"), nil
}

// Validate checks the same structural invariants used by the Qt importer.
func Validate(ctx context.Context, base string) error {
	// Do not use os.Stat(path) here: Android x86_64's seccomp policy rejects
	// the legacy lstat syscall that Go uses for path-based Stat. Opening the
	// directory and calling File.Stat uses fstat on an already-open descriptor.
	dataDir, err := os.Open(filepath.Join(base, "data"))
	if err != nil {
		return fmt.Errorf("rootfs: data directory is missing")
	}
	info, statErr := dataDir.Stat()
	closeErr := dataDir.Close()
	if statErr != nil || closeErr != nil || !info.IsDir() {
		return fmt.Errorf("rootfs: data directory is missing")
	}
	db, err := sql.Open("sqlite", filepath.Join(base, "meta.db"))
	if err != nil {
		return fmt.Errorf("rootfs: open installed database: %w", err)
	}
	defer db.Close()
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil || version != 3 {
		return fmt.Errorf("rootfs: invalid metadata version %d: %v", version, err)
	}
	var integrity string
	if err := db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		return fmt.Errorf("rootfs: metadata integrity check failed: %v", err)
	}
	for _, path := range []string{"", "/bin/sh"} {
		var n, size int
		if err := db.QueryRowContext(ctx, "SELECT p.inode, length(s.stat) FROM paths p JOIN stats s ON s.inode=p.inode WHERE p.path=? LIMIT 1", []byte(path)).Scan(&n, &size); err != nil || n <= 0 || size != 16 {
			return fmt.Errorf("rootfs: required path %q is missing or invalid", path)
		}
	}
	return nil
}
