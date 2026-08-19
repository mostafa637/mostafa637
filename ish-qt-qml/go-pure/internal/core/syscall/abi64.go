package syscall

import (
	"crypto/rand"
	"io"
	"runtime"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

// Number64 contains the native Linux x86-64 syscall numbers. It is kept
// separate from Number, whose values model the i386 ABI used by the existing
// guest implementation.
type Number64 uint64

const (
	Sys64Read         Number64 = 0
	Sys64Write        Number64 = 1
	Sys64Open         Number64 = 2
	Sys64Close        Number64 = 3
	Sys64Mmap         Number64 = 9
	Sys64Mprotect     Number64 = 10
	Sys64Munmap       Number64 = 11
	Sys64Brk          Number64 = 12
	Sys64Ioctl        Number64 = 16
	Sys64Readv        Number64 = 19
	Sys64Writev       Number64 = 20
	Sys64SchedYield   Number64 = 24
	Sys64Dup          Number64 = 32
	Sys64Dup2         Number64 = 33
	Sys64Nanosleep    Number64 = 35
	Sys64GetPID       Number64 = 39
	Sys64Socket       Number64 = 41
	Sys64Connect      Number64 = 42
	Sys64Accept       Number64 = 43
	Sys64Clone        Number64 = 56
	Sys64Fork         Number64 = 57
	Sys64Vfork        Number64 = 58
	Sys64Execve       Number64 = 59
	Sys64Exit         Number64 = 60
	Sys64Wait4        Number64 = 61
	Sys64Kill         Number64 = 62
	Sys64Uname        Number64 = 63
	Sys64Fcntl        Number64 = 72
	Sys64GetCWD       Number64 = 79
	Sys64Chdir        Number64 = 80
	Sys64Rename       Number64 = 82
	Sys64Mkdir        Number64 = 83
	Sys64Rmdir        Number64 = 84
	Sys64Unlink       Number64 = 87
	Sys64Readlink     Number64 = 89
	Sys64GetUID       Number64 = 102
	Sys64GetGID       Number64 = 104
	Sys64GetEUID      Number64 = 107
	Sys64GetEGID      Number64 = 108
	Sys64GetPPID      Number64 = 110
	Sys64Futex        Number64 = 202
	Sys64SetTIDAddr   Number64 = 218
	Sys64ExitGroup    Number64 = 231
	Sys64EpollWait    Number64 = 232
	Sys64EpollCtl     Number64 = 233
	Sys64InotifyAdd   Number64 = 254
	Sys64InotifyRm    Number64 = 255
	Sys64Openat       Number64 = 257
	Sys64Fstatat      Number64 = 262
	Sys64Unlinkat     Number64 = 263
	Sys64Renameat     Number64 = 264
	Sys64Linkat       Number64 = 265
	Sys64Symlinkat    Number64 = 266
	Sys64SetRobust    Number64 = 273
	Sys64Signalfd4    Number64 = 289
	Sys64Eventfd2     Number64 = 290
	Sys64EpollCreate1 Number64 = 291
	Sys64InotifyInit1 Number64 = 294
	Sys64Prlimit64    Number64 = 302
	Sys64Getrandom    Number64 = 318
	Sys64Rseq         Number64 = 334
)

// Handler64 follows the Linux x86-64 syscall register ABI after the SYSCALL
// instruction has transferred control to the guest runtime.
type Handler64 func(*Context64, [6]uint64) int64

type Context64 struct {
	Memory *corecpu.Memory64
	FS     *corefs.FS
	CWD    string
	PID    uint64
	TID    uint64
	Brk    uint64
	FDs    *corefd.Table
}

type Dispatcher64 struct {
	Context  *Context64
	handlers map[Number64]Handler64
}

func NewContext64(memory *corecpu.Memory64) *Context64 {
	return &Context64{Memory: memory, CWD: "/", FDs: corefd.New()}
}

const maxFD64 = uint64(^uint32(0) >> 1)

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
	d.Register(Sys64GetUID, func(ctx *Context64, args [6]uint64) int64 { return 0 })
	d.Register(Sys64GetGID, func(ctx *Context64, args [6]uint64) int64 { return 0 })
	d.Register(Sys64GetEUID, func(ctx *Context64, args [6]uint64) int64 { return 0 })
	d.Register(Sys64GetEGID, func(ctx *Context64, args [6]uint64) int64 { return 0 })
	d.Register(Sys64GetPPID, func(ctx *Context64, args [6]uint64) int64 { return 0 })
	d.Register(Sys64SchedYield, func(ctx *Context64, args [6]uint64) int64 {
		runtime.Gosched()
		return 0
	})
	d.Register(Sys64Read, read64)
	d.Register(Sys64Write, write64)
	d.Register(Sys64Close, close64)
	d.Register(Sys64Dup, dup64)
	d.Register(Sys64Dup2, dup264)
	d.Register(Sys64Getrandom, getrandom64)
	d.Register(Sys64Brk, brk64)
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
	handler := d.handlers[number]
	var result int64 = int64(ENOSYS)
	if handler != nil {
		result = handler(d.Context, args)
	}
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
