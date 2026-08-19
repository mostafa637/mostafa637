package fs

import (
	"errors"
	stdfs "io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/storage"
)

func newTestFS(t *testing.T) *FS {
	t.Helper()
	db, err := storage.Open(t.Context(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "root")
	fake, err := New(root, db)
	if err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fake.Close() })
	return fake
}

func TestFakeFSFileAndMetadataLifecycle(t *testing.T) {
	fake := newTestFS(t)
	if err := fake.Mkdir("/etc", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.WriteFile("/etc/config", []byte("hello"), 0o644, 1000, 1000); err != nil {
		t.Fatal(err)
	}
	if got, err := fake.ReadFile("/etc/config"); err != nil {
		t.Fatal(err)
	} else if string(got) != "hello" {
		t.Fatalf("content = %q, want hello", got)
	}

	info, err := fake.Stat("/etc/config")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode.Mode != ModeRegular|0o644 || info.Mode.UID != 1000 || info.Mode.GID != 1000 {
		t.Fatalf("metadata = %#v, want regular 0644 uid/gid 1000", info.Mode)
	}
	if info.Inode == 0 {
		t.Fatal("metadata inode is zero")
	}

	newMode := uint32(0o600)
	newUID := uint32(2000)
	if err := fake.SetAttr("/etc/config", &newMode, &newUID, nil); err != nil {
		t.Fatal(err)
	}
	info, err = fake.Stat("/etc/config")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode.Mode != ModeRegular|0o600 || info.Mode.UID != 2000 || info.Mode.GID != 1000 {
		t.Fatalf("updated metadata = %#v", info.Mode)
	}

	if err := fake.Link("/etc/config", "/etc/config.link"); err != nil {
		t.Fatal(err)
	}
	linked, err := fake.Stat("/etc/config.link")
	if err != nil {
		t.Fatal(err)
	}
	if linked.Inode != info.Inode {
		t.Fatalf("hard link inode = %d, original = %d", linked.Inode, info.Inode)
	}
	paths, err := fake.PathsForInode(info.Inode)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]byte{[]byte("/etc/config"), []byte("/etc/config.link")}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %q, want %q", paths, want)
	}

	if err := fake.Rename("/etc", "/var"); err != nil {
		t.Fatal(err)
	}
	if _, err := fake.Stat("/etc/config"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old path error = %v, want os.ErrNotExist", err)
	}
	moved, err := fake.Stat("/var/config.link")
	if err != nil {
		t.Fatal(err)
	}
	if moved.Inode != info.Inode || moved.Mode.UID != 2000 {
		t.Fatalf("moved metadata = %#v inode=%d", moved.Mode, moved.Inode)
	}

	if err := fake.Unlink("/var/config"); err != nil {
		t.Fatal(err)
	}
	if err := fake.Unlink("/var/config.link"); err != nil {
		t.Fatal(err)
	}
	if _, err := fake.Database().InodeReadStat(t.Context(), info.Inode); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("cleaned inode error = %v, want ErrNotFound", err)
	}
}

func TestFakeFSSymlinkAndReadDir(t *testing.T) {
	fake := newTestFS(t)
	if err := fake.Mkdir("/bin", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.WriteFile("/bin/sh", []byte("shell"), 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.Symlink("/bin/sh", "/bin/default", 0, 0); err != nil {
		t.Fatal(err)
	}
	if got, err := fake.Readlink("/bin/default"); err != nil {
		t.Fatal(err)
	} else if got != "/bin/sh" {
		t.Fatalf("link target = %q", got)
	}
	linkInfo, err := fake.Lstat("/bin/default")
	if err != nil {
		t.Fatal(err)
	}
	if linkInfo.Mode.Mode != ModeSymlink|0o777 || !linkInfo.Mode.isSymlink() {
		t.Fatalf("link metadata = %#v", linkInfo.Mode)
	}

	entries, err := fake.ReadDir("/bin")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
}

func TestFakeFSPathValidationAndStdlibFS(t *testing.T) {
	fake := newTestFS(t)
	if _, err := fake.Stat("relative"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("relative path error = %v", err)
	}
	if _, err := fake.Stat("/../escape"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("escape path error = %v", err)
	}
	if err := fake.WriteFile("/hello", []byte("world"), 0o644, 0, 0); err != nil {
		t.Fatal(err)
	}
	data, err := stdfs.ReadFile(fake.FS(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "world" {
		t.Fatalf("fs.ReadFile = %q", data)
	}
}
