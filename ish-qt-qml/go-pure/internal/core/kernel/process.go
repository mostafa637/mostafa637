// Package kernel owns the Go process boundary above CPU, memory and syscall.
// It is intentionally small: instruction execution and ELF loading are later
// layers, while this package already provides a stable process object for them.
package kernel

import (
	"io"
	"sync"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
	coresyscall "github.com/mostafa637/mostafa637/go-pure/internal/core/syscall"
)

type Process struct {
	mu sync.Mutex

	PID    uint32
	Memory *corecpu.Memory
	CPU    *corecpu.MachineState
	FS     *corefs.FS

	Context  *coresyscall.Context
	Syscalls *coresyscall.Dispatcher
	closed   bool
}

func NewProcess(pid uint32, fake *corefs.FS) *Process {
	memory := corecpu.NewMemory()
	state := corecpu.NewMachineState(memory)
	context := coresyscall.NewContext(memory)
	context.PID = pid
	return &Process{
		PID:      pid,
		Memory:   memory,
		CPU:      state,
		FS:       fake,
		Context:  context,
		Syscalls: coresyscall.NewDispatcher(context),
	}
}

func (p *Process) AttachFile(fd uint32, reader io.Reader, writer io.Writer) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	p.Context.Files[fd] = &coresyscall.File{Reader: reader, Writer: writer}
	return nil
}

func (p *Process) Syscall(number coresyscall.Number, args ...uint32) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return coresyscall.EFAULT
	}
	return p.Syscalls.Dispatch(p.CPU, number, args...)
}

func (p *Process) SyscallFromRegisters() int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return coresyscall.EFAULT
	}
	return p.Syscalls.DispatchState(p.CPU)
}

func (p *Process) ExitCode() (int32, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Context.ExitCode, p.Context.Exited
}

func (p *Process) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	memory := p.Memory
	p.mu.Unlock()
	memory.Close()
	return nil
}

var ErrClosed = errorString("kernel process is closed")

type errorString string

func (e errorString) Error() string { return string(e) }
