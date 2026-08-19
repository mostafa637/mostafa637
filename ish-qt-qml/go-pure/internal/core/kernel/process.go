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
	Executor *corecpu.Executor
	closed   bool
}

func NewProcess(pid uint32, fake *corefs.FS) *Process {
	memory := corecpu.NewMemory()
	state := corecpu.NewMachineState(memory)
	context := coresyscall.NewContext(memory)
	context.FS = fake
	context.PID = pid
	dispatcher := coresyscall.NewDispatcher(context)
	var executor *corecpu.Executor
	executor = corecpu.NewExecutor(func(machine *corecpu.MachineState) int32 {
		result := dispatcher.DispatchState(machine)
		if context.Exited {
			executor.Halted = true
		}
		return result
	})
	process := &Process{
		PID:      pid,
		Memory:   memory,
		CPU:      state,
		FS:       fake,
		Context:  context,
		Syscalls: dispatcher,
		Executor: executor,
	}
	context.Execve = process.execve
	return process
}

func (p *Process) AttachFile(fd uint32, reader io.Reader, writer io.Writer) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	return p.Context.InstallFile(fd, &coresyscall.File{Reader: reader, Writer: writer})
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

func (p *Process) Step() (corecpu.Instruction, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return corecpu.Instruction{}, ErrClosed
	}
	return p.Executor.Step(p.CPU)
}

func (p *Process) Run(maxSteps int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	return p.Executor.Run(p.CPU, maxSteps)
}

func (p *Process) ExitCode() (int32, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Context.ExitCode, p.Context.Exited
}

// SetExitStatus records termination from an architecture-specific guest runner.
// The i386 dispatcher writes the same fields internally; this method keeps the
// x86-64 JIT lifecycle from reaching into the process context without locking.
func (p *Process) SetExitStatus(code int32) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Context != nil {
		p.Context.ExitCode = code
		p.Context.Exited = true
	}
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

func (p *Process) SetWindowSize(cols, rows uint16) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	if cols == 0 || rows == 0 {
		return nil
	}
	p.Context.WinCols = cols
	p.Context.WinRows = rows
	return nil
}
