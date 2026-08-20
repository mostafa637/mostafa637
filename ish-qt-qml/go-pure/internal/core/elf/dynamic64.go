package elf

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// DynamicInfo64 contains the ELF64 dynamic tags needed by the guest loader.
// Addresses are image-relative virtual addresses; a loader applies its load
// bias before reading or writing them in guest memory.
type DynamicInfo64 struct {
	StrTab    uint64
	StrSz     uint64
	SymTab    uint64
	SymEnt    uint64
	SymSz     uint64
	Hash      uint64
	GNUHash   uint64
	Rel       uint64
	RelSz     uint64
	RelEnt    uint64
	Rela      uint64
	RelaSz    uint64
	RelaEnt   uint64
	JmpRel    uint64
	PltRel    uint64
	PltRelSz  uint64
	PltGot    uint64
	Init      uint64
	Fini      uint64
	Flags     uint64
	Flags1    uint64
	RelCount  uint64
	RelACount uint64

	Needed []string
	SONAME string

	sonameOffset uint64
}

const (
	dtNeeded64   = 1
	dtPltRelSz64 = 2
	dtPltGot64   = 3
	dtHash64     = 4
	dtStrTab64   = 5
	dtSymTab64   = 6
	dtRela64     = 7
	dtRelaSz64   = 8
	dtRelaEnt64  = 9
	dtStrSz64    = 10
	dtSymEnt64   = 11
	dtInit64     = 12
	dtFini64     = 13
	dtSoname64   = 14

	dtRel64       = 17
	dtRelSz64     = 18
	dtRelEnt64    = 19
	dtPltRel64    = 20
	dtJmpRel64    = 23
	dtFlags64     = 30
	dtRelCount64  = 0x6ffffffa
	dtFlags164    = 0x6ffffffb
	dtRelACount64 = 0x6ffffff9
	dtGNUHash64   = 0x6ffffef5
)

func parseDynamic64(r io.ReaderAt, size int64, programs []Segment64) (*DynamicInfo64, error) {
	var dynamic *Segment64
	for index := range programs {
		if programs[index].Type != elf.PT_DYNAMIC {
			continue
		}
		if dynamic != nil {
			return nil, fmt.Errorf("elf64: multiple PT_DYNAMIC segments")
		}
		dynamic = &programs[index]
	}
	if dynamic == nil {
		return nil, nil
	}
	if dynamic.FileSize == 0 || dynamic.FileSize%16 != 0 {
		return nil, fmt.Errorf("elf64: invalid PT_DYNAMIC size")
	}
	if size < 0 || dynamic.Offset > uint64(size) || dynamic.FileSize > uint64(size)-dynamic.Offset || dynamic.FileSize > uint64(maxInt()) {
		return nil, fmt.Errorf("elf64: PT_DYNAMIC exceeds image")
	}
	data := make([]byte, int(dynamic.FileSize))
	if _, err := io.ReadFull(io.NewSectionReader(r, int64(dynamic.Offset), int64(dynamic.FileSize)), data); err != nil {
		return nil, fmt.Errorf("elf64: read PT_DYNAMIC: %w", err)
	}

	info := &DynamicInfo64{}
	var neededOffsets []uint64
	for offset := 0; offset < len(data); offset += 16 {
		tag := int64(binary.LittleEndian.Uint64(data[offset:]))
		value := binary.LittleEndian.Uint64(data[offset+8:])
		if tag == 0 { // DT_NULL
			break
		}
		switch tag {
		case dtNeeded64:
			neededOffsets = append(neededOffsets, value)
		case dtSoname64:
			info.sonameOffset = value
		case dtStrTab64:
			info.StrTab = value
		case dtStrSz64:
			info.StrSz = value
		case dtSymTab64:
			info.SymTab = value
		case dtSymEnt64:
			info.SymEnt = value
		case dtHash64:
			info.Hash = value
		case dtGNUHash64:
			info.GNUHash = value
		case dtRel64:
			info.Rel = value
		case dtRelSz64:
			info.RelSz = value
		case dtRelEnt64:
			info.RelEnt = value
		case dtRela64:
			info.Rela = value
		case dtRelaSz64:
			info.RelaSz = value
		case dtRelaEnt64:
			info.RelaEnt = value
		case dtJmpRel64:
			info.JmpRel = value
		case dtPltRel64:
			info.PltRel = value
		case dtPltRelSz64:
			info.PltRelSz = value
		case dtPltGot64:
			info.PltGot = value
		case dtInit64:
			info.Init = value
		case dtFini64:
			info.Fini = value
		case dtFlags64:
			info.Flags = value
		case dtFlags164:
			info.Flags1 = value
		case dtRelCount64:
			info.RelCount = value
		case dtRelACount64:
			info.RelACount = value
		}
	}
	if info.StrTab == 0 && (len(neededOffsets) > 0 || info.sonameOffset != 0) {
		return nil, fmt.Errorf("elf64: dynamic string table is missing")
	}
	for _, offset := range neededOffsets {
		value, err := dynamicString64(r, size, programs, info.StrTab, info.StrSz, offset)
		if err != nil {
			return nil, fmt.Errorf("elf64: DT_NEEDED string: %w", err)
		}
		info.Needed = append(info.Needed, value)
	}
	if info.sonameOffset != 0 {
		value, err := dynamicString64(r, size, programs, info.StrTab, info.StrSz, info.sonameOffset)
		if err != nil {
			return nil, fmt.Errorf("elf64: DT_SONAME string: %w", err)
		}
		info.SONAME = value
	}
	if info.SymEnt != 0 && info.SymEnt != 24 {
		return nil, fmt.Errorf("elf64: invalid DT_SYMENT %d", info.SymEnt)
	}
	if info.RelaEnt != 0 && info.RelaEnt != 24 {
		return nil, fmt.Errorf("elf64: invalid DT_RELAENT %d", info.RelaEnt)
	}
	if info.RelEnt != 0 && info.RelEnt != 16 {
		return nil, fmt.Errorf("elf64: invalid DT_RELENT %d", info.RelEnt)
	}
	if info.RelaSz != 0 && info.RelaEnt == 0 {
		return nil, fmt.Errorf("elf64: DT_RELASZ without DT_RELAENT")
	}
	if info.RelSz != 0 && info.RelEnt == 0 {
		return nil, fmt.Errorf("elf64: DT_RELSZ without DT_RELENT")
	}
	if info.RelaSz > math.MaxInt64 || info.RelSz > math.MaxInt64 {
		return nil, fmt.Errorf("elf64: dynamic table is too large")
	}
	return info, nil
}

func dynamicString64(r io.ReaderAt, size int64, programs []Segment64, strtab, strsz, offset uint64) (string, error) {
	if strtab == 0 || offset >= strsz {
		return "", fmt.Errorf("invalid string-table offset %#x", offset)
	}
	for _, program := range programs {
		if !program.Loadable() || strtab < program.Vaddr {
			continue
		}
		delta := strtab - program.Vaddr
		if delta >= program.FileSize || offset > math.MaxUint64-delta {
			continue
		}
		fileOffset := program.Offset + delta + offset
		available := strsz - offset
		if program.Offset > math.MaxUint64-delta-offset || fileOffset > uint64(size) || available > uint64(size)-fileOffset {
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
