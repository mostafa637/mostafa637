package loader

import (
	"encoding/binary"
	"fmt"
	"math"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
)

// AuxEntry64 is the 64-bit form of an ELF auxiliary-vector pair.
type AuxEntry64 struct {
	Type  uint64
	Value uint64
}

type StackConfig64 struct {
	Top          corecpu.Address64
	Pages        uint64
	Argv         []string
	Env          []string
	ExecFilename string
	Platform     string
	Random       [16]byte
	Auxv         []AuxEntry64
}

type StackLayout64 struct {
	Start    corecpu.Address64
	Top      corecpu.Address64
	SP       corecpu.Address64
	Argv     corecpu.Address64
	Env      corecpu.Address64
	Auxv     corecpu.Address64
	Platform corecpu.Address64
	Random   corecpu.Address64
}

func DefaultStackConfig64() StackConfig64 {
	return StackConfig64{
		Top:      corecpu.Address64(0x00007fffffffe000),
		Pages:    32,
		Platform: "x86_64",
		Random:   [16]byte{0x69, 0x53, 0x48, 0x2d, 0x67, 0x6f, 0x2d, 0x70, 0x75, 0x72, 0x65, 0x2d, 0x36, 0x34, 0x00, 0x01},
	}
}

func BuildStack64ForImage(memory *corecpu.Memory64, space *AddressSpace64, config StackConfig64) (StackLayout64, error) {
	if space == nil || space.Image == nil {
		return StackLayout64{}, fmt.Errorf("loader64: nil address space")
	}
	if config.ExecFilename == "" {
		config.ExecFilename = "guest"
	}
	if config.Argv == nil {
		config.Argv = []string{config.ExecFilename}
	}
	if config.Auxv == nil {
		config.Auxv = []AuxEntry64{
			{Type: AT_PHDR, Value: uint64(space.Bias) + space.Image.Header.ProgramOff},
			{Type: AT_PHENT, Value: uint64(space.Image.Header.ProgramEnt)},
			{Type: AT_PHNUM, Value: uint64(space.Image.Header.ProgramNum)},
			{Type: AT_PAGESZ, Value: uint64(coreelf.PageSize)},
			{Type: AT_BASE, Value: 0},
			{Type: AT_FLAGS, Value: 0},
			{Type: AT_ENTRY, Value: uint64(space.Entry)},
			{Type: AT_UID, Value: 0},
			{Type: AT_EUID, Value: 0},
			{Type: AT_GID, Value: 0},
			{Type: AT_EGID, Value: 0},
			{Type: AT_SECURE, Value: 0},
			{Type: AT_RANDOM, Value: 0},
			{Type: AT_EXECFN, Value: 0},
			{Type: AT_PLATFORM, Value: 0},
		}
	}
	return BuildStack64(memory, config)
}

func BuildStack64(memory *corecpu.Memory64, config StackConfig64) (StackLayout64, error) {
	if memory == nil {
		return StackLayout64{}, fmt.Errorf("loader64: nil memory")
	}
	if config.Top == 0 {
		config.Top = DefaultStackConfig64().Top
	}
	if config.Pages == 0 {
		config.Pages = DefaultStackConfig64().Pages
	}
	if config.Platform == "" {
		config.Platform = "x86_64"
	}
	if uint64(config.Top)%corecpu.Page64Size != 0 {
		return StackLayout64{}, fmt.Errorf("loader64: stack top %#x is not page-aligned", config.Top)
	}
	if config.Pages > math.MaxUint64/corecpu.Page64Size {
		return StackLayout64{}, fmt.Errorf("loader64: stack size overflow")
	}
	stackSize := config.Pages * corecpu.Page64Size
	if uint64(config.Top) < stackSize {
		return StackLayout64{}, fmt.Errorf("loader64: invalid stack range")
	}
	start := corecpu.Address64(uint64(config.Top) - stackSize)
	if err := memory.Map(start, stackSize, corecpu.PRead|corecpu.PWrite); err != nil {
		return StackLayout64{}, fmt.Errorf("loader64: map stack: %w", err)
	}

	sp := uint64(config.Top)
	execFilenameAddress := uint64(0)
	var err error
	if config.ExecFilename != "" {
		execFilenameAddress, sp, err = pushString64(memory, sp, config.ExecFilename)
		if err != nil {
			return StackLayout64{}, err
		}
	}
	envAddress, sp, err := pushPackedStrings64(memory, sp, config.Env)
	if err != nil {
		return StackLayout64{}, err
	}
	argvAddress, sp, err := pushPackedStrings64(memory, sp, config.Argv)
	if err != nil {
		return StackLayout64{}, err
	}
	sp &= ^uint64(0xf)

	platformAddress, sp, err := pushString64(memory, sp, config.Platform)
	if err != nil {
		return StackLayout64{}, err
	}
	randomAddress, sp, err := pushBytes64(memory, sp, config.Random[:])
	if err != nil {
		return StackLayout64{}, err
	}

	auxv := make([]AuxEntry64, 0, len(config.Auxv)+1)
	for _, entry := range config.Auxv {
		if entry.Type != AT_NULL {
			auxv = append(auxv, entry)
		}
	}
	if len(auxv) == 0 {
		auxv = append(auxv, AuxEntry64{Type: AT_PAGESZ, Value: uint64(coreelf.PageSize)})
	}
	for index := range auxv {
		switch auxv[index].Type {
		case AT_RANDOM:
			auxv[index].Value = randomAddress
		case AT_PLATFORM:
			auxv[index].Value = platformAddress
		case AT_EXECFN:
			auxv[index].Value = execFilenameAddress
		}
	}
	pointerWords := uint64(1 + len(config.Argv) + 1 + len(config.Env) + 1 + (len(auxv)+1)*2)
	if pointerWords > math.MaxUint64/8 || pointerWords*8 > sp {
		return StackLayout64{}, fmt.Errorf("loader64: stack underflow")
	}
	sp -= pointerWords * 8
	sp &= ^uint64(0xf)
	p := sp
	if err := writeWord64(memory, p, uint64(len(config.Argv))); err != nil {
		return StackLayout64{}, err
	}
	p += 8
	for _, address := range argvAddress {
		if err := writeWord64(memory, p, address); err != nil {
			return StackLayout64{}, err
		}
		p += 8
	}
	if err := writeWord64(memory, p, 0); err != nil {
		return StackLayout64{}, err
	}
	p += 8
	for _, address := range envAddress {
		if err := writeWord64(memory, p, address); err != nil {
			return StackLayout64{}, err
		}
		p += 8
	}
	if err := writeWord64(memory, p, 0); err != nil {
		return StackLayout64{}, err
	}
	p += 8
	auxvAddress := corecpu.Address64(p)
	for _, entry := range auxv {
		if err := writeWord64(memory, p, entry.Type); err != nil {
			return StackLayout64{}, err
		}
		if err := writeWord64(memory, p+8, entry.Value); err != nil {
			return StackLayout64{}, err
		}
		p += 16
	}
	if err := writeWord64(memory, p, AT_NULL); err != nil {
		return StackLayout64{}, err
	}
	if err := writeWord64(memory, p+8, 0); err != nil {
		return StackLayout64{}, err
	}

	argvPointer := uint64(0)
	if len(argvAddress) > 0 {
		argvPointer = argvAddress[0]
	}
	envPointer := uint64(0)
	if len(envAddress) > 0 {
		envPointer = envAddress[0]
	}
	return StackLayout64{
		Start:    start,
		Top:      config.Top,
		SP:       corecpu.Address64(sp),
		Argv:     corecpu.Address64(argvPointer),
		Env:      corecpu.Address64(envPointer),
		Auxv:     auxvAddress,
		Platform: corecpu.Address64(platformAddress),
		Random:   corecpu.Address64(randomAddress),
	}, nil
}

func pushPackedStrings64(memory *corecpu.Memory64, sp uint64, values []string) ([]uint64, uint64, error) {
	if len(values) == 0 {
		return nil, sp, nil
	}
	total := 0
	for _, value := range values {
		if len(value) > math.MaxInt-total-1 {
			return nil, 0, fmt.Errorf("loader64: string vector too large")
		}
		total += len(value) + 1
	}
	if uint64(total) > sp {
		return nil, 0, fmt.Errorf("loader64: stack underflow")
	}
	sp -= uint64(total)
	addresses := make([]uint64, len(values))
	cursor := sp
	for index, value := range values {
		addresses[index] = cursor
		if err := writeBytes64(memory, corecpu.Address64(cursor), []byte(value)); err != nil {
			return nil, 0, err
		}
		if err := writeBytes64(memory, corecpu.Address64(cursor+uint64(len(value))), []byte{0}); err != nil {
			return nil, 0, err
		}
		cursor += uint64(len(value) + 1)
	}
	return addresses, sp, nil
}

func pushString64(memory *corecpu.Memory64, sp uint64, value string) (uint64, uint64, error) {
	addresses, next, err := pushPackedStrings64(memory, sp, []string{value})
	if err != nil {
		return 0, 0, err
	}
	return addresses[0], next, nil
}

func pushBytes64(memory *corecpu.Memory64, sp uint64, data []byte) (uint64, uint64, error) {
	if uint64(len(data)) > sp {
		return 0, 0, fmt.Errorf("loader64: stack underflow")
	}
	sp -= uint64(len(data))
	if err := writeBytes64(memory, corecpu.Address64(sp), data); err != nil {
		return 0, 0, err
	}
	return sp, sp, nil
}

func writeBytes64(memory *corecpu.Memory64, address corecpu.Address64, data []byte) error {
	if err := memory.Write(address, data); err != nil {
		return fmt.Errorf("loader64: write stack %#x: %w", address, err)
	}
	return nil
}

func writeWord64(memory *corecpu.Memory64, address uint64, value uint64) error {
	var data [8]byte
	binary.LittleEndian.PutUint64(data[:], value)
	return writeBytes64(memory, corecpu.Address64(address), data[:])
}
