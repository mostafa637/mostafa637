package session

import (
	"bytes"
	"context"
	"debug/elf"
	pathpkg "path"
	"strings"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"

	corekernel "github.com/mostafa637/mostafa637/go-pure/internal/core/kernel"
	coreloader "github.com/mostafa637/mostafa637/go-pure/internal/core/loader"
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
	image, err := coreelf.Parse64(bytesReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	var bias corecpu.Address64
	if image.Header.Type == elf.ET_DYN {
		bias = corecpu.Address64(0x0000000000400000)
	}
	space, err := coreloader.Load64(bytesReader(data), int64(len(data)), image, memory, bias)
	if err != nil {
		return err
	}
	stack := coreloader.DefaultStackConfig64()
	stack.Argv = []string{g.elfPath}
	stack.Env = []string{"PATH=/bin:/usr/bin"}
	stack.ExecFilename = g.elfPath
	layout, err := coreloader.BuildStack64ForImage(memory, space, stack)
	if err != nil {
		return err
	}

	state := corecpu.NewMachineState64(memory)
	state.RIP = uint64(space.Entry)
	state.Set(corecpu.RSP, uint64(layout.SP))
	state.RFLAGS = corecpu.Flag64IF

	sysContext := coresyscall.NewContext64(memory)
	sysContext.FS = fake
	sysContext.PID = uint64(process.PID)
	sysContext.TID = uint64(process.PID)
	sysContext.Brk = uint64(space.Brk)
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
		newImage, parseErr := coreelf.Parse64(bytesReader(imageData), int64(len(imageData)))
		if parseErr != nil {
			return int64(coresyscall.EINVAL)
		}
		newMemory := corecpu.NewMemory64()
		var newBias corecpu.Address64
		if newImage.Header.Type == elf.ET_DYN {
			newBias = corecpu.Address64(0x0000000000400000)
		}
		newSpace, loadErr := coreloader.Load64(bytesReader(imageData), int64(len(imageData)), newImage, newMemory, newBias)
		if loadErr != nil {
			return int64(coresyscall.EINVAL)
		}
		newStack := coreloader.DefaultStackConfig64()
		newStack.Argv = append([]string(nil), argv...)
		newStack.Env = append([]string(nil), env...)
		newStack.ExecFilename = resolved
		newLayout, stackErr := coreloader.BuildStack64ForImage(newMemory, newSpace, newStack)
		if stackErr != nil {
			return int64(coresyscall.EINVAL)
		}

		newState := corecpu.NewMachineState64(newMemory)
		newState.RIP = uint64(newSpace.Entry)
		newState.Set(corecpu.RSP, uint64(newLayout.SP))
		newState.RFLAGS = corecpu.Flag64IF
		*state = *newState
		sysContext.Memory = newMemory
		sysContext.Brk = uint64(newSpace.Brk)
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
