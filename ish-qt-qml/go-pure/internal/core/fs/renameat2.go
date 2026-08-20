package fs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/storage"
)

const (
	RenameNoReplace uint32 = 1 << iota
	RenameExchange
	RenameWhiteout
)

// RenameNoReplace performs a rename only when the destination does not exist.
func (f *FS) RenameNoReplace(src, dst string) error {
	srcHost, srcVirtual, err := f.hostPath(src)
	if err != nil {
		return err
	}
	dstHost, dstVirtual, err := f.hostPath(dst)
	if err != nil {
		return err
	}
	if srcVirtual == dstVirtual {
		return storage.ErrExists
	}
	if _, err := os.Lstat(srcHost); err != nil {
		return err
	}
	if _, err := os.Lstat(dstHost); err == nil {
		return fmt.Errorf("fakefs: destination %s exists: %w", dstVirtual, storage.ErrExists)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(srcHost, dstHost); err != nil {
		return fmt.Errorf("fakefs: rename %s -> %s: %w", srcVirtual, dstVirtual, err)
	}
	if err := f.db.PathRenameNoReplace(context.Background(), srcVirtual, dstVirtual); err != nil {
		_ = os.Rename(dstHost, srcHost)
		return err
	}
	return nil
}

// RenameExchange atomically swaps two existing path trees from fakefs's
// perspective while preserving each inode's identity in SQLite metadata.
func (f *FS) RenameExchange(src, dst string) error {
	srcHost, srcVirtual, err := f.hostPath(src)
	if err != nil {
		return err
	}
	dstHost, dstVirtual, err := f.hostPath(dst)
	if err != nil {
		return err
	}
	if srcVirtual == "/" || dstVirtual == "/" || srcVirtual == dstVirtual || pathIsNested(srcVirtual, dstVirtual) || pathIsNested(dstVirtual, srcVirtual) {
		return ErrInvalidPath
	}
	if _, err := os.Lstat(srcHost); err != nil {
		return err
	}
	if _, err := os.Lstat(dstHost); err != nil {
		return err
	}

	tempHost, err := uniqueRenameTemp(srcHost)
	if err != nil {
		return err
	}
	if err := os.Rename(srcHost, tempHost); err != nil {
		return fmt.Errorf("fakefs: exchange stage source: %w", err)
	}
	completed := false
	defer func() {
		if completed {
			return
		}
		_ = os.Rename(tempHost, srcHost)
	}()
	if err := os.Rename(dstHost, srcHost); err != nil {
		_ = os.Rename(tempHost, srcHost)
		return fmt.Errorf("fakefs: exchange stage destination: %w", err)
	}
	if err := os.Rename(tempHost, dstHost); err != nil {
		_ = os.Rename(srcHost, dstHost)
		_ = os.Rename(tempHost, srcHost)
		return fmt.Errorf("fakefs: exchange install source: %w", err)
	}
	if err := f.db.PathExchange(context.Background(), srcVirtual, dstVirtual); err != nil {
		// Best-effort rollback; the metadata store is single-connection and the
		// host swap is still reversible before returning to the guest.
		_ = os.Rename(dstHost, tempHost)
		_ = os.Rename(srcHost, dstHost)
		_ = os.Rename(tempHost, srcHost)
		return err
	}
	completed = true
	return nil
}

func uniqueRenameTemp(srcHost string) (string, error) {
	dir := filepath.Dir(srcHost)
	base := filepath.Base(srcHost)
	for attempt := 0; attempt < 16; attempt++ {
		candidate := filepath.Join(dir, fmt.Sprintf(".%s.ishgo-renameat2-%d-%d", base, os.Getpid(), time.Now().UnixNano()+int64(attempt)))
		if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("fakefs: unable to allocate renameat2 temporary path")
}

func pathIsNested(parent, child string) bool {
	if parent == "/" || parent == child {
		return parent == child
	}
	return strings.HasPrefix(child, strings.TrimRight(parent, "/")+"/")
}
