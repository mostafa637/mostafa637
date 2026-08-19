package loader

import (
	"encoding/binary"
	"fmt"
	"math"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	R386None     = 0
	R38632       = 1
	R386PC32     = 2
	R386GlobDat  = 6
	R386JMPSlot  = 7
	R386Relative = 8

	shnUndef = 0
	shnAbs   = 0xfff1

	dtRel  = 17
	dtRela = 7
)

// ApplyRelocations applies i386 relocations using the image's own symbol table.
// Relative and locally-defined symbol relocations are supported. Undefined
// symbols are rejected until a multi-object dynamic linker is available; this
// prevents silently corrupting the guest GOT/PLT.
func ApplyRelocations(memory *corecpu.Memory, space *AddressSpace) error {
	if memory == nil || space == nil || space.Image == nil || space.Image.Dynamic == nil {
		return nil
	}
	info := space.Image.Dynamic
	if err := applyRelocationTable(memory, space, info.Rel, info.RelSz, info.RelEnt); err != nil {
		return err
	}
	if info.JmpRel != 0 && info.PltRelSz != 0 {
		switch info.PltRel {
		case 0, dtRel:
			if err := applyRelocationTable(memory, space, info.JmpRel, info.PltRelSz, info.RelEnt); err != nil {
				return fmt.Errorf("loader: apply PLT relocations: %w", err)
			}
		case dtRela:
			return fmt.Errorf("loader: i386 DT_RELA PLT relocations are unsupported")
		default:
			return fmt.Errorf("loader: unsupported DT_PLTREL value %d", info.PltRel)
		}
	}
	return nil
}

func applyRelocationTable(memory *corecpu.Memory, space *AddressSpace, table, size, entrySize uint32) error {
	if table == 0 || size == 0 {
		return nil
	}
	if entrySize == 0 {
		entrySize = 8
	}
	if entrySize != 8 || size%entrySize != 0 {
		return fmt.Errorf("loader: unsupported i386 REL table entry size %d", entrySize)
	}
	count := size / entrySize
	for index := uint32(0); index < count; index++ {
		recordAddress, err := imageAddress(space.Bias, table+index*entrySize)
		if err != nil {
			return fmt.Errorf("loader: REL[%d] address: %w", index, err)
		}
		var record [8]byte
		if err := memory.Read(recordAddress, record[:]); err != nil {
			return fmt.Errorf("loader: read REL[%d] at %#x: %w", index, recordAddress, err)
		}
		offset := binary.LittleEndian.Uint32(record[:4])
		relocationInfo := binary.LittleEndian.Uint32(record[4:])
		relocationType := relocationInfo & 0xff
		symbolIndex := relocationInfo >> 8
		target, err := imageAddress(space.Bias, offset)
		if err != nil {
			return fmt.Errorf("loader: REL[%d] target: %w", index, err)
		}
		if err := applyOneRelocation(memory, space, relocationType, symbolIndex, target, index); err != nil {
			return err
		}
	}
	return nil
}

func applyOneRelocation(memory *corecpu.Memory, space *AddressSpace, relocationType, symbolIndex uint32, target corecpu.Address, index uint32) error {
	if relocationType == R386None {
		return nil
	}
	if relocationType == R386Relative {
		if symbolIndex != 0 {
			return fmt.Errorf("loader: R_386_RELATIVE[%d] has symbol index %d", index, symbolIndex)
		}
		var addend [4]byte
		if err := memory.Read(target, addend[:]); err != nil {
			return fmt.Errorf("loader: read RELATIVE addend at %#x: %w", target, err)
		}
		value := space.Bias + binary.LittleEndian.Uint32(addend[:])
		return writeRelocationValue(memory, target, value)
	}

	symbol, err := resolveSymbol(memory, space, symbolIndex)
	if err != nil {
		return fmt.Errorf("loader: resolve symbol %d for relocation[%d]: %w", symbolIndex, index, err)
	}
	var addend [4]byte
	if err := memory.Read(target, addend[:]); err != nil {
		return fmt.Errorf("loader: read relocation addend at %#x: %w", target, err)
	}
	A := binary.LittleEndian.Uint32(addend[:])
	var value uint32
	switch relocationType {
	case R38632:
		value = symbol + A
	case R386PC32:
		value = symbol + A - uint32(target)
	case R386GlobDat, R386JMPSlot:
		value = symbol
	default:
		return fmt.Errorf("loader: unsupported i386 relocation type %d at %#x", relocationType, target)
	}
	if err != nil {
		return fmt.Errorf("loader: relocation arithmetic at %#x: %w", target, err)
	}
	return writeRelocationValue(memory, target, value)
}

func resolveSymbol(memory *corecpu.Memory, space *AddressSpace, index uint32) (uint32, error) {
	info := space.Image.Dynamic
	if info == nil || info.SymTab == 0 {
		return 0, fmt.Errorf("symbol table is missing")
	}
	entrySize := info.SymEnt
	if entrySize == 0 {
		entrySize = 16
	}
	if entrySize != 16 {
		return 0, fmt.Errorf("unsupported symbol entry size %d", entrySize)
	}
	if info.SymSz != 0 && index >= info.SymSz/entrySize {
		return 0, fmt.Errorf("symbol index %d is outside table", index)
	}
	entryOffset64 := uint64(info.SymTab) + uint64(index)*uint64(entrySize)
	if entryOffset64 > math.MaxUint32 {
		return 0, fmt.Errorf("symbol address overflows 32-bit space")
	}
	address, err := imageAddress(space.Bias, uint32(entryOffset64))
	if err != nil {
		return 0, err
	}
	var raw [16]byte
	if err := memory.Read(address, raw[:]); err != nil {
		return 0, fmt.Errorf("read symbol at %#x: %w", address, err)
	}
	value := binary.LittleEndian.Uint32(raw[4:8])
	section := binary.LittleEndian.Uint16(raw[14:16])
	if section == shnUndef {
		return 0, fmt.Errorf("undefined symbol %q", symbolName(memory, space, binary.LittleEndian.Uint32(raw[:])))
	}
	if section != shnAbs {
		value, err = add32(space.Bias, value)
		if err != nil {
			return 0, err
		}
	}
	return value, nil
}

func symbolName(memory *corecpu.Memory, space *AddressSpace, offset uint32) string {
	info := space.Image.Dynamic
	if info == nil || info.StrTab == 0 || offset >= info.StrSz {
		return "?"
	}
	address, err := imageAddress(space.Bias, info.StrTab+offset)
	if err != nil {
		return "?"
	}
	name := make([]byte, 128)
	if err := memory.Read(address, name); err != nil {
		return "?"
	}
	for i, b := range name {
		if b == 0 {
			return string(name[:i])
		}
	}
	return "?"
}

func writeRelocationValue(memory *corecpu.Memory, target corecpu.Address, value uint32) error {
	var encoded [4]byte
	binary.LittleEndian.PutUint32(encoded[:], value)
	if err := memory.Write(target, encoded[:]); err != nil {
		return fmt.Errorf("loader: write relocation at %#x: %w", target, err)
	}
	return nil
}

func imageAddress(bias, offset uint32) (corecpu.Address, error) {
	if uint64(bias)+uint64(offset) > math.MaxUint32 {
		return 0, fmt.Errorf("address %#x+%#x overflows 32-bit space", bias, offset)
	}
	return corecpu.Address(bias + offset), nil
}

func add32(left, right uint32) (uint32, error) {
	if uint64(left)+uint64(right) > math.MaxUint32 {
		return 0, fmt.Errorf("addition overflows 32-bit space")
	}
	return left + right, nil
}

func sub32(left, right uint32) (uint32, error) {
	if left < right {
		return 0, fmt.Errorf("subtraction underflows 32-bit space")
	}
	return left - right, nil
}
