package elf

import (
	"bytes"
	"debug/elf"
	"fmt"
	"io"
)

// DynamicInfo contains the ELF32 dynamic tags needed by the guest loader.
// Addresses are image-relative virtual addresses; a loader applies its load
// bias before reading or writing them in guest memory.
type DynamicInfo struct {
	StrTab       uint32
	StrSz        uint32
	SymTab       uint32
	SymEnt       uint32
	SymSz        uint32
	Hash         uint32
	GNUHash      uint32
	Rel          uint32
	RelSz        uint32
	RelEnt       uint32
	Rela         uint32
	RelaSz       uint32
	RelaEnt      uint32
	JmpRel       uint32
	PltRel       uint32
	PltRelSz     uint32
	PltGot       uint32
	Init         uint32
	Fini         uint32
	TLS          uint32
	TLSSize      uint32
	Flags        uint32
	RelCount     uint32
	Needed       []string
	SONAME       string
	sonameOffset uint32
}

func parseDynamic(r io.ReaderAt, size int64, programs []*elf.Prog) (*DynamicInfo, error) {
	var dynamic *elf.Prog
	for _, program := range programs {
		if program.Type == elf.PT_DYNAMIC {
			if dynamic != nil {
				return nil, fmt.Errorf("elf32: multiple PT_DYNAMIC segments")
			}
			dynamic = program
		}
	}
	if dynamic == nil {
		return nil, nil
	}
	if dynamic.Filesz == 0 || dynamic.Filesz%8 != 0 {
		return nil, fmt.Errorf("elf32: invalid PT_DYNAMIC size")
	}
	if dynamic.Off > uint64(size) || dynamic.Filesz > uint64(size)-dynamic.Off || dynamic.Filesz > uint64(maxInt()) {
		return nil, fmt.Errorf("elf32: PT_DYNAMIC exceeds image")
	}
	data := make([]byte, int(dynamic.Filesz))
	if _, err := io.ReadFull(io.NewSectionReader(r, int64(dynamic.Off), int64(dynamic.Filesz)), data); err != nil {
		return nil, fmt.Errorf("elf32: read PT_DYNAMIC: %w", err)
	}

	info := &DynamicInfo{}
	var neededOffsets []uint32
	for offset := 0; offset < len(data); offset += 8 {
		tag := elf.DynTag(binary32(data[offset:]))
		value := binary32(data[offset+4:])
		if tag == elf.DT_NULL {
			break
		}
		switch tag {
		case elf.DT_NEEDED:
			neededOffsets = append(neededOffsets, value)
		case elf.DT_SONAME:
			// Keep the offset until the string table has been decoded.
			info.sonameOffset = value
		case elf.DT_STRTAB:
			info.StrTab = value
		case elf.DT_STRSZ:
			info.StrSz = value
		case elf.DT_SYMTAB:
			info.SymTab = value
		case elf.DT_SYMENT:
			info.SymEnt = value
		case elf.DT_HASH:
			info.Hash = value
		case elf.DT_GNU_HASH:
			info.GNUHash = value
		case elf.DT_REL:
			info.Rel = value
		case elf.DT_RELSZ:
			info.RelSz = value
		case elf.DT_RELENT:
			info.RelEnt = value
		case elf.DT_RELA:
			info.Rela = value
		case elf.DT_RELASZ:
			info.RelaSz = value
		case elf.DT_RELAENT:
			info.RelaEnt = value
		case elf.DT_JMPREL:
			info.JmpRel = value
		case elf.DT_PLTREL:
			info.PltRel = value
		case elf.DT_PLTRELSZ:
			info.PltRelSz = value
		case elf.DT_PLTGOT:
			info.PltGot = value
		case elf.DT_INIT:
			info.Init = value
		case elf.DT_FINI:
			info.Fini = value
		case elf.DT_FLAGS:
			info.Flags = value
		case elf.DT_RELCOUNT:
			info.RelCount = value
		}
	}
	if info.StrTab == 0 && (len(neededOffsets) > 0 || info.sonameOffset != 0) {
		return nil, fmt.Errorf("elf32: dynamic string table is missing")
	}
	for _, offset := range neededOffsets {
		value, err := dynamicString(r, size, programs, info.StrTab, info.StrSz, offset)
		if err != nil {
			return nil, fmt.Errorf("elf32: DT_NEEDED string: %w", err)
		}
		info.Needed = append(info.Needed, value)
	}
	if info.sonameOffset != 0 {
		value, err := dynamicString(r, size, programs, info.StrTab, info.StrSz, info.sonameOffset)
		if err != nil {
			return nil, fmt.Errorf("elf32: DT_SONAME string: %w", err)
		}
		info.SONAME = value
	}
	return info, nil
}

func dynamicString(r io.ReaderAt, size int64, programs []*elf.Prog, strtab, strsz, offset uint32) (string, error) {
	if strtab == 0 || offset >= strsz {
		return "", fmt.Errorf("invalid string-table offset %#x", offset)
	}
	for _, program := range programs {
		if program.Type != elf.PT_LOAD || strtab < uint32(program.Vaddr) || strtab >= uint32(program.Vaddr+program.Filesz) {
			continue
		}
		fileOffset := uint64(program.Off) + uint64(strtab-uint32(program.Vaddr)) + uint64(offset)
		available := uint64(strsz - offset)
		if fileOffset > uint64(size) || available > uint64(size)-fileOffset {
			return "", fmt.Errorf("string table exceeds image")
		}
		if available > uint64(maxInt()) {
			return "", fmt.Errorf("string table is too large")
		}
		data := make([]byte, int(available))
		if _, err := io.ReadFull(io.NewSectionReader(r, int64(fileOffset), int64(available)), data); err != nil {
			return "", err
		}
		if index := bytes.IndexByte(data, 0); index >= 0 {
			return string(data[:index]), nil
		}
		return "", fmt.Errorf("unterminated string")
	}
	return "", fmt.Errorf("string table is not in a PT_LOAD file range")
}

func binary32(data []byte) uint32 {
	return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
}

func maxInt() int {
	return int(^uint(0) >> 1)
}
