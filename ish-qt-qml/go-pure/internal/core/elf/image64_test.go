package elf_test

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"reflect"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/loader"
)

func minimalELF64(t *testing.T) []byte {
	t.Helper()
	const fileOffset = 0x1000
	data := make([]byte, fileOffset+4)
	copy(data[fileOffset:], []byte{0x90, 0x0f, 0x05, 0xc3})
	ident := data[:16]
	copy(ident[:4], []byte{0x7f, 'E', 'L', 'F'})
	ident[4] = byte(elf.ELFCLASS64)
	ident[5] = byte(elf.ELFDATA2LSB)
	ident[6] = byte(elf.EV_CURRENT)
	binary.LittleEndian.PutUint16(data[16:18], uint16(elf.ET_EXEC))
	binary.LittleEndian.PutUint16(data[18:20], uint16(elf.EM_X86_64))
	binary.LittleEndian.PutUint32(data[20:24], uint32(elf.EV_CURRENT))
	binary.LittleEndian.PutUint64(data[24:32], 0x400000)
	binary.LittleEndian.PutUint64(data[32:40], 64)
	binary.LittleEndian.PutUint16(data[52:54], 64)
	binary.LittleEndian.PutUint16(data[54:56], 56)
	binary.LittleEndian.PutUint16(data[56:58], 1)
	// elf.PT_LOAD: R|X, file offset 0x1000, virtual address 0x400000.
	ph := data[64 : 64+56]
	binary.LittleEndian.PutUint32(ph[0:4], uint32(elf.PT_LOAD))
	binary.LittleEndian.PutUint32(ph[4:8], uint32(elf.PF_R|elf.PF_X))
	binary.LittleEndian.PutUint64(ph[8:16], fileOffset)
	binary.LittleEndian.PutUint64(ph[16:24], 0x400000)
	binary.LittleEndian.PutUint64(ph[32:40], 4)
	binary.LittleEndian.PutUint64(ph[40:48], uint64(coreelf.PageSize))
	binary.LittleEndian.PutUint64(ph[48:56], uint64(coreelf.PageSize))
	return data
}

func TestParse64AndLoad64(t *testing.T) {
	data := minimalELF64(t)
	image, err := coreelf.Parse64(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if image.Header.Machine != elf.EM_X86_64 || image.Header.Entry != 0x400000 {
		t.Fatalf("header=%+v", image.Header)
	}
	memory := corecpu.NewMemory64()
	space, err := loader.Load64(bytes.NewReader(data), int64(len(data)), image, memory, 0)
	if err != nil {
		t.Fatal(err)
	}
	if space.Entry != 0x400000 || space.Start != 0x400000 || space.End != 0x401000 {
		t.Fatalf("space=%+v", space)
	}
	var code [4]byte
	if err := memory.Read(space.Entry, code[:]); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(code[:], data[0x1000:0x1004]) {
		t.Fatalf("code=%x", code)
	}
	flags, ok := memory.MappingFlags(corecpu.Page64(0x400000 >> corecpu.Page64Bits))
	if !ok || flags&(corecpu.PRead|corecpu.PExec) != corecpu.PRead|corecpu.PExec || flags&corecpu.PWrite != 0 {
		t.Fatalf("flags=%#x mapped=%v", flags, ok)
	}
}

func dynamicELF64(t *testing.T) []byte {
	t.Helper()
	const (
		programOffset = 64
		programSize   = 56
		loadVaddr     = 0x400000
		dynamicOffset = 0x1000
		stringOffset  = 0x1200
		interpOffset  = 0x1300
		tlsOffset     = 0x1400
		fileSize      = 0x1500
	)
	data := make([]byte, fileSize)
	ident := data[:16]
	copy(ident[:4], []byte{0x7f, 'E', 'L', 'F'})
	ident[4] = byte(elf.ELFCLASS64)
	ident[5] = byte(elf.ELFDATA2LSB)
	ident[6] = byte(elf.EV_CURRENT)
	binary.LittleEndian.PutUint16(data[16:18], uint16(elf.ET_DYN))
	binary.LittleEndian.PutUint16(data[18:20], uint16(elf.EM_X86_64))
	binary.LittleEndian.PutUint32(data[20:24], uint32(elf.EV_CURRENT))
	binary.LittleEndian.PutUint64(data[24:32], loadVaddr+0x1500)
	binary.LittleEndian.PutUint64(data[32:40], programOffset)
	binary.LittleEndian.PutUint16(data[52:54], 64)
	binary.LittleEndian.PutUint16(data[54:56], programSize)
	binary.LittleEndian.PutUint16(data[56:58], 4)

	putProgram := func(index int, typ elf.ProgType, flags elf.ProgFlag, offset, vaddr, filesz, memsz, align uint64) {
		ph := data[programOffset+index*programSize : programOffset+(index+1)*programSize]
		binary.LittleEndian.PutUint32(ph[0:4], uint32(typ))
		binary.LittleEndian.PutUint32(ph[4:8], uint32(flags))
		binary.LittleEndian.PutUint64(ph[8:16], offset)
		binary.LittleEndian.PutUint64(ph[16:24], vaddr)
		binary.LittleEndian.PutUint64(ph[32:40], filesz)
		binary.LittleEndian.PutUint64(ph[40:48], memsz)
		binary.LittleEndian.PutUint64(ph[48:56], align)
	}
	putProgram(0, elf.PT_LOAD, elf.PF_R|elf.PF_X, 0, loadVaddr, fileSize, fileSize+uint64(coreelf.PageSize), uint64(coreelf.PageSize))
	interp := []byte("/lib64/ld-linux-x86-64.so.2\x00")
	copy(data[interpOffset:], interp)
	putProgram(1, elf.PT_INTERP, elf.PF_R, interpOffset, loadVaddr+interpOffset, uint64(len(interp)), uint64(len(interp)), 1)
	tlsData := []byte{1, 2, 3, 4}
	copy(data[tlsOffset:], tlsData)
	putProgram(2, elf.PT_TLS, elf.PF_R, tlsOffset, loadVaddr+tlsOffset, uint64(len(tlsData)), 8, 8)

	strings := []byte("\x00libc.so.6\x00libm.so.6\x00main.so\x00")
	copy(data[stringOffset:], strings)
	libcOffset := uint64(bytes.Index(strings, []byte("libc.so.6")))
	libmOffset := uint64(bytes.Index(strings, []byte("libm.so.6")))
	sonameOffset := uint64(bytes.Index(strings, []byte("main.so")))
	dynamic := data[dynamicOffset : dynamicOffset+11*16]
	putDynamic := func(index int, tag, value uint64) {
		binary.LittleEndian.PutUint64(dynamic[index*16:], tag)
		binary.LittleEndian.PutUint64(dynamic[index*16+8:], value)
	}
	putDynamic(0, 5, loadVaddr+stringOffset) // DT_STRTAB
	putDynamic(1, 10, uint64(len(strings)))  // DT_STRSZ
	putDynamic(2, 1, libcOffset)             // DT_NEEDED
	putDynamic(3, 1, libmOffset)             // DT_NEEDED
	putDynamic(4, 14, sonameOffset)          // DT_SONAME
	putDynamic(5, 6, loadVaddr+stringOffset+0x40)
	putDynamic(6, 11, 24) // DT_SYMENT
	putDynamic(7, 7, loadVaddr+stringOffset+0x80)
	putDynamic(8, 8, 24) // DT_RELASZ
	putDynamic(9, 9, 24) // DT_RELAENT
	putDynamic(10, 0, 0) // DT_NULL
	putProgram(3, elf.PT_DYNAMIC, elf.PF_R, dynamicOffset, loadVaddr+dynamicOffset, uint64(len(dynamic)), uint64(len(dynamic)), 8)
	return data
}

func TestParse64DynamicInterpreterAndTLS(t *testing.T) {
	data := dynamicELF64(t)
	image, err := coreelf.Parse64(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if image.Interp != "/lib64/ld-linux-x86-64.so.2" {
		t.Fatalf("interp=%q", image.Interp)
	}
	if image.Dynamic == nil {
		t.Fatal("dynamic metadata is nil")
	}
	if got, want := image.Dynamic.Needed, []string{"libc.so.6", "libm.so.6"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("needed=%#v want %#v", got, want)
	}
	if image.Dynamic.SONAME != "main.so" || image.Dynamic.SymEnt != 24 || image.Dynamic.RelaSz != 24 || image.Dynamic.RelaEnt != 24 {
		t.Fatalf("dynamic=%+v", image.Dynamic)
	}
	if image.TLS == nil || image.TLS.FileSize != 4 || image.TLS.MemSize != 8 {
		t.Fatalf("tls=%+v", image.TLS)
	}
}

func TestParse64DynamicSymbolsFromHashWithoutSections(t *testing.T) {
	data := dynamicELF64(t)
	const (
		dynamicOffset = 0x1000
		hashOffset    = 0x1100
		symtabOffset  = 0x1140
		loadVaddr     = 0x400000
		stringOffset  = 0x1200
	)
	// Expand PT_DYNAMIC by one entry: replace the fixture's DT_NULL with
	// DT_HASH and place a new DT_NULL immediately after it.
	ph := data[64+3*56 : 64+4*56]
	binary.LittleEndian.PutUint64(ph[32:40], 12*16)
	binary.LittleEndian.PutUint64(ph[40:48], 12*16)
	dynamic := data[dynamicOffset : dynamicOffset+12*16]
	binary.LittleEndian.PutUint64(dynamic[10*16:], 4) // DT_HASH
	binary.LittleEndian.PutUint64(dynamic[10*16+8:], loadVaddr+hashOffset)
	binary.LittleEndian.PutUint64(dynamic[11*16:], 0) // DT_NULL
	binary.LittleEndian.PutUint64(dynamic[11*16+8:], 0)
	binary.LittleEndian.PutUint64(dynamic[5*16+8:], loadVaddr+symtabOffset)

	// DT_HASH: one bucket and two symbols (the mandatory undefined symbol plus
	// one defined symbol). The chain values are not needed for the count.
	binary.LittleEndian.PutUint32(data[hashOffset:hashOffset+4], 1)
	binary.LittleEndian.PutUint32(data[hashOffset+4:hashOffset+8], 2)
	nameOffset := uint32(bytes.Index(data[stringOffset:], []byte("main.so")))
	if nameOffset == ^uint32(0) {
		t.Fatal("main.so is missing from fixture string table")
	}
	// Symbol 0 remains zeroed. Write symbol 1 as a defined object symbol.
	symbol := data[symtabOffset+24 : symtabOffset+48]
	binary.LittleEndian.PutUint32(symbol[0:4], nameOffset)
	symbol[4] = 0x12 // STB_GLOBAL | STT_FUNC
	binary.LittleEndian.PutUint16(symbol[6:8], 1)
	binary.LittleEndian.PutUint64(symbol[8:16], 0x1600)
	binary.LittleEndian.PutUint64(symbol[16:24], 4)

	image, err := coreelf.Parse64(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(image.DynamicSymbols) != 2 {
		t.Fatalf("dynamic symbol count=%d, want 2", len(image.DynamicSymbols))
	}
	if image.DynamicSymbols[1].Name != "main.so" || image.DynamicSymbols[1].Value != 0x1600 || image.DynamicSymbols[1].Section != 1 {
		t.Fatalf("symbol=%+v", image.DynamicSymbols[1])
	}
}
