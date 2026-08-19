package fs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/storage"
)

var (
	ErrInvalidPath = errors.New("fakefs: invalid virtual path")
	ErrNotFound    = storage.ErrNotFound
)

// FS overlays iSH metadata on a real directory. File bytes and directory
// entries remain in the host directory while inode identity, mode, uid, gid,
// and device numbers live in the Pure Go SQLite metadata database.
type FS struct {
	root string
	db   *storage.DB
}

func Open(root, metaPath string) (*FS, error) {
	if root == "" {
		return nil, fmt.Errorf("fakefs: empty root path")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("fakefs: absolute root: %w", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("fakefs: create root: %w", err)
	}
	if metaPath == "" {
		metaPath = root + ".meta.db"
	}
	db, err := storage.Open(context.Background(), metaPath)
	if err != nil {
		return nil, err
	}
	fsys := &FS{root: root, db: db}
	if err := fsys.ensureRoot(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return fsys, nil
}

func New(root string, db *storage.DB) (*FS, error) {
	if db == nil {
		return nil, fmt.Errorf("fakefs: nil storage database")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("fakefs: absolute root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("fakefs: create root: %w", err)
	}
	fsys := &FS{root: abs, db: db}
	if err := fsys.ensureRoot(); err != nil {
		return nil, err
	}
	return fsys, nil
}

func (f *FS) Close() error {
	if f == nil || f.db == nil {
		return nil
	}
	return f.db.Close()
}

func (f *FS) Root() string {
	return f.root
}

func (f *FS) Database() *storage.DB {
	return f.db
}

func (f *FS) ensureRoot() error {
	_, _, exists, err := f.db.PathReadStat(context.Background(), "/")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = f.db.PathCreate(context.Background(), "/", IshStat{Mode: ModeDir | 0o755, UID: 0, GID: 0}.toStorage())
	if err != nil {
		return fmt.Errorf("fakefs: create root metadata: %w", err)
	}
	return nil
}

func (f *FS) virtualPath(name string) (string, error) {
	if name == "" {
		return "/", nil
	}
	if !strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("%w: %q is not absolute", ErrInvalidPath, name)
	}
	depth := 0
	for _, part := range strings.Split(strings.TrimPrefix(name, "/"), "/") {
		switch part {
		case "", ".":
			continue
		case "..":
			if depth == 0 {
				return "", fmt.Errorf("%w: %q escapes root", ErrInvalidPath, name)
			}
			depth--
		default:
			depth++
		}
	}
	clean := pathpkg.Clean(name)
	if clean == "." {
		clean = "/"
	}
	return clean, nil
}

func (f *FS) hostPath(name string) (string, string, error) {
	virtual, err := f.virtualPath(name)
	if err != nil {
		return "", "", err
	}
	if virtual == "/" {
		return f.root, virtual, nil
	}
	rel := filepath.FromSlash(strings.TrimPrefix(virtual, "/"))
	return filepath.Join(f.root, rel), virtual, nil
}

func (f *FS) Stat(name string) (FileInfo, error) {
	return f.stat(name, false)
}

func (f *FS) Lstat(name string) (FileInfo, error) {
	return f.stat(name, true)
}

func (f *FS) stat(name string, noFollow bool) (FileInfo, error) {
	host, virtual, err := f.hostPath(name)
	if err != nil {
		return FileInfo{}, err
	}
	var info os.FileInfo
	if noFollow {
		info, err = os.Lstat(host)
	} else {
		info, err = os.Stat(host)
	}
	if err != nil {
		return FileInfo{}, err
	}
	metadata, inode, exists, err := f.db.PathReadStat(context.Background(), virtual)
	if err != nil {
		return FileInfo{}, err
	}
	if !exists {
		return FileInfo{}, fmt.Errorf("fakefs: %s: %w", virtual, ErrNotFound)
	}
	return FileInfo{
		Name:    info.Name(),
		Size:    info.Size(),
		Mode:    ishStatFromStorage(metadata),
		Inode:   inode,
		ModTime: info.ModTime(),
	}, nil
}

func (f *FS) OpenFile(name string, flags int, perm os.FileMode, stat IshStat) (*os.File, error) {
	host, virtual, err := f.hostPath(name)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(host, flags, perm)
	if err != nil {
		return nil, err
	}
	if flags&os.O_CREATE != 0 {
		_, _, exists, statErr := f.db.PathReadStat(context.Background(), virtual)
		if statErr != nil {
			_ = file.Close()
			return nil, statErr
		}
		if !exists {
			if _, statErr := f.db.PathCreate(context.Background(), virtual, stat.toStorage()); statErr != nil {
				_ = file.Close()
				return nil, statErr
			}
		}
	}
	if _, _, exists, statErr := f.db.PathReadStat(context.Background(), virtual); statErr != nil {
		_ = file.Close()
		return nil, statErr
	} else if !exists {
		_ = file.Close()
		return nil, fmt.Errorf("fakefs: %s: %w", virtual, ErrNotFound)
	}
	return file, nil
}

func (f *FS) Create(name string, mode uint32, uid, gid uint32) (*os.File, error) {
	stat := IshStat{Mode: ModeRegular | (mode & 0o7777), UID: uid, GID: gid}
	return f.OpenFile(name, os.O_RDWR|os.O_CREATE, os.FileMode(mode&0o7777), stat)
}

func (f *FS) Mkdir(name string, mode uint32, uid, gid uint32) error {
	host, virtual, err := f.hostPath(name)
	if err != nil {
		return err
	}
	if err := os.Mkdir(host, os.FileMode(mode&0o7777)); err != nil {
		return err
	}
	_, err = f.db.PathCreate(context.Background(), virtual, IshStat{Mode: ModeDir | (mode & 0o7777), UID: uid, GID: gid}.toStorage())
	if err != nil {
		_ = os.Remove(host)
		return err
	}
	return nil
}

func (f *FS) Symlink(target, link string, uid, gid uint32) error {
	host, virtual, err := f.hostPath(link)
	if err != nil {
		return err
	}
	if err := os.Symlink(target, host); err != nil {
		return err
	}
	stat := IshStat{Mode: ModeSymlink | 0o777, UID: uid, GID: gid}
	if _, err := f.db.PathCreate(context.Background(), virtual, stat.toStorage()); err != nil {
		_ = os.Remove(host)
		return err
	}
	return nil
}

func (f *FS) Readlink(name string) (string, error) {
	info, err := f.Lstat(name)
	if err != nil {
		return "", err
	}
	if !info.Mode.isSymlink() {
		return "", fmt.Errorf("fakefs: %s: not a symlink", name)
	}
	host, _, err := f.hostPath(name)
	if err != nil {
		return "", err
	}
	return os.Readlink(host)
}

func (f *FS) Link(src, dst string) error {
	srcHost, srcVirtual, err := f.hostPath(src)
	if err != nil {
		return err
	}
	dstHost, dstVirtual, err := f.hostPath(dst)
	if err != nil {
		return err
	}
	if _, _, exists, err := f.db.PathReadStat(context.Background(), srcVirtual); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("fakefs: %s: %w", srcVirtual, ErrNotFound)
	}
	if err := os.Link(srcHost, dstHost); err != nil {
		return err
	}
	if err := f.db.PathLink(context.Background(), srcVirtual, dstVirtual); err != nil {
		_ = os.Remove(dstHost)
		return err
	}
	return nil
}

func (f *FS) Unlink(name string) error {
	host, virtual, err := f.hostPath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(host); err != nil {
		return err
	}
	inode, err := f.db.PathUnlink(context.Background(), virtual)
	if err != nil {
		return err
	}
	return f.db.TryCleanupInode(context.Background(), inode)
}

func (f *FS) Rename(src, dst string) error {
	srcHost, srcVirtual, err := f.hostPath(src)
	if err != nil {
		return err
	}
	dstHost, dstVirtual, err := f.hostPath(dst)
	if err != nil {
		return err
	}
	if _, _, exists, err := f.db.PathReadStat(context.Background(), srcVirtual); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("fakefs: %s: %w", srcVirtual, ErrNotFound)
	}
	if err := os.Rename(srcHost, dstHost); err != nil {
		return err
	}
	if err := f.db.PathRename(context.Background(), srcVirtual, dstVirtual); err != nil {
		return fmt.Errorf("fakefs: rename metadata %s -> %s: %w", srcVirtual, dstVirtual, err)
	}
	return nil
}

func (f *FS) SetAttr(name string, mode *uint32, uid, gid *uint32) error {
	metadata, inode, exists, err := f.db.PathReadStat(context.Background(), name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("fakefs: %s: %w", name, ErrNotFound)
	}
	if mode != nil {
		metadata.Mode = (metadata.Mode & ModeTypeMask) | (*mode &^ ModeTypeMask)
	}
	if uid != nil {
		metadata.UID = *uid
	}
	if gid != nil {
		metadata.GID = *gid
	}
	return f.db.InodeWriteStat(context.Background(), inode, metadata)
}

func (f *FS) Truncate(name string, size int64) error {
	host, _, err := f.hostPath(name)
	if err != nil {
		return err
	}
	return os.Truncate(host, size)
}

func (f *FS) ReadDir(name string) ([]FileInfo, error) {
	host, virtual, err := f.hostPath(name)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(host)
	if err != nil {
		return nil, err
	}
	result := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		child := pathpkg.Join(virtual, entry.Name())
		info, err := f.Lstat(child)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		result = append(result, info)
	}
	return result, nil
}

func (f *FS) PathsForInode(inode uint64) ([][]byte, error) {
	return f.db.PathsFromInode(context.Background(), inode)
}

func (f *FS) ReadFile(name string) ([]byte, error) {
	host, _, err := f.hostPath(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(host)
}

func (f *FS) WriteFile(name string, data []byte, mode uint32, uid, gid uint32) error {
	file, err := f.Create(name, mode, uid, gid)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Truncate(0); err != nil {
		return err
	}
	_, err = file.Write(data)
	return err
}

func (f *FS) CopyFile(src, dst string, mode uint32, uid, gid uint32) error {
	data, err := f.ReadFile(src)
	if err != nil {
		return err
	}
	return f.WriteFile(dst, data, mode, uid, gid)
}

func (f *FS) Touch(name string, now time.Time) error {
	host, _, err := f.hostPath(name)
	if err != nil {
		return err
	}
	return os.Chtimes(host, now, now)
}

func (f *FS) ReadAt(name string, p []byte, off int64) (int, error) {
	host, _, err := f.hostPath(name)
	if err != nil {
		return 0, err
	}
	file, err := os.Open(host)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return file.ReadAt(p, off)
}

func (f *FS) WriteAt(name string, p []byte, off int64) (int, error) {
	host, _, err := f.hostPath(name)
	if err != nil {
		return 0, err
	}
	file, err := os.OpenFile(host, os.O_WRONLY, 0)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return file.WriteAt(p, off)
}

func (f *FS) CopyTo(w io.Writer, name string) error {
	data, err := f.ReadFile(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

var _ fs.FS = (*stdlibFS)(nil)

type stdlibFS struct{ owner *FS }

func (f *FS) FS() fs.FS {
	return &stdlibFS{owner: f}
}

func (s *stdlibFS) Open(name string) (fs.File, error) {
	if name == "." {
		name = "/"
	}
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	return s.owner.OpenFile(name, os.O_RDONLY, 0, IshStat{})
}
