package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestOpenCreatesCompatibleSchema(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var version int
	if err := db.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != schemaVersion {
		t.Fatalf("user_version = %d, want %d", version, schemaVersion)
	}

	var count int
	if err := db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM meta").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("meta rows = %d, want 1", count)
	}
}

func TestStorageCRUDAndRename(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	root := Stat{Mode: 0o40755, UID: 501, GID: 20, Rdev: 0}
	file := Stat{Mode: 0o100644, UID: 501, GID: 20, Rdev: 0}
	rootInode, err := db.PathCreate(ctx, "/dir", root)
	if err != nil {
		t.Fatal(err)
	}
	fileInode, err := db.PathCreate(ctx, "/dir/file", file)
	if err != nil {
		t.Fatal(err)
	}
	if rootInode == fileInode || rootInode == 0 || fileInode == 0 {
		t.Fatalf("unexpected inodes root=%d file=%d", rootInode, fileInode)
	}

	got, gotInode, exists, err := db.PathReadStat(ctx, "/dir/file")
	if err != nil {
		t.Fatal(err)
	}
	if !exists || gotInode != fileInode || !reflect.DeepEqual(got, file) {
		t.Fatalf("PathReadStat = %#v, inode=%d, exists=%v", got, gotInode, exists)
	}

	if err := db.PathLink(ctx, "/dir/file", "/dir/linked"); err != nil {
		t.Fatal(err)
	}
	paths, err := db.PathsFromInode(ctx, fileInode)
	if err != nil {
		t.Fatal(err)
	}
	wantPaths := [][]byte{[]byte("/dir/file"), []byte("/dir/linked")}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("paths = %q, want %q", paths, wantPaths)
	}

	if err := db.PathRename(ctx, "/dir", "/moved"); err != nil {
		t.Fatal(err)
	}
	if _, _, exists, err := db.PathReadStat(ctx, "/dir/file"); err != nil {
		t.Fatal(err)
	} else if exists {
		t.Fatal("old path still exists after rename")
	}
	if got, gotInode, exists, err := db.PathReadStat(ctx, "/moved/linked"); err != nil {
		t.Fatal(err)
	} else if !exists || gotInode != fileInode || !reflect.DeepEqual(got, file) {
		t.Fatalf("renamed linked stat = %#v, inode=%d, exists=%v", got, gotInode, exists)
	}

	if _, err := db.PathUnlink(ctx, "/moved/file"); err != nil {
		t.Fatal(err)
	}
	if err := db.TryCleanupInode(ctx, fileInode); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InodeReadStat(ctx, fileInode); err != nil {
		t.Fatalf("inode must remain while hard link exists: %v", err)
	}
	if _, err := db.PathUnlink(ctx, "/moved/linked"); err != nil {
		t.Fatal(err)
	}
	if err := db.TryCleanupInode(ctx, fileInode); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InodeReadStat(ctx, fileInode); !errors.Is(err, ErrNotFound) {
		t.Fatalf("inode read error = %v, want ErrNotFound", err)
	}
}

func TestChangePrefixSQLFunction(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var got []byte
	if err := db.db.QueryRowContext(ctx, "SELECT change_prefix(?, ?, ?)", []byte("/old/file"), 4, []byte("/new")).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if string(got) != "/new/file" {
		t.Fatalf("change_prefix = %q, want %q", got, "/new/file")
	}
}

func TestMigrateV0ToV3(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "meta.db")
	seed, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = seed.Exec(`
		CREATE TABLE meta (id INTEGER UNIQUE DEFAULT 0, db_inode INTEGER);
		INSERT INTO meta (db_inode) VALUES (0);
		CREATE TABLE stats (inode INTEGER PRIMARY KEY, stat BLOB);
		CREATE TABLE paths (path BLOB PRIMARY KEY, inode INTEGER);
		INSERT INTO stats (inode, stat) VALUES (1, zeroblob(16));
		INSERT INTO paths (path, inode) VALUES (x'2F', 1);
		PRAGMA user_version = 0;
	`)
	if err != nil {
		seed.Close()
		t.Fatal(err)
	}
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var version int
	if err := store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 3 {
		t.Fatalf("migrated version = %d, want 3", version)
	}
	var foreignKey string
	if err := store.db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='paths'").Scan(&foreignKey); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(foreignKey, "REFERENCES stats") {
		t.Fatalf("migrated paths table has no foreign key: %s", foreignKey)
	}
}

func TestStatEncoding(t *testing.T) {
	in := Stat{Mode: 1, UID: 2, GID: 3, Rdev: 4}
	blob, err := in.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var out Stat
	if err := out.UnmarshalBinary(blob); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("decoded stat = %#v, want %#v", out, in)
	}
}
