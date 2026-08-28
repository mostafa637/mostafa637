package rootfs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestInstallCreatesFakefsLayout(t *testing.T) {
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gz)
	entries := []struct {
		name     string
		mode     int64
		body     string
		typeflag byte
	}{
		{"bin", 0o755, "", tar.TypeDir},
		{"usr", 0o755, "", tar.TypeDir},
		{"usr/bin", 0o755, "", tar.TypeDir},
		{"bin/sh", 0o755, "#!/bin/sh\necho ok\n", tar.TypeReg},
		{"usr/bin/python", 0o755, "python\n", tar.TypeReg},
	}
	for _, entry := range entries {
		if err := tarWriter.WriteHeader(&tar.Header{Name: entry.name, Mode: entry.mode, Size: int64(len(entry.body)), Typeflag: entry.typeflag}); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(tarWriter, entry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.WriteHeader(&tar.Header{Name: "usr/bin/python3", Linkname: "usr/bin/python", Typeflag: tar.TypeLink, Mode: 0o755}); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	base := t.TempDir()
	if err := Install(context.Background(), bytes.NewReader(archive.Bytes()), base); err != nil {
		t.Fatal(err)
	}
	if err := Validate(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"data/bin/sh", "data/usr/bin/python", "data/usr/bin/python3", "meta.db"} {
		if _, err := os.Stat(filepath.Join(base, path)); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
	original, err := os.Stat(filepath.Join(base, "data/usr/bin/python"))
	if err != nil {
		t.Fatal(err)
	}
	linked, err := os.Stat(filepath.Join(base, "data/usr/bin/python3"))
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(original, linked) {
		t.Fatal("hardlink does not share inode")
	}

	db, err := sql.Open("sqlite", filepath.Join(base, "meta.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow("SELECT count(*) FROM paths WHERE path = ?", []byte("")).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("root path count = %d, want 1", count)
	}
	var inodeA, inodeB int64
	if err := db.QueryRow("SELECT inode FROM paths WHERE path = ?", []byte("/usr/bin/python")).Scan(&inodeA); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT inode FROM paths WHERE path = ?", []byte("/usr/bin/python3")).Scan(&inodeB); err != nil {
		t.Fatal(err)
	}
	if inodeA != inodeB {
		t.Fatalf("hardlink database inode mismatch: %d != %d", inodeA, inodeB)
	}
}

func TestNormalizePathRejectsTraversal(t *testing.T) {
	if _, err := normalizePath("../../etc/passwd"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
	if got, err := normalizePath("./bin/../sh"); err == nil || got != "" {
		t.Fatal("expected embedded traversal to be rejected")
	}
}
