package syscall

import (
	"crypto/rand"
	"io"
	"os"
	"path"
	"runtime"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

// Number64 contains the native Linux x86-64 syscall numbers. It is kept
// separate from Number, whose values model the i386 ABI used by the existing
// guest implementation.
type Number64 uint64

const (
	Sys64Read          Number64 = 0
	Sys64Write         Number64 = 1
	Sys64Poll          Number64 = 7
	Sys64Select        Number64 = 23
	Sys64Open          Number64 = 2
	Sys64Close         Number64 = 3
	Sys64Stat          Number64 = 4
	Sys64Fstat         Number64 = 5
	Sys64Mmap          Number64 = 9
	Sys64Mprotect      Number64 = 10
	Sys64Munmap        Number64 = 11
	Sys64Brk           Number64 = 12
	Sys64Gettimeofday  Number64 = 96
	Sys64Times         Number64 = 100
	Sys64Lseek         Number64 = 8
	Sys64Ioctl         Number64 = 16
	Sys64RtSigaction   Number64 = 13
	Sys64RtSigprocmask Number64 = 14
	Sys64RtSigreturn   Number64 = 15
	Sys64Readv         Number64 = 19
	Sys64Writev        Number64 = 20
	Sys64SchedYield    Number64 = 24
	Sys64Pselect6      Number64 = 270
	Sys64Ppoll         Number64 = 271
	Sys64Dup           Number64 = 32
	Sys64Dup2          Number64 = 33
	Sys64Nanosleep     Number64 = 35
	Sys64GetPID        Number64 = 39
	Sys64Chown         Number64 = 92
	Sys64Fchown        Number64 = 93
	Sys64Lchown        Number64 = 94
	Sys64Umask         Number64 = 95
	Sys64Fchmod        Number64 = 91

	Sys64GetRlimit Number64 = 97
	Sys64Prctl     Number64 = 157
	Sys64Fchownat  Number64 = 260
	Sys64Fchmodat  Number64 = 268

	Sys64GetRUsage        Number64 = 98
	Sys64GetGroups        Number64 = 115
	Sys64SetGroups        Number64 = 116
	Sys64Socket           Number64 = 41
	Sys64Connect          Number64 = 42
	Sys64Accept           Number64 = 43
	Sys64Sendto           Number64 = 44
	Sys64Recvfrom         Number64 = 45
	Sys64Shutdown         Number64 = 48
	Sys64Bind             Number64 = 49
	Sys64Listen           Number64 = 50
	Sys64Getsockname      Number64 = 51
	Sys64Socketpair       Number64 = 53
	Sys64Clone            Number64 = 56
	Sys64Fork             Number64 = 57
	Sys64Vfork            Number64 = 58
	Sys64Execve           Number64 = 59
	Sys64Exit             Number64 = 60
	Sys64Wait4            Number64 = 61
	Sys64Kill             Number64 = 62
	Sys64Uname            Number64 = 63
	Sys64Fcntl            Number64 = 72
	Sys64GetCWD           Number64 = 79
	Sys64Chdir            Number64 = 80
	Sys64Rename           Number64 = 82
	Sys64Mkdir            Number64 = 83
	Sys64Rmdir            Number64 = 84
	Sys64Unlink           Number64 = 87
	Sys64Readlink         Number64 = 89
	Sys64GetUID           Number64 = 102
	Sys64GetGID           Number64 = 104
	Sys64GetEUID          Number64 = 107
	Sys64GetEGID          Number64 = 108
	Sys64GetPPID          Number64 = 110
	Sys64GetTID           Number64 = 186
	Sys64Tkill            Number64 = 200
	Sys64SchedSetAffinity Number64 = 203
	Sys64SchedGetAffinity Number64 = 204
	Sys64Futex            Number64 = 202
	Sys64SetTIDAddr       Number64 = 218
	Sys64ClockGettime     Number64 = 228
	Sys64ClockGetres      Number64 = 229
	Sys64ExitGroup        Number64 = 231
	Sys64EpollWait        Number64 = 232
	Sys64EpollCtl         Number64 = 233
	Sys64InotifyAdd       Number64 = 254
	Sys64InotifyRm        Number64 = 255
	Sys64Openat           Number64 = 257
	Sys64Fstatat          Number64 = 262
	Sys64Unlinkat         Number64 = 263
	Sys64Renameat         Number64 = 264
	Sys64Linkat           Number64 = 265
	Sys64Symlinkat        Number64 = 266
	Sys64SetRobust        Number64 = 273
	Sys64GetRobust        Number64 = 274
	Sys64Signalfd4        Number64 = 289
	Sys64Eventfd2         Number64 = 290
	Sys64EpollCreate1     Number64 = 291
	Sys64Tgkill           Number64 = 234
	Sys64SetRlimit        Number64 = 160
	Sys64ArchPrctl        Number64 = 158
	Sys64Accept4          Number64 = 288
	Sys64InotifyInit1     Number64 = 294
	Sys64Prlimit64        Number64 = 302
	Sys64Getrandom        Number64 = 318
	Sys64GetCPU           Number64 = 309
	Sys64Statx            Number64 = 332
	Sys64Rseq             Number64 = 334
	Sys64Getdents64       Number64 = 217
	Sys64TimerfdCreate    Number64 = 283
	Sys64TimerfdSettime   Number64 = 286
	Sys64TimerfdGettime   Number64 = 287
)

// Handler64 follows the Linux x86-64 syscall register ABI after the SYSCALL
// instruction has transferred control to the guest runtime.
type Handler64 func(*Context64, [6]uint64) int64

type Context64 struct {
	Memory         *corecpu.Memory64
	FS             *corefs.FS
	CWD            string
	WinCols        uint16
	WinRows        uint16
	Termios        [44]byte
	PID            uint64
	ParentPID      uint64
	TID            uint64
	TIDAddress     uint64
	Children       *ChildRegistry
	RobustListHead uint64
	RobustListLen  uint64
	Brk            uint64
	Futexes        *FutexRegistry64
	RseqAddress    uint64
	RseqLength     uint64
	RseqSignature  uint64
	RLimits        map[uint64]ResourceLimit64
	Groups         []uint32
	SignalMask     uint64
	SignalActions  map[uint64][32]byte
	StartTime      time.Time
	FSBase         uint64
	GSBase         uint64
	CPUIDEnabled   bool
	ProcessName    [16]byte
	Dumpable       bool
	NoNewPrivs     bool
	ParentDeathSig uint64
	AffinityMask   uint64
	Umask          uint32

	// Execve is provided by the guest session. It replaces the current ELF
	// image while preserving process identity and the descriptor table.
	Execve func(path string, argv, env []string) int64

	FDs       *corefd.Table
	Mappings  []GuestMapping64
	signalFDs map[*signalFD64]struct{}
}

type Dispatcher64 struct {
	Context  *Context64
	handlers map[Number64]Handler64
}

func NewContext64(memory *corecpu.Memory64) *Context64 {
	return &Context64{Memory: memory, CWD: "/", WinCols: 80, WinRows: 24, FDs: corefd.New(), Futexes: NewFutexRegistry64(), Children: NewChildRegistry(), RLimits: defaultResourceLimits64(), SignalActions: make(map[uint64][32]byte), StartTime: time.Now(), CPUIDEnabled: true, Dumpable: true, AffinityMask: ^uint64(0), Umask: 0o022, signalFDs: make(map[*signalFD64]struct{})}
}

const maxFD64 = uint64(^uint32(0) >> 1)

// CloseOnExec closes descriptors marked close-on-exec after a successful image replacement.
func (c *Context64) CloseOnExec() {
	if c == nil || c.FDs == nil {
		return
	}
	c.FDs.CloseOnExec()
}

func (c *Context64) InstallFile(fd uint64, file *corefd.File) error {
	if c == nil || c.FDs == nil || file == nil || fd > maxFD64 {
		return corefd.ErrBadFD
	}
	return c.FDs.InstallAt(int32(fd), file, true)
}

func (c *Context64) GetFile(fd uint64) (*corefd.File, error) {
	if c == nil || c.FDs == nil || fd > maxFD64 {
		return nil, corefd.ErrBadFD
	}
	return c.FDs.Get(int32(fd))
}

func NewDispatcher64(context *Context64) *Dispatcher64 {
	d := &Dispatcher64{Context: context, handlers: make(map[Number64]Handler64)}
	d.Register(Sys64Exit, func(ctx *Context64, args [6]uint64) int64 {
		return int64(args[0])
	})
	d.Register(Sys64ExitGroup, func(ctx *Context64, args [6]uint64) int64 {
		return int64(args[0])
	})
	d.Register(Sys64GetPID, func(ctx *Context64, args [6]uint64) int64 {
		return int64(ctx.PID)
	})
	d.Register(Sys64Execve, execve64)
	d.Register(Sys64Clone, clone64Stub)
	d.Register(Sys64Fork, fork64Stub)
	d.Register(Sys64Vfork, vfork64Stub)
	d.Register(Sys64Wait4, wait4_64)
	d.Register(Sys64SetRobust, setRobustList64)
	d.Register(Sys64GetRobust, getRobustList64)
	d.Register(Sys64Gettimeofday, gettimeofday64)
	d.Register(Sys64ClockGettime, clockGettime64)
	d.Register(Sys64ClockGetres, clockGetres64)
	d.Register(Sys64GetRUsage, getrusage64)
	d.Register(Sys64Times, times64)
	d.Register(Sys64GetRlimit, getrlimit64)
	d.Register(Sys64SetRlimit, setrlimit64)
	d.Register(Sys64GetGroups, getgroups64)
	d.Register(Sys64SetGroups, setgroups64)
	d.Register(Sys64RtSigaction, rtSigaction64)
	d.Register(Sys64RtSigprocmask, rtSigprocmask64)
	d.Register(Sys64RtSigreturn, signalStub64)
	d.Register(Sys64Tkill, tkill64)
	d.Register(Sys64Tgkill, tgkill64)
	d.Register(Sys64GetUID, func(ctx *Context64, args [6]uint64) int64 { return 0 })
	d.Register(Sys64GetGID, func(ctx *Context64, args [6]uint64) int64 { return 0 })
	d.Register(Sys64GetEUID, func(ctx *Context64, args [6]uint64) int64 { return 0 })
	d.Register(Sys64GetEGID, func(ctx *Context64, args [6]uint64) int64 { return 0 })
	d.Register(Sys64GetPPID, func(ctx *Context64, args [6]uint64) int64 {
		if ctx == nil {
			return int64(ESRCH)
		}
		return int64(ctx.ParentPID)
	})
	d.Register(Sys64GetTID, gettid64)
	d.Register(Sys64Poll, poll64)
	d.Register(Sys64Select, selectSyscall64)
	d.Register(Sys64Pselect6, pselect6Syscall64)
	d.Register(Sys64Ppoll, ppoll64)

	d.Register(Sys64Fchmod, fchmod64)
	d.Register(Sys64Fchmodat, fchmodat64)
	d.Register(Sys64Chown, chown64)
	d.Register(Sys64Fchown, fchown64)
	d.Register(Sys64Lchown, lchown64)
	d.Register(Sys64Fchownat, fchownat64)
	d.Register(Sys64Umask, umask64)
	d.Register(Sys64SetTIDAddr, setTIDAddress64)
	d.Register(Sys64Fcntl, fcntl64_64)
	d.Register(Sys64ArchPrctl, archPrctl64)
	d.Register(Sys64Prctl, prctl64)
	d.Register(Sys64SchedSetAffinity, schedSetAffinity64)
	d.Register(Sys64SchedGetAffinity, schedGetAffinity64)
	d.Register(Sys64GetCPU, getcpu64)
	d.Register(Sys64Ioctl, ioctl64)
	d.Register(Sys64Nanosleep, nanosleep64)
	d.Register(Sys64Futex, futex64)
	d.Register(Sys64Rseq, rseq64)
	d.Register(Sys64SchedYield, func(ctx *Context64, args [6]uint64) int64 {
		runtime.Gosched()
		return 0
	})
	d.Register(Sys64Read, read64)
	d.Register(Sys64Write, write64)
	d.Register(Sys64Readv, readv64)
	d.Register(Sys64Writev, writev64)
	d.Register(Sys64Close, close64)
	d.Register(Sys64Dup, dup64)
	d.Register(Sys64Dup2, dup264)
	d.Register(Sys64Open, open64)
	d.Register(Sys64Openat, openat64)
	d.Register(Sys64Stat, stat64Guest)
	d.Register(Sys64Fstat, fstat64Guest)
	d.Register(Sys64Fstatat, fstatat64Guest)
	d.Register(Sys64Statx, statx64Guest)
	d.Register(Sys64Unlinkat, unlinkat64Guest)
	d.Register(Sys64Renameat, renameat64Guest)
	d.Register(Sys64Linkat, linkat64Guest)
	d.Register(Sys64Symlinkat, symlinkat64Guest)
	d.Register(Sys64Prlimit64, prlimit64_64)
	d.Register(Sys64Mkdir, mkdir64Guest)
	d.Register(Sys64Rmdir, rmdir64Guest)
	d.Register(Sys64Unlink, unlink64Guest)
	d.Register(Sys64Rename, rename64Guest)
	d.Register(Sys64Getrandom, getrandom64)
	d.Register(Sys64Uname, uname64)
	d.Register(Sys64GetCWD, getcwd64)
	d.Register(Sys64Chdir, chdir64)
	d.Register(Sys64Readlink, readlink64)
	d.Register(Sys64Getdents64, getdents64Guest)
	d.Register(Sys64Brk, brk64)
	d.Register(Sys64Mmap, mmap64)
	d.Register(Sys64Mprotect, mprotect64)
	d.Register(Sys64Munmap, munmap64)
	d.Register(Sys64Lseek, lseek64)
	d.Register(Sys64Socket, socket64)
	d.Register(Sys64Socketpair, socketpair64)
	d.Register(Sys64Bind, bind64)
	d.Register(Sys64Listen, listen64)
	d.Register(Sys64Connect, connect64)
	d.Register(Sys64Accept, accept64)
	d.Register(Sys64Accept4, accept464)
	d.Register(Sys64Getsockname, getsockname64)
	d.Register(Sys64Sendto, sendto64)
	d.Register(Sys64Recvfrom, recvfrom64)
	d.Register(Sys64Shutdown, shutdown64)
	d.Register(Sys64Kill, kill64)
	d.Register(Sys64Signalfd4, signalfd4_64)
	d.Register(Sys64Eventfd2, eventfd2_64)
	d.Register(Sys64TimerfdCreate, timerfdCreate64)
	d.Register(Sys64TimerfdSettime, timerfdSettime64)
	d.Register(Sys64TimerfdGettime, timerfdGettime64)
	d.Register(Sys64EpollCreate1, epollCreate164)
	d.Register(Sys64EpollCtl, epollCtl64)
	d.Register(Sys64EpollWait, epollWait64)
	d.Register(Sys64InotifyInit1, inotifyInit164)
	d.Register(Sys64InotifyAdd, inotifyAddWatch64)
	d.Register(Sys64InotifyRm, inotifyRmWatch64)
	return d

}

func (d *Dispatcher64) Register(number Number64, handler Handler64) {
	if d.handlers == nil {
		d.handlers = make(map[Number64]Handler64)
	}
	d.handlers[number] = handler
}

// Dispatch reads the syscall number and six arguments from the architectural
// registers and writes the signed Linux result back to RAX. The boolean result
// says whether the guest should resume after the syscall; exit/exit_group set
// Halted and return false.
func (d *Dispatcher64) Dispatch(state *corecpu.MachineState64) (bool, error) {
	if d == nil || d.Context == nil || state == nil {
		return false, nil
	}
	number := Number64(state.Get(corecpu.RAX))
	args := [6]uint64{
		state.Get(corecpu.RDI),
		state.Get(corecpu.RSI),
		state.Get(corecpu.RDX),
		state.Get(corecpu.R10),
		state.Get(corecpu.R8),
		state.Get(corecpu.R9),
	}
	d.Context.FSBase = state.FSBase
	d.Context.GSBase = state.GSBase
	handler := d.handlers[number]

	var result int64 = int64(ENOSYS)
	if handler != nil {
		result = handler(d.Context, args)
	}
	state.FSBase = d.Context.FSBase
	state.GSBase = d.Context.GSBase
	state.Set(corecpu.RAX, uint64(result))
	if number == Sys64Exit || number == Sys64ExitGroup {
		state.Halted = true
		return false, nil
	}
	state.TrapNo = 0
	return true, nil
}

func read64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	if args[2] > 1<<20 {
		return int64(EINVAL)
	}
	buffer := make([]byte, int(args[2]))
	n, readErr := file.Read(buffer)
	if n > 0 {
		if err := ctx.Memory.Write(corecpu.Address64(args[1]), buffer[:n]); err != nil {
			return int64(EFAULT)
		}
	}
	if readErr != nil && readErr != io.EOF && n == 0 {
		if readErr == errWouldBlock64 {
			return int64(EAGAIN)
		}
		return int64(EIO)
	}
	return int64(n)
}

func write64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	if args[2] > 1<<20 {
		return int64(EINVAL)
	}
	buffer := make([]byte, int(args[2]))
	if err := ctx.Memory.Read(corecpu.Address64(args[1]), buffer); err != nil {
		return int64(EFAULT)
	}
	n, writeErr := file.Write(buffer)
	if writeErr != nil && n == 0 {
		return int64(EIO)
	}
	return int64(n)
}

func close64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FDs == nil || args[0] > maxFD64 {
		return int64(EBADF)
	}
	if err := ctx.FDs.Close(int32(args[0])); err != nil {
		return int64(EBADF)
	}
	return 0
}

func dup64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FDs == nil || args[0] > maxFD64 {
		return int64(EBADF)
	}
	fd, err := ctx.FDs.Dup(int32(args[0]))
	if err != nil {
		return int64(EBADF)
	}
	return int64(fd)
}

func dup264(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FDs == nil || args[0] > maxFD64 || args[1] > maxFD64 {
		return int64(EBADF)
	}
	fd, err := ctx.FDs.Dup2(int32(args[0]), int32(args[1]))
	if err != nil {
		return int64(EBADF)
	}
	return int64(fd)
}

func readGuestString64(ctx *Context64, address corecpu.Address64, limit int) (string, bool) {
	if ctx == nil || ctx.Memory == nil || limit <= 0 {
		return "", false
	}
	buffer := make([]byte, 0, limit)
	var one [1]byte
	for i := 0; i < limit; i++ {
		if err := ctx.Memory.Read(address+corecpu.Address64(i), one[:]); err != nil {
			return "", false
		}
		if one[0] == 0 {
			return string(buffer), true
		}
		buffer = append(buffer, one[0])
	}
	return "", false
}

const atFDCWD64 uint64 = ^uint64(99)

func openResolvedPath64(ctx *Context64, name string, flags, mode uint64) int64 {
	if ctx != nil && uint32(flags)&guestOpenCreat != 0 {
		mode &= ^uint64(ctx.Umask)
	}
	if ctx == nil || ctx.FS == nil || ctx.FDs == nil {
		return int64(ENOSYS)
	}
	if flags > uint64(^uint32(0)) || mode > uint64(^uint32(0)) {
		return int64(EINVAL)
	}
	hostFlags, ok := hostOpenFlags(uint32(flags))
	if !ok {
		return int64(EINVAL)
	}
	file, err := ctx.FS.OpenFile(name, hostFlags, os.FileMode(uint32(mode)&0o7777), corefs.IshStat{Mode: corefs.ModeRegular | (uint32(mode) & 0o7777), UID: 0, GID: 0})
	if err != nil {
		return int64(errnoForOpen(err))
	}
	info, statErr := ctx.FS.Stat(name)
	if statErr != nil {
		_ = file.Close()
		return int64(errnoForOpen(statErr))
	}
	if uint32(flags)&guestOpenDirectory != 0 && !info.IsDir() {
		_ = file.Close()
		return int64(ENOTDIR)
	}
	guestFile := &corefd.File{Reader: file, Writer: file, Closer: file, Seeker: file, Path: name, Cloexec: uint32(flags)&guestOpenCloexec != 0}
	fd, err := ctx.FDs.Open(guestFile)
	if err != nil {
		_ = file.Close()
		return int64(ENOMEM)
	}
	return int64(fd)
}

func open64(ctx *Context64, args [6]uint64) int64 {
	name, ok := readGuestString64(ctx, corecpu.Address64(args[0]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	if !path.IsAbs(name) {
		name = path.Join(ctx.CWD, name)
	}
	return openResolvedPath64(ctx, name, args[1], args[2])
}

func openat64(ctx *Context64, args [6]uint64) int64 {
	if args[0] != atFDCWD64 {
		return int64(EBADF)
	}
	return open64(ctx, [6]uint64{args[1], args[2], args[3]})
}

func getrandom64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	length := args[1]
	if length > 1<<20 {
		return int64(EINVAL)
	}
	buf := make([]byte, int(length))
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return int64(EIO)
	}
	if err := ctx.Memory.Write(corecpu.Address64(args[0]), buf); err != nil {
		return int64(EFAULT)
	}
	return int64(length)
}

func brk64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(EINVAL)
	}
	requested := args[0]
	if requested != 0 {
		ctx.Brk = requested
	}
	return int64(ctx.Brk)
}
