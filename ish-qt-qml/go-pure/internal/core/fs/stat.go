package fs

import (
	"io/fs"
	"time"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/storage"
)

const (
	ModeTypeMask = 0o170000
	ModeFIFO     = 0o010000
	ModeChar     = 0o020000
	ModeDir      = 0o040000
	ModeBlock    = 0o060000
	ModeRegular  = 0o100000
	ModeSymlink  = 0o120000
	ModeSocket   = 0o140000
)

// IshStat is byte-for-byte compatible with the original C struct ish_stat.
// The fakefs database stores exactly these four uint32 fields.
type IshStat struct {
	Mode uint32
	UID  uint32
	GID  uint32
	Rdev uint32
}

func (s IshStat) isDir() bool {
	return s.Mode&ModeTypeMask == ModeDir
}

func (s IshStat) isSymlink() bool {
	return s.Mode&ModeTypeMask == ModeSymlink
}

func (s IshStat) toStorage() storage.Stat {
	return storage.Stat{Mode: s.Mode, UID: s.UID, GID: s.GID, Rdev: s.Rdev}
}

func ishStatFromStorage(s storage.Stat) IshStat {
	return IshStat{Mode: s.Mode, UID: s.UID, GID: s.GID, Rdev: s.Rdev}
}

// FileInfo combines real filesystem size/timestamps with iSH fake metadata.
type FileInfo struct {
	Name    string
	Size    int64
	Mode    IshStat
	Inode   uint64
	ModTime time.Time
}

func (i FileInfo) IsDir() bool {
	return i.Mode.isDir()
}

func (i FileInfo) Type() fs.FileMode {
	switch i.Mode.Mode & ModeTypeMask {
	case ModeDir:
		return fs.ModeDir
	case ModeSymlink:
		return fs.ModeSymlink
	case ModeNamedPipe:
		return fs.ModeNamedPipe
	default:
		return 0
	}
}

func (i FileInfo) Perm() fs.FileMode {
	return fs.FileMode(i.Mode.Mode & 0o7777)
}

// ModeNamedPipe is kept local because the POSIX mode value is not exposed
// consistently by syscall packages on every supported Go target.
const ModeNamedPipe = ModeFIFO
