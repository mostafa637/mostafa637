package kernel

import (
	"bytes"
	"debug/elf"
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
	Image            *coreelf.Image
	Space            *coreloader.AddressSpace
	Stack            coreloader.StackLayout
	Interpreter      *coreelf.Image
	InterpreterSpace *coreloader.AddressSpace
	Registry         *coreloader.ObjectRegistry
	TLS              *coreloader.TLSBlock
	TLSLayout        *coreloader.TLSLayout
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
	mainBias := bias
	if image.Header.Type == elf.ET_DYN && mainBias == 0 {
		mainBias = 0x10000000
	}
	space, err := coreloader.Load(r, size, image, p.Memory, mainBias)
	if err != nil {
		return nil, err
	}
	registry := coreloader.NewObjectRegistry(p.Memory)

	mainName := filename
	if mainName == "" {
		mainName = "<main>"
	}
	if _, err := registry.AddWithReader(mainName, space, r, size); err != nil {
		return nil, err
	}
	entry := space.Entry
	var interpreterImage *coreelf.Image
	var interpreterSpace *coreloader.AddressSpace
	if image.Interp != "" {
		if p.FS == nil {
			return nil, fmt.Errorf("kernel: PT_INTERP %q requires fakefs", image.Interp)
		}
		interpreterData, readErr := p.FS.ReadFile(image.Interp)
		if readErr != nil {
			return nil, fmt.Errorf("kernel: read interpreter %q: %w", image.Interp, readErr)
		}
		interpreterImage, err = coreelf.Parse(bytes.NewReader(interpreterData), int64(len(interpreterData)))
		if err != nil {
			return nil, fmt.Errorf("kernel: parse interpreter %q: %w", image.Interp, err)
		}
		if interpreterImage.Interp != "" {
			return nil, fmt.Errorf("kernel: nested PT_INTERP is unsupported")
		}
		interpreterBias := uint32(0)
		if interpreterImage.Header.Type == elf.ET_DYN {
			start, _, rangeErr := interpreterImage.LoadRange()
			if rangeErr != nil || start > 0x40000000 {
				return nil, fmt.Errorf("kernel: invalid interpreter load range")
			}
			interpreterBias = 0x40000000 - start
		}
		interpreterReader := bytes.NewReader(interpreterData)
		interpreterSpace, err = coreloader.Load(interpreterReader, int64(len(interpreterData)), interpreterImage, p.Memory, interpreterBias)
		if err != nil {
			return nil, fmt.Errorf("kernel: load interpreter %q: %w", image.Interp, err)
		}
		if _, err := registry.AddWithReader(image.Interp, interpreterSpace, interpreterReader, int64(len(interpreterData))); err != nil {
			return nil, err
		}

		stack.Auxv = dynamicAuxv(space, interpreterSpace)

		entry = interpreterSpace.Entry
	}
	if err := p.loadNeededObjects(registry); err != nil {
		return nil, err
	}
	tlsLayout, err := p.loadTLSModules(registry)
	if err != nil {
		return nil, fmt.Errorf("kernel: load TLS modules: %w", err)
	}
	if err := coreloader.ApplyAllRelocations(p.Memory, registry); err != nil {
		return nil, fmt.Errorf("kernel: apply dynamic relocations: %w", err)
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
	p.CPU.EIP = uint32(entry)
	p.CPU.Regs[corecpu.ESP] = uint32(layout.SP)
	p.CPU.EFlags = 0
	p.CPU.CF = 0
	p.CPU.OF = 0
	p.CPU.Lazy = 0
	p.CPU.FaultAt = 0
	p.CPU.FaultWrite = false
	p.CPU.TrapNo = 0
	p.CPU.FCW = 0x037f
	if tlsLayout != nil && len(tlsLayout.Modules) > 0 {
		p.CPU.FSBase = uint32(tlsLayout.ThreadPointer)
		p.CPU.GSBase = uint32(tlsLayout.ThreadPointer)
		p.CPU.TLS = uint32(tlsLayout.ThreadPointer)
		p.CPU.TLSDTV = uint32(tlsLayout.DTVStart)
	} else {
		p.CPU.FSBase = 0
		p.CPU.GSBase = 0
		p.CPU.TLS = 0
		p.CPU.TLSDTV = 0
	}
	var tls *coreloader.TLSBlock
	if tlsLayout != nil && len(tlsLayout.Modules) > 0 {
		tls = &tlsLayout.Modules[0].Block
	}
	p.Executor.Halted = false
	return &LoadedImage{
		Image:            image,
		Space:            space,
		Stack:            layout,
		Interpreter:      interpreterImage,
		InterpreterSpace: interpreterSpace,
		Registry:         registry,
		TLS:              tls,
		TLSLayout:        tlsLayout,
	}, nil

}

func (p *Process) loadNeededObjects(registry *coreloader.ObjectRegistry) error {
	if p == nil || p.FS == nil || registry == nil {
		return nil
	}
	objects := registry.Objects()
	nextBias := uint32(0x50000000)
	for index := 0; index < len(objects); index++ {
		object := objects[index]
		if object == nil || object.Image == nil || object.Image.Dynamic == nil {
			continue
		}
		for _, needed := range object.Image.Dynamic.Needed {
			if registry.Has(needed) {
				continue
			}
			filename, data, err := readNeededObject(p.FS, needed)
			if err != nil {
				return fmt.Errorf("kernel: load DT_NEEDED %q for %s: %w", needed, object.Name, err)
			}
			image, err := coreelf.Parse(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				return fmt.Errorf("kernel: parse shared object %q: %w", filename, err)
			}
			if image.Interp != "" {
				return fmt.Errorf("kernel: shared object %q has unsupported PT_INTERP %q", filename, image.Interp)
			}
			if image.Header.Type != elf.ET_DYN {
				return fmt.Errorf("kernel: shared object %q is not ET_DYN", filename)
			}
			bias, updatedBias, err := nextObjectBias(image, nextBias)
			if err != nil {
				return fmt.Errorf("kernel: choose bias for shared object %q: %w", filename, err)
			}
			reader := bytes.NewReader(data)
			space, err := coreloader.Load(reader, int64(len(data)), image, p.Memory, bias)
			if err != nil {
				return fmt.Errorf("kernel: load shared object %q: %w", filename, err)
			}
			if _, err := registry.AddWithReader(filename, space, reader, int64(len(data))); err != nil {
				return fmt.Errorf("kernel: register shared object %q: %w", filename, err)
			}
			nextBias = updatedBias
			objects = registry.Objects()
		}
	}
	return nil
}

func (p *Process) loadTLSModules(registry *coreloader.ObjectRegistry) (*coreloader.TLSLayout, error) {
	if p == nil || registry == nil {
		return nil, nil
	}
	objects := registry.Objects()
	specs := make([]coreloader.TLSModuleSpec, 0, len(objects))
	moduleID := uint32(1)
	for _, object := range objects {
		if object == nil || object.Reader == nil || object.Size <= 0 || object.Image == nil || !hasTLS(object.Image) {
			continue
		}
		specs = append(specs, coreloader.TLSModuleSpec{
			ID: moduleID, Name: object.Name, Reader: object.Reader, Size: object.Size,
			Image: object.Image, Bias: object.Space.Bias,
		})
		moduleID++
	}
	return coreloader.LoadTLSModules(p.Memory, specs, coreloader.DefaultTLSBase(), 0)
}

func hasTLS(image *coreelf.Image) bool {
	if image == nil {
		return false
	}
	for _, segment := range image.Segments {
		if segment.Type == 7 && segment.MemSize != 0 {
			return true
		}
	}
	return false
}

func readNeededObject(fsys *corefs.FS, needed string) (string, []byte, error) {
	candidates := []string{
		needed,
		pathpkg.Join("/lib", needed),
		pathpkg.Join("/lib/i386-linux-gnu", needed),
		pathpkg.Join("/usr/lib", needed),
		pathpkg.Join("/usr/lib/i386-linux-gnu", needed),
	}
	var lastErr error
	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		candidate = pathpkg.Clean(candidate)
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		data, err := fsys.ReadFile(candidate)
		if err == nil {
			return candidate, data, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = corefs.ErrNotFound
	}
	return "", nil, lastErr
}

func nextObjectBias(image *coreelf.Image, next uint32) (uint32, uint32, error) {
	start, end, err := image.LoadRange()
	if err != nil {
		return 0, 0, err
	}
	if end <= start {
		return 0, 0, fmt.Errorf("empty load range")
	}
	if start > next {
		return 0, 0, fmt.Errorf("load range starts above allocation window")
	}
	bias := (next - start) &^ (coreelf.PageSize - 1)
	actualEnd64 := uint64(bias) + uint64(end)
	if actualEnd64 > 0xffffffff {
		return 0, 0, fmt.Errorf("load range overflows 32-bit address space")
	}
	nextEnd := uint32(actualEnd64)
	if nextEnd > 0xffffffff-coreelf.PageSize {
		return 0, 0, fmt.Errorf("shared-object allocation window exhausted")
	}
	return bias, (nextEnd+coreelf.PageSize-1)&^(coreelf.PageSize-1) + coreelf.PageSize, nil
}

func dynamicAuxv(main, interpreter *coreloader.AddressSpace) []coreloader.AuxEntry {
	return []coreloader.AuxEntry{
		{Type: coreloader.AT_PHDR, Value: main.Bias + main.Image.Header.ProgramOff},
		{Type: coreloader.AT_PHENT, Value: uint32(main.Image.Header.ProgramEnt)},
		{Type: coreloader.AT_PHNUM, Value: uint32(main.Image.Header.ProgramNum)},
		{Type: coreloader.AT_PAGESZ, Value: coreelf.PageSize},
		{Type: coreloader.AT_BASE, Value: interpreter.Bias},
		{Type: coreloader.AT_FLAGS, Value: 0},
		{Type: coreloader.AT_ENTRY, Value: uint32(main.Entry)},
		{Type: coreloader.AT_UID, Value: 0},
		{Type: coreloader.AT_EUID, Value: 0},
		{Type: coreloader.AT_GID, Value: 0},
		{Type: coreloader.AT_EGID, Value: 0},
		{Type: coreloader.AT_SECURE, Value: 0},
		{Type: coreloader.AT_RANDOM, Value: 0},
		{Type: coreloader.AT_EXECFN, Value: 0},
		{Type: coreloader.AT_PLATFORM, Value: 0},
	}
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
	p.Context.CloseOnExec()
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
