// Package syscall contains the first Go syscall boundary for the iSH core.
// It follows the i386 register ABI used by calls.c: eax is the number/result
// and ebx..ebp carry up to six arguments.
package syscall

import (
	"encoding/binary"
	"errors"
	"io"
	"runtime"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

type Number uint32

const (
	SysExit       Number = 1
	SysRead       Number = 3
	SysWrite      Number = 4
	SysGetPID     Number = 20
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
	EBADF  int32 = -9
	ENOMEM int32 = -12
	EFAULT int32 = -14
	EINVAL int32 = -22
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
	Memory *cpu.Memory
	PID    uint32

	StartBrk uint32
	Brk      uint32

	Files map[uint32]*File

	Exited   bool
	ExitCode int32
}

type File struct {
	Reader io.Reader
	Writer io.Writer
}

type Handler func(*Context, *cpu.MachineState, [6]uint32) int32

type Dispatcher struct {
	Context  *Context
	handlers map[Number]Handler
}

func NewContext(memory *cpu.Memory) *Context {
	return &Context{Memory: memory, Files: make(map[uint32]*File)}
}

func NewDispatcher(context *Context) *Dispatcher {
	d := &Dispatcher{Context: context, handlers: make(map[Number]Handler)}
	d.Register(SysExit, exit)
	d.Register(SysExitGroup, exit)
	d.Register(SysRead, read)
	d.Register(SysWrite, write)
	d.Register(SysGetPID, getpid)
	d.Register(SysBrk, brk)
	d.Register(SysMmap, mmapLegacy)
	d.Register(SysMmap2, mmap2)
	d.Register(SysMunmap, munmap)
	d.Register(SysMprotect, mprotect)
	d.Register(SysMadvise, success)
	d.Register(SysSchedYield, func(*Context, *cpu.MachineState, [6]uint32) int32 {
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

func (d *Dispatcher) Dispatch(state *cpu.MachineState, number Number, args ...uint32) int32 {
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

func (d *Dispatcher) DispatchState(state *cpu.MachineState) int32 {
	if state == nil {
		return EFAULT
	}
	return d.Dispatch(state, Number(state.EAXValue()), state.Get(cpu.EBX), state.Get(cpu.ECX), state.Get(cpu.EDX), state.Get(cpu.ESI), state.Get(cpu.EDI), state.Get(cpu.EBP))
}

func exit(context *Context, _ *cpu.MachineState, args [6]uint32) int32 {
	context.Exited = true
	context.ExitCode = int32(args[0])
	return 0
}

func getpid(context *Context, _ *cpu.MachineState, _ [6]uint32) int32 {
	return int32(context.PID)
}

func read(context *Context, state *cpu.MachineState, args [6]uint32) int32 {
	file := context.Files[args[0]]
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
		if writeMemory(context, state, cpu.Address(args[1]), buffer[:n]) != nil {
			return EFAULT
		}
	}
	if err != nil && err != io.EOF && n == 0 {
		return EFAULT
	}
	return int32(n)
}

func write(context *Context, state *cpu.MachineState, args [6]uint32) int32 {
	file := context.Files[args[0]]
	if file == nil || file.Writer == nil {
		return EBADF
	}
	length, ok := safeLength(args[2])
	if !ok {
		return EINVAL
	}
	buffer := make([]byte, length)
	if err := context.Memory.Read(cpu.Address(args[1]), buffer); err != nil {
		return EFAULT
	}
	n, err := file.Writer.Write(buffer)
	if err != nil && n == 0 {
		return EFAULT
	}
	return int32(n)
}

func mmapLegacy(context *Context, _ *cpu.MachineState, args [6]uint32) int32 {
	if context.Memory == nil {
		return ENOMEM
	}
	raw := make([]byte, 6*4)
	if err := context.Memory.Read(cpu.Address(args[0]), raw); err != nil {
		return EFAULT
	}
	var values [6]uint32
	for i := range values {
		values[i] = binary.LittleEndian.Uint32(raw[i*4:])
	}
	return doMmap(context, values[0], values[1], values[2], values[3], values[4], values[5])
}

func mmap2(context *Context, _ *cpu.MachineState, args [6]uint32) int32 {
	return doMmap(context, args[0], args[1], args[2], args[3], args[4], args[5]*cpu.PageSize)
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
	page := cpu.Page(addr >> cpu.PageBits)
	if addr != 0 {
		if addr&(cpu.PageSize-1) != 0 {
			return EINVAL
		}
		if flags&MapFixed == 0 && !context.Memory.IsHole(page, pages) {
			page = context.Memory.FindHole(pages)
		}
	} else {
		page = context.Memory.FindHole(pages)
	}
	if page == cpu.BadPage {
		return ENOMEM
	}
	memoryFlags := memoryFlags(prot)
	if flags&MapShared != 0 {
		memoryFlags |= cpu.PShared
	}
	if err := context.Memory.Map(page, pages, memoryFlags); err != nil {
		return ENOMEM
	}
	_ = fd
	_ = offset
	return int32(uint32(page) << cpu.PageBits)
}

func munmap(context *Context, _ *cpu.MachineState, args [6]uint32) int32 {
	if context.Memory == nil || args[0]&(cpu.PageSize-1) != 0 || args[1] == 0 {
		return EINVAL
	}
	if err := context.Memory.UnmapAlways(cpu.Page(args[0]>>cpu.PageBits), pagesFor(args[1])); err != nil {
		return EINVAL
	}
	return 0
}

func mprotect(context *Context, _ *cpu.MachineState, args [6]uint32) int32 {
	if context.Memory == nil || args[0]&(cpu.PageSize-1) != 0 || args[2]&^(ProtRead|ProtWrite|ProtExec) != 0 {
		return EINVAL
	}
	if err := context.Memory.SetFlags(cpu.Page(args[0]>>cpu.PageBits), pagesFor(args[1]), memoryFlags(args[2])); err != nil {
		return ENOMEM
	}
	return 0
}

func brk(context *Context, _ *cpu.MachineState, args [6]uint32) int32 {
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
			pages := cpu.Pages(end - start)
			if !context.Memory.IsHole(start, pages) || context.Memory.Map(start, pages, cpu.PWrite) != nil {
				return int32(context.Brk)
			}
		}
	} else if newBrk < oldBrk {
		start := cpu.Page(newBrk >> cpu.PageBits)
		end := cpu.Page(oldBrk >> cpu.PageBits)
		if end > start {
			_ = context.Memory.UnmapAlways(start, cpu.Pages(end-start))
		}
	}
	context.Brk = newBrk
	return int32(context.Brk)
}

func success(*Context, *cpu.MachineState, [6]uint32) int32 { return 0 }

func writeMemory(context *Context, state *cpu.MachineState, addr cpu.Address, data []byte) error {
	if context.Memory == nil || state == nil {
		return errFault
	}
	return context.Memory.Write(addr, data)
}

func memoryFlags(prot uint32) cpu.Flags {
	var flags cpu.Flags
	if prot&ProtRead != 0 {
		flags |= cpu.PRead
	}
	if prot&ProtWrite != 0 {
		flags |= cpu.PWrite
	}
	if prot&ProtExec != 0 {
		flags |= cpu.PExec
	}
	return flags
}

func pagesFor(length uint32) cpu.Pages {
	return cpu.Pages((length + cpu.PageSize - 1) / cpu.PageSize)
}

func pageRoundUp(value uint32) cpu.Page {
	return cpu.Page((value + cpu.PageSize - 1) / cpu.PageSize)
}

func safeLength(value uint32) (int, bool) {
	maxInt := int(^uint(0) >> 1)
	if uint64(value) > uint64(maxInt) {
		return 0, false
	}
	return int(value), true
}
