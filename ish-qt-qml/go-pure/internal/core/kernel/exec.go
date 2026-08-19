package kernel

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	pathpkg "path"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
	coreloader "github.com/mostafa637/mostafa637/go-pure/internal/core/loader"
)

type LoadedImage struct {
	Image *coreelf.Image
	Space *coreloader.AddressSpace
	Stack coreloader.StackLayout
}

// LoadELF maps an i386 ELF image into the process address space and constructs
// the initial Linux/iSH-compatible user stack. The caller retains ownership of
// the ReaderAt; all mapped bytes are copied into Memory.
func (p *Process) LoadELF(r io.ReaderAt, size int64, filename string, bias uint32, stack coreloader.StackConfig) (*LoadedImage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.loadELF(r, size, filename, bias, stack)
}

func (p *Process) loadELF(r io.ReaderAt, size int64, filename string, bias uint32, stack coreloader.StackConfig) (*LoadedImage, error) {
	if p.closed {
		return nil, ErrClosed
	}
	if r == nil || size <= 0 {
		return nil, fmt.Errorf("kernel: invalid ELF reader")
	}
	image, err := coreelf.Parse(r, size)
	if err != nil {
		return nil, err
	}
	space, err := coreloader.Load(r, size, image, p.Memory, bias)
	if err != nil {
		return nil, err
	}
	if filename != "" {
		stack.ExecFilename = filename
	}
	if stack.ExecFilename == "" {
		stack.ExecFilename = "guest"
	}
	if len(stack.Argv) == 0 {
		stack.Argv = []string{stack.ExecFilename}
	}
	layout, err := coreloader.BuildStackForImage(p.Memory, space, stack)
	if err != nil {
		return nil, err
	}

	p.Context.StartBrk = uint32(space.Brk)
	p.Context.Brk = uint32(space.Brk)
	p.Context.Exited = false
	p.Context.ExitCode = 0
	for index := range p.CPU.Regs {
		p.CPU.Regs[index] = 0
	}
	p.CPU.EIP = uint32(space.Entry)
	p.CPU.Regs[corecpu.ESP] = uint32(layout.SP)
	p.CPU.EFlags = 0
	p.CPU.CF = 0
	p.CPU.OF = 0
	p.CPU.Lazy = 0
	p.CPU.FaultAt = 0
	p.CPU.FaultWrite = false
	p.CPU.TrapNo = 0
	p.CPU.FCW = 0x037f
	p.Executor.Halted = false
	return &LoadedImage{Image: image, Space: space, Stack: layout}, nil
}

// execve replaces the current image while retaining PID, descriptors, fakefs,
// and the process's syscall context. It is called from inside Process.Run, so
// it must not acquire p.mu again.
func (p *Process) execve(filename string, argv, env []string) int32 {
	if p == nil || p.closed || p.FS == nil || p.Context == nil {
		return -14 // EFAULT
	}
	resolved := filename
	if !pathpkg.IsAbs(resolved) {
		cwd := p.Context.CWD
		if cwd == "" {
			cwd = "/"
		}
		resolved = pathpkg.Join(cwd, resolved)
	}
	resolved = pathpkg.Clean(resolved)
	data, err := p.FS.ReadFile(resolved)
	if err != nil {
		return execErrno(err)
	}
	if len(argv) == 0 {
		argv = []string{filename}
	}
	stack := coreloader.DefaultStackConfig()
	stack.Argv = append([]string(nil), argv...)
	stack.Env = append([]string(nil), env...)
	stack.ExecFilename = filename

	// Build the replacement image on a separate address space. A failed execve
	// must leave the old program image and register state untouched.
	oldMemory := p.Memory
	oldCPU := *p.CPU
	oldContext := *p.Context
	oldHalted := p.Executor.Halted
	candidate := corecpu.NewMemory()
	p.Memory = candidate
	p.CPU.Memory = candidate
	p.Context.Memory = candidate
	if _, err := p.loadELF(bytes.NewReader(data), int64(len(data)), filename, 0, stack); err != nil {
		*p.CPU = oldCPU
		*p.Context = oldContext
		p.Memory = oldMemory
		p.Executor.Halted = oldHalted
		candidate.Close()
		return -8 // ENOEXEC
	}
	if oldMemory != nil {
		oldMemory.Close()
	}
	return 0
}

func execErrno(err error) int32 {
	switch {
	case errors.Is(err, corefs.ErrInvalidPath):
		return -22 // EINVAL
	case errors.Is(err, corefs.ErrNotFound), errors.Is(err, fs.ErrNotExist):
		return -2 // ENOENT
	case errors.Is(err, fs.ErrPermission):
		return -13 // EACCES
	default:
		return -5 // EIO
	}
}
