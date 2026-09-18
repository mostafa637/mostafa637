// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/sys_file.c — the file-descriptor syscalls.

package linux

import (
	"encoding/binary"
	"syscall"

	"github.com/mostafa637/mostafa637/abi"
	"github.com/mostafa637/mostafa637/mem"
)

func init() {
	Register(map[uint64]Handler{
		Sysopenat:     sysOpenat,
		Sysclose:      sysClose,
		Sysread:       sysRead,
		Syswrite:      sysWrite,
		Sysreadv:      sysReadv,
		Syswritev:     sysWritev,
		Syspread64:    sysPread,
		Syspwrite64:   sysPwrite,
		Syspreadv:     sysPreadv,
		Syspwritev:    sysPwritev,
		Syslseek:      sysLseek,
		Sysfstat:      sysFstat,
		Sysnewfstatat: sysNewFstatat,
		Sysreadlinkat: sysReadlinkat,
		Sysgetcwd:     sysGetcwd,
		Sysgetdents64: sysGetdents64,
		Sysioctl:      sysIoctl,
		Sysfcntl:      sysFcntl,
		Sysdup:        sysDup,
		Sysdup3:       sysDup3,
		Syspipe2:      sysPipe2,
		Sysunlinkat:   sysUnlinkat,
		Sysmkdirat:    sysMkdirat,
		Sysrenameat:   sysRenameat,
		Syssymlinkat:  sysSymlinkat,
		Syslinkat:     sysLinkat,
		Sysfaccessat:  sysFaccessat,
		Sysftruncate:  sysFtruncate,
		Sysfchmod:     sysFchmod,
		Sysfchmodat:   sysFchmodat,
		Sysfchown:     sysFchown,
		Sysfchownat:   sysFchownat,
		Sysutimensat:  sysUtimensat,
		Sysfsync:      sysFsync,
		Sysfdatasync:  sysFdatasync,
		Syssync:       sysSync,
		Syschdir:      sysChdir,
		Sysfchdir:     sysFchdir,
		Sysmknodat:    sysMknodat,
		Sysstatx:      sysStatx,
	})
}

// ---- guest struct conversion ---------------------------------------------

// GuestStat is asm-generic struct stat for arm64 (128 bytes). The host's
// struct stat is a different shape on x86-64, so the fields are marshalled
// one by one rather than copied.
const statSize = 128

func putStat(buf []byte, st *syscall.Stat_t) {
	le := binary.LittleEndian
	le.PutUint64(buf[0:], uint64(st.Dev))
	le.PutUint64(buf[8:], st.Ino)
	le.PutUint32(buf[16:], uint32(st.Mode))
	le.PutUint32(buf[20:], uint32(st.Nlink))
	le.PutUint32(buf[24:], st.Uid)
	le.PutUint32(buf[28:], st.Gid)
	le.PutUint64(buf[32:], uint64(st.Rdev))
	le.PutUint64(buf[40:], 0)
	le.PutUint64(buf[48:], uint64(st.Size))
	le.PutUint32(buf[56:], uint32(st.Blksize))
	le.PutUint32(buf[60:], 0)
	le.PutUint64(buf[64:], uint64(st.Blocks))
	le.PutUint64(buf[72:], uint64(st.Atim.Sec))
	le.PutUint64(buf[80:], uint64(st.Atim.Nsec))
	le.PutUint64(buf[88:], uint64(st.Mtim.Sec))
	le.PutUint64(buf[96:], uint64(st.Mtim.Nsec))
	le.PutUint64(buf[104:], uint64(st.Ctim.Sec))
	le.PutUint64(buf[112:], uint64(st.Ctim.Nsec))
	le.PutUint64(buf[120:], 0)
}

// ---- I/O ------------------------------------------------------------------

func sysWrite(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	buf := make([]byte, a[2])
	if err := mem.CopyFromGuest(t.CPU, buf, a[1]); err != nil {
		return retErr(err)
	}
	n, err := syscall.Write(fd, buf)
	if n > 0 {
		return retOK(uint64(n))
	}
	return retErr(err)
}

func sysRead(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	buf := make([]byte, a[2])
	n, err := syscall.Read(fd, buf)
	if n > 0 {
		if err := mem.CopyToGuest(t.CPU, a[1], buf[:n]); err != nil {
			return retErr(err)
		}
		return retOK(uint64(n))
	}
	return retErr(err)
}

func sysPread(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	buf := make([]byte, a[2])
	n, err := syscall.Pread(fd, buf, int64(a[3]))
	if n > 0 {
		if err := mem.CopyToGuest(t.CPU, a[1], buf[:n]); err != nil {
			return retErr(err)
		}
		return retOK(uint64(n))
	}
	return retErr(err)
}

func sysPwrite(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	buf := make([]byte, a[2])
	if err := mem.CopyFromGuest(t.CPU, buf, a[1]); err != nil {
		return retErr(err)
	}
	n, err := syscall.Pwrite(fd, buf, int64(a[3]))
	if n > 0 {
		return retOK(uint64(n))
	}
	return retErr(err)
}

// iovec is the guest's struct iovec: {void *base; size_t len}.
func guestIOVs(t *Task, va, count uint64) ([][]byte, uint64, error) {
	buf := make([]byte, 16*count)
	if err := mem.CopyFromGuest(t.CPU, buf, va); err != nil {
		return nil, 0, err
	}
	iovs := make([][]byte, count)
	total := uint64(0)
	for i := uint64(0); i < count; i++ {
		base := binary.LittleEndian.Uint64(buf[i*16:])
		l := binary.LittleEndian.Uint64(buf[i*16+8:])
		iovs[i] = make([]byte, l)
		total += l
		if l > 0 {
			if err := mem.CopyFromGuest(t.CPU, iovs[i], base); err != nil {
				return nil, 0, err
			}
		}
	}
	return iovs, total, nil
}

func sysWritev(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	iovs, _, err := guestIOVs(t, a[1], a[2])
	if err != nil {
		return retErr(err)
	}
	written := 0
	for _, b := range iovs {
		n, err := syscall.Write(fd, b)
		if n > 0 {
			written += n
		}
		if err != nil && n <= 0 && written == 0 {
			return retErr(err)
		}
		if n < len(b) {
			break
		}
	}
	return retOK(uint64(written))
}

func sysReadv(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	// Read the descriptor table, then read into each buffer in turn and copy
	// the bytes back as they arrive.
	buf := make([]byte, 16*a[2])
	if err := mem.CopyFromGuest(t.CPU, buf, a[1]); err != nil {
		return retErr(err)
	}
	total := uint64(0)
	for i := uint64(0); i < a[2]; i++ {
		base := binary.LittleEndian.Uint64(buf[i*16:])
		l := binary.LittleEndian.Uint64(buf[i*16+8:])
		b := make([]byte, l)
		n, err := syscall.Read(fd, b)
		if n > 0 {
			if err := mem.CopyToGuest(t.CPU, base, b[:n]); err != nil {
				return retErr(err)
			}
			total += uint64(n)
		}
		if err != nil && n <= 0 && total == 0 {
			return retErr(err)
		}
		if uint64(n) < l {
			break
		}
	}
	return retOK(total)
}

func sysPreadv(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	buf := make([]byte, 16*a[2])
	if err := mem.CopyFromGuest(t.CPU, buf, a[1]); err != nil {
		return retErr(err)
	}
	total := uint64(0)
	off := int64(a[3])
	for i := uint64(0); i < a[2]; i++ {
		base := binary.LittleEndian.Uint64(buf[i*16:])
		l := binary.LittleEndian.Uint64(buf[i*16+8:])
		b := make([]byte, l)
		n, err := syscall.Pread(fd, b, off)
		if n > 0 {
			if err := mem.CopyToGuest(t.CPU, base, b[:n]); err != nil {
				return retErr(err)
			}
			total += uint64(n)
			off += int64(n)
		}
		if err != nil && n <= 0 && total == 0 {
			return retErr(err)
		}
		if uint64(n) < l {
			break
		}
	}
	return retOK(total)
}

func sysPwritev(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	buf := make([]byte, 16*a[2])
	if err := mem.CopyFromGuest(t.CPU, buf, a[1]); err != nil {
		return retErr(err)
	}
	total := uint64(0)
	off := int64(a[3])
	for i := uint64(0); i < a[2]; i++ {
		base := binary.LittleEndian.Uint64(buf[i*16:])
		l := binary.LittleEndian.Uint64(buf[i*16+8:])
		b := make([]byte, l)
		if l > 0 {
			if err := mem.CopyFromGuest(t.CPU, b, base); err != nil {
				return retErr(err)
			}
		}
		n, err := syscall.Pwrite(fd, b, off)
		if n > 0 {
			total += uint64(n)
			off += int64(n)
		}
		if err != nil && n <= 0 && total == 0 {
			return retErr(err)
		}
		if uint64(n) < l {
			break
		}
	}
	return retOK(total)
}

// ---- descriptor management -----------------------------------------------

func sysOpenat(t *Task, a [6]uint64) uint64 {
	dirfd := int32(int64(a[0]))
	pathStr, err := mem.CopyStringFromGuest(t.CPU, a[1], 4096)
	if err != nil {
		return retErr(err)
	}
	hostPath, err := t.ResolveAt(dirfd, pathStr)
	if err != nil {
		return retErr(err)
	}
	flags := int(a[2]) &^ 0x100000 // drop O_LARGEFILE-ish guest-only bits we carry
	mode := uint32(a[3])
	fd, err := syscall.Open(hostPath, flags, mode)
	if fd >= 0 {
		t.Own(0) // never marks anything: the guest owns what open returns
	}
	return retErr(err) + uint64(max0(fd))
}

func max0(fd int) int {
	if fd < 0 {
		return 0
	}
	return fd
}

func sysClose(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	return retErr(syscall.Close(fd))
}

func sysDup(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	nfd, err := syscall.Dup(fd)
	return retErr(err) + uint64(max0(nfd))
}

func sysDup3(t *Task, a [6]uint64) uint64 {
	oldfd, newfd := int(int32(a[0])), int(int32(a[1]))
	if !t.GuestFD(oldfd) {
		return negErrno(abi.EBADF)
	}
	return retErr(syscall.Dup3(oldfd, newfd, int(a[2])))
}

func sysPipe2(t *Task, a [6]uint64) uint64 {
	var p [2]int
	if err := syscall.Pipe2(p[:], int(a[1])); err != nil {
		return retErr(err)
	}
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf[0:], uint32(p[0]))
	binary.LittleEndian.PutUint32(buf[4:], uint32(p[1]))
	if err := mem.CopyToGuest(t.CPU, a[0], buf); err != nil {
		return retErr(err)
	}
	return 0
}

func sysFcntl(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	r, err := hostFcntl(uintptr(fd), int(a[1]), uintptr(a[2]))
	if err != nil {
		return retErr(err)
	}
	return retOK(uint64(r))
}

func sysIoctl(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	r, err := hostIoctl(uintptr(fd), uintptr(a[1]), uintptr(a[2]))
	if err != nil {
		return retErr(err)
	}
	return retOK(uint64(r))
}

func sysLseek(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	off, err := syscall.Seek(fd, int64(a[1]), int(a[2]))
	if err != nil {
		return retErr(err)
	}
	return retOK(uint64(off))
}

// ---- stat and friends ----------------------------------------------------

func sysFstat(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		return retErr(err)
	}
	buf := make([]byte, statSize)
	putStat(buf, &st)
	return retErr(mem.CopyToGuest(t.CPU, a[1], buf))
}

func sysNewFstatat(t *Task, a [6]uint64) uint64 {
	dirfd := int32(int64(a[0]))
	pathStr, err := mem.CopyStringFromGuest(t.CPU, a[1], 4096)
	if err != nil {
		return retErr(err)
	}
	hostPath, err := t.ResolveAt(dirfd, pathStr)
	if err != nil {
		return retErr(err)
	}
	var st syscall.Stat_t
	flags := int(a[3])
	if flags&AT_EMPTY_PATH != 0 && pathStr == "" {
		if err := syscall.Fstat(int(dirfd), &st); err != nil {
			return retErr(err)
		}
	} else if flags&AT_SYMLINK_NOFOLLOW != 0 {
		if err := syscall.Lstat(hostPath, &st); err != nil {
			return retErr(err)
		}
	} else {
		if err := syscall.Stat(hostPath, &st); err != nil {
			return retErr(err)
		}
	}
	buf := make([]byte, statSize)
	putStat(buf, &st)
	return retErr(mem.CopyToGuest(t.CPU, a[2], buf))
}

func sysStatx(t *Task, a [6]uint64) uint64 {
	dirfd := int32(int64(a[0]))
	pathStr, err := mem.CopyStringFromGuest(t.CPU, a[1], 4096)
	if err != nil {
		return retErr(err)
	}
	hostPath, err := t.ResolveAt(dirfd, pathStr)
	if err != nil {
		return retErr(err)
	}
	var st syscall.Stat_t
	if int(a[2])&AT_SYMLINK_NOFOLLOW != 0 {
		err = syscall.Lstat(hostPath, &st)
	} else {
		err = syscall.Stat(hostPath, &st)
	}
	if err != nil {
		return retErr(err)
	}
	// struct statx is 256 bytes; the fields a libc asks for are filled and the
	// mask says which of them are valid.
	buf := make([]byte, 256)
	le := binary.LittleEndian
	le.PutUint32(buf[0:], 0xfff)              // mask: everything below
	le.PutUint32(buf[4:], uint32(st.Blksize)) // blksize
	le.PutUint64(buf[8:], 0)                  // attributes
	le.PutUint32(buf[16:], uint32(st.Nlink))
	le.PutUint32(buf[20:], st.Uid)
	le.PutUint32(buf[24:], st.Gid)
	le.PutUint16(buf[28:], uint16(st.Mode))
	le.PutUint64(buf[32:], st.Ino)
	le.PutUint64(buf[40:], uint64(st.Size))
	le.PutUint64(buf[48:], uint64(st.Blocks))
	le.PutUint32(buf[72:], uint64lo(uint64(st.Dev)))
	le.PutUint32(buf[76:], uint64hi(uint64(st.Dev)))
	le.PutUint32(buf[80:], uint64lo(uint64(st.Rdev)))
	le.PutUint32(buf[84:], uint64hi(uint64(st.Rdev)))
	putTimespecAt(buf[128:], st.Atim)
	putTimespecAt(buf[144:], st.Mtim)
	putTimespecAt(buf[160:], st.Ctim)
	return retErr(mem.CopyToGuest(t.CPU, a[3], buf))
}

func putTimespecAt(buf []byte, ts syscall.Timespec) {
	binary.LittleEndian.PutUint64(buf[0:], uint64(ts.Sec))
	binary.LittleEndian.PutUint32(buf[8:], uint32(ts.Nsec))
}

func uint64lo(v uint64) uint32 { return uint32(v & 0xffffffff) }
func uint64hi(v uint64) uint32 { return uint32(v >> 32) }

// ---- directories ---------------------------------------------------------

func sysGetcwd(t *Task, a [6]uint64) uint64 {
	guest := t.WorkDir
	if len(guest)+1 > int(a[1]) {
		return negErrno(abi.ERANGE)
	}
	buf := append([]byte(guest), 0)
	return retErr(mem.CopyToGuest(t.CPU, a[0], buf))
}

func sysChdir(t *Task, a [6]uint64) uint64 {
	pathStr, err := mem.CopyStringFromGuest(t.CPU, a[0], 4096)
	if err != nil {
		return retErr(err)
	}
	hostPath, err := t.Resolve(pathStr)
	if err != nil {
		return retErr(err)
	}
	var st syscall.Stat_t
	if err := syscall.Stat(hostPath, &st); err != nil {
		return retErr(err)
	}
	if st.Mode&syscall.S_IFMT != syscall.S_IFDIR {
		return negErrno(abi.ENOTDIR)
	}
	t.WorkDir = joinPath(t.WorkDir, pathStr)
	return 0
}

func sysFchdir(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	dir, err := t.DirPath(fd)
	if err != nil {
		return retErr(err)
	}
	t.WorkDir = dir
	return retErr(syscall.Fchdir(fd))
}

// sysGetdents64 reads directory entries through the host's getdents64 and
// copies them out unchanged: struct linux_dirent64 has the same layout on both
// architectures (d_ino, d_off, d_reclen, d_type, d_name[]).
func sysGetdents64(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	buf := make([]byte, a[2])
	n, err := hostGetdents64(fd, buf)
	if err != nil {
		return retErr(err)
	}
	if n > 0 {
		if err := mem.CopyToGuest(t.CPU, a[1], buf[:n]); err != nil {
			return retErr(err)
		}
	}
	return retOK(uint64(n))
}

func sysReadlinkat(t *Task, a [6]uint64) uint64 {
	dirfd := int32(int64(a[0]))
	pathStr, err := mem.CopyStringFromGuest(t.CPU, a[1], 4096)
	if err != nil {
		return retErr(err)
	}
	hostPath, err := t.ResolveAt(dirfd, pathStr)
	if err != nil {
		return retErr(err)
	}
	buf := make([]byte, a[3])
	n, err := syscall.Readlink(hostPath, buf)
	if err != nil {
		return retErr(err)
	}
	return retErr(mem.CopyToGuest(t.CPU, a[2], buf[:n]))
}

func sysMkdirat(t *Task, a [6]uint64) uint64 {
	dirfd := int32(int64(a[0]))
	pathStr, err := mem.CopyStringFromGuest(t.CPU, a[1], 4096)
	if err != nil {
		return retErr(err)
	}
	hostPath, err := t.ResolveAt(dirfd, pathStr)
	if err != nil {
		return retErr(err)
	}
	return retErr(syscall.Mkdir(hostPath, uint32(a[2])))
}

func sysUnlinkat(t *Task, a [6]uint64) uint64 {
	dirfd := int32(int64(a[0]))
	pathStr, err := mem.CopyStringFromGuest(t.CPU, a[1], 4096)
	if err != nil {
		return retErr(err)
	}
	hostPath, err := t.ResolveAt(dirfd, pathStr)
	if err != nil {
		return retErr(err)
	}
	if a[2]&AT_REMOVEDIR != 0 {
		return retErr(syscall.Rmdir(hostPath))
	}
	return retErr(syscall.Unlink(hostPath))
}

func sysRenameat(t *Task, a [6]uint64) uint64 {
	oldStr, err := mem.CopyStringFromGuest(t.CPU, a[1], 4096)
	if err != nil {
		return retErr(err)
	}
	newStr, err := mem.CopyStringFromGuest(t.CPU, a[3], 4096)
	if err != nil {
		return retErr(err)
	}
	oldPath, err := t.ResolveAt(int32(int64(a[0])), oldStr)
	if err != nil {
		return retErr(err)
	}
	newPath, err := t.ResolveAt(int32(int64(a[2])), newStr)
	if err != nil {
		return retErr(err)
	}
	return retErr(syscall.Rename(oldPath, newPath))
}

func sysSymlinkat(t *Task, a [6]uint64) uint64 {
	target, err := mem.CopyStringFromGuest(t.CPU, a[0], 4096)
	if err != nil {
		return retErr(err)
	}
	linkStr, err := mem.CopyStringFromGuest(t.CPU, a[2], 4096)
	if err != nil {
		return retErr(err)
	}
	linkPath, err := t.ResolveAt(int32(int64(a[1])), linkStr)
	if err != nil {
		return retErr(err)
	}
	return retErr(syscall.Symlink(target, linkPath))
}

func sysLinkat(t *Task, a [6]uint64) uint64 {
	oldStr, err := mem.CopyStringFromGuest(t.CPU, a[1], 4096)
	if err != nil {
		return retErr(err)
	}
	newStr, err := mem.CopyStringFromGuest(t.CPU, a[3], 4096)
	if err != nil {
		return retErr(err)
	}
	oldPath, err := t.ResolveAt(int32(int64(a[0])), oldStr)
	if err != nil {
		return retErr(err)
	}
	newPath, err := t.ResolveAt(int32(int64(a[2])), newStr)
	if err != nil {
		return retErr(err)
	}
	return retErr(syscall.Link(oldPath, newPath))
}

func sysMknodat(t *Task, a [6]uint64) uint64 {
	pathStr, err := mem.CopyStringFromGuest(t.CPU, a[1], 4096)
	if err != nil {
		return retErr(err)
	}
	hostPath, err := t.ResolveAt(int32(int64(a[0])), pathStr)
	if err != nil {
		return retErr(err)
	}
	return retErr(syscall.Mknod(hostPath, uint32(a[2]), int(a[3])))
}

// ---- attributes ----------------------------------------------------------

func sysFaccessat(t *Task, a [6]uint64) uint64 {
	pathStr, err := mem.CopyStringFromGuest(t.CPU, a[1], 4096)
	if err != nil {
		return retErr(err)
	}
	hostPath, err := t.ResolveAt(int32(int64(a[0])), pathStr)
	if err != nil {
		return retErr(err)
	}
	// faccessat(2) uses the real ids, as does the host call.
	return retErr(hostFaccessat2(0, hostPath, uint32(a[2]), 0))
}

func sysFtruncate(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	return retErr(syscall.Ftruncate(fd, int64(a[1])))
}

func sysFchmod(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	return retErr(syscall.Fchmod(fd, uint32(a[1])))
}

func sysFchmodat(t *Task, a [6]uint64) uint64 {
	pathStr, err := mem.CopyStringFromGuest(t.CPU, a[1], 4096)
	if err != nil {
		return retErr(err)
	}
	hostPath, err := t.ResolveAt(int32(int64(a[0])), pathStr)
	if err != nil {
		return retErr(err)
	}
	if int(a[3])&AT_SYMLINK_NOFOLLOW != 0 {
		return 0 // no lchmod on Linux: NOP, as the kernel's does
	}
	return retErr(syscall.Chmod(hostPath, uint32(a[2])))
}

func sysFchown(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	return retErr(syscall.Fchown(fd, int(a[1]), int(a[2])))
}

func sysFchownat(t *Task, a [6]uint64) uint64 {
	pathStr, err := mem.CopyStringFromGuest(t.CPU, a[1], 4096)
	if err != nil {
		return retErr(err)
	}
	hostPath, err := t.ResolveAt(int32(int64(a[0])), pathStr)
	if err != nil {
		return retErr(err)
	}
	if int(a[4])&AT_SYMLINK_NOFOLLOW != 0 {
		return retErr(syscall.Lchown(hostPath, int(a[2]), int(a[3])))
	}
	return retErr(syscall.Chown(hostPath, int(a[2]), int(a[3])))
}

// sysUtimensat sets file times. The guest passes an array of two timespecs
// (atime, mtime), with UTIME_NOW and UTIME_OMIT as special nsec values.
func sysUtimensat(t *Task, a [6]uint64) uint64 {
	pathStr, err := mem.CopyStringFromGuest(t.CPU, a[1], 4096)
	if err != nil {
		return retErr(err)
	}
	hostPath, err := t.ResolveAt(int32(int64(a[0])), pathStr)
	if err != nil {
		return retErr(err)
	}
	var ts [2]syscall.Timespec
	if a[2] != 0 {
		buf := make([]byte, 32)
		if err := mem.CopyFromGuest(t.CPU, buf, a[2]); err != nil {
			return retErr(err)
		}
		for i := 0; i < 2; i++ {
			ts[i].Sec = int64(binary.LittleEndian.Uint64(buf[i*16:]))
			ts[i].Nsec = int64(binary.LittleEndian.Uint64(buf[i*16+8:]))
		}
	} else {
		ts[0].Nsec = utimeNow
		ts[1].Nsec = utimeNow
	}
	if ts[0].Nsec == utimeOmit && ts[1].Nsec == utimeOmit {
		return 0
	}
	return retErr(syscall.UtimesNano(hostPath, ts[:]))
}

const (
	utimeNow  = (1 << 30) - 1
	utimeOmit = (1 << 30) - 2
)

func sysFsync(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	return retErr(syscall.Fsync(fd))
}

func sysFdatasync(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	return retErr(syscall.Fdatasync(fd))
}

func sysSync(t *Task, a [6]uint64) uint64 {
	syscall.Sync()
	return 0
}
