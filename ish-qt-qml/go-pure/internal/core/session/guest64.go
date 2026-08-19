package session

import (
	"bytes"
	"context"
	"debug/elf"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"

	corekernel "github.com/mostafa637/mostafa637/go-pure/internal/core/kernel"
	coreloader "github.com/mostafa637/mostafa637/go-pure/internal/core/loader"
	coresyscall "github.com/mostafa637/mostafa637/go-pure/internal/core/syscall"
)

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
	dispatcher := coresyscall.NewDispatcher64(sysContext)

	jit := corecpu.NewJIT64(memory)
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
