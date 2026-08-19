package kernel

import (
	"fmt"
	"io"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
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
