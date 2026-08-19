package fs

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
)

// BootstrapMetadata populates missing fakefs records from the existing rootfs.
// It never overwrites an inode already present in meta.db, which makes it safe
// to run after an interrupted import or on every session startup.
func (f *FS) BootstrapMetadata(uid, gid uint32) error {
	return filepath.WalkDir(f.root, func(host string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if host == f.root {
			return nil
		}
		rel, err := filepath.Rel(f.root, host)
		if err != nil {
			return err
		}
		virtual := "/" + filepath.ToSlash(rel)
		virtual, err = f.virtualPath(virtual)
		if err != nil {
			return err
		}
		_, _, exists, err := f.db.PathReadStat(context.Background(), virtual)
		if err != nil {
			return err
		}
		if exists {
			if entry.IsDir() {
				return nil
			}
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("fakefs: inspect %s: %w", virtual, err)
		}
		mode := modeFromFileMode(info.Mode())
		if _, err := f.db.PathCreate(context.Background(), virtual, IshStat{
			Mode: mode,
			UID:  uid,
			GID:  gid,
		}.toStorage()); err != nil {
			return fmt.Errorf("fakefs: bootstrap %s: %w", virtual, err)
		}
		return nil
	})
}

func modeFromFileMode(mode os.FileMode) uint32 {
	var kind uint32
	switch {
	case mode&os.ModeSymlink != 0:
		kind = ModeSymlink
	case mode.IsDir():
		kind = ModeDir
	case mode&os.ModeNamedPipe != 0:
		kind = ModeFIFO
	case mode&os.ModeSocket != 0:
		kind = ModeSocket
	default:
		kind = ModeRegular
	}
	return kind | uint32(mode.Perm())
}

func virtualFromHost(root, host string) string {
	rel, err := filepath.Rel(root, host)
	if err != nil || rel == "." {
		return "/"
	}
	return pathpkg.Clean("/" + strings.ReplaceAll(filepath.ToSlash(rel), "\\", "/"))
}
