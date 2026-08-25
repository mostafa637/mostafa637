package rootfs

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// InstallBundled installs the data tree from a trusted bundled root archive and
// copies a prebuilt fakefs metadata database. It deliberately does not open
// SQLite. Android x86_64 rejects the legacy lstat syscall used by the pure-Go
// SQLite VFS, while the bundled database is generated from the same archive on
// Linux and is consumed by the native iSH fakefs core.
func InstallBundled(ctx context.Context, archive io.Reader, metadata io.Reader, base string) error {
	if archive == nil {
		return errors.New("rootfs: nil archive")
	}
	if metadata == nil {
		return errors.New("rootfs: nil bundled metadata")
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

	if err := extractData(ctx, archive, filepath.Join(stage, "data")); err != nil {
		return err
	}
	if err := copyBundledMetadata(metadata, filepath.Join(stage, "meta.db")); err != nil {
		return err
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
	return ValidateBundled(base)
}

func extractData(ctx context.Context, archive io.Reader, dataRoot string) error {
	gz, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("rootfs: open gzip stream: %w", err)
	}
	defer gz.Close()

	t := tar.NewReader(gz)
	hardlinks := make([]hardlink, 0)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
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
		if h.Typeflag == tar.TypeLink {
			target, err := normalizePath(h.Linkname)
			if err != nil {
				return fmt.Errorf("rootfs: unsafe hardlink target %q: %w", h.Linkname, err)
			}
			hardlinks = append(hardlinks, hardlink{path: path, target: target})
			continue
		}

		out := bundledDataPath(dataRoot, path)
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
			// files and reads their guest type from the bundled metadata database.
			f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				return fmt.Errorf("rootfs: create %q: %w", path, err)
			}
			if h.Typeflag == tar.TypeReg || h.Typeflag == tar.TypeRegA {
				if _, err := io.Copy(f, t); err != nil {
					_ = f.Close()
					return fmt.Errorf("rootfs: copy %q: %w", path, err)
				}
			} else if h.Typeflag == tar.TypeSymlink {
				if _, err := io.WriteString(f, h.Linkname); err != nil {
					_ = f.Close()
					return fmt.Errorf("rootfs: write symlink %q: %w", path, err)
				}
			}
			if err := f.Close(); err != nil {
				return fmt.Errorf("rootfs: close %q: %w", path, err)
			}
		default:
			return fmt.Errorf("rootfs: unsupported tar type %d for %q", h.Typeflag, path)
		}
	}

	for _, link := range hardlinks {
		if err := materializeParents(bundledDataPath(dataRoot, link.path)); err != nil {
			return err
		}
		if err := os.Link(bundledDataPath(dataRoot, link.target), bundledDataPath(dataRoot, link.path)); err != nil {
			return fmt.Errorf("rootfs: hardlink %q -> %q: %w", link.path, link.target, err)
		}
	}
	return nil
}

func copyBundledMetadata(metadata io.Reader, path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("rootfs: create bundled metadata: %w", err)
	}
	n, copyErr := io.Copy(f, metadata)
	closeErr := f.Close()
	if copyErr != nil {
		return fmt.Errorf("rootfs: copy bundled metadata: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("rootfs: close bundled metadata: %w", closeErr)
	}
	if n < 100 {
		return errors.New("rootfs: bundled metadata is not a SQLite database")
	}
	return nil
}

// ValidateBundled checks the Android install without opening SQLite. The
// database is trusted because it is embedded in the APK and generated from the
// matching root archive during the release build.
func ValidateBundled(base string) error {
	dataDir, err := os.Open(filepath.Join(base, "data"))
	if err != nil {
		return fmt.Errorf("rootfs: data directory is missing")
	}
	info, statErr := dataDir.Stat()
	closeErr := dataDir.Close()
	if statErr != nil || closeErr != nil || !info.IsDir() {
		return fmt.Errorf("rootfs: data directory is missing")
	}

	meta, err := os.Open(filepath.Join(base, "meta.db"))
	if err != nil {
		return fmt.Errorf("rootfs: metadata database is missing")
	}
	defer meta.Close()
	info, err = meta.Stat()
	if err != nil || info.Size() < 100 {
		return fmt.Errorf("rootfs: metadata database is missing or empty")
	}
	var header [16]byte
	if _, err := io.ReadFull(meta, header[:]); err != nil || string(header[:15]) != "SQLite format 3" || header[15] != 0 {
		return errors.New("rootfs: metadata database has an invalid SQLite header")
	}
	return nil
}

func bundledDataPath(dataRoot, path string) string {
	return filepath.Join(dataRoot, strings.TrimPrefix(path, "/"))
}
