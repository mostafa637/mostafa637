package session

import (
	"bytes"
	"fmt"
	"io"
	"path"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
	coreloader "github.com/mostafa637/mostafa637/go-pure/internal/core/loader"
)

func loadGuestImage64(fake *corefs.FS, filename string, data []byte, argv, env []string, memory *corecpu.Memory64) (*coreloader.DynamicLoad64, coreloader.StackLayout64, error) {
	if fake == nil || len(data) == 0 || memory == nil {
		return nil, coreloader.StackLayout64{}, fmt.Errorf("guest64 loader: invalid image argument")
	}
	open := func(name string) (reader io.ReaderAt, size int64, resolved string, err error) {
		for _, candidate := range guestELFCandidates64(name) {
			candidate = path.Clean(candidate)
			payload, readErr := fake.ReadFile(candidate)
			if readErr == nil {
				return bytes.NewReader(payload), int64(len(payload)), candidate, nil
			}
			err = readErr
		}
		return nil, 0, "", err
	}
	loaded, err := coreloader.LoadDynamic64(bytes.NewReader(data), int64(len(data)), filename, 0, memory, open)
	if err != nil {
		return nil, coreloader.StackLayout64{}, err
	}
	stack := coreloader.DefaultStackConfig64()
	stack.Argv = append([]string(nil), argv...)
	stack.Env = append([]string(nil), env...)
	stack.ExecFilename = filename
	stack.Auxv = dynamicAuxv64(loaded)
	layout, err := coreloader.BuildStack64ForImage(memory, loaded.Main.Space, stack)
	if err != nil {
		return nil, coreloader.StackLayout64{}, err
	}
	return loaded, layout, nil
}

func guestELFCandidates64(name string) []string {
	if name == "" {
		return nil
	}
	candidates := []string{name}
	if !path.IsAbs(name) {
		candidates = append(candidates,
			path.Join("/lib", name),
			path.Join("/lib/x86_64-linux-gnu", name),
			path.Join("/usr/lib", name),
			path.Join("/usr/lib/x86_64-linux-gnu", name),
		)
	}
	seen := make(map[string]struct{}, len(candidates))
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = path.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		result = append(result, candidate)
	}
	return result
}

func dynamicAuxv64(loaded *coreloader.DynamicLoad64) []coreloader.AuxEntry64 {
	if loaded == nil || loaded.Main == nil || loaded.Main.Space == nil || loaded.Main.Image == nil {
		return nil
	}
	main := loaded.Main.Space
	entries := []coreloader.AuxEntry64{
		{Type: coreloader.AT_PHDR, Value: uint64(main.Bias) + main.Image.Header.ProgramOff},
		{Type: coreloader.AT_PHENT, Value: uint64(main.Image.Header.ProgramEnt)},
		{Type: coreloader.AT_PHNUM, Value: uint64(main.Image.Header.ProgramNum)},
		{Type: coreloader.AT_PAGESZ, Value: uint64(coreelf.PageSize)},
		{Type: coreloader.AT_BASE, Value: 0},
		{Type: coreloader.AT_FLAGS, Value: 0},
		{Type: coreloader.AT_ENTRY, Value: uint64(main.Entry)},
		{Type: coreloader.AT_UID, Value: 0},
		{Type: coreloader.AT_EUID, Value: 0},
		{Type: coreloader.AT_GID, Value: 0},
		{Type: coreloader.AT_EGID, Value: 0},
		{Type: coreloader.AT_SECURE, Value: 0},
		{Type: coreloader.AT_RANDOM, Value: 0},
		{Type: coreloader.AT_EXECFN, Value: 0},
		{Type: coreloader.AT_PLATFORM, Value: 0},
	}
	if loaded.Interpreter != nil && loaded.Interpreter.Space != nil {
		entries[4].Value = uint64(loaded.Interpreter.Space.Bias)
	}
	return entries
}

func attachTLS64(state *corecpu.MachineState64, loaded *coreloader.DynamicLoad64) {
	if state == nil || loaded == nil || loaded.TLS == nil || len(loaded.TLS.Modules) == 0 {
		return
	}
	_ = coreloader.AttachTLS64(state, loaded.TLS)
}
