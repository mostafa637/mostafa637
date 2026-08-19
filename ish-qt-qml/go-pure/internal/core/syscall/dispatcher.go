// Package syscall contains the first Go syscall boundary for the iSH core.
// It follows the i386 register ABI used by calls.c: eax is the number/result
// and ebx..ebp carry up to six arguments.
package syscall

import (
	"encoding/binary"
	"errors"
	"io"
	"runtime"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

type Number uint32

const (
	SysExit          Number = 1
	SysExecve        Number = 11
	SysFork          Number = 2
	SysRead          Number = 3
	SysOpen          Number = 5
	SysWrite         Number = 4
	SysReadv         Number = 145
	SysWritev        Number = 146
	SysPipe          Number = 42
	SysPoll          Number = 168
	SysPipe2         Number = 331
	SysAccess        Number = 33
	SysIoctl         Number = 54
	SysReadlink      Number = 85
	SysUname         Number = 122
	SysChdir         Number = 12
	SysClose         Number = 6
	SysLseek         Number = 19
	SysGetPID        Number = 20
	SysGetUID        Number = 24
	SysGetGID        Number = 47
	SysGetEUID       Number = 49
	SysGetEGID       Number = 50
	SysGetPPID       Number = 64
	SysKill          Number = 37
	SysSignal        Number = 48
	SysWait4         Number = 114
	SysClone         Number = 120
	SysGetCWD        Number = 183
	SysStat64        Number = 195
	SysFstat64       Number = 197
	SysDup           Number = 41
	SysDup2          Number = 63
	SysBrk           Number = 45
	SysMmap          Number = 90
	SysMunmap        Number = 91
	SysMprotect      Number = 125
	SysMmap2         Number = 192
	SysOpenat        Number = 295
	SysFstatat64     Number = 300
	SysExitGroup     Number = 252
	SysMadvise       Number = 219
	SysGetdents64    Number = 220
	SysFcntl64       Number = 221
	SysGetTID        Number = 224
	SysSchedYield    Number = 158
	SysRtSigaction   Number = 174
	SysRtSigprocmask Number = 175
	SysSetTIDAddress Number = 258
	SysSetThreadArea Number = 243
	SysGetThreadArea Number = 244
	SysSetRobustList Number = 311
	SysGetRobustList Number = 312
	SysGettimeofday  Number = 78
	SysNanosleep     Number = 162
	SysGetrlimit     Number = 76
	SysSetrlimit     Number = 75
	SysGetgroups32   Number = 205
	SysSetgroups32   Number = 206
	SysClockGettime  Number = 265
	SysStatfs64      Number = 268
	SysFstatfs64     Number = 269
)

var errFault = errors.New("syscall: bad guest address")

const (
	EPERM        int32 = -1
	ENOENT       int32 = -2
	EACCES       int32 = -13
	EEXIST       int32 = -17
	EBADF        int32 = -9
	EMFILE       int32 = -24
	ENOMEM       int32 = -12
	EFAULT       int32 = -14
	EINVAL       int32 = -22
	EIO          int32 = -5
	ENOTDIR      int32 = -20
	ENAMETOOLONG int32 = -36
	ENOTTY       int32 = -25
	ENOSYS       int32 = -38
	ECHILD       int32 = -10
	ESRCH        int32 = -3
	EINTR        int32 = -4
)

const (
	ProtRead  uint32 = 1
	ProtWrite uint32 = 2
	ProtExec  uint32 = 4

	MapShared    uint32 = 1
	MapPrivate   uint32 = 2
	MapFixed     uint32 = 0x10
	MapAnonymous uint32 = 0x20
)

type Context struct {
	Memory         *corecpu.Memory
	FS             *corefs.FS
	PID            uint32
	ParentPID      uint32
	CWD            string
	TIDAddress     uint32
	Children       *ChildRegistry
	WinCols        uint16
	WinRows        uint16
	TLSBase        uint32
	RobustListHead uint32
	RobustListLen  uint32
	RLimits        map[uint32]ResourceLimit
	Groups         []uint32
	StartTime      time.Time
	Mappings       []GuestMapping

	StartBrk uint32
	Brk      uint32

	// FDs is the authoritative guest descriptor table. Files is retained as a
	// compatibility map for old callers and tests; new code should use FDs.
	FDs   *corefd.Table
	Files map[uint32]*File

	Exited   bool
	ExitCode int32

	// Execve is provided by kernel.Process. It replaces the current image while
	// preserving the process identity and descriptor table.
	Execve func(path string, argv, env []string) int32
}

type File = corefd.File

type Handler func(*Context, *corecpu.MachineState, [6]uint32) int32

type Dispatcher struct {
	Context  *Context
	handlers map[Number]Handler
}

func NewContext(memory *corecpu.Memory) *Context {
	return &Context{
		Memory: memory, CWD: "/", FDs: corefd.New(), Files: make(map[uint32]*File),
		Children: NewChildRegistry(), WinCols: 80, WinRows: 24,
		RLimits: defaultResourceLimits(), Groups: []uint32{0}, StartTime: time.Now(),
	}
}

// InstallFile installs a descriptor in both the new table and the legacy map.
// CloseOnExec closes descriptors marked with O_CLOEXEC after a successful execve.
func (c *Context) CloseOnExec() {
	if c == nil || c.FDs == nil {
		return
	}
	for _, fd := range c.FDs.CloseOnExec() {
		if c.Files != nil {
			delete(c.Files, uint32(fd))
		}
	}
}

func (c *Context) InstallFile(number uint32, file *File) error {
	if c == nil || file == nil || number > uint32(^uint32(0)>>1) {
		return corefd.ErrBadFD
	}
	if c.Files == nil {
		c.Files = make(map[uint32]*File)
	}
	c.Files[number] = file
	if c.FDs == nil {
		c.FDs = corefd.New()
	}
	return c.FDs.InstallAt(int32(number), file, true)
}

func NewDispatcher(context *Context) *Dispatcher {
	d := &Dispatcher{Context: context, handlers: make(map[Number]Handler)}
	d.Register(SysExit, exit)
	d.Register(SysExitGroup, exit)
	d.Register(SysExecve, execve)
	d.Register(SysFork, forkStub)
	d.Register(SysClone, cloneStub)
	d.Register(SysWait4, wait4)
	d.Register(SysKill, kill)
	d.Register(SysSignal, signalStub)
	d.Register(SysRtSigaction, signalStub)
	d.Register(SysRtSigprocmask, signalStub)
	d.Register(SysSetTIDAddress, setTIDAddress)
	d.Register(SysSetThreadArea, setThreadArea)
	d.Register(SysGetThreadArea, getThreadArea)
	d.Register(SysSetRobustList, setRobustList)
	d.Register(SysGetRobustList, getRobustList)
	d.Register(SysGettimeofday, gettimeofday)
	d.Register(SysClockGettime, clockGettime)
	d.Register(SysNanosleep, nanosleep)
	d.Register(SysGetrlimit, getrlimit)
	d.Register(SysSetrlimit, setrlimit)
	d.Register(SysGetgroups32, getgroups32)
	d.Register(SysSetgroups32, setgroups32)
	d.Register(SysStatfs64, statfs64)
	d.Register(SysFstatfs64, fstatfs64)
	d.Register(SysGetTID, gettid)
	d.Register(SysOpen, open)
	d.Register(SysOpenat, openat)
	d.Register(SysAccess, access)
	d.Register(SysIoctl, ioctl)
	d.Register(SysReadlink, readlink)
	d.Register(SysUname, uname)
	d.Register(SysFcntl64, fcntl64)
	d.Register(SysChdir, chdir)
	d.Register(SysGetCWD, getcwd)
	d.Register(SysStat64, stat64)
	d.Register(SysFstat64, fstat64)
	d.Register(SysFstatat64, fstatat64)
	d.Register(SysGetdents64, getdents64)
	d.Register(SysRead, read)
	d.Register(SysWrite, write)
	d.Register(SysReadv, readv)
	d.Register(SysWritev, writev)
	d.Register(SysPipe, pipe)
	d.Register(SysPoll, poll)
	d.Register(SysPipe2, pipe2)
	d.Register(SysClose, closeFD)
	d.Register(SysLseek, lseek)
	d.Register(SysGetPID, getpid)
	d.Register(SysGetUID, getuid)
	d.Register(SysGetGID, getgid)
	d.Register(SysGetEUID, getuid)
	d.Register(SysGetEGID, getgid)
	d.Register(SysGetPPID, getppid)
	d.Register(SysDup, dup)
	d.Register(SysDup2, dup2)
	d.Register(SysBrk, brk)
	d.Register(SysMmap, mmapLegacy)
	d.Register(SysMmap2, mmap2)
	d.Register(SysMunmap, munmap)
	d.Register(SysMprotect, mprotect)
	d.Register(SysMadvise, success)
	d.Register(SysSchedYield, func(*Context, *corecpu.MachineState, [6]uint32) int32 {
		runtime.Gosched()
		return 0
	})
	return d
}

func (d *Dispatcher) Register(number Number, handler Handler) {
	if d.handlers == nil {
		d.handlers = make(map[Number]Handler)
	}
	d.handlers[number] = handler
}

func (d *Dispatcher) Dispatch(state *corecpu.MachineState, number Number, args ...uint32) int32 {
	var callArgs [6]uint32
	copy(callArgs[:], args)
	handler := d.handlers[number]
	var result int32 = ENOSYS
	if handler != nil && d.Context != nil && state != nil {
		result = handler(d.Context, state, callArgs)
	}
	if state != nil {
		state.SetEAX(uint32(result))
	}
	return result
}

func (d *Dispatcher) DispatchState(state *corecpu.MachineState) int32 {
	if state == nil {
		return EFAULT
	}
	return d.Dispatch(state, Number(state.EAXValue()), state.Get(corecpu.EBX), state.Get(corecpu.ECX), state.Get(corecpu.EDX), state.Get(corecpu.ESI), state.Get(corecpu.EDI), state.Get(corecpu.EBP))
}

func exit(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	context.Exited = true
	context.ExitCode = int32(args[0])
	return 0
}

func getpid(context *Context, _ *corecpu.MachineState, _ [6]uint32) int32 {
	return int32(context.PID)
}

func open(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.FS == nil {
		return ENOSYS
	}
	path, ok := readGuestString(context, state, corecpu.Address(args[0]), 4096)
	if !ok {
		return EFAULT
	}
	path, ok = resolveGuestPath(context, path)
	if !ok {
		return ENOENT
	}
	return openResolvedPath(context, path, args[1], args[2])
}

func read(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	file := context.file(args[0])
	if file == nil || file.Reader == nil {
		return EBADF
	}
	length, ok := safeLength(args[2])
	if !ok {
		return EINVAL
	}
	buffer := make([]byte, length)
	n, err := file.Reader.Read(buffer)
	if n > 0 {
		if writeMemory(context, state, corecpu.Address(args[1]), buffer[:n]) != nil {
			return EFAULT
		}
	}
	if err != nil && err != io.EOF && n == 0 {
		return EFAULT
	}
	return int32(n)
}

func write(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	file := context.file(args[0])
	if file == nil || file.Writer == nil {
		return EBADF
	}
	length, ok := safeLength(args[2])
	if !ok {
		return EINVAL
	}
	buffer := make([]byte, length)
	if err := context.Memory.Read(corecpu.Address(args[1]), buffer); err != nil {
		return EFAULT
	}
	n, err := file.Writer.Write(buffer)
	if err != nil && n == 0 {
		return EFAULT
	}
	return int32(n)
}

func mmapLegacy(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context.Memory == nil {
		return ENOMEM
	}
	raw := make([]byte, 6*4)
	if err := context.Memory.Read(corecpu.Address(args[0]), raw); err != nil {
		return EFAULT
	}
	var values [6]uint32
	for i := range values {
		values[i] = binary.LittleEndian.Uint32(raw[i*4:])
	}
	return doMmap(context, values[0], values[1], values[2], values[3], values[4], values[5])
}

func mmap2(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	return doMmap(context, args[0], args[1], args[2], args[3], args[4], args[5]*corecpu.PageSize)
}

func doMmap(context *Context, addr, length, prot, flags, fd, offset uint32) int32 {
	if context == nil || context.Memory == nil || length == 0 || prot&^(ProtRead|ProtWrite|ProtExec) != 0 {
		return EINVAL
	}
	if flags&MapPrivate != 0 && flags&MapShared != 0 {
		return EINVAL
	}
	anonymous := flags&MapAnonymous != 0
	var backing *corefd.File
	var fileSize int64
	if !anonymous {
		if fd == ^uint32(0) || offset&(corecpu.PageSize-1) != 0 {
			return EBADF
		}
		var err int32
		backing, err = getFile(context, fd)
		if err != 0 || backing == nil || backing.Path == "" || context.FS == nil {
			return EBADF
		}
	}
	pages := pagesFor(length)
	page := corecpu.Page(addr >> corecpu.PageBits)
	if addr != 0 {
		if addr&(corecpu.PageSize-1) != 0 {
			return EINVAL
		}
		if flags&MapFixed == 0 && !context.Memory.IsHole(page, pages) {
			page = context.Memory.FindHole(pages)
		}
	} else {
		page = context.Memory.FindHole(pages)
	}
	if page == corecpu.BadPage {
		return ENOMEM
	}
	finalFlags := memoryFlags(prot)
	mapFlags := finalFlags | corecpu.PWrite
	if anonymous {
		mapFlags |= corecpu.PAnonymous
	}
	if flags&MapShared != 0 {
		mapFlags |= corecpu.PShared
	}
	if err := context.Memory.Map(page, pages, mapFlags); err != nil {
		return ENOMEM
	}
	base := uint32(page) << corecpu.PageBits
	if backing != nil {
		info, err := context.FS.Stat(backing.Path)
		if err != nil {
			_ = context.Memory.UnmapAlways(page, pages)
			return errnoForOpen(err)
		}
		fileSize = info.Size
		available := uint64(0)
		if info.Size > int64(offset) {
			available = uint64(info.Size - int64(offset))
		}
		if available > uint64(length) {
			available = uint64(length)
		}
		if available > 0 {
			buffer := make([]byte, int(available))
			n, readErr := context.FS.ReadAt(backing.Path, buffer, int64(offset))
			if n > 0 {
				if err := context.Memory.Write(corecpu.Address(base), buffer[:n]); err != nil {
					_ = context.Memory.UnmapAlways(page, pages)
					return EFAULT
				}
			}
			if readErr != nil && n == 0 {
				_ = context.Memory.UnmapAlways(page, pages)
				return EIO
			}
		}
	}
	protectedFlags := finalFlags
	if flags&MapShared != 0 {
		protectedFlags |= corecpu.PShared
	}
	if err := context.Memory.SetFlags(page, pages, protectedFlags); err != nil {
		_ = context.Memory.UnmapAlways(page, pages)
		return ENOMEM
	}
	if backing != nil {
		context.addMapping(GuestMapping{
			Base: base, Length: uint32(pages) * corecpu.PageSize, Pages: pages,
			Path: backing.Path, Offset: uint64(offset), FileSize: fileSize,
			Prot: prot, Shared: flags&MapShared != 0,
		})
	}
	return int32(base)
}

func munmap(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil || args[0]&(corecpu.PageSize-1) != 0 || args[1] == 0 {
		return EINVAL
	}
	if result := context.flushMappings(args[0], args[1]); result != 0 {
		return result
	}
	pages := pagesFor(args[1])
	if err := context.Memory.UnmapAlways(corecpu.Page(args[0]>>corecpu.PageBits), pages); err != nil {
		return EINVAL
	}
	context.removeMappings(args[0], uint32(pages)*corecpu.PageSize)
	return 0
}

func mprotect(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context.Memory == nil || args[0]&(corecpu.PageSize-1) != 0 || args[2]&^(ProtRead|ProtWrite|ProtExec) != 0 {
		return EINVAL
	}
	if err := context.Memory.SetFlags(corecpu.Page(args[0]>>corecpu.PageBits), pagesFor(args[1]), memoryFlags(args[2])); err != nil {
		return ENOMEM
	}
	return 0
}

func brk(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context.Memory == nil {
		return 0
	}
	newBrk := args[0]
	if newBrk < context.StartBrk {
		return int32(context.Brk)
	}
	oldBrk := context.Brk
	if newBrk > oldBrk {
		start := pageRoundUp(oldBrk)
		end := pageRoundUp(newBrk)
		if end > start {
			pages := corecpu.Pages(end - start)
			if !context.Memory.IsHole(start, pages) || context.Memory.Map(start, pages, corecpu.PWrite) != nil {
				return int32(context.Brk)
			}
		}
	} else if newBrk < oldBrk {
		start := corecpu.Page(newBrk >> corecpu.PageBits)
		end := corecpu.Page(oldBrk >> corecpu.PageBits)
		if end > start {
			_ = context.Memory.UnmapAlways(start, corecpu.Pages(end-start))
		}
	}
	context.Brk = newBrk
	return int32(context.Brk)
}

func success(*Context, *corecpu.MachineState, [6]uint32) int32 { return 0 }

func (c *Context) file(number uint32) *File {
	if c == nil {
		return nil
	}
	if c.FDs != nil {
		if file, err := c.FDs.Get(int32(number)); err == nil {
			return file
		}
	}
	return c.Files[number]
}

func closeFD(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil {
		return EBADF
	}
	if context.FDs != nil {
		if err := context.FDs.Close(int32(args[0])); err == nil {
			delete(context.Files, args[0])
			return 0
		}
	}
	if context.Files[args[0]] == nil {
		return EBADF
	}
	delete(context.Files, args[0])
	return 0
}

func dup(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.FDs == nil {
		return EBADF
	}
	newFD, err := context.FDs.Dup(int32(args[0]))
	if err != nil {
		return EBADF
	}
	if file, err := context.FDs.Get(newFD); err == nil {
		context.Files[uint32(newFD)] = file
	}
	return newFD
}

func dup2(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.FDs == nil {
		return EBADF
	}
	newFD, err := context.FDs.Dup2(int32(args[0]), int32(args[1]))
	if err != nil {
		return EBADF
	}
	if file, err := context.FDs.Get(newFD); err == nil {
		context.Files[uint32(newFD)] = file
	}
	return newFD
}

func lseek(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	file := context.file(args[0])
	if file == nil || file.Seeker == nil {
		return EBADF
	}
	position, err := file.Seek(int64(int32(args[1])), int(args[2]))
	if err != nil || position < 0 || position > int64(^uint32(0)) {
		return EINVAL
	}
	return int32(position)
}

func writeMemory(context *Context, state *corecpu.MachineState, addr corecpu.Address, data []byte) error {
	if context.Memory == nil || state == nil {
		return errFault
	}
	return context.Memory.Write(addr, data)
}

func memoryFlags(prot uint32) corecpu.Flags {
	var flags corecpu.Flags
	if prot&ProtRead != 0 {
		flags |= corecpu.PRead
	}
	if prot&ProtWrite != 0 {
		flags |= corecpu.PWrite
	}
	if prot&ProtExec != 0 {
		flags |= corecpu.PExec
	}
	return flags
}

func pagesFor(length uint32) corecpu.Pages {
	return corecpu.Pages((length + corecpu.PageSize - 1) / corecpu.PageSize)
}

func pageRoundUp(value uint32) corecpu.Page {
	return corecpu.Page((value + corecpu.PageSize - 1) / corecpu.PageSize)
}

func safeLength(value uint32) (int, bool) {
	maxInt := int(^uint(0) >> 1)
	if uint64(value) > uint64(maxInt) {
		return 0, false
	}
	return int(value), true
}
