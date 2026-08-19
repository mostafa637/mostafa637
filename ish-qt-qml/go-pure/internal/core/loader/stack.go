package loader

import (
	"encoding/binary"
	"fmt"
	"math"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
)

const (
	AT_NULL         = 0
	AT_PHDR         = 3
	AT_PHENT        = 4
	AT_PHNUM        = 5
	AT_PAGESZ       = 6
	AT_BASE         = 7
	AT_FLAGS        = 8
	AT_ENTRY        = 9
	AT_UID          = 11
	AT_EUID         = 12
	AT_GID          = 13
	AT_EGID         = 14
	AT_PLATFORM     = 15
	AT_HWCAP        = 16
	AT_CLKTCK       = 17
	AT_SECURE       = 23
	AT_RANDOM       = 25
	AT_EXECFN       = 31
	AT_SYSINFO      = 32
	AT_SYSINFO_EHDR = 33
)

type AuxEntry struct {
	Type  uint32
	Value uint32
}

type StackConfig struct {
	Top          corecpu.Address
	Pages        corecpu.Pages
	Argv         []string
	Env          []string
	ExecFilename string
	Platform     string
	Random       [16]byte
	Auxv         []AuxEntry
}

type StackLayout struct {
	Start    corecpu.Address
	Top      corecpu.Address
	SP       corecpu.Address
	Argv     corecpu.Address
	Env      corecpu.Address
	Auxv     corecpu.Address
	Platform corecpu.Address
	Random   corecpu.Address
}

func DefaultStackConfig() StackConfig {
	return StackConfig{
		Top:      corecpu.Address(0xffffe000),
		Pages:    8,
		Platform: "i686",
		Random:   [16]byte{0x69, 0x53, 0x48, 0x2d, 0x67, 0x6f, 0x2d, 0x70, 0x75, 0x72, 0x65, 0x2d, 0x67, 0x6f, 0x00, 0x01},
	}
}

func BuildStack(memory *corecpu.Memory, config StackConfig) (StackLayout, error) {
	if memory == nil {
		return StackLayout{}, fmt.Errorf("loader: nil memory")
	}
	if config.Top == 0 {
		config.Top = corecpu.Address(0xffffe000)
	}
	if config.Pages == 0 {
		config.Pages = 8
	}
	if config.Platform == "" {
		config.Platform = "i686"
	}
	if config.Top&corecpu.Address(coreelf.PageSize-1) != 0 {
		return StackLayout{}, fmt.Errorf("loader: stack top %#x is not page-aligned", config.Top)
	}
	topPage := corecpu.Page(config.Top >> corecpu.PageBits)
	if topPage == 0 || corecpu.Pages(config.Pages) > corecpu.Pages(topPage) {
		return StackLayout{}, fmt.Errorf("loader: invalid stack range")
	}
	startPage := topPage - corecpu.Page(config.Pages)
	if err := memory.MapNothing(startPage, config.Pages, corecpu.PRead|corecpu.PWrite|corecpu.PGrowDown); err != nil {
		return StackLayout{}, fmt.Errorf("loader: map stack: %w", err)
	}
	sp := uint32(config.Top)
	var err error

	execFilenameAddress := uint32(0)
	if config.ExecFilename != "" {
		execFilenameAddress, sp, err = pushString(memory, sp, config.ExecFilename)
		if err != nil {
			return StackLayout{}, err
		}
	}
	envAddress, sp, err := pushPackedStrings(memory, sp, config.Env)
	if err != nil {
		return StackLayout{}, err
	}
	argvAddress, sp, err := pushPackedStrings(memory, sp, config.Argv)
	if err != nil {
		return StackLayout{}, err
	}
	sp &= ^uint32(0xf)

	platformAddress, sp, err := pushString(memory, sp, config.Platform)
	if err != nil {
		return StackLayout{}, err
	}
	randomAddress, sp, err := pushBytes(memory, sp, config.Random[:])
	if err != nil {
		return StackLayout{}, err
	}

	auxv := make([]AuxEntry, 0, len(config.Auxv)+1)
	for _, entry := range config.Auxv {
		if entry.Type != AT_NULL {
			auxv = append(auxv, entry)
		}
	}
	if len(auxv) == 0 {
		auxv = append(auxv, AuxEntry{Type: AT_PAGESZ, Value: coreelf.PageSize})
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
	pointerWords := 1 + len(config.Argv) + 1 + len(config.Env) + 1 + (len(auxv)+1)*2
	sp -= uint32(pointerWords * 4)
	sp &= ^uint32(0xf)
	p := sp
	if err := writeWord(memory, p, uint32(len(config.Argv))); err != nil {
		return StackLayout{}, err
	}
	p += 4
	for _, address := range argvAddress {
		if err := writeWord(memory, p, address); err != nil {
			return StackLayout{}, err
		}
		p += 4
	}
	if err := writeWord(memory, p, 0); err != nil {
		return StackLayout{}, err
	}
	p += 4
	for _, address := range envAddress {
		if err := writeWord(memory, p, address); err != nil {
			return StackLayout{}, err
		}
		p += 4
	}
	if err := writeWord(memory, p, 0); err != nil {
		return StackLayout{}, err
	}
	p += 4
	auxvAddress := corecpu.Address(p)
	for _, entry := range auxv {
		if err := writeWord(memory, p, entry.Type); err != nil {
			return StackLayout{}, err
		}
		if err := writeWord(memory, p+4, entry.Value); err != nil {
			return StackLayout{}, err
		}
		p += 8
	}
	if err := writeWord(memory, p, AT_NULL); err != nil {
		return StackLayout{}, err
	}
	if err := writeWord(memory, p+4, 0); err != nil {
		return StackLayout{}, err
	}

	argvPointer := uint32(0)
	if len(argvAddress) > 0 {
		argvPointer = argvAddress[0]
	}
	envPointer := uint32(0)
	if len(envAddress) > 0 {
		envPointer = envAddress[0]
	}
	return StackLayout{
		Start:    corecpu.Address(startPage << corecpu.PageBits),
		Top:      config.Top,
		SP:       corecpu.Address(sp),
		Argv:     corecpu.Address(argvPointer),
		Env:      corecpu.Address(envPointer),
		Auxv:     auxvAddress,
		Platform: corecpu.Address(platformAddress),
		Random:   corecpu.Address(randomAddress),
	}, nil
}

func BuildStackForImage(memory *corecpu.Memory, space *AddressSpace, config StackConfig) (StackLayout, error) {
	if space == nil || space.Image == nil {
		return StackLayout{}, fmt.Errorf("loader: nil address space")
	}
	if config.ExecFilename == "" {
		config.ExecFilename = "guest"
	}
	if config.Argv == nil {
		config.Argv = []string{config.ExecFilename}
	}
	if config.Auxv == nil {
		config.Auxv = []AuxEntry{
			{Type: AT_PHDR, Value: space.Bias + space.Image.Header.ProgramOff},
			{Type: AT_PHENT, Value: uint32(space.Image.Header.ProgramEnt)},
			{Type: AT_PHNUM, Value: uint32(space.Image.Header.ProgramNum)},
			{Type: AT_PAGESZ, Value: coreelf.PageSize},
			{Type: AT_BASE, Value: 0},
			{Type: AT_FLAGS, Value: 0},
			{Type: AT_ENTRY, Value: uint32(space.Entry)},
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
	return BuildStack(memory, config)
}

func pushPackedStrings(memory *corecpu.Memory, sp uint32, values []string) ([]uint32, uint32, error) {
	if len(values) == 0 {
		return nil, sp, nil
	}
	total := 0
	for _, value := range values {
		if len(value) > math.MaxInt-total-1 {
			return nil, 0, fmt.Errorf("loader: string vector too large")
		}
		total += len(value) + 1
	}
	if uint32(total) > sp {
		return nil, 0, fmt.Errorf("loader: stack underflow")
	}
	sp -= uint32(total)
	addresses := make([]uint32, len(values))
	cursor := sp
	for index, value := range values {
		addresses[index] = cursor
		if err := writeBytes(memory, cursor, []byte(value)); err != nil {
			return nil, 0, err
		}
		if err := writeBytes(memory, cursor+uint32(len(value)), []byte{0}); err != nil {
			return nil, 0, err
		}
		cursor += uint32(len(value) + 1)
	}
	return addresses, sp, nil
}

func pushString(memory *corecpu.Memory, sp uint32, value string) (uint32, uint32, error) {
	addresses, next, err := pushPackedStrings(memory, sp, []string{value})
	if err != nil {
		return 0, 0, err
	}
	return addresses[0], next, nil
}

func pushBytes(memory *corecpu.Memory, sp uint32, data []byte) (uint32, uint32, error) {
	if uint32(len(data)) > sp {
		return 0, 0, fmt.Errorf("loader: stack underflow")
	}
	sp -= uint32(len(data))
	if err := writeBytes(memory, sp, data); err != nil {
		return 0, 0, err
	}
	return sp, sp, nil
}

func writeBytes(memory *corecpu.Memory, address uint32, data []byte) error {
	if err := memory.Write(corecpu.Address(address), data); err != nil {
		return fmt.Errorf("loader: write stack %#x: %w", address, err)
	}
	return nil
}

func writeWord(memory *corecpu.Memory, address uint32, value uint32) error {
	var data [4]byte
	binary.LittleEndian.PutUint32(data[:], value)
	return writeBytes(memory, address, data[:])
}
