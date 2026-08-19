// Package syscall contains the first Go syscall boundary for the iSH core.
// It follows the i386 register ABI used by calls.c: eax is the number/result
// and ebx..ebp carry up to six arguments.
package syscall

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"runtime"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

type Number uint32

const (
	SysExit       Number = 1
	SysRead       Number = 3
	SysOpen       Number = 5
	SysWrite      Number = 4
	SysClose      Number = 6
	SysLseek      Number = 19
	SysGetPID     Number = 20
	SysDup        Number = 41
	SysDup2       Number = 63
	SysBrk        Number = 45
	SysMmap       Number = 90
	SysMunmap     Number = 91
	SysMprotect   Number = 125
	SysMmap2      Number = 192
	SysExitGroup  Number = 252
	SysMadvise    Number = 219
	SysSchedYield Number = 158
)

var errFault = errors.New("syscall: bad guest address")

const (
	EPERM  int32 = -1
	ENOENT int32 = -2
	EACCES int32 = -13
	EEXIST int32 = -17
	EBADF  int32 = -9
	ENOMEM int32 = -12
	EFAULT int32 = -14
	EINVAL int32 = -22
	EIO    int32 = -5
	ENOSYS int32 = -38
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
	Memory *corecpu.Memory
	FS     *corefs.FS
	PID    uint32

	StartBrk uint32
	Brk      uint32

	// FDs is the authoritative guest descriptor table. Files is retained as a
	// compatibility map for old callers and tests; new code should use FDs.
	FDs   *corefd.Table
	Files map[uint32]*File

	Exited   bool
	ExitCode int32
}

type File = corefd.File

type Handler func(*Context, *corecpu.MachineState, [6]uint32) int32

type Dispatcher struct {
	Context  *Context
	handlers map[Number]Handler
}

func NewContext(memory *corecpu.Memory) *Context {
	return &Context{Memory: memory, FDs: corefd.New(), Files: make(map[uint32]*File)}
}

// InstallFile installs a descriptor in both the new table and the legacy map.
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
	d.Register(SysOpen, open)
	d.Register(SysRead, read)
	d.Register(SysWrite, write)
	d.Register(SysClose, closeFD)
	d.Register(SysLseek, lseek)
	d.Register(SysGetPID, getpid)
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
	flags, ok := hostOpenFlags(args[1])
	if !ok {
		return EINVAL
	}
	file, err := context.FS.OpenFile(path, flags, os.FileMode(args[2]&0o7777), corefs.IshStat{Mode: corefs.ModeRegular | (args[2] & 0o7777), UID: 0, GID: 0})
	if err != nil {
		return errnoForOpen(err)
	}
	fd, err := context.FDs.Open(&corefd.File{Reader: file, Writer: file, Closer: file, Seeker: file})
	if err != nil {
		_ = file.Close()
		return ENOMEM
	}
	context.Files[uint32(fd)] = &corefd.File{Reader: file, Writer: file, Closer: file, Seeker: file}
	return fd
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
	if context.Memory == nil || length == 0 || prot&^(ProtRead|ProtWrite|ProtExec) != 0 {
		return EINVAL
	}
	if flags&MapPrivate != 0 && flags&MapShared != 0 {
		return EINVAL
	}
	if flags&MapAnonymous == 0 {
		return EBADF
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
	memoryFlags := memoryFlags(prot)
	if flags&MapShared != 0 {
		memoryFlags |= corecpu.PShared
	}
	if err := context.Memory.Map(page, pages, memoryFlags); err != nil {
		return ENOMEM
	}
	_ = fd
	_ = offset
	return int32(uint32(page) << corecpu.PageBits)
}

func munmap(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context.Memory == nil || args[0]&(corecpu.PageSize-1) != 0 || args[1] == 0 {
		return EINVAL
	}
	if err := context.Memory.UnmapAlways(corecpu.Page(args[0]>>corecpu.PageBits), pagesFor(args[1])); err != nil {
		return EINVAL
	}
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
