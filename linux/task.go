// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/machine.h, src/syscall.c and src/path.c.
//
// Package linux is the Linux-user layer: the syscall dispatch, the process and
// file-descriptor state it needs, and the rootfs containment that keeps a
// guest path inside the guest's tree.
package linux

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/mostafa637/mostafa637/abi"
	"github.com/mostafa637/mostafa637/core"
	"github.com/mostafa637/mostafa637/jit"
	"github.com/mostafa637/mostafa637/mem"
)

// Task is one guest thread of execution — the CPU, its address space and the
// process-level state the syscall layer keeps (struct Machine in the C core,
// minus the parts that belong to a whole process rather than a thread).
type Task struct {
	CPU *core.CPU
	AS  *mem.AddrSpace

	// Rootfs is the guest root. Every guest path is resolved inside it; see
	// Resolve.
	Rootfs string
	// WorkDir is the guest's current directory ("/" at start).
	WorkDir string

	// PID and TID are the guest-visible identifiers.
	PID, TID int

	// ClearChildTID is set by set_tid_address: the address a thread zeroes and
	// futex-wakes as it exits.
	ClearChildTID uint64
	// RobustList is set by set_robust_list.
	RobustList uint64
	RobustLen  uint64

	// Strace, when non-nil, receives one line per guest syscall.
	Strace io.Writer
	// Warn, when non-nil, receives the one-shot unimplemented-syscall warning.
	Warn io.Writer

	ExitCode int
	Exiting  bool

	// JIT is the compiled-code cache, when the emulator was started with
	// -j/--jit. Nil means the interpreter runs everything.
	JIT *jit.Engine

	sig *sigState
	// trampVA is the hidden page holding the rt_sigreturn trampoline
	// (see signal.go); it is mapped the first time a handler runs.
	trampVA uint64 // signal dispositions (sys_sig.go)

	mu     sync.Mutex
	owned  map[int]bool // descriptors that belong to the emulator, not the guest
	closed map[int]bool
}

// NewTask creates a task: an address space, a CPU bound to it, and an empty
// descriptor table.
func NewTask(rootfs string) *Task {
	as := mem.NewAddrSpace()
	cpu := core.NewCPU()
	mem.Bind(cpu, as)
	t := &Task{
		CPU: cpu, AS: as, Rootfs: rootfs, WorkDir: "/",
		PID: os.Getpid(), TID: os.Getpid(),
		owned:  map[int]bool{},
		closed: map[int]bool{},
	}
	cpu.Thread.Owner = as
	return t
}

// Resolve turns a guest path into a host path.
//
// This is the rootfs containment (src/path.c in the C core). An absolute guest
// path is looked up under the rootfs; a relative one under the guest's current
// directory; "." and ".." are resolved lexically against the guest tree, so a
// guest cannot escape the rootfs by walking up out of it. Symlinks that point
// outside the rootfs are, for now, followed and then re-checked — the full
// path.c pins every component against a re-opened descriptor, which is a later
// file in this port.
func (t *Task) Resolve(guest string) (string, error) {
	if !strings.HasPrefix(guest, "/") {
		guest = path.Join(t.WorkDir, guest)
	}
	clean := path.Clean("/" + guest)
	host := path.Join(t.Rootfs, clean)
	if resolved, err := filepathEval(host); err == nil {
		host = resolved
	}
	if !strings.HasPrefix(host, strings.TrimSuffix(t.Rootfs, "/")+"/") && host != t.Rootfs {
		return "", abi.EACCES
	}
	return host, nil
}

// ResolveAt is Resolve for the *at() family: dirfd is a guest descriptor (or
// AT_FDCWD) and the path may be relative to it.
func (t *Task) ResolveAt(dirfd int32, guest string) (string, error) {
	if !path.IsAbs(guest) && dirfd != ATFDCWD {
		dir, err := t.DirPath(int(dirfd))
		if err != nil {
			return "", err
		}
		guest = path.Join(dir, guest)
	}
	return t.Resolve(guest)
}

// DirPath returns the guest path a directory descriptor refers to, which the
// *at() syscalls need as their base. It is read from /proc/self/fd on the host
// and mapped back into guest terms.
func (t *Task) DirPath(fd int) (string, error) {
	link, err := os.Readlink(hostProcFD(fd))
	if err != nil {
		return "", abi.EBADF
	}
	if !strings.HasPrefix(link, t.Rootfs) {
		return "/", nil
	}
	return path.Clean("/" + strings.TrimPrefix(link, t.Rootfs)), nil
}

func hostProcFD(fd int) string { return "/proc/self/fd/" + itoa(fd) }

// filepathEval resolves symlinks in a host path, so a rootfs component that is
// a symlink (a distro's /lib -> /usr/lib, or a rootfs reached through one)
// still measures as inside it.
func filepathEval(p string) (string, error) { return filepath.EvalSymlinks(p) }

// joinPath applies a possibly-relative path change to a guest current
// directory, the way chdir(2) does.
func joinPath(cwd, p string) string {
	if path.IsAbs(p) {
		return path.Clean(p)
	}
	return path.Clean(path.Join(cwd, p))
}

// Own marks a host descriptor as the emulator's own: the guest must never see
// it, and close() of it fails EBADF.
func (t *Task) Own(fd int) {
	t.mu.Lock()
	t.owned[fd] = true
	t.mu.Unlock()
}

// IsOwn reports whether fd belongs to the emulator.
func (t *Task) IsOwn(fd int) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.owned[fd]
}

// GuestFD reports whether a descriptor number is one the guest may use.
//
// The C core keeps guest fd == host fd and brackets the descriptors it opens
// for itself in a "fd window" so a guest cannot see them; this is the same
// check, from the other side.
func (t *Task) GuestFD(fd int) bool {
	if fd < 0 {
		return false
	}
	return !t.IsOwn(fd)
}

const (
	// ATFDCWD is the AT_* "use the current directory" descriptor value.
	ATFDCWD = -100
	// AT_SYMLINK_NOFOLLOW and friends (the generic Linux values).
	AT_SYMLINK_NOFOLLOW   = 0x100
	AT_REMOVEDIR          = 0x200
	AT_SYMLINK_FOLLOW     = 0x400
	AT_EMPTY_PATH         = 0x1000
	AT_STATX_SYNC_AS_STAT = 0x0000
)

// itoa is the tiny integer formatting the /proc paths need (kept local so the
// hot paths do not reach for strconv).
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [24]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// retErr turns a host error into the negative errno the guest sees. Host and
// guest are both Linux, so the numbers agree; the conversion is only about
// which Go type carries them.
func retErr(err error) uint64 {
	switch e := err.(type) {
	case nil:
		return 0
	case abi.Errno:
		return -uint64(e)
	case syscall.Errno:
		if e == 0 {
			return 0
		}
		return -uint64(e)
	case *os.PathError:
		return retErr(e.Err)
	case *os.LinkError:
		return retErr(e.Err)
	case *os.SyscallError:
		return retErr(e.Err)
	}
	if err == mem.ErrFault {
		return negErrno(abi.EFAULT)
	}
	return negErrno(abi.EINVAL)
}

// retOK returns a successful value.
func retOK(v uint64) uint64 { return v }

// negErrno returns -errno as the guest sees it in x0.
//
// It is written through a variable because negErrno(abi.EBADF) is a constant
// expression, and negating a constant unsigned is a compile error in Go: the
// number the guest is handed is a runtime quantity even though the errno that
// produced it is not.
func negErrno(e abi.Errno) uint64 {
	v := uint64(e)
	if v == 0 {
		return 0
	}
	return -v
}

// isErr reports whether a syscall return value is an -errno: the kernel's
// contract is that a return in [-4095,-1] is an error and anything else,
// however large, is a result.
func isErr(v uint64) bool { return int64(v) < 0 && int64(v) > -4096 }

// warnOnce prints a diagnostic to the task's warning stream, once per message
// per process: a libc that probes a call and falls back is not a failure, and
// repeating the same line a million times is worse than saying it once.
func (t *Task) warnOnce(msg string) {
	if t.Warn == nil {
		return
	}
	if _, dup := warnedOnce.LoadOrStore(msg, true); dup {
		return
	}
	fmt.Fprintf(t.Warn, "arm64chroot: %s\n", msg)
}

// guestToHostID and hostToGuestID are the --fake-id identity remap: the uid and
// gid a guest believes it has versus the host identity it stands for. They are
// the identity today (sys_getuid answers the host's), and every place that
// moves an id between the two worlds — stat, SCM_CREDENTIALS, SO_PEERCRED —
// goes through them, so the remap has one definition when -u lands.
func (t *Task) guestToHostID(uid, gid uint32) (uint32, uint32) { return uid, gid }

func (t *Task) hostToGuestID(uid, gid uint32) (uint32, uint32) { return uid, gid }
