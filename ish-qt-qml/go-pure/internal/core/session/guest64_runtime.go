package session

import (
	"debug/elf"
	"encoding/binary"
	"sync"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
	coreloader "github.com/mostafa637/mostafa637/go-pure/internal/core/loader"
	coresyscall "github.com/mostafa637/mostafa637/go-pure/internal/core/syscall"
)

type guest64Runtime struct {
	transport  *guestTransport
	context    *coresyscall.Context64
	state      *corecpu.MachineState64
	dispatcher *coresyscall.Dispatcher64
	jit        *corecpu.JIT64
	parent     *coresyscall.Context64
	pid        uint64
	done       chan struct{}
	once       sync.Once
}

func (g *guestTransport) nextChildPID(parent *coresyscall.Context64) uint64 {
	g.runtimeMu.Lock()
	defer g.runtimeMu.Unlock()
	if g.nextPID <= parent.PID {
		g.nextPID = parent.PID + 1
	}
	for {
		candidate := g.nextPID
		g.nextPID++
		used := false
		for _, runtime := range g.runtimes {
			if runtime != nil && runtime.pid == candidate {
				used = true
				break
			}
		}
		if !used {
			return candidate
		}
	}
}

func (g *guestTransport) installRuntime(runtime *guest64Runtime) {
	if g == nil || runtime == nil || runtime.context == nil {
		return
	}
	g.runtimeMu.Lock()
	if g.runtimes == nil {
		g.runtimes = make(map[*coresyscall.Context64]*guest64Runtime)
	}
	g.runtimes[runtime.context] = runtime
	g.runtimeMu.Unlock()
}

func (g *guestTransport) runtimeForPID(pid uint64) *guest64Runtime {
	if g == nil {
		return nil
	}
	g.runtimeMu.Lock()
	defer g.runtimeMu.Unlock()
	for _, runtime := range g.runtimes {
		if runtime != nil && runtime.pid == pid {
			return runtime
		}
	}
	return nil
}

func (g *guestTransport) createChild64(parent *coresyscall.Context64, request coresyscall.CloneRequest64) int64 {
	if g == nil || parent == nil || parent.Memory == nil || parent.Machine == nil {
		return int64(coresyscall.EFAULT)
	}
	pid := g.nextChildPID(parent)
	memory := parent.Memory.Clone()
	if request.Flags&coresyscall.CloneVMFlag64 != 0 { // CLONE_VM: threads share the address space.
		memory = parent.Memory
	}
	state := parent.Machine.Clone(memory)
	if state == nil {
		return int64(coresyscall.ENOMEM)
	}
	state.Set(corecpu.RAX, 0)
	state.Halted = false
	state.TrapNo = corecpu.Trap64None
	if request.ChildStack != 0 {
		state.Set(corecpu.RSP, request.ChildStack)
	}
	if request.Flags&coresyscall.CloneSetTLSFlag64 != 0 { // CLONE_SETTLS
		state.FSBase = request.TLS
	}
	child := parent.CloneForChild64(memory, pid)
	if child == nil {
		return int64(coresyscall.ENOMEM)
	}
	if request.Flags&coresyscall.CloneFilesFlag64 != 0 { // CLONE_FILES
		child.FDs = parent.FDs
	}
	child.Machine = state
	child.ProcessFactory = g.createChild64
	child.ChildStarter = g.startChild64
	if request.Flags&coresyscall.CloneSetTLSFlag64 != 0 {
		child.FSBase = request.TLS
	}
	runtime := &guest64Runtime{
		transport: g,
		context:   child,
		state:     state,
		parent:    parent,
		pid:       pid,
		done:      make(chan struct{}),
	}
	runtime.dispatcher = coresyscall.NewDispatcher64(child)
	runtime.jit = corecpu.NewJIT64(memory)
	runtime.jit.OnSyscall64 = func(machine *corecpu.MachineState64) (bool, error) {
		return runtime.dispatcher.Dispatch(machine)
	}
	child.Execve = func(path string, argv, env []string) int64 {
		return runtime.execve64(path, argv, env)
	}
	g.installRuntime(runtime)
	return int64(pid)
}

func (g *guestTransport) startChild64(parent *coresyscall.Context64, childPID int64, request coresyscall.CloneRequest64) {
	if childPID <= 0 {
		return
	}
	runtime := g.runtimeForPID(uint64(childPID))
	if runtime == nil || runtime.context == nil {
		return
	}
	if request.Flags&coresyscall.CloneChildTIDFlag64 != 0 { // CLONE_CHILD_SETTID, in child memory.
		var raw [8]byte
		binary.LittleEndian.PutUint64(raw[:], uint64(childPID))
		if runtime.context.Memory == nil || runtime.context.Memory.Write(corecpu.Address64(request.ChildTID), raw[:]) != nil {
			runtime.context.ExitCode = 128 + int32(coresyscall.EFAULT)
			runtime.context.Exited = true
			if parent != nil && parent.Children != nil {
				parent.Children.MarkExited(uint32(childPID), runtime.context.ExitCode)
			}
			return
		}
	}
	go runtime.run()
}

func (runtime *guest64Runtime) run() {
	if runtime == nil || runtime.jit == nil || runtime.state == nil || runtime.context == nil {
		return
	}
	defer runtime.once.Do(func() { close(runtime.done) })
	for {
		trap := runtime.jit.RunToInterrupt(runtime.state)
		switch trap {
		case corecpu.Trap64Timer:
			continue
		case corecpu.Trap64Exit:
			if !runtime.context.Exited {
				runtime.context.ExitCode = int32(runtime.state.Get(corecpu.RAX))
				runtime.context.Exited = true
			}
			runtime.publishExit()
			return
		default:
			runtime.context.ExitCode = 128 + int32(trap)
			runtime.context.Exited = true
			runtime.publishExit()
			return
		}
	}
}

func (runtime *guest64Runtime) publishExit() {
	if runtime == nil || runtime.parent == nil || runtime.parent.Children == nil {
		return
	}
	runtime.parent.Children.MarkExited(uint32(runtime.pid), runtime.context.ExitCode)
}

func (g *guestTransport) pokeRuntimes() {
	if g == nil {
		return
	}
	g.runtimeMu.Lock()
	defer g.runtimeMu.Unlock()
	for _, runtime := range g.runtimes {
		if runtime != nil && runtime.jit != nil {
			runtime.jit.Poke()
		}
	}
}

func (runtime *guest64Runtime) execve64(path string, argv, env []string) int64 {
	if runtime == nil || runtime.context == nil || runtime.context.FS == nil || runtime.state == nil {
		return int64(coresyscall.EFAULT)
	}
	resolved, ok := resolveSessionPath64(runtime.context.CWD, path)
	if !ok || resolved == "" {
		return int64(coresyscall.ENOENT)
	}
	imageData, readErr := runtime.context.FS.ReadFile(resolved)
	if readErr != nil {
		return int64(coresyscall.ENOENT)
	}
	image, parseErr := coreelf.Parse64(bytesReader(imageData), int64(len(imageData)))
	if parseErr != nil {
		return int64(coresyscall.EINVAL)
	}
	newMemory := corecpu.NewMemory64()
	var bias corecpu.Address64
	if image.Header.Type == elf.ET_DYN {
		bias = corecpu.Address64(0x0000000000400000)
	}
	space, loadErr := coreloader.Load64(bytesReader(imageData), int64(len(imageData)), image, newMemory, bias)
	if loadErr != nil {
		return int64(coresyscall.EINVAL)
	}
	stack := coreloader.DefaultStackConfig64()
	stack.Argv = append([]string(nil), argv...)
	stack.Env = append([]string(nil), env...)
	stack.ExecFilename = resolved
	layout, stackErr := coreloader.BuildStack64ForImage(newMemory, space, stack)
	if stackErr != nil {
		return int64(coresyscall.EINVAL)
	}
	newState := corecpu.NewMachineState64(newMemory)
	newState.RIP = uint64(space.Entry)
	newState.Set(corecpu.RSP, uint64(layout.SP))
	newState.RFLAGS = corecpu.Flag64IF
	*runtime.state = *newState
	runtime.context.Memory = newMemory
	runtime.context.Machine = runtime.state
	runtime.context.Brk = uint64(space.Brk)
	runtime.context.CloseOnExec()
	runtime.jit = corecpu.NewJIT64(newMemory)
	runtime.jit.OnSyscall64 = func(machine *corecpu.MachineState64) (bool, error) {
		return runtime.dispatcher.Dispatch(machine)
	}
	return 0
}
