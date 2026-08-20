package fs

import (
	"time"

	"golang.org/x/sys/unix"
)

// Times returns the host-backed timestamps for a virtual path. The fakefs
// already uses the host inode timestamps for FileInfo.ModTime; exposing both
// values keeps utimensat consistent without adding a second clock store.
func (f *FS) Times(name string, noFollow bool) (time.Time, time.Time, error) {
	host, _, err := f.hostPath(name)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	var stat unix.Stat_t
	if noFollow {
		err = unix.Lstat(host, &stat)
	} else {
		err = unix.Stat(host, &stat)
	}
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return time.Unix(stat.Atim.Sec, stat.Atim.Nsec), time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec), nil
}

// SetTimes updates atime and mtime. A nil value preserves the existing time,
// which is the UTIME_OMIT behavior required by utimensat.
func (f *FS) SetTimes(name string, atime, mtime *time.Time, noFollow bool) error {
	host, _, err := f.hostPath(name)
	if err != nil {
		return err
	}
	if atime == nil || mtime == nil {
		currentAtime, currentMtime, statErr := f.Times(name, noFollow)
		if statErr != nil {
			return statErr
		}
		if atime == nil {
			atime = &currentAtime
		}
		if mtime == nil {
			mtime = &currentMtime
		}
	}
	flags := 0
	if noFollow {
		flags = unix.AT_SYMLINK_NOFOLLOW
	}
	return unix.UtimesNanoAt(unix.AT_FDCWD, host, []unix.Timespec{
		{Sec: atime.Unix(), Nsec: int64(atime.Nanosecond())},
		{Sec: mtime.Unix(), Nsec: int64(mtime.Nanosecond())},
	}, flags)
}
