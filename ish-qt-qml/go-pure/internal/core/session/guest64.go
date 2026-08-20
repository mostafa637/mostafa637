package session

import (
	"bytes"
	"context"
	"debug/elf"
	pathpkg "path"
	"strings"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"

	corekernel "github.com/mostafa637/mostafa637/go-pure/internal/core/kernel"
	coresyscall "github.com/mostafa637/mostafa637/go-pure/internal/core/syscall"
)

func resolveSessionPath64(cwd, name string) (string, bool) {
	if name == "" {
		return "", false
	}
	if strings.HasPrefix(name, "/") {
		return pathpkg.Clean(name), true
	}
	if cwd == "" {
		cwd = "/"
	}
	resolved := pathpkg.Clean(pathpkg.Join(cwd, name))
	if !strings.HasPrefix(resolved, "/") {
		return "", false
	}
	return resolved, true
}

func isELF64Image(data []byte) bool {
	return len(data) >= 5 && bytes.Equal(data[:4], []byte{0x7f, 'E', 'L', 'F'}) && data[4] == byte(elf.ELFCLASS64)
}

func (g *guestTransport) start64(ctx context.Context, process *corekernel.Process, fake *corefs.FS, data []byte, cols, rows int) error {
	if g == nil || process == nil || fake == nil {
		return transportError("guest64 session: nil process or fakefs")
	}
	if err := process.SetWindowSize(uint16(cols), uint16(rows)); err != nil {
		return err
	}
	reader := &guestInput{chunks: g.input, done: g.done}
	writer := &guestOutput{chunks: g.output, done: g.done}

	memory := corecpu.NewMemory64()
	loaded, layout, err := loadGuestImage64(fake, g.elfPath, data, []string{g.elfPath}, []string{"PATH=/bin:/usr/bin"}, memory)
	if err != nil {
		return err
	}

	state := corecpu.NewMachineState64(memory)
	state.RIP = uint64(loaded.Entry)
	state.Set(corecpu.RSP, uint64(layout.SP))
	state.RFLAGS = corecpu.Flag64IF
	attachTLS64(state, loaded)

	sysContext := coresyscall.NewContext64(memory)
	sysContext.FS = fake
	sysContext.PID = uint64(process.PID)
	sysContext.TID = uint64(process.PID)
	sysContext.Machine = state
	sysContext.Brk = uint64(loaded.Main.Space.Brk)
	if err := sysContext.InstallFile(0, &corefd.File{Reader: reader}); err != nil {
		return err
	}
	if err := sysContext.InstallFile(1, &corefd.File{Writer: writer}); err != nil {
		return err
	}
	if err := sysContext.InstallFile(2, &corefd.File{Writer: writer}); err != nil {
		return err
	}

	var jit *corecpu.JIT64
	var dispatcher *coresyscall.Dispatcher64
	sysContext.Execve = func(path string, argv, env []string) int64 {
		resolved, ok := resolveSessionPath64(sysContext.CWD, path)
		if !ok || resolved == "" {
			return int64(coresyscall.ENOENT)
		}
		imageData, readErr := fake.ReadFile(resolved)
		if readErr != nil {
			return int64(coresyscall.ENOENT)
		}
		newMemory := corecpu.NewMemory64()
		newLoaded, newLayout, loadErr := loadGuestImage64(fake, resolved, imageData, argv, env, newMemory)
		if loadErr != nil {
			return int64(coresyscall.EINVAL)
		}

		newState := corecpu.NewMachineState64(newMemory)
		newState.RIP = uint64(newLoaded.Entry)
		newState.Set(corecpu.RSP, uint64(newLayout.SP))
		newState.RFLAGS = corecpu.Flag64IF
		attachTLS64(newState, newLoaded)

		*state = *newState
		sysContext.Memory = newMemory
		sysContext.Machine = state
		sysContext.Brk = uint64(newLoaded.Main.Space.Brk)

		sysContext.CloseOnExec()
		jit = corecpu.NewJIT64(newMemory)
		jit.OnSyscall64 = func(machine *corecpu.MachineState64) (bool, error) {
			return dispatcher.Dispatch(machine)
		}
		return 0
	}

	dispatcher = coresyscall.NewDispatcher64(sysContext)
	jit = corecpu.NewJIT64(memory)
	jit.OnSyscall64 = func(machine *corecpu.MachineState64) (bool, error) {
		return dispatcher.Dispatch(machine)
	}
	sysContext.ProcessFactory = g.createChild64
	sysContext.ChildStarter = g.startChild64
	sysContext.VForkWaiter = g.waitVFork64
	g.runtimeMu.Lock()
	if g.runtimes == nil {
		g.runtimes = make(map[*coresyscall.Context64]*guest64Runtime)
	}
	g.nextPID = uint64(process.PID)
	g.runtimes[sysContext] = &guest64Runtime{transport: g, context: sysContext, state: state, dispatcher: dispatcher, jit: jit, pid: uint64(process.PID), done: make(chan struct{})}
	g.runtimeMu.Unlock()
	g.process = process
	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		defer g.closeOutput()
		for {
			select {
			case <-g.done:
				return
			default:
			}
			trap := jit.RunToInterrupt(state)
			switch trap {
			case corecpu.Trap64Timer:
				continue
			case corecpu.Trap64Exit:
				process.SetExitStatus(int32(state.Get(corecpu.RAX)))
				return
			default:
				process.SetExitStatus(int32(128 + trap))
				return
			}
		}
	}()
	if ctx != nil {
		go func() {
			select {
			case <-ctx.Done():
				g.stop()
				jit.Poke()
			case <-runDone:
			}
		}()
	}
	return nil
}
